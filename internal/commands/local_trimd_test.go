package commands

import (
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/AlekSi/shoulda"
	"github.com/AlekSi/shoulda/musta"

	"github.com/AlekSi/hardcache/internal/caches/local/localtest"
	"github.com/AlekSi/hardcache/internal/unit"
)

func TestLocalTrimdStatistics(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	dir := localtest.Setup(t)
	var output strings.Builder
	l := slog.New(slog.NewJSONHandler(&output, &slog.HandlerOptions{Level: slog.LevelDebug}))
	musta.NoError(t, LocalTrimd(ctx, &LocalTrimdOpts{
		Dir:      dir,
		MaxSize:  "50MB",
		Interval: unit.Duration(time.Hour),
	}, l))

	logs := decodeLocalTrimLogs(t, output.String())
	if len(logs) < 2 {
		t.Fatalf("got %d log entries, expected at least 2", len(logs))
	}

	shoulda.BeDeepEqual(t, logs[len(logs)-2:], []localTrimLog{
		{
			Level:       "DEBUG",
			Message:     "Local cache trimmed",
			BeforeBytes: 109_518_524,
			FreedBytes:  60_023_595,
		},
		{
			Level:     "INFO",
			Message:   "Local cache trimmed",
			Directory: dir,
			Before:    "109MB (109518524 bytes)",
			Freed:     "60MB (60023595 bytes)",
		},
	})
}
