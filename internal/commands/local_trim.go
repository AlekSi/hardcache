package commands

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/AlekSi/hardcache/internal/caches/local"
	"github.com/AlekSi/hardcache/internal/unit"
)

// LocalTrimOpts contains flag values for [LocalTrim].
type LocalTrimOpts struct {
	Dir       string
	UnusedFor unit.Duration
	MaxSize   string
}

// LocalTrim force-trims a local cache according to the given parameters.
func LocalTrim(opts *LocalTrimOpts, l *slog.Logger) error {
	if time.Duration(opts.UnusedFor) > 5*24*time.Hour {
		l.Info("Note: this command should be invoked more often than once per day to keep the cache.")
	}

	return localTrim(opts, time.Now, l)
}

// localTrim force-trims a local cache according to the given parameters.
func localTrim(opts *LocalTrimOpts, now func() time.Time, l *slog.Logger) error {
	if opts.UnusedFor < 0 {
		return fmt.Errorf("--unused-for cannot be negative: %d", opts.UnusedFor)
	}

	var cutoff *time.Time
	if opts.UnusedFor > 0 {
		cutoff = new(now().Add(-time.Duration(opts.UnusedFor)))
	}

	var b unit.Bytes
	if strings.HasSuffix(opts.MaxSize, "%") {
		var p unit.Percentage
		if err := p.UnmarshalText([]byte(opts.MaxSize)); err != nil {
			return err
		}

		total, _, err := local.DiskInfo(opts.Dir)
		if err != nil {
			return err
		}

		b = unit.Bytes(total / 100 * int64(p))

		l.Debug(
			"Calculated max size from percentage of total disk size",
			slog.Int64("disk_size_bytes", total),
			slog.String("disk_size", unit.Bytes(total).String()),
			slog.Int64("max_size_bytes", int64(b)),
			slog.String("max_size", b.String()),
		)
	} else {
		if err := b.UnmarshalText([]byte(opts.MaxSize)); err != nil {
			return err
		}

		l.Debug("Max size", slog.Int64("max_size_bytes", int64(b)), slog.String("max_size", b.String()))
	}

	if b < 0 {
		return fmt.Errorf("--max-size cannot be negative: %d", b)
	}

	var maxSize *int64
	if b > 0 {
		maxSize = new(int64(b))
	}

	c, err := local.New(opts.Dir, cutoff, maxSize, l)
	if err != nil {
		return err
	}

	before, freed := c.TrimForce()
	l.Debug(
		"Local cache trimmed",
		slog.Int64("before_bytes", before),
		slog.Int64("freed_bytes", freed),
	)
	l.Info(
		"Local cache trimmed",
		slog.String("directory", opts.Dir),
		slog.String("before", fmt.Sprintf("%s (%d bytes)", unit.Bytes(before), before)),
		slog.String("freed", fmt.Sprintf("%s (%d bytes)", unit.Bytes(freed), freed)),
	)

	return nil
}
