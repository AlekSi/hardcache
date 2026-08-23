module github.com/AlekSi/hardcache

go 1.26

// https://go.dev/doc/godebug#go-120
// https://pkg.go.dev/archive/tar#Reader.Next
// https://pkg.go.dev/archive/zip#OpenReader
godebug (
	tarinsecurepath=0
	zipinsecurepath=0
)

require (
	github.com/AlekSi/lazyerrors v0.6.0
	github.com/AlekSi/shoulda v0.0.0-20260823083631-9dc9dbf21e59
	github.com/alecthomas/kong v1.16.1
	github.com/alecthomas/units v0.0.0-20240927000941-0f3dac36c52b
	github.com/xhit/go-str2duration/v2 v2.1.0
	golang.org/x/sys v0.47.0
)

require github.com/sanity-io/litter v1.5.8 // indirect
