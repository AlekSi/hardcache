package commands

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/AlekSi/hardcache/internal/unit"
)

// LocalTrimdOpts contains flag values for [LocalTrimd].
type LocalTrimdOpts struct {
	Dir            string
	UnusedFor      unit.Duration
	MaxSize        string
	KeepUsedWithin unit.Duration
	HardMaxSize    string // empty value disables it, like 0
	Interval       unit.Duration
}

// LocalTrimd continuously trims a local cache until ctx is canceled.
func LocalTrimd(ctx context.Context, opts *LocalTrimdOpts, l *slog.Logger) error {
	if opts.Interval <= 0 {
		return fmt.Errorf("--interval must be positive: %d", opts.Interval)
	}

	warnKeepUsedWithin(opts.KeepUsedWithin, l)

	trimOpts := &LocalTrimOpts{
		Dir:            opts.Dir,
		UnusedFor:      opts.UnusedFor,
		MaxSize:        opts.MaxSize,
		KeepUsedWithin: opts.KeepUsedWithin,
		HardMaxSize:    opts.HardMaxSize,
	}

	t := time.Tick(time.Duration(opts.Interval))

	for {
		if err := localTrim(trimOpts, time.Now, l); err != nil {
			return err
		}

		select {
		case <-ctx.Done():
			return nil
		case <-t:
		}
	}
}
