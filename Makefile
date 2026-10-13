# Native build. Docker is an optional toolchain, not a runtime requirement.
GO ?= go
GOOS ?= $(shell $(GO) env GOOS)
EXT := so
ifeq ($(GOOS),darwin)
EXT := dylib
endif
ifeq ($(GOOS),windows)
EXT := dll
endif
OUTPUT ?= build/factory.$(EXT)

.PHONY: build test vet verify
build:
	CGO_ENABLED=1 $(GO) build -buildmode=c-shared -o $(OUTPUT) ./cmd/factory

test:
	$(GO) test -count=1 ./...

vet:
	$(GO) vet ./...

verify: test vet build
