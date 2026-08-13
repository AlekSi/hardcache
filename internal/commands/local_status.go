package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math"
	"time"

	"github.com/AlekSi/hardcache/internal/caches/local"
	"github.com/AlekSi/hardcache/internal/unit"
)

type localStatusOutput struct {
	Directory string `json:"directory"`
	Cache     struct {
		Entries           int     `json:"entries"`
		Bytes             int64   `json:"bytes"`
		Human             string  `json:"human"`
		Oldest            *string `json:"oldest"`
		Newest            *string `json:"newest"`
		LeastRecentlyUsed *string `json:"least_recently_used"`
		MostRecentlyUsed  *string `json:"most_recently_used"`
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

// LocalStatusOpts contains flag values for [LocalStatus].
type LocalStatusOpts struct {
	Dir  string
	JSON bool
}

// LocalStatus writes local cache and disk usage statistics to out.
func LocalStatus(opts *LocalStatusOpts, out io.Writer, l *slog.Logger) error {
	c, err := local.New(opts.Dir, nil, nil, l)
	if err != nil {
		return err
	}

	stats := c.Stats()
	total, free, err := local.DiskInfo(opts.Dir)
	if err != nil {
		return err
	}

	output := newLocalStatusOutput(opts.Dir, stats, total, free)
	if opts.JSON {
		return json.NewEncoder(out).Encode(output)
	}

	_, err = fmt.Fprint(out, output)
	return err
}

func newLocalStatusOutput(dir string, stats *local.Stats, total, free int64) localStatusOutput {
	used := max(total-free, 0)
	percent := func(value int64) float64 {
		if total <= 0 {
			return 0
		}

		return math.Round(float64(value)/float64(total)*10_000) / 100
	}

	res := localStatusOutput{
		Directory:           dir,
		CacheOfTotalPercent: percent(stats.Bytes),
	}
	res.Cache.Entries = stats.Entries
	res.Cache.Bytes = stats.Bytes
	res.Cache.Human = unit.Bytes(stats.Bytes).String()
	formatTime := func(t *time.Time) *string {
		if t == nil {
			return nil
		}

		return new(t.Local().Format(time.RFC3339))
	}
	res.Cache.Oldest = formatTime(stats.Oldest)
	res.Cache.Newest = formatTime(stats.Newest)
	res.Cache.LeastRecentlyUsed = formatTime(stats.LeastRecentlyUsed)
	res.Cache.MostRecentlyUsed = formatTime(stats.MostRecentlyUsed)
	res.Disk.TotalBytes = total
	res.Disk.TotalHuman = unit.Bytes(total).String()
	res.Disk.UsedBytes = used
	res.Disk.UsedHuman = unit.Bytes(used).String()
	res.Disk.UsedPercent = percent(used)
	res.Disk.FreeBytes = free
	res.Disk.FreeHuman = unit.Bytes(free).String()
	res.Disk.FreePercent = percent(free)

	return res
}

func (s localStatusOutput) String() string {
	formatTime := func(t *string) string {
		if t == nil {
			return "n/a"
		}

		return *t
	}

	return fmt.Sprintf(`Directory: %s
Cache entries: %d
Cache size: %s (%d bytes)
Oldest entry: %s
Newest entry: %s
Least recently used: %s
Most recently used: %s
Disk total: %s (%d bytes)
Disk used: %s (%d bytes) (%.2f%%)
Disk free: %s (%d bytes) (%.2f%%)
Cache of total disk: %.2f%%
`,
		s.Directory,
		s.Cache.Entries,
		s.Cache.Human, s.Cache.Bytes,
		formatTime(s.Cache.Oldest),
		formatTime(s.Cache.Newest),
		formatTime(s.Cache.LeastRecentlyUsed),
		formatTime(s.Cache.MostRecentlyUsed),
		s.Disk.TotalHuman, s.Disk.TotalBytes,
		s.Disk.UsedHuman, s.Disk.UsedBytes, s.Disk.UsedPercent,
		s.Disk.FreeHuman, s.Disk.FreeBytes, s.Disk.FreePercent,
		s.CacheOfTotalPercent,
	)
}
