# hardcache

![Hardcache logo](hardcache.jpeg)

Hardcache is a tool for managing the Go build cache.

The initial public version supports only a more flexible trimming policy of a standard local cache.
More functionality will be published soon, including support for `GOCACHEPROG`.

## Installation

```
go install github.com/AlekSi/hardcache@latest
```

Alternatively, just run it with

```
go run github.com/AlekSi/hardcache@latest -h
```

## Usage

### Local cache

Commands for a standard local cache.
The cache location defaults to the value of `go env GOCACHE` (supporting `go env -w`).
It can be overridden with `--dir` flag:

```
hardcache local --dir=/tmp/cache ...
```

#### Manual trimming

Go standard build cache does not support [disabling trimming](https://github.com/golang/go/issues/69565),
[trimming based on disk usage](https://github.com/golang/go/issues/29561),
or [making trimming cutoff configurable](https://github.com/golang/go/issues/69879).
`hardcache local trim` command makes all that possible while staying compatible with the standard build cache
and all `go` commands (by reusing the `go` command internal code).

There are, however, two downsides:
1. The implementation has to traverse the entire cache directory tree, which makes it relatively slow.
   It is fast enough for manual usage, but the slowdown would be noticeable
   if it were hooked into most `go` commands via `GOCACHEPROG`.
2. Because of the previous reason, if you want to keep your cache for longer than 5 days,
   you have to run `hardcache local trim` manually at least every 23 hours (for example, twice per day)
   to prevent the standard trimming mechanism from triggering.

See the next section for the solution to those problems.

```
hardcache local trim --unused-for=2w --max-size=10GB
```

This command would first remove cache entries that were unused for 2 weeks.
Then, if the remaining cache size is greater than 10GB, it will remove the least-recently used items
until the cache is small enough.

Maximum cache size can also be configured as a percentage of the total disk space, for example: `--max-size=5%`.

#### Continuous trimming

Both problems above can be solved by running trimming continuously:

```
hardcache local trimd --unused-for=2w --max-size=10GB --interval=1h
```

Using `trimd` subcommand instead of `trim` triggers trimming every specified interval.

## Credits

This tool is written by me, Alexey Palazhchenko,
but it uses the internal Go code from https://github.com/golang/go/tree/master/src/cmd/go/internal.
"The Go Authors" remain the copyright holders, who, conveniently, include me.
