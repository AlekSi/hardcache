package commands

import (
	"context"
	"log/slog"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/AlekSi/shoulda"
	"github.com/AlekSi/shoulda/musta"

	"github.com/AlekSi/hardcache/internal/caches/local"
	"github.com/AlekSi/hardcache/internal/caches/local/localtest"
	"github.com/AlekSi/hardcache/internal/unit"
)

func TestLocalTrimdStatistics(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	dir := localtest.Setup(t)
	var output strings.Builder
	l := slog.New(slog.NewTextHandler(&output, nil))
	musta.NoError(t, LocalTrimd(ctx, &LocalTrimdOpts{
		Dir:      dir,
		MaxSize:  "50MB",
		Interval: unit.Duration(time.Hour),
	}, l))

	stats := musta.NotFail(local.New(dir, nil, nil, l))(t).Stats()
	shoulda.BeEqual(t, stats.Bytes, int64(49_494_929))
	shoulda.BeGreater(t, stats.Entries, 0)

	actual := output.String()
	shoulda.SatisfyWith(t, actual, `msg="Local cache trimmed"`, strings.Contains)
	shoulda.SatisfyWith(t, actual, `before="109MB (109518524 bytes)"`, strings.Contains)
	shoulda.SatisfyWith(t, actual, `freed="60MB (60023595 bytes)"`, strings.Contains)
	shoulda.SatisfyWith(t, actual, `cache.entries=`+strconv.Itoa(stats.Entries), strings.Contains)
	shoulda.SatisfyWith(t, actual, `cache.size="49MB (49494929 bytes)"`, strings.Contains)
	shoulda.SatisfyWith(t, actual, `cache.oldest=`, strings.Contains)
	shoulda.SatisfyWith(t, actual, `cache.newest=`, strings.Contains)
	shoulda.SatisfyWith(t, actual, `cache.least_recently_used=`, strings.Contains)
	shoulda.SatisfyWith(t, actual, `cache.most_recently_used=`, strings.Contains)
	shoulda.SatisfyWith(t, actual, `disk.total=`, strings.Contains)
}
