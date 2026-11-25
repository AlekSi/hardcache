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
			UnusedFor unit.Duration `default:"5d" help:"${local_unused_for_help}"`
			MaxSize   string        `default:"0GB" help:"${local_max_size_help}"`
		} `cmd:"" help:"Trim local cache."`

		Trimd struct {
			UnusedFor unit.Duration `default:"5d" help:"${local_unused_for_help}"`
			MaxSize   string        `default:"0GB" help:"${local_max_size_help}"`
			Interval  unit.Duration `short:"i" default:"1h" help:"Interval between trimmings."`
		} `cmd:"" help:"Trim local cache continuously."`
	} `cmd:""`

	Debug bool `help:"Enable debug logging."`
}

func main() {
	opts := []kong.Option{
		kong.Name("hardcache"),
		kong.Description("Tool for managing the Go build cache."),
		kong.Vars{
			"local_dir_default":     GOCACHE(),
			"local_unused_for_help": "Always remove entries unused for this duration. Pass 0 to disable.",
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

	var err error
	switch kongCtx.Command() {
	case "local status":
		err = commands.LocalStatus(&commands.LocalStatusOpts{
			Dir:  cli.Local.Dir,
			JSON: cli.Local.Status.JSON,
		}, os.Stdout, l)

	case "local trim":
		err = commands.LocalTrim(&commands.LocalTrimOpts{
			Dir:       cli.Local.Dir,
			UnusedFor: cli.Local.Trim.UnusedFor,
			MaxSize:   cli.Local.Trim.MaxSize,
		}, l)

	case "local trimd":
		err = commands.LocalTrimd(ctx, &commands.LocalTrimdOpts{
			Dir:       cli.Local.Dir,
			UnusedFor: cli.Local.Trimd.UnusedFor,
			MaxSize:   cli.Local.Trimd.MaxSize,
			Interval:  cli.Local.Trimd.Interval,
		}, l)

	default:
		kongCtx.Fatalf("unknown command: %q", kongCtx.Command())
	}

	kongCtx.FatalIfErrorf(err)
}
