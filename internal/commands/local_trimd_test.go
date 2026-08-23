package commands

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/AlekSi/shoulda"
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

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	shoulda.BeEqual(t, len(lines), 810)

	lines = lines[len(lines)-3:]

	var actual []map[string]any
	for _, line := range lines {
		t.Log(line)

		var m map[string]any
		musta.NoError(t, json.Unmarshal([]byte(line), &m))

		timestamp, err := time.Parse(time.RFC3339Nano, m["time"].(string))
		musta.NoError(t, err)

		shoulda.Satisfy(t, timestamp, started.Before)
		shoulda.Satisfy(t, timestamp, finished.After)
		delete(m, "time")

		actual = append(actual, m)
	}

	shoulda.BeDeepEqual(t, actual, []map[string]any{
		{
			"level":        "DEBUG",
			"msg":          "trim.txt updated",
			"before_bytes": 109518524.0,
			"after_bytes":  49494929.0,
			"freed_bytes":  60023595.0,
		},
		{
			"level":        "DEBUG",
			"msg":          "Local cache trimmed",
			"before_bytes": 109518524.0,
			"freed_bytes":  60023595.0,
		},
		{
			"level":     "INFO",
			"msg":       "Local cache trimmed",
			"directory": dir,
			"before":    "109MB (109518524 bytes)",
			"freed":     "60MB (60023595 bytes)",
		},
	})
}
