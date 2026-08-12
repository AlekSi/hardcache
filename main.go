package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/exec"
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
		Dir string `default:"${local_dir_default}" type:"path" help:"Directory to use."`

		Status struct {
			JSON bool `help:"Output as compact JSON."`
		} `cmd:"" help:"Show local cache status."`

		Trim struct {
			UnusedFor unit.Duration `default:"5d" help:"Always remove entries unused for this duration. Pass 0 to disable."`
			MaxSize   string        `default:"0GB" help:"${local_max_size_help}"`
		} `cmd:"" help:"Trim local cache."`

		Trimd struct {
			UnusedFor unit.Duration `default:"5d" help:"Always remove entries unused for this duration. Pass 0 to disable."`
			MaxSize   string        `default:"0GB" help:"${local_max_size_help}"`
			Interval  unit.Duration `short:"i" default:"1h" help:"Interval between trimmings."`
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

// localTrim force-trims local cache according to parameters.
func localTrim(dir string, unusedFor unit.Duration, maxSizeValue string, l *slog.Logger) error {
	if unusedFor < 0 {
		return fmt.Errorf("--unused-for cannot be negative: %d", unusedFor)
	}

	var cutoff *time.Time
	if unusedFor > 0 {
		c := time.Now().Add(-time.Duration(unusedFor))
		cutoff = &c
	}

	var b unit.Bytes
	if strings.HasSuffix(maxSizeValue, "%") {
		var p unit.Percentage
		if err := p.UnmarshalText([]byte(maxSizeValue)); err != nil {
			return err
		}

		total, _, err := local.DiskInfo(dir)
		if err != nil {
			return err
		}

		b = unit.Bytes(total / 100 * int64(p))

		l.Debug(
			"Calculated max size from percentage of total disk size",
			slog.Int64("disk_size_bytes", total),
			slog.String("disk_size", unit.Bytes(total).String()),
			slog.Int64("max_size_bytes", int64(b)),
			slog.String("max_size", b.String()),
		)
	} else {
		if err := b.UnmarshalText([]byte(maxSizeValue)); err != nil {
			return err
		}

		l.Debug("Max size", slog.Int64("max_size_bytes", int64(b)), slog.String("max_size", b.String()))
	}

	if b < 0 {
		return fmt.Errorf("--max-size cannot be negative: %d", b)
	}

	var maxSize *int64
	if b > 0 {
		maxSize = (*int64)(&b)
	}

	c, err := local.New(dir, cutoff, maxSize, l)
	if err != nil {
		return err
	}

	before, freed := c.TrimForce()
	l.Debug(
		"Local cache trimmed",
		slog.Int64("before_bytes", before), slog.Int64("freed_bytes", freed),
	)
	l.Info(
		"Local cache trimmed",
		slog.String("before", unit.Bytes(before).String()), slog.String("freed", unit.Bytes(freed).String()),
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
	case "local status":
		err := localStatus(cli.Local.Dir, cli.Local.Status.JSON, l)
		kongCtx.FatalIfErrorf(err)

	case "local trim":
		if time.Duration(cli.Local.Trim.UnusedFor) > 5*24*time.Hour {
			l.Info("Note: this command should be invoked more often than once per day to keep the cache.")
		}

		err := localTrim(cli.Local.Dir, cli.Local.Trim.UnusedFor, cli.Local.Trim.MaxSize, l)
		kongCtx.FatalIfErrorf(err)

	case "local trimd":
		for {
			err := localTrim(cli.Local.Dir, cli.Local.Trimd.UnusedFor, cli.Local.Trimd.MaxSize, l)
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
