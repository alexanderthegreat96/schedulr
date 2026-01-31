.PHONY: test test-verbose test-coverage test-core test-cmd clean help build run

help:
	@echo "Available targets:"
	@echo "  test              - Run all tests"
	@echo "  test-verbose      - Run all tests with verbose output"
	@echo "  test-coverage     - Run tests with coverage report"
	@echo "  test-core         - Run only core package tests"
	@echo "  test-cmd          - Run only cmd package tests"
	@echo "  clean             - Remove test cache and build artifacts"
	@echo "  build             - Build the application"
	@echo "  run               - Run the application"

test:
	go test ./...

test-verbose:
	go test -v ./...

test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

test-core:
	go test -v ./core

test-cmd:
	go test -v ./cmd

clean:
	go clean -testcache
	rm -f coverage.out coverage.html
	rm -rf core/app-logs core/task-logs

build:
	go build -o bin/schedulr

run: build
	./bin/schedulr
