package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/alecthomas/kong"

	"github.com/AlekSi/hardcache/internal/caches/local"
	"github.com/AlekSi/hardcache/internal/caches/stat"
	"github.com/AlekSi/hardcache/internal/prog"
	"github.com/AlekSi/hardcache/internal/sigterm"
	"github.com/AlekSi/hardcache/internal/unit"
)

// cli represents CLI arguments and flags.
//
//nolint:vet // for readability
var cli struct {
	Local struct {
		Dir       string        `default:"${local_dir}" type:"path" help:"Directory to use."`
		UnusedFor unit.Duration `default:"5d" help:"Remove entries unused for this duration."`
		MaxSize   unit.Bytes    `default:"0GB" help:"Remove entries if cache size is larger than this."`

		Use  struct{} `cmd:"" hidden:""`
		Trim struct{} `cmd:"" help:"Trim local cache."`
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

// localCache returns local cache, configured according to CLI flags.
func localCache(l *slog.Logger) (*local.Cache, error) {
	var cutoff *time.Time
	if cli.Local.UnusedFor > 0 {
		d := time.Duration(cli.Local.UnusedFor)
		if d > 5*unit.Day {
			l.Info("Note: this command should be invoked more often than once per day to keep the cache.")
		}

		c := time.Now().Add(-d)
		cutoff = &c
	}

	var maxSize *int64
	if cli.Local.MaxSize > 0 {
		m := int64(cli.Local.MaxSize)
		maxSize = &m
	}

	return local.New(cli.Local.Dir, cutoff, maxSize, l)
}

func main() {
	opts := []kong.Option{
		kong.Name("hardcache"),
		kong.Description("Tool for managing the Go build cache."),
		kong.Vars{
			"local_dir": GOCACHE(),
		},
		kong.ConfigureHelp(kong.HelpOptions{
			Tree: true,
		}),
	}
	kongCtx := kong.Parse(&cli, opts...)

	log.SetPrefix("hardcache: ")
	log.SetFlags(0)

	if cli.Debug {
		slog.SetLogLoggerLevel(slog.LevelDebug)
	}

	l := slog.Default()

	ctx, cancel := sigterm.Ctx(context.Background())
	defer cancel()

	switch kongCtx.Command() {
	case "local use":
		c, err := localCache(l)
		kongCtx.FatalIfErrorf(err)

		p := prog.New(stat.New(c, l), l, os.Stdin, os.Stdout)
		err = p.Run(ctx)
		kongCtx.FatalIfErrorf(err)

	case "local trim":
		c, err := localCache(l)
		kongCtx.FatalIfErrorf(err)

		before, freed := c.TrimForce()

		l.Info(
			"Local cache trimmed",
			slog.Int64("before", int64(unit.Bytes(before))), slog.Int64("freed", int64(unit.Bytes(freed))),
		)

	default:
		kongCtx.Fatalf("unknown command: %q", kongCtx.Command())
	}
}
