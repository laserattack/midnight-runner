#!/usr/bin/env bash

set -e

BINARY_NAME="mr"
DIR_FOR_BINS=".bin"

echo "Building for Linux (amd64)..."
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o "${DIR_FOR_BINS}/${BINARY_NAME}-linux-amd64" .
echo "OK"

echo "Building for Windows (amd64)..."
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o "${DIR_FOR_BINS}/${BINARY_NAME}-windows-amd64.exe" .
echo "OK"

echo "Done!"
echo "Binaries:"
ls -l ./"${DIR_FOR_BINS}"/*
