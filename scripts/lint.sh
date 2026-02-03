#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

cd "$PROJECT_ROOT"

if ! command -v golangci-lint &> /dev/null; then
    echo "golangci-lint not found. Installing..."
    go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
fi

echo "Running golangci-lint..."

if [[ "$1" == "--fix" ]]; then
    echo "Auto-fixing issues..."
    golangci-lint run --fix ./...
else
    golangci-lint run ./...
fi

echo "Linting passed!"
