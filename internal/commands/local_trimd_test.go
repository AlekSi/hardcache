package commands

import (
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/AlekSi/hardcache/internal/unit"
	"github.com/AlekSi/shoulda"
	"github.com/AlekSi/shoulda/musta"
)

func TestLocalTrimdStatistics(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var output strings.Builder
	l := slog.New(slog.NewTextHandler(&output, nil))
	musta.NoError(t, LocalTrimd(ctx, &LocalTrimdOpts{
		Dir:      t.TempDir(),
		MaxSize:  "0GB",
		Interval: unit.Duration(time.Hour),
	}, l))

	actual := output.String()
	shoulda.SatisfyWith(t, actual, `msg="Local cache trimmed"`, strings.Contains)
	shoulda.SatisfyWith(t, actual, `cache.entries=0`, strings.Contains)
	shoulda.SatisfyWith(t, actual, `cache.size="0B (0 bytes)"`, strings.Contains)
	shoulda.SatisfyWith(t, actual, `cache.oldest=n/a`, strings.Contains)
	shoulda.SatisfyWith(t, actual, `cache.newest=n/a`, strings.Contains)
	shoulda.SatisfyWith(t, actual, `disk.total=`, strings.Contains)
}
