.PHONY: test lint check build

SRC_DIR = src

default: all

test:
	cd $(SRC_DIR) && go test -v ./...

lint:
	cd $(SRC_DIR) && golangci-lint run

build:
	./cross-build.sh

all: test lint build
