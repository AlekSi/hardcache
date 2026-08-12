package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/AlekSi/hardcache/internal/caches/local"
	"github.com/AlekSi/hardcache/internal/unit"
	"github.com/AlekSi/shoulda"
	"github.com/AlekSi/shoulda/musta"
)

func TestLocalStatusOutput(t *testing.T) {
	t.Parallel()

	src := filepath.Join("internal", "testdata", "local")
	dst := t.TempDir()
	b, err := exec.Command("cp", "-a", src, dst).CombinedOutput()
	musta.NoErrorf(t, err, "%s", b)
	dir := filepath.Join(dst, "local")

	c := musta.NotFail(local.New(dir, nil, nil, slog.Default()))(t)
	stats := c.Status()
	total, free := musta.NotFail2(local.DiskInfo(dir))(t)
	output := newLocalStatusOutput(dir, stats, total, free)

	oldest := time.Date(2025, time.November, 17, 17, 12, 57, 524467000, time.UTC).Local().Format(time.RFC3339)
	newest := time.Date(2025, time.November, 17, 17, 13, 7, 284400000, time.UTC).Local().Format(time.RFC3339)

	shoulda.BeEqual(t, output.Cache.Entries, 1219)
	shoulda.BeEqual(t, output.Cache.Bytes, int64(109_518_524))
	shoulda.BeEqual(t, output.Cache.Human, "109MB")
	shoulda.BeEqual(t, *output.Cache.Oldest, oldest)
	shoulda.BeEqual(t, *output.Cache.Newest, newest)

	t.Run("text", func(t *testing.T) {
		actual := output.String()
		shoulda.SatisfyWith(t, actual, "Directory: "+dir, strings.Contains)
		shoulda.SatisfyWith(t, actual, "Cache entries: 1219", strings.Contains)
		shoulda.SatisfyWith(t, actual, "Cache size: 109MB (109518524 bytes)", strings.Contains)
		shoulda.SatisfyWith(t, actual, "Oldest entry: "+oldest, strings.Contains)
		shoulda.SatisfyWith(t, actual, "Newest entry: "+newest, strings.Contains)
		shoulda.SatisfyWith(t, actual,
			fmt.Sprintf("Disk total: %s (%d bytes)", unit.Bytes(total), total), strings.Contains)
		shoulda.SatisfyWith(t, actual,
			fmt.Sprintf("Disk free: %s (%d bytes)", unit.Bytes(free), free), strings.Contains)
	})

	t.Run("JSON", func(t *testing.T) {
		actual, err := json.Marshal(output)
		musta.NoError(t, err)
		shoulda.BeZero(t, strings.Contains(string(actual), "\n"))

		var got localStatusOutput
		musta.NoError(t, json.Unmarshal(actual, &got))
		shoulda.BeDeepEqual(t, got, output)
	})
}

func TestLocalStatusOutputEmpty(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	c := musta.NotFail(local.New(dir, nil, nil, slog.Default()))(t)
	stats := c.Status()
	total, free := musta.NotFail2(local.DiskInfo(dir))(t)
	output := newLocalStatusOutput(dir, stats, total, free)

	actual := output.String()
	shoulda.SatisfyWith(t, actual, "Cache entries: 0", strings.Contains)
	shoulda.SatisfyWith(t, actual, "Cache size: 0B (0 bytes)", strings.Contains)
	shoulda.SatisfyWith(t, actual, "Oldest entry: n/a", strings.Contains)
	shoulda.SatisfyWith(t, actual, "Newest entry: n/a", strings.Contains)
}
