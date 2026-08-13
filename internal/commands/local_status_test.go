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

func TestLocalStatus(t *testing.T) {
	t.Parallel()

	dir := localtest.Setup(t)
	oldest := time.Date(2025, time.November, 17, 17, 12, 57, 524467000, time.UTC).Local().Format(time.RFC3339)
	newest := time.Date(2025, time.November, 17, 17, 13, 7, 284400000, time.UTC).Local().Format(time.RFC3339)

	t.Run("text", func(t *testing.T) {
		var output strings.Builder
		musta.NoError(t, LocalStatus(&LocalStatusOpts{Dir: dir}, &output, slog.Default()))

		actual := output.String()
		shoulda.SatisfyWith(t, actual, "Directory: "+dir, strings.Contains)
		shoulda.SatisfyWith(t, actual, "Cache entries: 1219", strings.Contains)
		shoulda.SatisfyWith(t, actual, "Cache size: 109MB (109518524 bytes)", strings.Contains)
		shoulda.SatisfyWith(t, actual, "Oldest entry: "+oldest, strings.Contains)
		shoulda.SatisfyWith(t, actual, "Newest entry: "+newest, strings.Contains)
		shoulda.SatisfyWith(t, actual, "Disk total: ", strings.Contains)
		shoulda.SatisfyWith(t, actual, "Disk free: ", strings.Contains)
	})

	t.Run("JSON", func(t *testing.T) {
		var output strings.Builder
		musta.NoError(t, LocalStatus(&LocalStatusOpts{Dir: dir, JSON: true}, &output, slog.Default()))

		actual := output.String()
		shoulda.SatisfyWith(t, actual, "\n", strings.HasSuffix)
		shoulda.BeEqual(t, strings.Count(actual, "\n"), 1)

		var got localStatusOutput
		musta.NoError(t, json.Unmarshal([]byte(actual), &got))
		shoulda.BeEqual(t, got.Directory, dir)
		shoulda.BeEqual(t, got.Cache.Entries, 1219)
		shoulda.BeEqual(t, got.Cache.Bytes, int64(109_518_524))
		shoulda.BeEqual(t, got.Cache.Human, "109MB")
		musta.NotBeZero(t, got.Cache.Oldest)
		musta.NotBeZero(t, got.Cache.Newest)
		shoulda.BeEqual(t, *got.Cache.Oldest, oldest)
		shoulda.BeEqual(t, *got.Cache.Newest, newest)
		shoulda.BeGreater(t, got.Disk.TotalBytes, int64(0))
		shoulda.BeEqual(t, got.Disk.UsedBytes+got.Disk.FreeBytes, got.Disk.TotalBytes)
	})
}

func TestLocalStatusEmpty(t *testing.T) {
	t.Parallel()

	dir := localtest.Setup(t)
	c := musta.NotFail(local.New(dir, nil, new(int64(0)), slog.Default()))(t)
	before, freed := c.TrimForce()
	shoulda.BeEqual(t, before, int64(109_518_524))
	shoulda.BeEqual(t, freed, before)

	var output strings.Builder
	musta.NoError(t, LocalStatus(&LocalStatusOpts{Dir: dir}, &output, slog.Default()))

	actual := output.String()
	shoulda.SatisfyWith(t, actual, "Cache entries: 0", strings.Contains)
	shoulda.SatisfyWith(t, actual, "Cache size: 0B (0 bytes)", strings.Contains)
	shoulda.SatisfyWith(t, actual, "Oldest entry: n/a", strings.Contains)
	shoulda.SatisfyWith(t, actual, "Newest entry: n/a", strings.Contains)
}
