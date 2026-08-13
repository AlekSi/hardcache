all: test run

init:
	go mod tidy
	go -C internal/testdata build -o ../../bin/modtimes modtimes.go
	cd internal/testdata && ../../bin/modtimes -apply

test:
	go test -race -count=1 ./...

run:
	go build -race -o bin/
	bin/hardcache --help
	mkdir -p tmp/cache
	env GORACE=halt_on_error=1 GOCACHEPROG='bin/hardcache local use --dir=tmp/cache --debug' go build -o bin/
