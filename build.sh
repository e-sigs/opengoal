#!/bin/bash

# Build script for cross-platform compilation
# Builds binaries for macOS, Linux, and Windows

set -e

VERSION="1.1.1"
OUTPUT_DIR="dist"

echo "Building OpenCode Goal Tracker v${VERSION}"
echo "=========================================="

# Clean output directory
rm -rf "$OUTPUT_DIR"
mkdir -p "$OUTPUT_DIR"

# Build for different platforms
PLATFORMS=(
    "darwin/amd64"
    "darwin/arm64"
    "linux/amd64"
    "linux/arm64"
    "windows/amd64"
)

for platform in "${PLATFORMS[@]}"; do
    IFS='/' read -r -a array <<< "$platform"
    GOOS="${array[0]}"
    GOARCH="${array[1]}"
    
    OUTPUT_NAME="goals-${GOOS}-${GOARCH}"
    
    if [ "$GOOS" = "windows" ]; then
        OUTPUT_NAME="${OUTPUT_NAME}.exe"
    fi
    
    echo "Building ${GOOS}/${GOARCH}..."
    
    GOOS=$GOOS GOARCH=$GOARCH go build \
        -ldflags="-s -w" \
        -o "$OUTPUT_DIR/$OUTPUT_NAME" \
        main.go
    
    # Compress binary
    if command -v upx &> /dev/null; then
        echo "  Compressing with UPX..."
        upx -q "$OUTPUT_DIR/$OUTPUT_NAME" 2>/dev/null || true
    fi
    
    # Get file size
    SIZE=$(ls -lh "$OUTPUT_DIR/$OUTPUT_NAME" | awk '{print $5}')
    echo "  ✓ Built $OUTPUT_NAME ($SIZE)"
done

# Create checksums
echo ""
echo "Generating checksums..."
cd "$OUTPUT_DIR"
shasum -a 256 * > checksums.txt
cd ..

echo ""
echo "Build complete! Binaries are in ./$OUTPUT_DIR/"
echo ""
ls -lh "$OUTPUT_DIR"
