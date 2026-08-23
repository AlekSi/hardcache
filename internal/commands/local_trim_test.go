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

func TestLocalTrimStatistics(t *testing.T) {
	t.Parallel()

	dir := localtest.Setup(t)
	var output strings.Builder
	l := slog.New(slog.NewJSONHandler(&output, &slog.HandlerOptions{Level: slog.LevelDebug}))

	started := time.Now()
	musta.NoError(t, LocalTrim(&LocalTrimOpts{Dir: dir, MaxSize: "50MB"}, l))
	finished := time.Now()

	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	actual := make([]map[string]any, 2)
	for i, line := range lines[len(lines)-2:] {
		musta.NoError(t, json.Unmarshal([]byte(line), &actual[i]))
		timestamp := musta.NotFail(time.Parse(time.RFC3339Nano, actual[i]["time"].(string)))(t)
		shoulda.BeFalse(t, timestamp.Before(started))
		shoulda.BeFalse(t, timestamp.After(finished))
		delete(actual[i], "time")
	}

	shoulda.BeDeepEqual(t, actual, []map[string]any{
		{
			"level":        "DEBUG",
			"msg":          "Local cache trimmed",
			"before_bytes": float64(109_518_524),
			"freed_bytes":  float64(60_023_595),
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
