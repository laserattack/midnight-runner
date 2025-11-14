.PHONY: test lint build-all clean

BINARY_NAME = mr
SRC_DIR = src
BUILD_DIR = .bin

default: all

test:
	cd $(SRC_DIR) && go test -v ./...

lint:
	cd $(SRC_DIR) && golangci-lint run

build-all:
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

all: test build-all

clean:
	rm -rf $(BUILD_DIR)
