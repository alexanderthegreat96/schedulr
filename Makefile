.PHONY: help test test-verbose test-coverage test-race test-core test-cmd build-local run clean-cache build clean

# test*, build-local, run and clean-cache need nothing but the Go toolchain and
# work on any platform. build and clean shell out to POSIX tools, so on Windows
# they want Git Bash or MSYS2.

help:
	@echo "Available targets:"
	@echo ""
	@echo "  Portable (Go toolchain only):"
	@echo "    test            - Run all tests"
	@echo "    test-verbose    - Run all tests with verbose output"
	@echo "    test-coverage   - Run tests with coverage report (coverage.html)"
	@echo "    test-race       - Run all tests under the race detector"
	@echo "    test-core       - Run only core package tests"
	@echo "    test-cmd        - Run only cmd package tests"
	@echo "    build-local     - Build a binary for this machine into bin/"
	@echo "    run             - Run the application from source"
	@echo "    clean-cache     - Clear the Go test cache"
	@echo ""
	@echo "  Needs a POSIX shell (bash / Git Bash / MSYS2):"
	@echo "    build           - Cross-build every release target via build.sh"
	@echo "    clean           - clean-cache plus removing generated files"

test:
	go test ./...

test-verbose:
	go test -v ./...

test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

test-race:
	go test -race ./...

test-core:
	go test -v ./core

test-cmd:
	go test -v ./cmd

# -o with a trailing separator writes the binary into that directory under its
# own name, so this picks up the .exe suffix on Windows by itself
build-local:
	go build -o bin/ .

# `run` used to depend on build and then execute ./bin/schedulr, which build.sh
# never produces - it names its output schedulr-<os>-<arch>-<timestamp> - so the
# target was broken on every platform. Running from source needs no artifact.
run:
	go run .

clean-cache:
	go clean -testcache

clean: clean-cache
	rm -f coverage.out coverage.html
	rm -rf core/app-logs core/task-logs
