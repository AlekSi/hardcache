package commands

import (
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/AlekSi/shoulda"
	"github.com/AlekSi/shoulda/musta"

	"github.com/AlekSi/hardcache/internal/caches/local"
	"github.com/AlekSi/hardcache/internal/unit"
)

func TestLocalTrimStatistics(t *testing.T) {
	t.Parallel()

	dir := setup(t)
	var output strings.Builder
	l := slog.New(slog.NewTextHandler(&output, nil))

	musta.NoError(t, LocalTrim(dir, 0, "50MB", l))

	stats := musta.NotFail(local.New(dir, nil, nil, l))(t).Status()
	shoulda.BeEqual(t, stats.Bytes, int64(49_494_929))

	actual := output.String()
	for _, expected := range []string{
		`msg="Local cache trimmed"`,
		`directory=`,
		`before="109MB (109518524 bytes)"`,
		`freed="60MB (60023595 bytes)"`,
		`cache.entries=`,
		`cache.size="49MB (49494929 bytes)"`,
		`cache.oldest=`,
		`cache.newest=`,
		`disk.total=`,
		`disk.used=`,
		`disk.used_percent=`,
		`disk.free=`,
		`disk.free_percent=`,
		`cache_of_total_disk=`,
	} {
		shoulda.SatisfyWith(t, actual, expected, strings.Contains)
	}
}

func TestLocalTrimdStatistics(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var output strings.Builder
	l := slog.New(slog.NewTextHandler(&output, nil))
	musta.NoError(t, LocalTrimd(ctx, t.TempDir(), 0, "0GB", unit.Duration(time.Hour), l))

	actual := output.String()
	shoulda.SatisfyWith(t, actual, `msg="Local cache trimmed"`, strings.Contains)
	shoulda.SatisfyWith(t, actual, `cache.entries=0`, strings.Contains)
	shoulda.SatisfyWith(t, actual, `cache.size="0B (0 bytes)"`, strings.Contains)
	shoulda.SatisfyWith(t, actual, `cache.oldest=n/a`, strings.Contains)
	shoulda.SatisfyWith(t, actual, `cache.newest=n/a`, strings.Contains)
	shoulda.SatisfyWith(t, actual, `disk.total=`, strings.Contains)
}
