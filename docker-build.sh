#!/bin/bash

set -e

DOCKER_USER="dimahkiin"
IMAGE_NAME="osm2ch"
VERSION="${VERSION:-$(git describe --tags --abbrev=0 2>/dev/null || echo "latest")}"
BUILD_TIME=$(date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")

FULL_IMAGE_NAME="${DOCKER_USER}/${IMAGE_NAME}"

echo "Building Docker image: ${FULL_IMAGE_NAME}:${VERSION}"
echo "  Build time: ${BUILD_TIME}"
echo "  Git commit: ${GIT_COMMIT}"
echo ""

docker build \
    --build-arg VERSION="${VERSION}" \
    --build-arg BUILD_TIME="${BUILD_TIME}" \
    --build-arg GIT_COMMIT="${GIT_COMMIT}" \
    -t "${FULL_IMAGE_NAME}:${VERSION}" \
    -t "${FULL_IMAGE_NAME}:latest" \
    .

echo ""
echo "Docker image built:"
echo "  ${FULL_IMAGE_NAME}:${VERSION}"
echo "  ${FULL_IMAGE_NAME}:latest"
echo ""
echo "To push to Docker Hub:"
echo "  docker push ${FULL_IMAGE_NAME}:${VERSION}"
echo "  docker push ${FULL_IMAGE_NAME}:latest"
