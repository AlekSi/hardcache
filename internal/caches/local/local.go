// Package local provides local Go build cache.
package local

import (
	"io"
	"log/slog"
	"time"

	"github.com/AlekSi/lazyerrors"

	"github.com/AlekSi/hardcache/internal/go/cache"
)

// Cache represents a local Go build cache, compatible with a built-in one.
// It provides more configuration options for trimming.
type Cache struct {
	dc      *cache.DiskCache
	cutoff  *time.Time
	maxSize *int64
	l       *slog.Logger
}

// New creates a new [Cache].
func New(dir string, cutoff *time.Time, maxSize *int64, l *slog.Logger) (*Cache, error) {
	dc, err := cache.Open(dir)
	if err != nil {
		return nil, lazyerrors.Error(err)
	}

	return &Cache{
		dc:      dc,
		cutoff:  cutoff,
		maxSize: maxSize,
		l:       l,
	}, nil
}

// Get implements [cache.Cache].
func (c *Cache) Get(id cache.ActionID) (cache.Entry, error) {
	return c.dc.Get(id)
}

// Put implements [cache.Cache].
func (c *Cache) Put(id cache.ActionID, rs io.ReadSeeker) (cache.OutputID, int64, error) {
	return c.dc.Put(id, rs)
}

// Close implements [cache.Cache].
func (c *Cache) Close() error {
	c.dc.TrimExtra(c.cutoff, c.maxSize, c.l)
	return nil
}

// OutputFile implements [cache.Cache].
func (c *Cache) OutputFile(id cache.OutputID) string {
	return c.dc.OutputFile(id)
}

// FuzzDir implements [cache.Cache].
func (c *Cache) FuzzDir() string {
	return c.dc.FuzzDir()
}

// TrimForce removes cache entries (starting from least recently used),
// enforcing both cutoff date and max cache size, if set.
// It ignores the last trim time, but updates it.
func (c *Cache) TrimForce() (before, freed int64) {
	return c.dc.TrimForce(c.cutoff, c.maxSize, c.l)
}

// Stats returns current local cache statistics.
func (c *Cache) Stats() *cache.Stats {
	s := c.dc.Stats(c.l)
	return &cache.Stats{
		Entries:           s.Entries,
		Bytes:             s.Bytes,
		Oldest:            s.Oldest,
		Newest:            s.Newest,
		LeastRecentlyUsed: s.LeastRecentlyUsed,
		MostRecentlyUsed:  s.MostRecentlyUsed,
	}
}

// check interfaces
var (
	_ cache.Cache = (*Cache)(nil)
)
