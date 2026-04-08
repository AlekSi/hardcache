module github.com/AlekSi/hardcache

go 1.25.0

// https://go.dev/doc/godebug#go-120
// https://pkg.go.dev/archive/tar#Reader.Next
// https://pkg.go.dev/archive/zip#OpenReader
godebug (
	tarinsecurepath=0
	zipinsecurepath=0
)

require (
	github.com/AlekSi/lazyerrors v0.6.0
	github.com/alecthomas/kong v1.15.0
	github.com/alecthomas/units v0.0.0-20240927000941-0f3dac36c52b
	github.com/stretchr/testify v1.11.1
	github.com/xhit/go-str2duration/v2 v2.1.0
	golang.org/x/sys v0.42.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
