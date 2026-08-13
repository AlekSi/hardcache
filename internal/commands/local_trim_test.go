package commands

import (
	"log/slog"
	"strings"
	"testing"

	"github.com/AlekSi/shoulda"
	"github.com/AlekSi/shoulda/musta"

	"github.com/AlekSi/hardcache/internal/caches/local"
	"github.com/AlekSi/hardcache/internal/caches/local/localtest"
)

func TestLocalTrimStatistics(t *testing.T) {
	t.Parallel()

	dir := localtest.Setup(t)
	var output strings.Builder
	l := slog.New(slog.NewTextHandler(&output, nil))

	musta.NoError(t, LocalTrim(&LocalTrimOpts{Dir: dir, MaxSize: "50MB"}, l))

	stats := musta.NotFail(local.New(dir, nil, nil, l))(t).Stats()
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
		`cache.least_recently_used=`,
		`cache.most_recently_used=`,
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
