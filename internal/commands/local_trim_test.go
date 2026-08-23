package commands

import (
	"log/slog"
	"strings"
	"testing"

	"github.com/AlekSi/shoulda"
	"github.com/AlekSi/shoulda/musta"

	"github.com/AlekSi/hardcache/internal/caches/local/localtest"
)

func TestLocalTrimStatistics(t *testing.T) {
	t.Parallel()

	dir := localtest.Setup(t)
	var output strings.Builder
	l := slog.New(slog.NewTextHandler(&output, nil))

	musta.NoError(t, LocalTrim(&LocalTrimOpts{Dir: dir, MaxSize: "50MB"}, l))

	actual := output.String()
	for _, expected := range []string{
		`msg="Local cache trimmed"`,
		`directory=`,
		`before="109MB (109518524 bytes)"`,
		`freed="60MB (60023595 bytes)"`,
	} {
		shoulda.SatisfyWith(t, actual, expected, strings.Contains)
	}
}
