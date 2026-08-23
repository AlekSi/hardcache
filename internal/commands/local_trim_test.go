package commands

import (
	"encoding/json"
	"io"
	"log/slog"
	"strings"
	"testing"

	"github.com/AlekSi/shoulda"
	"github.com/AlekSi/shoulda/musta"

	"github.com/AlekSi/hardcache/internal/caches/local/localtest"
)

type localTrimLog struct {
	Level       string `json:"level"`
	Message     string `json:"msg"`
	Directory   string `json:"directory"`
	Before      string `json:"before"`
	Freed       string `json:"freed"`
	BeforeBytes int64  `json:"before_bytes"`
	FreedBytes  int64  `json:"freed_bytes"`
}

func decodeLocalTrimLogs(t *testing.T, output string) []localTrimLog {
	t.Helper()

	d := json.NewDecoder(strings.NewReader(output))
	var logs []localTrimLog
	for {
		var l localTrimLog
		err := d.Decode(&l)
		if err == io.EOF {
			return logs
		}

		musta.NoError(t, err)
		logs = append(logs, l)
	}
}

func TestLocalTrimStatistics(t *testing.T) {
	t.Parallel()

	dir := localtest.Setup(t)
	var output strings.Builder
	l := slog.New(slog.NewJSONHandler(&output, &slog.HandlerOptions{Level: slog.LevelDebug}))

	musta.NoError(t, LocalTrim(&LocalTrimOpts{Dir: dir, MaxSize: "50MB"}, l))

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
