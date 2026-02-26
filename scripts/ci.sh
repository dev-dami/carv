#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

cd "$PROJECT_ROOT"

echo "========================================"
echo "Carv CI Pipeline"
echo "========================================"
echo ""

echo "[1/4] Formatting check..."
if ! go fmt ./... | grep -q .; then
    echo "  Formatting OK"
else
    echo "  Warning: Some files were not formatted"
fi

echo ""
echo "[2/4] Running linter..."
if command -v golangci-lint &> /dev/null; then
    if golangci-lint run ./... 2>&1; then
        echo "  Linting OK"
    else
        echo "  Linting found issues (see above)"
        exit 1
    fi
else
    echo "  Skipping (golangci-lint not installed)"
fi

echo ""
echo "[3/4] Running tests..."
if go test -race ./... 2>&1; then
    echo "  Tests OK"
else
    echo "  Tests failed"
    exit 1
fi

echo ""
echo "[4/4] Building..."
mkdir -p build
if go build -o build/carv ./cmd/carv/ 2>&1; then
    echo "  Build OK"
else
    echo "  Build failed"
    exit 1
fi

echo ""
echo "[5/5] Running docs samples..."
if ./build/carv run docs/samples/hello.carv > /dev/null 2>&1; then
    echo "  docs/samples/hello.carv OK"
else
    echo "  docs/samples/hello.carv failed"
fi

echo ""
echo "========================================"
echo "CI Pipeline Complete!"
echo "========================================"
