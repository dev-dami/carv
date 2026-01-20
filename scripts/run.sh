#!/bin/bash
set -e

cd "$(dirname "$0")/.."

if [ ! -f "build/carv" ]; then
    echo "Building first..."
    ./scripts/build.sh
fi

./build/carv "$@"
