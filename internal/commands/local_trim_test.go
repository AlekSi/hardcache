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
	"github.com/AlekSi/hardcache/internal/unit"
)

func TestLocalTrim(t *testing.T) {
	t.Parallel()

	dir := localtest.Setup(t)

	var buf strings.Builder
	l := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	started := time.Now().Add(-time.Second).Truncate(time.Second)

	musta.NoError(t, LocalTrim(&LocalTrimOpts{
		Dir:     dir,
		MaxSize: "50MB",
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

// records parses JSON log records, removing their time.
func records(t testing.TB, logs string) []map[string]any {
	t.Helper()

	var res []map[string]any
	for line := range strings.SplitSeq(strings.TrimSpace(logs), "\n") {
		var m map[string]any
		musta.NoError(t, json.Unmarshal([]byte(line), &m))
		delete(m, "time")
		res = append(res, m)
	}

	return res
}

// countMsg returns the number of records with the given message.
func countMsg(records []map[string]any, msg string) int {
	var res int
	for _, r := range records {
		if r["msg"] == msg {
			res++
		}
	}

	return res
}

// hotNow returns a fixed time that makes testdata entries used since 17:13:02 hot
// for --keep-used-within=3s.
func hotNow() time.Time {
	return time.Date(2025, time.November, 17, 17, 13, 5, 0, time.UTC)
}

func TestLocalTrimKeepHot(t *testing.T) {
	t.Parallel()

	dir := localtest.Setup(t)

	var buf strings.Builder
	l := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	musta.NoError(t, localTrim(&LocalTrimOpts{
		Dir:            dir,
		MaxSize:        "50MB",
		KeepUsedWithin: unit.Duration(3 * time.Second),
	}, hotNow, l))

	actual := records(t, buf.String())
	shoulda.BeEqual(t, countMsg(actual, "Removed entry by max size"), 722)
	shoulda.BeEqual(t, countMsg(actual, "Removed hot entry by hard max size"), 0)

	shoulda.BeDeepEqual(t, actual[len(actual)-3:], []map[string]any{
		{
			"level":        "DEBUG",
			"msg":          "trim.txt updated",
			"before_bytes": 109518524.0,
			"after_bytes":  60715695.0,
			"freed_bytes":  48802829.0,
		},
		{
			"level":        "DEBUG",
			"msg":          "Local cache trimmed",
			"before_bytes": 109518524.0,
			"freed_bytes":  48802829.0,
			"surge_bytes":  10715695.0,
		},
		{
			"level":     "INFO",
			"msg":       "Local cache trimmed",
			"directory": dir,
			"before":    "109MB (109518524 bytes)",
			"freed":     "48MB (48802829 bytes)",
			"surge":     "10MB (10715695 bytes)",
		},
	})
}

func TestLocalTrimHardMaxSize(t *testing.T) {
	t.Parallel()

	dir := localtest.Setup(t)

	var buf strings.Builder
	l := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	musta.NoError(t, localTrim(&LocalTrimOpts{
		Dir:            dir,
		MaxSize:        "50MB",
		KeepUsedWithin: unit.Duration(3 * time.Second),
		HardMaxSize:    "55MB",
	}, hotNow, l))

	actual := records(t, buf.String())
	shoulda.BeEqual(t, countMsg(actual, "Removed entry by max size"), 722)
	shoulda.BeEqual(t, countMsg(actual, "Removed hot entry by hard max size"), 9)

	shoulda.BeDeepEqual(t, actual[len(actual)-3:], []map[string]any{
		{
			"level":        "DEBUG",
			"msg":          "trim.txt updated",
			"before_bytes": 109518524.0,
			"after_bytes":  54580204.0,
			"freed_bytes":  54938320.0,
		},
		{
			"level":        "DEBUG",
			"msg":          "Local cache trimmed",
			"before_bytes": 109518524.0,
			"freed_bytes":  54938320.0,
			"surge_bytes":  4580204.0,
		},
		{
			"level":     "INFO",
			"msg":       "Local cache trimmed",
			"directory": dir,
			"before":    "109MB (109518524 bytes)",
			"freed":     "54MB (54938320 bytes)",
			"surge":     "4MB (4580204 bytes)",
		},
	})
}

func TestLocalTrimHardMaxSizeOnly(t *testing.T) {
	t.Parallel()

	dir := localtest.Setup(t)

	var buf strings.Builder
	l := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	// the same result as TestLocalTrim
	musta.NoError(t, localTrim(&LocalTrimOpts{
		Dir:            dir,
		MaxSize:        "0GB",
		KeepUsedWithin: unit.Duration(3 * time.Second),
		HardMaxSize:    "50MB",
	}, hotNow, l))

	actual := records(t, buf.String())
	shoulda.BeEqual(t, countMsg(actual, "Removed entry by max size"), 0)
	shoulda.BeEqual(t, countMsg(actual, "Removed entry by hard max size"), 722)
	shoulda.BeEqual(t, countMsg(actual, "Removed hot entry by hard max size"), 84)

	shoulda.BeDeepEqual(t, actual[len(actual)-1], map[string]any{
		"level":     "INFO",
		"msg":       "Local cache trimmed",
		"directory": dir,
		"before":    "109MB (109518524 bytes)",
		"freed":     "60MB (60023595 bytes)",
	})
}

func TestLocalTrimKeepUsedWithinWarning(t *testing.T) {
	t.Parallel()

	var buf strings.Builder
	l := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	musta.NoError(t, LocalTrim(&LocalTrimOpts{
		Dir:            localtest.Setup(t),
		MaxSize:        "0GB",
		KeepUsedWithin: unit.Duration(30 * time.Minute),
	}, l))

	shoulda.BeDeepEqual(t, records(t, buf.String())[0], map[string]any{
		"level": "WARN",
		"msg": "--keep-used-within is shorter than 1h: " +
			"Go updates entry modification times at most once per hour, " +
			"so recently used entries could still be removed. Consider using 2h or more.",
		"keep_used_within": "30m",
	})
}

func TestLocalTrimErrors(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		opts LocalTrimOpts
		err  string
	}{
		"UnusedForNegative": {
			opts: LocalTrimOpts{UnusedFor: unit.Duration(-time.Hour), MaxSize: "0GB"},
			err:  "--unused-for cannot be negative: -3600000000000",
		},
		"KeepUsedWithinNegative": {
			opts: LocalTrimOpts{KeepUsedWithin: unit.Duration(-time.Hour), MaxSize: "0GB"},
			err:  "--keep-used-within cannot be negative: -3600000000000",
		},
		"UnusedForEqualKeepUsedWithin": {
			opts: LocalTrimOpts{
				UnusedFor:      unit.Duration(3 * time.Hour),
				KeepUsedWithin: unit.Duration(3 * time.Hour),
				MaxSize:        "0GB",
			},
			err: "--unused-for (3h) must be longer than --keep-used-within (3h)",
		},
		"UnusedForShorterThanKeepUsedWithin": {
			opts: LocalTrimOpts{
				UnusedFor:      unit.Duration(time.Hour),
				KeepUsedWithin: unit.Duration(24 * time.Hour),
				MaxSize:        "0GB",
			},
			err: "--unused-for (1h) must be longer than --keep-used-within (1d)",
		},
		"MaxSizeNegative": {
			opts: LocalTrimOpts{MaxSize: "-1GB"},
			err:  "--max-size cannot be negative: -1000000000",
		},
		"HardMaxSizeNegative": {
			opts: LocalTrimOpts{MaxSize: "0GB", HardMaxSize: "-1GB"},
			err:  "--hard-max-size cannot be negative: -1000000000",
		},
		"HardMaxSizeLessThanMaxSize": {
			opts: LocalTrimOpts{MaxSize: "50MB", HardMaxSize: "40MB"},
			err:  "--hard-max-size (40MB, 40000000 bytes) cannot be less than --max-size (50MB, 50000000 bytes)",
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var buf strings.Builder
			l := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

			opts := tc.opts
			opts.Dir = t.TempDir()

			err := localTrim(&opts, hotNow, l)
			musta.Error(t, err)
			shoulda.BeEqual(t, err.Error(), tc.err)
		})
	}
}
