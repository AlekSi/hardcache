package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/alecthomas/kong"

	"github.com/AlekSi/hardcache/internal/caches/local"
	"github.com/AlekSi/hardcache/internal/sigterm"
	"github.com/AlekSi/hardcache/internal/unit"
)

// cli represents CLI arguments and flags.
//
//nolint:vet // for readability
var cli struct {
	Local struct {
		Dir       string        `default:"${local_dir_default}" type:"path" help:"Directory to use."`
		UnusedFor unit.Duration `default:"5d" help:"Always remove entries unused for this duration. Pass 0 to disable."`
		MaxSize   string        `default:"0GB" help:"${local_max_size_help}"`

		Trim  struct{} `cmd:"" help:"Trim local cache."`
		Trimd struct {
			Interval unit.Duration `short:"i" default:"1h" help:"Interval between trimmings."`
		} `cmd:"" help:"Trim local cache continuously."`
	} `cmd:""`

	Debug bool `help:"Enable debug logging."`
}

// GOCACHE returns the Go build cache directory.
var GOCACHE = sync.OnceValue(func() string {
	// in theory, someone might not have go in the PATH
	if v := os.Getenv("GOCACHE"); v != "" {
		return v
	}

	// that handles `go env -w` and the default value
	b, err := exec.Command("go", "env", "GOCACHE").Output()
	if err != nil {
		panic(err)
	}

	return strings.TrimSpace(string(b))
})

// localTrim force-trims local cache according to CLI flags.
func localTrim(l *slog.Logger) error {
	var cutoff *time.Time
	if cli.Local.UnusedFor > 0 {
		c := time.Now().Add(-time.Duration(cli.Local.UnusedFor))
		cutoff = &c
	}

	var maxSize *int64
	maxSizeS, p := strings.CutSuffix(cli.Local.MaxSize, "%")
	maxSizeI, err := strconv.ParseInt(maxSizeS, 10, 64)
	if err != nil {
		return err
	}

	if p {
		var total int64
		if total, _, err = local.DiskInfo(cli.Local.Dir); err != nil {
			return err
		}

		maxSizeI = total * maxSizeI / 100
		l.Debug("Calculated max size from percentage", slog.Int64("disk_size", total), slog.Int64("max_size", maxSizeI))
	}

	if maxSizeI > 0 {
		maxSize = &maxSizeI
	}

	c, err := local.New(cli.Local.Dir, cutoff, maxSize, l)
	if err != nil {
		return err
	}

	before, freed := c.TrimForce()
	l.Info(
		"Local cache trimmed",
		slog.Int64("before", int64(unit.Bytes(before))), slog.Int64("freed", int64(unit.Bytes(freed))),
	)

	return nil
}

func main() {
	opts := []kong.Option{
		kong.Name("hardcache"),
		kong.Description("Tool for managing the Go build cache."),
		kong.Vars{
			"local_dir_default": GOCACHE(),
			"local_max_size_help": "Remove entries, starting from least recently used, " +
				"if cache size is larger than this value. " +
				"Supports MiB, GB, etc. suffixes, or percentage of the total disk space (e.g., 5%). " +
				"Pass 0 to disable.",
		},
		kong.ConfigureHelp(kong.HelpOptions{
			Tree:           true,
			WrapUpperBound: 120,
		}),
	}
	kongCtx := kong.Parse(&cli, opts...)

	log.SetPrefix("hardcache: ")
	log.SetFlags(log.Lmicroseconds)

	if cli.Debug {
		slog.SetLogLoggerLevel(slog.LevelDebug)
	}

	l := slog.Default()

	ctx, cancel := sigterm.Ctx(context.Background())
	defer cancel()

	switch kongCtx.Command() {
	case "local trim":
		if time.Duration(cli.Local.UnusedFor) > 5*24*time.Hour {
			l.Info("Note: this command should be invoked more often than once per day to keep the cache.")
		}

		err := localTrim(l)
		kongCtx.FatalIfErrorf(err)

	case "local trimd":
		for {
			err := localTrim(l)
			kongCtx.FatalIfErrorf(err)

			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Duration(cli.Local.Trimd.Interval)):
				// nothing
			}
		}

	default:
		kongCtx.Fatalf("unknown command: %q", kongCtx.Command())
	}
}
