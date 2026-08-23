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

const (
	actionEntryTimeSize   = 20
	actionEntryTimeOffset = entrySize - actionEntryTimeSize - 1
)

// Stats describes cache state derived from a full directory scan.
// Oldest and Newest are add times from action entries (-a); data entries (-d) do not store add times.
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
	LastUse time.Time // file (or directory for executable) last use time (mtime)
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
	before, freed, _ = c.trimForce(cutoff, maxSize, false, l)
	return
}

// TrimForceWithStats is like [DiskCache.TrimForce], but also returns statistics
// derived from the same cache scan used for trimming.
func (c *DiskCache) TrimForceWithStats(
	cutoff *time.Time,
	maxSize *int64,
	l *slog.Logger,
) (before, freed int64, stats *Stats) {
	return c.trimForce(cutoff, maxSize, true, l)
}

func (c *DiskCache) trimForce(
	cutoff *time.Time,
	maxSize *int64,
	wantStats bool,
	l *slog.Logger,
) (before, freed int64, stats *Stats) {
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

	if cutoff == nil && maxSize == nil && !wantStats {
		return
	}

	files, bytes := c.read(l)
	if cutoff != nil || maxSize != nil {
		before = bytes
	}

	if cutoff == nil && maxSize != nil && before <= *maxSize {
		if wantStats {
			stats = c.stats(files, l)
		}
		return
	}

	if cutoff != nil {
		for i, fi := range files {
			if fi.Name == "" || !fi.LastUse.Before(*cutoff) {
				continue
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

	if sortForMaxSize(files, before-freed, maxSize) {
		for i, fi := range files {
			if before-freed <= *maxSize {
				break
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
			files[i] = fileInfo{}
		}
	}

	if wantStats {
		stats = c.stats(files, l)
	}

	return
}

func sortForMaxSize(files []fileInfo, size int64, maxSize *int64) bool {
	if maxSize == nil || size <= *maxSize {
		return false
	}

	slices.SortFunc(files, func(f1, f2 fileInfo) int {
		return f1.LastUse.Compare(f2.LastUse)
	})
	return true
}

// Stats scans the cache directory and returns aggregate statistics.
func (c *DiskCache) Stats(l *slog.Logger) *Stats {
	stats := new(Stats)
	c.walk(l, func(fi fileInfo) {
		c.addStats(stats, fi, l)
	})
	return stats
}

func (c *DiskCache) stats(files []fileInfo, l *slog.Logger) *Stats {
	stats := new(Stats)
	for _, fi := range files {
		if fi.Name == "" {
			continue
		}

		c.addStats(stats, fi, l)
	}
	return stats
}

func (c *DiskCache) addStats(stats *Stats, fi fileInfo, l *slog.Logger) {
	stats.Entries++
	stats.Bytes += fi.Size

	if stats.LeastRecentlyUsed == nil || fi.LastUse.Before(*stats.LeastRecentlyUsed) {
		stats.LeastRecentlyUsed = &fi.LastUse
	}
	if stats.MostRecentlyUsed == nil || fi.LastUse.After(*stats.MostRecentlyUsed) {
		stats.MostRecentlyUsed = &fi.LastUse
	}

	if !strings.HasSuffix(fi.Name, "-a") {
		return
	}

	path := filepath.Join(c.dir, fi.Name[:2], fi.Name)
	entry, err := os.ReadFile(path)
	if err != nil {
		l.Debug("Failed to read action entry", slog.String("name", path), slog.String("error", err.Error()))
		return
	}
	if len(entry) != entrySize {
		l.Debug("Invalid action entry", slog.String("name", path))
		return
	}

	b := entry[actionEntryTimeOffset : actionEntryTimeOffset+actionEntryTimeSize]
	ns, err := strconv.ParseInt(strings.TrimLeft(string(b), " "), 10, 64)
	if err != nil {
		l.Debug("Failed to parse action entry time", slog.String("name", path), slog.String("error", err.Error()))
		return
	}
	if ns < 0 {
		l.Debug("Invalid action entry time", slog.String("name", path), slog.Int64("timestamp", ns))
		return
	}

	added := time.Unix(0, ns)
	if stats.Oldest == nil || added.Before(*stats.Oldest) {
		stats.Oldest = &added
	}
	if stats.Newest == nil || added.After(*stats.Newest) {
		stats.Newest = &added
	}
}

// read reads the entire cache directory.
func (c *DiskCache) read(l *slog.Logger) (files []fileInfo, before int64) {
	files = make([]fileInfo, 0, 256)
	c.walk(l, func(fi fileInfo) {
		files = append(files, fi)
		before += fi.Size
	})
	return
}

// walk calls yield for every valid cache entry in the cache directory.
func (c *DiskCache) walk(l *slog.Logger, yield func(fileInfo)) {
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
			lastUse := fi.ModTime()

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

			yield(fileInfo{
				LastUse: lastUse,
				Name:    name,
				Size:    size,
			})
		}
	}
}
