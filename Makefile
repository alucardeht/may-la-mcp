.PHONY: build test clean install fmt lint

BINARY_NAME=mayla
BUILD_DIR=bin

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS=-ldflags "-s -w \
	-X github.com/alucardeht/may-la-mcp/pkg/version.Version=$(VERSION) \
	-X github.com/alucardeht/may-la-mcp/pkg/version.Commit=$(COMMIT) \
	-X github.com/alucardeht/may-la-mcp/pkg/version.BuildDate=$(BUILD_DATE)"

build:
	CGO_ENABLED=0 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/mayla

test:
	go test -v ./...

clean:
	rm -rf $(BUILD_DIR)

install: build
	cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/

fmt:
	go fmt ./...

lint:
	golangci-lint run

.DEFAULT_GOAL := build
