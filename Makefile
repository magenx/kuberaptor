.PHONY: build clean test install lint

BINARY_NAME=kuberaptor
VERSION=0.0.0
BUILD_DIR=dist
INSTALL_PATH=/usr/local/bin

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=$(GOCMD) fmt

# Build flags
LDFLAGS=-ldflags "-X github.com/magenx/kuberaptor/pkg/version.Version=$(VERSION) -s -w"

all: test build

build:
	@echo "Building $(BINARY_NAME)"
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/kuberaptor

build-linux:
	@echo "Building $(BINARY_NAME) for Linux amd64"
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd/kuberaptor

build-linux-arm:
	@echo "Building $(BINARY_NAME) for Linux arm64"
	GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 ./cmd/kuberaptor

build-darwin-arm:
	@echo "Building $(BINARY_NAME) for macOS arm64"
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 ./cmd/kuberaptor

build-all: build-linux build-linux-arm build-darwin-arm

test:
	@echo "Running tests"
	$(GOTEST) -v -race -coverprofile=coverage.out ./...

coverage:
	@echo "Generating coverage report"
	$(GOTEST) -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out

clean:
	@echo "Cleaning"
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)
	rm -f coverage.out

install: build
	@echo "Installing to $(INSTALL_PATH)"
	@mkdir -p $(INSTALL_PATH)
	@cp $(BUILD_DIR)/$(BINARY_NAME) $(INSTALL_PATH)/$(BINARY_NAME)
	@echo "Installed $(BINARY_NAME) to $(INSTALL_PATH)"

deps:
	@echo "Downloading dependencies"
	$(GOMOD) download
	$(GOMOD) tidy

lint:
	@echo "Running linters"
	@which golangci-lint > /dev/null 2>&1 || (echo "golangci-lint not found, install it from https://golangci-lint.run/usage/install/" && exit 1)
	golangci-lint run ./...

fmt:
	@echo "Formatting code"
	$(GOFMT) ./...

.DEFAULT_GOAL := build
