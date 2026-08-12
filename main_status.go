package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"os"
	"strings"
	"time"

	"github.com/AlekSi/hardcache/internal/caches/local"
	"github.com/AlekSi/hardcache/internal/unit"
)

// localStatusReport contains all calculated values used by status output modes.
type localStatusReport struct {
	Directory           string
	CacheEntries        int
	CacheBytes          int64
	CacheOldest         *time.Time
	CacheNewest         *time.Time
	DiskTotalBytes      int64
	DiskUsedBytes       int64
	DiskFreeBytes       int64
	DiskUsedPercent     float64
	DiskFreePercent     float64
	CacheOfTotalPercent float64
}

type jsonLocalStatus struct {
	Directory string `json:"directory"`
	Cache     struct {
		Entries int     `json:"entries"`
		Bytes   int64   `json:"bytes"`
		Human   string  `json:"human"`
		Oldest  *string `json:"oldest"`
		Newest  *string `json:"newest"`
	} `json:"cache"`
	Disk struct {
		TotalBytes  int64   `json:"total_bytes"`
		TotalHuman  string  `json:"total_human"`
		UsedBytes   int64   `json:"used_bytes"`
		UsedHuman   string  `json:"used_human"`
		UsedPercent float64 `json:"used_percent"`
		FreeBytes   int64   `json:"free_bytes"`
		FreeHuman   string  `json:"free_human"`
		FreePercent float64 `json:"free_percent"`
	} `json:"disk"`
	CacheOfTotalPercent float64 `json:"cache_of_total_percent"`
}

func localStatus(dir string, asJSON bool, l *slog.Logger) error {
	c, err := local.New(dir, nil, nil, l)
	if err != nil {
		return err
	}

	cacheStats := c.Status()
	total, free, err := local.DiskInfo(dir)
	if err != nil {
		return err
	}

	report := newLocalStatusReport(dir, cacheStats, total, free)

	var out string
	if asJSON {
		out, err = renderLocalStatusJSON(report)
		if err != nil {
			return err
		}
	} else {
		out = renderLocalStatusText(report)
	}

	_, err = os.Stdout.WriteString(out)
	return err
}

func newLocalStatusReport(dir string, stats local.Stats, total, free int64) localStatusReport {
	used := total - free
	if used < 0 {
		used = 0
	}

	return localStatusReport{
		Directory:           dir,
		CacheEntries:        stats.Entries,
		CacheBytes:          stats.Bytes,
		CacheOldest:         stats.Oldest,
		CacheNewest:         stats.Newest,
		DiskTotalBytes:      total,
		DiskUsedBytes:       used,
		DiskFreeBytes:       free,
		DiskUsedPercent:     round2(percentage(used, total)),
		DiskFreePercent:     round2(percentage(free, total)),
		CacheOfTotalPercent: round2(percentage(stats.Bytes, total)),
	}
}

func renderLocalStatusText(report localStatusReport) string {
	var b strings.Builder

	fmt.Fprintf(&b, "Directory: %s\n", report.Directory)
	fmt.Fprintf(&b, "Cache entries: %d\n", report.CacheEntries)
	fmt.Fprintf(&b, "Cache size: %s\n", formatSizeWithRaw(report.CacheBytes))
	fmt.Fprintf(&b, "Oldest entry: %s\n", formatLocalTime(report.CacheOldest))
	fmt.Fprintf(&b, "Newest entry: %s\n", formatLocalTime(report.CacheNewest))
	fmt.Fprintf(&b, "Disk total: %s\n", formatSizeWithRaw(report.DiskTotalBytes))
	fmt.Fprintf(&b, "Disk used: %s (%.2f%%)\n", formatSizeWithRaw(report.DiskUsedBytes), report.DiskUsedPercent)
	fmt.Fprintf(&b, "Disk free: %s (%.2f%%)\n", formatSizeWithRaw(report.DiskFreeBytes), report.DiskFreePercent)
	fmt.Fprintf(&b, "Cache of total disk: %.2f%%\n", report.CacheOfTotalPercent)

	return b.String()
}

func renderLocalStatusJSON(report localStatusReport) (string, error) {
	payload := jsonLocalStatus{
		Directory:           report.Directory,
		CacheOfTotalPercent: report.CacheOfTotalPercent,
	}
	payload.Cache.Entries = report.CacheEntries
	payload.Cache.Bytes = report.CacheBytes
	payload.Cache.Human = unit.Bytes(report.CacheBytes).String()
	payload.Cache.Oldest = formatLocalTimePtr(report.CacheOldest)
	payload.Cache.Newest = formatLocalTimePtr(report.CacheNewest)
	payload.Disk.TotalBytes = report.DiskTotalBytes
	payload.Disk.TotalHuman = unit.Bytes(report.DiskTotalBytes).String()
	payload.Disk.UsedBytes = report.DiskUsedBytes
	payload.Disk.UsedHuman = unit.Bytes(report.DiskUsedBytes).String()
	payload.Disk.UsedPercent = report.DiskUsedPercent
	payload.Disk.FreeBytes = report.DiskFreeBytes
	payload.Disk.FreeHuman = unit.Bytes(report.DiskFreeBytes).String()
	payload.Disk.FreePercent = report.DiskFreePercent

	res, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	return string(res) + "\n", nil
}

func formatSizeWithRaw(size int64) string {
	return fmt.Sprintf("%s (%d bytes)", unit.Bytes(size).String(), size)
}

func formatLocalTime(ts *time.Time) string {
	if ts == nil {
		return "n/a"
	}

	return ts.Local().Format(time.RFC3339)
}

func formatLocalTimePtr(ts *time.Time) *string {
	if ts == nil {
		return nil
	}

	res := ts.Local().Format(time.RFC3339)
	return &res
}

func percentage(value, total int64) float64 {
	if total <= 0 {
		return 0
	}

	return float64(value) / float64(total) * 100
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}
