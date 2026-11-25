// This file provides additional functionality for this package,
// not available in the Go repository.

package cache

import (
	"bytes"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/AlekSi/hardcache/internal/go/lockedfile"
)

// EntryNotFoundError is exported for use in other packages.
type EntryNotFoundError = entryNotFoundError

// fileInfo represents information about a file or directory with executable in the cache.
// The order of fields is weird to make struct smaller.
type fileInfo struct {
	ModTime time.Time // file modification time, or directory modification time
	Name    string    // file name, or directory name for executable
	Size    int64     // file size, or executable size
}

// TrimLimits contains limits enforced by [DiskCache.TrimExtra] and [DiskCache.TrimForce].
// Nil fields are not enforced.
//
// Entries are ordered by their modification time, which Go updates when an entry is used,
// but at most once per mtimeInterval (1 hour).
type TrimLimits struct {
	// Cutoff removes all entries last used before this time.
	Cutoff *time.Time

	// MaxSize removes entries, starting from least recently used,
	// until the cache size is not larger than this value.
	// Hot entries (see KeepAfter) are kept, so the cache may remain larger.
	MaxSize *int64

	// KeepAfter makes entries last used at or after this time hot:
	// they are not removed by MaxSize (but still could be removed by Cutoff or HardMaxSize).
	KeepAfter *time.Time

	// HardMaxSize removes entries, starting from least recently used, hot ones included,
	// until the cache size is not larger than this value.
	HardMaxSize *int64
}

// TrimExtra removes cache entries (starting from least recently used),
// enforcing the given limits.
// Like [Trim], it honors the last trim time, doing nothing if the last trim was recent.
//
// See [DiskCache.TrimForce] for the returned values.
func (c *DiskCache) TrimExtra(limits TrimLimits, l *slog.Logger) (before, freed, surge int64) {
	// see DiskCache.Trim
	if data, err := lockedfile.Read(filepath.Join(c.dir, "trim.txt")); err == nil {
		if t, err := strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64); err == nil {
			lastTrim := time.Unix(t, 0)
			if d := c.now().Sub(lastTrim); d < trimInterval && d > -mtimeInterval {
				before = -1
				return
			}
		}
	}

	before, freed, surge = c.TrimForce(limits, l)
	return
}

// TrimForce removes cache entries (starting from least recently used),
// enforcing the given limits.
// It ignores the last trim time, but updates it.
//
// It returns the cache size before trimming (or -1 if no limits are set),
// the number of freed bytes,
// and the number of bytes the cache remains above MaxSize because hot entries were kept.
//
// Passed logger is used for debug messages only.
func (c *DiskCache) TrimForce(limits TrimLimits, l *slog.Logger) (before, freed, surge int64) {
	before = -1
	now := c.now()

	defer func() {
		var b bytes.Buffer
		fmt.Fprintf(&b, "%d", now.Unix())
		if err := lockedfile.Write(filepath.Join(c.dir, "trim.txt"), &b, 0o666); err != nil {
			l.Debug("Failed to write trim.txt", slog.String("error", err.Error()))
			return
		}

		l.Debug(
			"trim.txt updated",
			slog.Int64("before_bytes", before),
			slog.Int64("after_bytes", before-freed),
			slog.Int64("freed_bytes", freed),
		)
	}()

	if limits.Cutoff == nil && limits.MaxSize == nil && limits.HardMaxSize == nil {
		return
	}

	files, before := c.read(l)

	// over returns true if the cache is currently larger than the given limit.
	over := func(limit *int64) bool {
		return limit != nil && before-freed > *limit
	}

	if limits.Cutoff == nil && !over(limits.MaxSize) && !over(limits.HardMaxSize) {
		return
	}

	slices.SortFunc(files, func(f1, f2 fileInfo) int {
		return f1.ModTime.Compare(f2.ModTime)
	})

	// hot returns true if the given entry should not be removed by max size.
	hot := func(fi fileInfo) bool {
		return limits.KeepAfter != nil && !fi.ModTime.Before(*limits.KeepAfter)
	}

	if limits.Cutoff != nil {
		for i, fi := range files {
			if !fi.ModTime.Before(*limits.Cutoff) {
				break
			}

			p := filepath.Join(c.dir, fi.Name[:2], fi.Name)
			if err := os.RemoveAll(p); err != nil {
				l.Debug("Failed to remove entry by cutoff", slog.String("name", p), slog.String("error", err.Error()))
				continue
			}

			l.Debug("Removed entry by cutoff", slog.String("name", p))
			freed += fi.Size
			files[i] = fileInfo{}
		}
	}

	var keptHot bool

	if limits.MaxSize != nil {
		for i, fi := range files {
			if !over(limits.MaxSize) {
				break
			}

			if fi.Name == "" {
				continue
			}

			// entries are sorted, so all remaining entries are hot too
			if hot(fi) {
				keptHot = true
				break
			}

			p := filepath.Join(c.dir, fi.Name[:2], fi.Name)
			if err := os.RemoveAll(p); err != nil {
				l.Debug("Failed to remove entry by max size", slog.String("name", p), slog.String("error", err.Error()))
				continue
			}

			l.Debug("Removed entry by max size", slog.String("name", p))
			freed += fi.Size
			files[i] = fileInfo{}
		}
	}

	if limits.HardMaxSize != nil {
		for i, fi := range files {
			if !over(limits.HardMaxSize) {
				break
			}

			if fi.Name == "" {
				continue
			}

			p := filepath.Join(c.dir, fi.Name[:2], fi.Name)
			if err := os.RemoveAll(p); err != nil {
				l.Debug("Failed to remove entry by hard max size", slog.String("name", p), slog.String("error", err.Error()))
				continue
			}

			if hot(fi) {
				l.Debug("Removed hot entry by hard max size", slog.String("name", p))
			} else {
				l.Debug("Removed entry by hard max size", slog.String("name", p))
			}

			freed += fi.Size
			files[i] = fileInfo{}
		}
	}

	if keptHot && over(limits.MaxSize) {
		surge = before - freed - *limits.MaxSize
	}

	return
}

// read reads the entire cache directory.
func (c *DiskCache) read(l *slog.Logger) (files []fileInfo, before int64) {
	files = make([]fileInfo, 0, 256)

	for i := range 256 {
		subdir := filepath.Join(c.dir, fmt.Sprintf("%02x", i))

		f, err := os.Open(subdir)
		if err != nil {
			l.Debug("Failed to open subdir", slog.String("name", subdir), slog.String("error", err.Error()))
			continue
		}

		fis, err := f.Readdir(-1)
		_ = f.Close()
		if err != nil {
			l.Debug("Failed to read subdir", slog.String("name", subdir), slog.String("error", err.Error()))
			continue
		}

		for _, fi := range fis {
			name := fi.Name()

			if !strings.HasSuffix(name, "-a") && !strings.HasSuffix(name, "-d") {
				continue
			}

			size := fi.Size()
			modTime := fi.ModTime()

			if fi.IsDir() {
				entrs, err := os.ReadDir(filepath.Join(subdir, name))
				if err != nil {
					l.Debug("Failed to read executable subdir", slog.String("name", name), slog.String("error", err.Error()))
					continue
				}

				if len(entrs) != 1 {
					l.Debug("Unexpected executable subdir", slog.String("name", name), slog.Int("entries", len(entrs)))
					continue
				}

				fi, err = entrs[0].Info()
				if err != nil {
					l.Debug("Failed to get executable info", slog.String("name", name), slog.String("error", err.Error()))
					continue
				}

				size = fi.Size()
			}

			before += size
			files = append(files, fileInfo{
				ModTime: modTime,
				Name:    name,
				Size:    size,
			})
		}
	}

	return
}
