// Package local provides local Go build cache.
package local

import (
	"io"
	"log/slog"

	"github.com/AlekSi/lazyerrors"

	"github.com/AlekSi/hardcache/internal/go/cache"
)

// TrimLimits contains limits enforced by trimming.
type TrimLimits = cache.TrimLimits

// Cache represents a local Go build cache, compatible with a built-in one.
// It provides more configuration options for trimming.
type Cache struct {
	dc     *cache.DiskCache
	limits TrimLimits
	l      *slog.Logger
}

// New creates a new [Cache].
func New(dir string, limits TrimLimits, l *slog.Logger) (*Cache, error) {
	dc, err := cache.Open(dir)
	if err != nil {
		return nil, lazyerrors.Error(err)
	}

	return &Cache{
		dc:     dc,
		limits: limits,
		l:      l,
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
	c.dc.TrimExtra(c.limits, c.l)
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
// enforcing trim limits.
// It ignores the last trim time, but updates it.
//
// See [cache.DiskCache.TrimForce] for the returned values.
func (c *Cache) TrimForce() (before, freed, surge int64) {
	return c.dc.TrimForce(c.limits, c.l)
}

// check interfaces
var (
	_ cache.Cache = (*Cache)(nil)
)
