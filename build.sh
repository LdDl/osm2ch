#!/bin/bash

set -e

APP_NAME="osm2ch"
VERSION="${VERSION:-$(git describe --tags --abbrev=0 2>/dev/null || echo "latest")}"
BUILD_TIME=$(date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Build directories
BUILD_DIR="./build"
DIST_DIR="./dist"

# ldflags for version info
LDFLAGS="-s -w -X main.version=${VERSION} -X main.buildTime=${BUILD_TIME} -X main.gitCommit=${GIT_COMMIT}"

echo "Building ${APP_NAME} ${VERSION}"
echo "  Build time: ${BUILD_TIME}"
echo "  Git commit: ${GIT_COMMIT}"
echo ""

# Clean previous builds
rm -rf "${BUILD_DIR}" "${DIST_DIR}"
mkdir -p "${BUILD_DIR}" "${DIST_DIR}"

# Build for Linux amd64
echo "Building for linux/amd64..."
GOOS=linux GOARCH=amd64 go build -ldflags "${LDFLAGS}" -o "${BUILD_DIR}/${APP_NAME}-linux-amd64/${APP_NAME}" ./cmd/osm2ch

# Build for Linux arm64
echo "Building for linux/arm64..."
GOOS=linux GOARCH=arm64 go build -ldflags "${LDFLAGS}" -o "${BUILD_DIR}/${APP_NAME}-linux-arm64/${APP_NAME}" ./cmd/osm2ch

# Build for Windows amd64
echo "Building for windows/amd64..."
GOOS=windows GOARCH=amd64 go build -ldflags "${LDFLAGS}" -o "${BUILD_DIR}/${APP_NAME}-windows-amd64/${APP_NAME}.exe" ./cmd/osm2ch

# Build for macOS amd64
echo "Building for darwin/amd64..."
GOOS=darwin GOARCH=amd64 go build -ldflags "${LDFLAGS}" -o "${BUILD_DIR}/${APP_NAME}-darwin-amd64/${APP_NAME}" ./cmd/osm2ch

# Build for macOS arm64
echo "Building for darwin/arm64..."
GOOS=darwin GOARCH=arm64 go build -ldflags "${LDFLAGS}" -o "${BUILD_DIR}/${APP_NAME}-darwin-arm64/${APP_NAME}" ./cmd/osm2ch

echo ""
echo "Packaging..."

# Package builds
for arch in amd64 arm64; do
    tar -czvf "${DIST_DIR}/${APP_NAME}-${VERSION}-linux-${arch}.tar.gz" -C "${BUILD_DIR}" "${APP_NAME}-linux-${arch}"
done

for arch in amd64 arm64; do
    tar -czvf "${DIST_DIR}/${APP_NAME}-${VERSION}-darwin-${arch}.tar.gz" -C "${BUILD_DIR}" "${APP_NAME}-darwin-${arch}"
done

cd "${BUILD_DIR}"
zip -r "../${DIST_DIR}/${APP_NAME}-${VERSION}-windows-amd64.zip" "${APP_NAME}-windows-amd64"
cd ..

echo ""
echo "Build complete!"
echo "Artifacts in ${DIST_DIR}:"
ls -la "${DIST_DIR}"
