package commands

import (
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/AlekSi/shoulda"
	"github.com/AlekSi/shoulda/musta"

	"github.com/AlekSi/hardcache/internal/caches/local"
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

	checkLocalTrimOutput(t, dir, &buf, l, started, finished)
}

func checkLocalTrimOutput(t *testing.T, dir string, buf *strings.Builder, l *slog.Logger, started time.Time, finished time.Time) {
	t.Helper()

	c := musta.NotFail(local.New(dir, nil, nil, l))(t)
	_, _, stats := c.TrimForce()
	shoulda.BeEqual(t, stats.Bytes, int64(49_494_929))
	shoulda.BeGreater(t, stats.Entries, 0)

	status := newLocalStatusOutput(dir, stats, 0, 0)
	musta.NotBeZero(t, status.Cache.LeastRecentlyUsed)
	musta.NotBeZero(t, status.Cache.MostRecentlyUsed)

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

		shoulda.NotSatisfy(t, timestamp, started.After)
		shoulda.NotSatisfy(t, timestamp, finished.Before)
		delete(m, "time")

		actual = append(actual, m)
	}

	disk := actual[2]["disk"].(map[string]any)
	shoulda.BeEqual(t, len(disk), 5)
	for _, name := range []string{"total", "used", "free"} {
		shoulda.SatisfyWith(t, disk[name].(string), " bytes)", strings.HasSuffix)
	}
	for _, name := range []string{"used_percent", "free_percent"} {
		shoulda.SatisfyWith(t, disk[name].(string), "%", strings.HasSuffix)
	}
	shoulda.SatisfyWith(t, actual[2]["cache_of_total_disk"].(string), "%", strings.HasSuffix)
	delete(actual[2], "disk")
	delete(actual[2], "cache_of_total_disk")

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
			"after_bytes":  49494929.0,
			"freed_bytes":  60023595.0,
		},
		{
			"level":     "INFO",
			"msg":       "Local cache trimmed",
			"directory": dir,
			"before":    "109MB (109518524 bytes)",
			"freed":     "60MB (60023595 bytes)",
			"cache": map[string]any{
				"entries":             float64(status.Cache.Entries),
				"size":                "49MB (49494929 bytes)",
				"least_recently_used": *status.Cache.LeastRecentlyUsed,
				"most_recently_used":  *status.Cache.MostRecentlyUsed,
			},
		},
	})
}
