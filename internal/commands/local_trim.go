package commands

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/AlekSi/hardcache/internal/caches/local"
	"github.com/AlekSi/hardcache/internal/unit"
)

// LocalTrim force-trims a local cache according to the given parameters.
func LocalTrim(dir string, unusedFor unit.Duration, maxSizeValue string, l *slog.Logger) error {
	if time.Duration(unusedFor) > 5*24*time.Hour {
		l.Info("Note: this command should be invoked more often than once per day to keep the cache.")
	}

	return localTrim(dir, unusedFor, maxSizeValue, l)
}

func localTrim(dir string, unusedFor unit.Duration, maxSizeValue string, l *slog.Logger) error {
	if unusedFor < 0 {
		return fmt.Errorf("--unused-for cannot be negative: %d", unusedFor)
	}

	var cutoff *time.Time
	if unusedFor > 0 {
		c := time.Now().Add(-time.Duration(unusedFor))
		cutoff = &c
	}

	var b unit.Bytes
	if strings.HasSuffix(maxSizeValue, "%") {
		var p unit.Percentage
		if err := p.UnmarshalText([]byte(maxSizeValue)); err != nil {
			return err
		}

		total, _, err := local.DiskInfo(dir)
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
		if err := b.UnmarshalText([]byte(maxSizeValue)); err != nil {
			return err
		}

		l.Debug("Max size", slog.Int64("max_size_bytes", int64(b)), slog.String("max_size", b.String()))
	}

	if b < 0 {
		return fmt.Errorf("--max-size cannot be negative: %d", b)
	}

	var maxSize *int64
	if b > 0 {
		maxSize = (*int64)(&b)
	}

	c, err := local.New(dir, cutoff, maxSize, l)
	if err != nil {
		return err
	}

	before, freed := c.TrimForce()
	l.Debug(
		"Local cache trimmed",
		slog.Int64("before_bytes", before), slog.Int64("freed_bytes", freed),
	)
	l.Info(
		"Local cache trimmed",
		slog.String("before", unit.Bytes(before).String()), slog.String("freed", unit.Bytes(freed).String()),
	)

	return nil
}

// LocalTrimd continuously trims a local cache until ctx is canceled.
func LocalTrimd(
	ctx context.Context,
	dir string,
	unusedFor unit.Duration,
	maxSizeValue string,
	interval unit.Duration,
	l *slog.Logger,
) error {
	for {
		if err := localTrim(dir, unusedFor, maxSizeValue, l); err != nil {
			return err
		}

		select {
		case <-ctx.Done():
			return nil
		case <-time.After(time.Duration(interval)):
		}
	}
}
