#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

cd "$PROJECT_ROOT"

VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_DIR="build"
BINARY_NAME="carv"

echo "Building Carv $VERSION..."

mkdir -p "$BUILD_DIR"

if [[ "$1" == "--release" ]]; then
    echo "Release build (optimized)..."
    go build -ldflags "-s -w -X main.version=$VERSION" -o "$BUILD_DIR/$BINARY_NAME" ./cmd/carv/
else
    go build -ldflags "-X main.version=$VERSION" -o "$BUILD_DIR/$BINARY_NAME" ./cmd/carv/
fi

echo "Built: $BUILD_DIR/$BINARY_NAME"
"$BUILD_DIR/$BINARY_NAME" --version 2>/dev/null || echo "Version: $VERSION"
