all: test run

init:
	go mod tidy

test:
	go test -race -count=1 ./...

run:
	go build -race -o bin/
	bin/hardcache --help
	mkdir -p tmp/cache
