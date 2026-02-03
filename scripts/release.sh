#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

cd "$PROJECT_ROOT"

if [[ -z "$1" ]]; then
    echo "Usage: $0 <version>"
    echo "Example: $0 v0.4.0"
    exit 1
fi

VERSION="$1"

if [[ ! "$VERSION" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    echo "Error: Version must be in format vX.Y.Z (e.g., v0.4.0)"
    exit 1
fi

echo "Preparing release $VERSION..."

echo "Running tests..."
go test ./...

echo "Running linter..."
golangci-lint run ./... || true

echo "Building release binaries..."
mkdir -p dist

platforms=("linux/amd64" "linux/arm64" "darwin/amd64" "darwin/arm64" "windows/amd64")

for platform in "${platforms[@]}"; do
    GOOS="${platform%/*}"
    GOARCH="${platform#*/}"
    output_name="carv-${GOOS}-${GOARCH}"
    
    if [[ "$GOOS" == "windows" ]]; then
        output_name="${output_name}.exe"
    fi
    
    echo "Building $output_name..."
    GOOS=$GOOS GOARCH=$GOARCH go build -ldflags "-s -w -X main.version=$VERSION" -o "dist/$output_name" ./cmd/carv/
done

echo ""
echo "Release binaries created in dist/:"
ls -la dist/

echo ""
echo "To create the release:"
echo "  git tag -a $VERSION -m 'Release $VERSION'"
echo "  git push origin $VERSION"
echo "  gh release create $VERSION dist/* --title '$VERSION' --notes 'Release $VERSION'"
