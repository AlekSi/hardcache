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
	Dir            string
	UnusedFor      unit.Duration
	MaxSize        string
	KeepUsedWithin unit.Duration
	HardMaxSize    string // empty value disables it, like 0
}

// LocalTrim force-trims a local cache according to the given parameters.
func LocalTrim(opts *LocalTrimOpts, l *slog.Logger) error {
	if time.Duration(opts.UnusedFor) > 5*24*time.Hour {
		l.Info("Note: this command should be invoked more often than once per day to keep the cache.")
	}

	warnKeepUsedWithin(opts.KeepUsedWithin, l)

	return localTrim(opts, time.Now, l)
}

// warnKeepUsedWithin logs a warning if --keep-used-within value protects less than it says.
func warnKeepUsedWithin(keepUsedWithin unit.Duration, l *slog.Logger) {
	if keepUsedWithin <= 0 || time.Duration(keepUsedWithin) >= time.Hour {
		return
	}

	l.Warn(
		"--keep-used-within is shorter than 1h: "+
			"Go updates entry modification times at most once per hour, "+
			"so recently used entries could still be removed. Consider using 2h or more.",
		slog.String("keep_used_within", keepUsedWithin.String()),
	)
}

// localTrim force-trims a local cache according to the given parameters.
func localTrim(opts *LocalTrimOpts, now func() time.Time, l *slog.Logger) error {
	if opts.UnusedFor < 0 {
		return fmt.Errorf("--unused-for cannot be negative: %d", opts.UnusedFor)
	}

	if opts.KeepUsedWithin < 0 {
		return fmt.Errorf("--keep-used-within cannot be negative: %d", opts.KeepUsedWithin)
	}

	if opts.UnusedFor > 0 && opts.KeepUsedWithin > 0 && opts.UnusedFor <= opts.KeepUsedWithin {
		return fmt.Errorf(
			"--unused-for (%s) must be longer than --keep-used-within (%s)",
			opts.UnusedFor, opts.KeepUsedWithin,
		)
	}

	maxSize, err := parseSize("--max-size", opts.MaxSize, opts.Dir, l)
	if err != nil {
		return err
	}

	var hardMaxSize unit.Bytes
	if opts.HardMaxSize != "" {
		if hardMaxSize, err = parseSize("--hard-max-size", opts.HardMaxSize, opts.Dir, l); err != nil {
			return err
		}
	}

	if maxSize > 0 && hardMaxSize > 0 && hardMaxSize < maxSize {
		return fmt.Errorf(
			"--hard-max-size (%s, %d bytes) cannot be less than --max-size (%s, %d bytes)",
			hardMaxSize, hardMaxSize, maxSize, maxSize,
		)
	}

	t := now()

	var limits local.TrimLimits

	if opts.UnusedFor > 0 {
		limits.Cutoff = new(t.Add(-time.Duration(opts.UnusedFor)))
	}

	if maxSize > 0 {
		limits.MaxSize = new(int64(maxSize))
	}

	if opts.KeepUsedWithin > 0 {
		limits.KeepAfter = new(t.Add(-time.Duration(opts.KeepUsedWithin)))
	}

	if hardMaxSize > 0 {
		limits.HardMaxSize = new(int64(hardMaxSize))
	}

	c, err := local.New(opts.Dir, limits, l)
	if err != nil {
		return err
	}

	before, freed, surge := c.TrimForce()

	debugAttrs := []any{
		slog.Int64("before_bytes", before),
		slog.Int64("freed_bytes", freed),
	}
	infoAttrs := []any{
		slog.String("directory", opts.Dir),
		slog.String("before", fmt.Sprintf("%s (%d bytes)", unit.Bytes(before), before)),
		slog.String("freed", fmt.Sprintf("%s (%d bytes)", unit.Bytes(freed), freed)),
	}

	// how much the cache remains above --max-size because hot entries were kept
	if surge > 0 {
		debugAttrs = append(debugAttrs, slog.Int64("surge_bytes", surge))
		infoAttrs = append(infoAttrs, slog.String("surge", fmt.Sprintf("%s (%d bytes)", unit.Bytes(surge), surge)))
	}

	l.Debug("Local cache trimmed", debugAttrs...)
	l.Info("Local cache trimmed", infoAttrs...)

	return nil
}

// parseSize parses the value of the given size flag:
// a size with MiB, GB, etc. suffix, or a percentage of the total disk space.
func parseSize(flag, value, dir string, l *slog.Logger) (unit.Bytes, error) {
	var b unit.Bytes
	if strings.HasSuffix(value, "%") {
		var p unit.Percentage
		if err := p.UnmarshalText([]byte(value)); err != nil {
			return 0, err
		}

		total, _, err := local.DiskInfo(dir)
		if err != nil {
			return 0, err
		}

		b = unit.Bytes(total / 100 * int64(p))

		l.Debug(
			"Calculated size limit from percentage of total disk size",
			slog.String("flag", flag),
			slog.Int64("disk_size_bytes", total),
			slog.String("disk_size", unit.Bytes(total).String()),
			slog.Int64("size_bytes", int64(b)),
			slog.String("size", b.String()),
		)
	} else {
		if err := b.UnmarshalText([]byte(value)); err != nil {
			return 0, err
		}

		l.Debug("Size limit", slog.String("flag", flag), slog.Int64("size_bytes", int64(b)), slog.String("size", b.String()))
	}

	if b < 0 {
		return 0, fmt.Errorf("%s cannot be negative: %d", flag, b)
	}

	return b, nil
}
