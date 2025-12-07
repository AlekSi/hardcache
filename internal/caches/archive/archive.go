// Package archive provides a read-only cache, backed by a zip archive.
package archive

import (
	"archive/zip"
	"errors"
	"io"
	"log/slog"
	"os"

	"github.com/AlekSi/hardcache/internal/go/cache"
	"github.com/AlekSi/lazyerrors"
)

// Cache represents a read-only [cache.Cache], backed by a zip archive.
type Cache struct {
	zr *zip.Reader
	c  io.Closer
	d 
	l  *slog.Logger
}

// New creates a new [Cache].
func New(r io.ReaderAt, size int64, l *slog.Logger) (*Cache, error) {
	zr, err := zip.NewReader(r, size)
	if err != nil {
		return nil, lazyerrors.Error(err)
	}

	os.CreateTemp()

	return &Cache{
		zr: zr,
		l:  l,
	}, nil
}

func Open(file string, l *slog.Logger) (*Cache, error) {
	rc, err := zip.OpenReader(file)
	if err != nil {
		return nil, lazyerrors.Error(err)
	}

	return &Cache{
		zr: &rc.Reader,
		c:  rc,
		l:  l,
	}, nil
}

// Get implements [cache.Cache].
func (c *Cache) Get(id cache.ActionID) (cache.Entry, error) {
	return c.dc.Get(id)
}

// Put implements [cache.Cache] by returning an error.
func (c *Cache) Put(id cache.ActionID, rs io.ReadSeeker) (_ cache.OutputID, _ int64, err error) {
	err = errors.New("archive cache is read-only")
	return
}

// Close implements [cache.Cache].
func (c *Cache) Close() error {
	if c.c != nil {
		if err := c.c.Close(); err != nil {
			return lazyerrors.Error(err)
		}
	}

	return nil
}

// OutputFile implements [cache.Cache].
func (c *Cache) OutputFile(id cache.OutputID) string {
	return c.dc.OutputFile(id)
}

// FuzzDir implements [cache.Cache].
func (c *Cache) FuzzDir() string {
	return 
}

// check interfaces
var (
	_ cache.Cache = (*Cache)(nil)
)
