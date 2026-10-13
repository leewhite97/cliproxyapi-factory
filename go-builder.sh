#!/bin/sh
set -eu
BASE=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
CACHE=${FACTORY_BUILD_CACHE:-${XDG_CACHE_HOME:-$HOME/.cache}/cliproxyapi-factory}
mkdir -p "$CACHE/go-build" "$CACHE/go-mod"
CACHE=$(CDPATH= cd -- "$CACHE" && pwd)
exec docker run --rm --user "$(id -u):$(id -g)" --cpus 2 \
  -e GOMAXPROCS=2 -e GOCACHE=/cache -e GOPATH=/gopath \
  -v "$BASE:/work" -v "$CACHE/go-build:/cache" -v "$CACHE/go-mod:/gopath" \
  -w /work \
  golang@sha256:2a0ba12e116687098780d3ce700f9ce3cb340783779646aafbabed748fa6677c "$@"
