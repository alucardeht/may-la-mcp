.PHONY: build build-daemon build-all test test-e2e bench clean install run-daemon fmt lint

BINARY_NAME=mayla
DAEMON_NAME=mayla-daemon
BUILD_DIR=bin
BUILD_FLAGS=-tags sqlite_fts5

build:
	CGO_ENABLED=1 go build $(BUILD_FLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/mayla

build-daemon:
	CGO_ENABLED=1 go build $(BUILD_FLAGS) -o $(BUILD_DIR)/$(DAEMON_NAME) ./cmd/mayla-daemon

build-all: build build-daemon

test:
	CGO_ENABLED=1 go test $(BUILD_FLAGS) -v ./...

test-e2e:
	CGO_ENABLED=1 go test $(BUILD_FLAGS) -v ./tests/... -timeout 5m

bench:
	CGO_ENABLED=1 go test $(BUILD_FLAGS) -bench=. -benchmem ./...

clean:
	rm -rf $(BUILD_DIR)
	rm -f ~/.mayla/daemon.sock

install: build-all
	cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/
	cp $(BUILD_DIR)/$(DAEMON_NAME) /usr/local/bin/

run-daemon:
	./$(BUILD_DIR)/$(DAEMON_NAME)

fmt:
	go fmt ./...

lint:
	golangci-lint run

.DEFAULT_GOAL := build-all
