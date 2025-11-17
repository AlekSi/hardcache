#!/bin/bash

set -ex

export GOCACHE=$(pwd)/local
rm -fr local
go version
date -Iseconds

go install -v math/bits
sleep 1

go install -v math
sleep 1

go test -count=1 math/bits
sleep 1

go test -count=1 math
sleep 1

go tool mytool
