#!/bin/bash
set -e

cd "$(dirname "$0")/.."

echo "Building carv..."
go build -o build/carv ./cmd/carv/

echo "Build complete: build/carv"
