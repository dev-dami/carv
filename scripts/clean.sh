#!/bin/bash

cd "$(dirname "$0")/.."

echo "Cleaning..."
rm -rf build/
rm -f examples/*.c examples/hello examples/class examples/builtins_test examples/simple
echo "Done"
