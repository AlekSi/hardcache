package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/AlekSi/hardcache/internal/caches/local"
	"github.com/AlekSi/shoulda"
	"github.com/AlekSi/shoulda/musta"
)

func TestStatusTextFormatting(t *testing.T) {
	oldest := time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC)
	newest := time.Date(2026, time.January, 3, 4, 5, 6, 0, time.UTC)
	report := newLocalStatusReport("/tmp/cache", local.Stats{
		Entries: 3,
		Bytes:   1024,
		Oldest:  &oldest,
		Newest:  &newest,
	}, 10*1024, 4*1024)

	actual := renderLocalStatusText(report)

	shoulda.SatisfyWith(t, actual, "Directory: /tmp/cache", strings.Contains)
	shoulda.SatisfyWith(t, actual, "Cache entries: 3", strings.Contains)
	shoulda.SatisfyWith(t, actual, fmt.Sprintf("Cache size: %s", formatSizeWithRaw(1024)), strings.Contains)
	shoulda.SatisfyWith(t, actual, fmt.Sprintf("Oldest entry: %s", oldest.Local().Format(time.RFC3339)), strings.Contains)
	shoulda.SatisfyWith(t, actual, fmt.Sprintf("Newest entry: %s", newest.Local().Format(time.RFC3339)), strings.Contains)
	shoulda.SatisfyWith(t, actual, fmt.Sprintf("Disk total: %s", formatSizeWithRaw(10*1024)), strings.Contains)
	shoulda.SatisfyWith(t, actual, fmt.Sprintf("Disk used: %s (60.00%%)", formatSizeWithRaw(6*1024)), strings.Contains)
	shoulda.SatisfyWith(t, actual, fmt.Sprintf("Disk free: %s (40.00%%)", formatSizeWithRaw(4*1024)), strings.Contains)
	shoulda.SatisfyWith(t, actual, "Cache of total disk: 10.00%", strings.Contains)
}

func TestStatusTextFormattingEmpty(t *testing.T) {
	report := newLocalStatusReport("/tmp/cache", local.Stats{}, 100, 25)

	actual := renderLocalStatusText(report)

	shoulda.SatisfyWith(t, actual, "Cache entries: 0", strings.Contains)
	shoulda.SatisfyWith(t, actual, "Cache size: 0B (0 bytes)", strings.Contains)
	shoulda.SatisfyWith(t, actual, "Oldest entry: n/a", strings.Contains)
	shoulda.SatisfyWith(t, actual, "Newest entry: n/a", strings.Contains)
}

func TestStatusJSONCompact(t *testing.T) {
	oldest := time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC)
	newest := time.Date(2026, time.January, 3, 4, 5, 6, 0, time.UTC)
	report := newLocalStatusReport("/tmp/cache", local.Stats{
		Entries: 3,
		Bytes:   1024,
		Oldest:  &oldest,
		Newest:  &newest,
	}, 10*1024, 4*1024)

	actual, err := renderLocalStatusJSON(report)
	musta.NoError(t, err)
	shoulda.SatisfyWith(t, actual, "\n", strings.HasSuffix)

	var got jsonLocalStatus
	err = json.Unmarshal([]byte(actual), &got)
	musta.NoError(t, err)

	shoulda.BeEqual(t, got.Directory, "/tmp/cache")
	shoulda.BeEqual(t, got.Cache.Entries, 3)
	shoulda.BeEqual(t, got.Cache.Bytes, int64(1024))
	shoulda.NotBeZero(t, got.Cache.Human)
	musta.NotBeZero(t, got.Cache.Oldest)
	musta.NotBeZero(t, got.Cache.Newest)
	shoulda.BeEqual(t, *got.Cache.Oldest, oldest.Local().Format(time.RFC3339))
	shoulda.BeEqual(t, *got.Cache.Newest, newest.Local().Format(time.RFC3339))
	shoulda.BeEqual(t, got.Disk.TotalBytes, int64(10*1024))
	shoulda.BeEqual(t, got.Disk.UsedBytes, int64(6*1024))
	shoulda.BeEqual(t, got.Disk.FreeBytes, int64(4*1024))
	shoulda.BeEqual(t, got.Disk.UsedPercent, 60)
	shoulda.BeEqual(t, got.Disk.FreePercent, 40)
	shoulda.BeEqual(t, got.CacheOfTotalPercent, 10)
}
