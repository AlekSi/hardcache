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
	Dir       string
	UnusedFor unit.Duration
	MaxSize   string
	Interval  unit.Duration
}

// LocalTrimd continuously trims a local cache until ctx is canceled.
func LocalTrimd(ctx context.Context, opts *LocalTrimdOpts, l *slog.Logger) error {
	if opts.UnusedFor < 0 {
		return fmt.Errorf("--unused-for cannot be negative: %d", opts.UnusedFor)
	}

	trimOpts := &LocalTrimOpts{
		Dir:       opts.Dir,
		UnusedFor: opts.UnusedFor,
		MaxSize:   opts.MaxSize,
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
