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
	stats := c.Stats()
	total, free, err := local.DiskInfo(opts.Dir)
	if err != nil {
		return err
	}

	status := newLocalStatusOutput(opts.Dir, stats, total, free)
	if before < 0 {
		before = stats.Bytes + freed
	}

	formatTime := func(t *string) string {
		if t == nil {
			return "n/a"
		}

		return *t
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
			slog.String("oldest", formatTime(status.Cache.Oldest)),
			slog.String("newest", formatTime(status.Cache.Newest)),
			slog.String("least_recently_used", formatTime(status.Cache.LeastRecentlyUsed)),
			slog.String("most_recently_used", formatTime(status.Cache.MostRecentlyUsed)),
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
