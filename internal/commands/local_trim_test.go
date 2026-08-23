package commands

import (
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/AlekSi/shoulda"
	"github.com/AlekSi/shoulda/musta"

	"github.com/AlekSi/hardcache/internal/caches/local/localtest"
)

func TestLocalTrim(t *testing.T) {
	t.Parallel()

	dir := localtest.Setup(t)

	var buf strings.Builder
	l := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	started := time.Now()
	musta.NoError(t, LocalTrim(&LocalTrimOpts{
		Dir:     dir,
		MaxSize: "50MB",
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

		shoulda.BeFalse(t, timestamp.Before(started))
		shoulda.BeFalse(t, timestamp.After(finished))
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
