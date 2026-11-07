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

	started := time.Now().Add(-time.Second).Truncate(time.Second)

	musta.NoError(t, LocalTrimd(ctx, &LocalTrimdOpts{
		Dir:      dir,
		MaxSize:  "50MB",
		Interval: unit.Duration(time.Hour),
	}, l))

	finished := time.Now().Add(time.Second).Truncate(time.Second)

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

func TestLocalTrimdHardMaxSize(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	dir := localtest.Setup(t)

	var buf strings.Builder
	l := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	// testdata entries are old, so none are hot, and the result is the same as TestLocalTrimd
	musta.NoError(t, LocalTrimd(ctx, &LocalTrimdOpts{
		Dir:            dir,
		MaxSize:        "0GB",
		KeepUsedWithin: unit.Duration(30 * time.Minute),
		HardMaxSize:    "50MB",
		Interval:       unit.Duration(time.Hour),
	}, l))

	actual := records(t, buf.String())
	shoulda.BeEqual(t, actual[0]["level"].(string), "WARN")
	shoulda.BeEqual(t, countMsg(actual, "Removed entry by hard max size"), 806)
	shoulda.BeEqual(t, countMsg(actual, "Removed hot entry by hard max size"), 0)

	shoulda.BeDeepEqual(t, actual[len(actual)-1], map[string]any{
		"level":     "INFO",
		"msg":       "Local cache trimmed",
		"directory": dir,
		"before":    "109MB (109518524 bytes)",
		"freed":     "60MB (60023595 bytes)",
	})
}

func TestLocalTrimdErrors(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	for name, tc := range map[string]struct {
		opts LocalTrimdOpts
		err  string
	}{
		"UnusedForShorterThanKeepUsedWithin": {
			opts: LocalTrimdOpts{
				UnusedFor:      unit.Duration(time.Hour),
				KeepUsedWithin: unit.Duration(3 * time.Hour),
				MaxSize:        "0GB",
			},
			err: "--unused-for (1h) must be longer than --keep-used-within (3h)",
		},
		"HardMaxSizeLessThanMaxSize": {
			opts: LocalTrimdOpts{MaxSize: "50MB", HardMaxSize: "40MB"},
			err:  "--hard-max-size (40MB, 40000000 bytes) cannot be less than --max-size (50MB, 50000000 bytes)",
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var buf strings.Builder
			l := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

			opts := tc.opts
			opts.Dir = t.TempDir()
			opts.Interval = unit.Duration(time.Hour)

			err := LocalTrimd(ctx, &opts, l)
			musta.Error(t, err)
			shoulda.BeEqual(t, err.Error(), tc.err)
		})
	}
}
