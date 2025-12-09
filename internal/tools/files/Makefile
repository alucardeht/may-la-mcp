.PHONY: test test-unit test-integration test-coverage clean lint fmt

test:
	@echo "Running all tests..."
	go test -v -race ./...

test-unit:
	@echo "Running unit tests..."
	go test -v -run "^Test[^I]" ./...

test-integration:
	@echo "Running integration tests..."
	go test -v -run "^TestComplete" ./...

test-coverage:
	@echo "Generating coverage report..."
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

bench:
	@echo "Running benchmarks..."
	go test -bench=. -benchmem ./...

clean:
	@echo "Cleaning..."
	rm -f coverage.out coverage.html

fmt:
	@echo "Formatting code..."
	go fmt ./...

lint:
	@echo "Linting code..."
	go vet ./...

help:
	@echo "Available targets:"
	@echo "  make test            - Run all tests"
	@echo "  make test-unit       - Run unit tests only"
	@echo "  make test-integration- Run integration tests only"
	@echo "  make test-coverage   - Generate coverage report"
	@echo "  make bench           - Run benchmarks"
	@echo "  make fmt             - Format code"
	@echo "  make lint            - Lint code"
	@echo "  make clean           - Clean build artifacts"
