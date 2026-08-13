// This file provides additional functionality for this package,
// not available in the Go repository.

package cache

import (
	"bytes"
	"errors"
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

// Stats describes cache state derived from a full directory scan.
// Oldest and Newest are add times from action entries. Data entries do not store add times.
// LeastRecentlyUsed and MostRecentlyUsed are approximate last-use times from filesystem mtimes.
type Stats struct {
	Entries           int
	Bytes             int64
	Oldest            *time.Time
	Newest            *time.Time
	LeastRecentlyUsed *time.Time
	MostRecentlyUsed  *time.Time
}

// fileInfo represents information about a file or directory with executable in the cache.
// The order of fields is weird to make struct smaller.
type fileInfo struct {
	ModTime time.Time // file (or directory for executable) last use time
	Name    string    // file name, or directory name for executable
	Size    int64     // file size, or executable size
}

// TrimExtra removes cache entries (starting from least recently used),
// enforcing both cutoff date and max cache size, if set.
// Like [Trim], it honors the last trim time, doing nothing if the last trim was recent.
func (c *DiskCache) TrimExtra(cutoff *time.Time, maxSize *int64, l *slog.Logger) (before, freed int64) {
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

	before, freed = c.TrimForce(cutoff, maxSize, l)
	return
}

// TrimForce removes cache entries (starting from least recently used),
// enforcing both cutoff date and max cache size, if set.
// It ignores the last trim time, but updates it.
//
// Passed logger is used for debug messages only.
func (c *DiskCache) TrimForce(cutoff *time.Time, maxSize *int64, l *slog.Logger) (before, freed int64) {
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

	if cutoff == nil && maxSize == nil {
		return
	}

	files, before := c.read(l)

	if cutoff == nil && before <= *maxSize {
		return
	}

	slices.SortFunc(files, func(f1, f2 fileInfo) int {
		return f1.ModTime.Compare(f2.ModTime)
	})

	if cutoff != nil {
		for i, fi := range files {
			if !fi.ModTime.Before(*cutoff) {
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

	if maxSize != nil {
		for _, fi := range files {
			if before-freed <= *maxSize {
				return
			}

			if fi.Name == "" {
				continue
			}

			p := filepath.Join(c.dir, fi.Name[:2], fi.Name)
			if err := os.RemoveAll(p); err != nil {
				l.Debug("Failed to remove entry by max size", slog.String("name", p), slog.String("error", err.Error()))
				continue
			}

			l.Debug("Removed entry by max size", slog.String("name", p))
			freed += fi.Size
		}
	}

	return
}

// Stats scans the cache directory and returns aggregate statistics.
func (c *DiskCache) Stats(l *slog.Logger) *Stats {
	files, bytes := c.read(l)
	stats := &Stats{
		Entries: len(files),
		Bytes:   bytes,
	}

	for _, fi := range files {
		modTime := fi.ModTime
		if stats.LeastRecentlyUsed == nil || modTime.Before(*stats.LeastRecentlyUsed) {
			stats.LeastRecentlyUsed = new(modTime)
		}
		if stats.MostRecentlyUsed == nil || modTime.After(*stats.MostRecentlyUsed) {
			stats.MostRecentlyUsed = new(modTime)
		}

		if !strings.HasSuffix(fi.Name, "-a") {
			continue
		}

		path := filepath.Join(c.dir, fi.Name[:2], fi.Name)
		entry, err := os.ReadFile(path)
		if err != nil {
			l.Debug("Failed to read action entry", slog.String("name", path), slog.String("error", err.Error()))
			continue
		}
		if !validEntry(entry) {
			l.Debug("Invalid action entry", slog.String("name", path))
			continue
		}

		added, err := parseEntryTime(entry[entryTimeOffset : entryTimeOffset+entryTimeSize])
		if err != nil {
			l.Debug("Failed to parse action entry time", slog.String("name", path), slog.String("error", err.Error()))
			continue
		}
		if stats.Oldest == nil || added.Before(*stats.Oldest) {
			stats.Oldest = new(added)
		}
		if stats.Newest == nil || added.After(*stats.Newest) {
			stats.Newest = new(added)
		}
	}

	return stats
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

func parseEntryTime(b []byte) (time.Time, error) {
	ns, err := strconv.ParseInt(strings.TrimLeft(string(b), " "), 10, 64)
	if err != nil {
		return time.Time{}, err
	}
	if ns < 0 {
		return time.Time{}, errors.New("negative timestamp")
	}

	return time.Unix(0, ns), nil
}

func validEntry(entry []byte) bool {
	return len(entry) == entrySize &&
		entry[0] == 'v' && entry[1] == '1' && entry[2] == ' ' &&
		entry[3+hexSize] == ' ' &&
		entry[3+hexSize+1+hexSize] == ' ' &&
		entry[3+hexSize+1+hexSize+1+20] == ' ' &&
		entry[entrySize-1] == '\n'
}
