#!/usr/bin/env bash

set -e

BINARY_NAME="mr"
DIR_FOR_BINS=".bin"

mkdir -p "${DIR_FOR_BINS}"

echo "Building for Linux (amd64)..."
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o "${DIR_FOR_BINS}/${BINARY_NAME}-linux-amd64" .
echo "OK"
echo "Building for Linux (arm64)..."
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o "${DIR_FOR_BINS}/${BINARY_NAME}-linux-arm64" .
echo "OK"
echo "Building for Windows (amd64)..."
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o "${DIR_FOR_BINS}/${BINARY_NAME}-windows-amd64.exe" .
echo "OK"
echo "Building for Windows (arm64)..."
GOOS=windows GOARCH=arm64 CGO_ENABLED=0 go build -o "${DIR_FOR_BINS}/${BINARY_NAME}-windows-arm64.exe" .
echo "OK"
echo "Building for macOS (amd64)..."
GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -o "${DIR_FOR_BINS}/${BINARY_NAME}-macos-amd64" .
echo "OK"
echo "Building for macOS (arm64)..."
GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -o "${DIR_FOR_BINS}/${BINARY_NAME}-macos-arm64" .
echo "OK"
echo "Building for FreeBSD (amd64)..."
GOOS=freebsd GOARCH=amd64 CGO_ENABLED=0 go build -o "${DIR_FOR_BINS}/${BINARY_NAME}-freebsd-amd64" .
echo "OK"
echo "Building for FreeBSD (arm64)..."
GOOS=freebsd GOARCH=arm64 CGO_ENABLED=0 go build -o "${DIR_FOR_BINS}/${BINARY_NAME}-freebsd-arm64" .
echo "OK"
echo "Building for OpenBSD (amd64)..."
GOOS=openbsd GOARCH=amd64 CGO_ENABLED=0 go build -o "${DIR_FOR_BINS}/${BINARY_NAME}-openbsd-amd64" .
echo "OK"
echo "Building for NetBSD (amd64)..."
GOOS=netbsd GOARCH=amd64 CGO_ENABLED=0 go build -o "${DIR_FOR_BINS}/${BINARY_NAME}-netbsd-amd64" .
echo "OK"
echo "Building for DragonFly BSD (amd64)..."
GOOS=dragonfly GOARCH=amd64 CGO_ENABLED=0 go build -o "${DIR_FOR_BINS}/${BINARY_NAME}-dragonfly-amd64" .
echo "OK"

echo "Done!"
echo "Binaries:"
ls -l "${DIR_FOR_BINS}"/*
