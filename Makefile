.PHONY: test lint build clean all

BINARY_NAME = midnight-runner
SRC_DIR = src
BUILD_DIR = .bin

default: all

test:
	@echo "\n[INFO] Running tests"
	cd $(SRC_DIR) && go test -v ./...
	@echo "[INFO] Done!"

lint:
	@echo "\n[INFO] Running linter"
	cd $(SRC_DIR) && golangci-lint run
	@echo "[INFO] Done!"

build:
	@echo "\n[INFO] Building for all platforms"
	@mkdir -p $(BUILD_DIR)
	cd $(SRC_DIR) && GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o ../$(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 .
	cd $(SRC_DIR) && GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o ../$(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 .
	cd $(SRC_DIR) && GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o ../$(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe .
	cd $(SRC_DIR) && GOOS=windows GOARCH=arm64 CGO_ENABLED=0 go build -o ../$(BUILD_DIR)/$(BINARY_NAME)-windows-arm64.exe .
	cd $(SRC_DIR) && GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -o ../$(BUILD_DIR)/$(BINARY_NAME)-macos-amd64 .
	cd $(SRC_DIR) && GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -o ../$(BUILD_DIR)/$(BINARY_NAME)-macos-arm64 .
	cd $(SRC_DIR) && GOOS=freebsd GOARCH=amd64 CGO_ENABLED=0 go build -o ../$(BUILD_DIR)/$(BINARY_NAME)-freebsd-amd64 .
	cd $(SRC_DIR) && GOOS=freebsd GOARCH=arm64 CGO_ENABLED=0 go build -o ../$(BUILD_DIR)/$(BINARY_NAME)-freebsd-arm64 .
	cd $(SRC_DIR) && GOOS=openbsd GOARCH=amd64 CGO_ENABLED=0 go build -o ../$(BUILD_DIR)/$(BINARY_NAME)-openbsd-amd64 .
	cd $(SRC_DIR) && GOOS=netbsd GOARCH=amd64 CGO_ENABLED=0 go build -o ../$(BUILD_DIR)/$(BINARY_NAME)-netbsd-amd64 .
	cd $(SRC_DIR) && GOOS=dragonfly GOARCH=amd64 CGO_ENABLED=0 go build -o ../$(BUILD_DIR)/$(BINARY_NAME)-dragonfly-amd64 .
	@echo "[INFO] Done!"

all:
	@$(MAKE) test
	@$(MAKE) lint
	@$(MAKE) build

clean:
	rm -rf $(BUILD_DIR)
