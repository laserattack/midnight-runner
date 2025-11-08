#!/usr/bin/env bash

set -e

BINARY_NAME="mr"

echo "Building for Linux (amd64)..."
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o "${BINARY_NAME}-linux" .
echo "OK"

echo "Building for Windows (amd64)..."
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o "${BINARY_NAME}.exe" .
echo "OK"

echo "Done!"
echo "Binaries:"
ls -l "${BINARY_NAME}-linux" "${BINARY_NAME}.exe"
