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
	stats := c.Status()
	total, free, err := local.DiskInfo(dir)
	if err != nil {
		return err
	}

	status := newLocalStatusOutput(dir, stats, total, free)
	if before < 0 {
		before = stats.Bytes + freed
	}

	oldest, newest := "n/a", "n/a"
	if status.Cache.Oldest != nil {
		oldest = *status.Cache.Oldest
	}
	if status.Cache.Newest != nil {
		newest = *status.Cache.Newest
	}

	l.Debug(
		"Local cache trimmed",
		slog.Int64("before_bytes", before),
		slog.Int64("after_bytes", stats.Bytes),
		slog.Int64("freed_bytes", freed),
	)
	l.Info(
		"Local cache trimmed",
		slog.String("directory", status.Directory),
		slog.String("before", fmt.Sprintf("%s (%d bytes)", unit.Bytes(before), before)),
		slog.String("freed", fmt.Sprintf("%s (%d bytes)", unit.Bytes(freed), freed)),
		slog.Group("cache",
			slog.Int("entries", status.Cache.Entries),
			slog.String("size", fmt.Sprintf("%s (%d bytes)", status.Cache.Human, status.Cache.Bytes)),
			slog.String("oldest", oldest),
			slog.String("newest", newest),
		),
		slog.Group("disk",
			slog.String("total", fmt.Sprintf("%s (%d bytes)", status.Disk.TotalHuman, status.Disk.TotalBytes)),
			slog.String("used", fmt.Sprintf("%s (%d bytes)", status.Disk.UsedHuman, status.Disk.UsedBytes)),
			slog.String("used_percent", fmt.Sprintf("%.2f%%", status.Disk.UsedPercent)),
			slog.String("free", fmt.Sprintf("%s (%d bytes)", status.Disk.FreeHuman, status.Disk.FreeBytes)),
			slog.String("free_percent", fmt.Sprintf("%.2f%%", status.Disk.FreePercent)),
		),
		slog.String("cache_of_total_disk", fmt.Sprintf("%.2f%%", status.CacheOfTotalPercent)),
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
