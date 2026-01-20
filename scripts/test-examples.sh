#!/bin/bash
set -e

cd "$(dirname "$0")/.."

if [ ! -f "build/carv" ]; then
    ./scripts/build.sh
fi

echo "Testing examples..."

for file in examples/*.carv; do
    echo "Running: $file"
    ./build/carv run "$file"
    echo ""
done

echo "All examples passed!"
