package commands

import (
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/AlekSi/shoulda"
	"github.com/AlekSi/shoulda/musta"

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

	actual := output.String()
	shoulda.SatisfyWith(t, actual, `msg="Local cache trimmed"`, strings.Contains)
	shoulda.SatisfyWith(t, actual, `before="109MB (109518524 bytes)"`, strings.Contains)
	shoulda.SatisfyWith(t, actual, `freed="60MB (60023595 bytes)"`, strings.Contains)
}
