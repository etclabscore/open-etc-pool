# Convenience wrapper around the standard Go and npm commands. Nothing here is
# required — `go build -o open-etc-pool .` and `go test -race ./...` work on
# their own; these targets just stamp the version and bundle the common steps.

VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -X main.version=$(VERSION)

.PHONY: all build test vet fmt web clean

all: build

# Build the pool binary with the version stamped in.
build:
	go build -ldflags "$(LDFLAGS)" -o open-etc-pool .

# Run the test suite with the race detector. Storage/API/payout tests need a
# local Redis on 127.0.0.1:6379 (e.g. `redis-server` or `docker run -p 6379:6379 redis:8`).
test:
	go test -race ./...

vet:
	go vet ./...

fmt:
	gofmt -w -l .

# Build the static frontend into web/dist.
web:
	cd web && npm ci && npm run build

clean:
	rm -f open-etc-pool
	go clean -cache
