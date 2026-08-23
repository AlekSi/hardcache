package commands

import (
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/AlekSi/shoulda/musta"

	"github.com/AlekSi/hardcache/internal/caches/local/localtest"
	"github.com/AlekSi/hardcache/internal/unit"
)

func TestLocalTrimd(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	dir := localtest.Setup(t)

	var buf strings.Builder
	l := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	started := time.Now()

	musta.NoError(t, LocalTrimd(ctx, &LocalTrimdOpts{
		Dir:      dir,
		MaxSize:  "50MB",
		Interval: unit.Duration(time.Hour),
	}, l))

	finished := time.Now()

	checkLocalTrimOutput(t, dir, &buf, l, started, finished)
}
