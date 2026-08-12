package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/alecthomas/kong"

	"github.com/AlekSi/hardcache/internal/commands"
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
		err := commands.LocalStatus(cli.Local.Dir, cli.Local.Status.JSON, os.Stdout, l)
		kongCtx.FatalIfErrorf(err)

	case "local trim":
		err := commands.LocalTrim(cli.Local.Dir, cli.Local.Trim.UnusedFor, cli.Local.Trim.MaxSize, l)
		kongCtx.FatalIfErrorf(err)

	case "local trimd":
		err := commands.LocalTrimd(
			ctx, cli.Local.Dir, cli.Local.Trimd.UnusedFor, cli.Local.Trimd.MaxSize, cli.Local.Trimd.Interval, l,
		)
		kongCtx.FatalIfErrorf(err)

	default:
		kongCtx.Fatalf("unknown command: %q", kongCtx.Command())
	}
}
