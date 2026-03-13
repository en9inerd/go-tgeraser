GO=go
BUILD_DIR=build
DIST_DIR=dist
BINARY_NAME=$(shell basename $(PWD))
BINARY_PATH=$(BUILD_DIR)/$(BINARY_NAME)

all: build

build:
	$(GO) build -o $(BINARY_PATH) ./cmd/tgeraser/

build-prod:
	bash scripts/build.sh

clean:
	rm -rf $(BUILD_DIR)
	rm -rf $(DIST_DIR)

format:
	$(GO) fmt ./...

test:
	$(GO) test -race ./...

vet:
	$(GO) vet ./...

run:
	@test -f .env && set -a && . ./.env && set +a; $(GO) run ./cmd/tgeraser/

run-verbose:
	@test -f .env && set -a && . ./.env && set +a; $(GO) run ./cmd/tgeraser/ --verbose

.PHONY: all build build-prod clean format test vet run run-verbose
