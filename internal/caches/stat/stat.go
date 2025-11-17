// Package stat provides a cache wrapper that logs statistics.
package stat

import (
	"io"
	"log/slog"
	"time"

	"github.com/AlekSi/hardcache/internal/go/cache"
	"github.com/AlekSi/hardcache/internal/unit"
)

// Cache is a cache wrapper that logs statistics.
type Cache struct {
	c       cache.Cache
	l       *slog.Logger
	start   time.Time
	hits    int
	misses  int
	puts    int
	putErrs int
	largest int64
}

// New creates a new [Cache].
func New(c cache.Cache, l *slog.Logger) *Cache {
	return &Cache{
		c:     c,
		l:     l,
		start: time.Now(),
	}
}

// Get implements [cache.Cache].
func (c *Cache) Get(id cache.ActionID) (cache.Entry, error) {
	e, err := c.c.Get(id)
	if err == nil {
		c.hits++
	} else {
		c.misses++
	}

	c.largest = max(c.largest, e.Size)

	return e, err
}

// Put implements [cache.Cache].
func (c *Cache) Put(id cache.ActionID, rs io.ReadSeeker) (cache.OutputID, int64, error) {
	o, size, err := c.c.Put(id, rs)
	if err == nil {
		c.puts++
	} else {
		c.putErrs++
	}

	c.largest = max(c.largest, size)

	return o, size, err
}

// Close implements [cache.Cache].
func (c *Cache) Close() error {
	err := c.c.Close()
	c.l.Info(
		"Cache closed",
		slog.Int("hits", c.hits), slog.Int("misses", c.misses),
		slog.Int("puts", c.puts), slog.Int("putErrs", c.putErrs),
		slog.String("largest", unit.Bytes(c.largest).String()),
		slog.Duration("duration", time.Since(c.start)),
	)
	return err
}

// OutputFile implements [cache.Cache].
func (c *Cache) OutputFile(id cache.OutputID) string {
	return c.c.OutputFile(id)
}

// FuzzDir implements [cache.Cache].
func (c *Cache) FuzzDir() string {
	return c.c.FuzzDir()
}

// check interfaces
var (
	_ cache.Cache = (*Cache)(nil)
)
