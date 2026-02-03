#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

cd "$PROJECT_ROOT"

echo "Running Carv tests..."

if [[ "$1" == "--coverage" ]]; then
    echo "Running with coverage..."
    mkdir -p build
    go test -coverprofile=build/coverage.out ./...
    go tool cover -func=build/coverage.out
    echo ""
    echo "Coverage report: build/coverage.out"
    echo "Run 'go tool cover -html=build/coverage.out' for HTML report"
elif [[ "$1" == "--race" ]]; then
    echo "Running with race detector..."
    go test -race -v ./...
elif [[ "$1" == "--verbose" ]] || [[ "$1" == "-v" ]]; then
    go test -v ./...
else
    go test ./...
fi

echo "All tests passed!"
