.PHONY: build test install clean fmt lint run help

BINARY_NAME=carv
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_DIR=build
INSTALL_DIR=/usr/local/bin

GO=go
GOFLAGS=-ldflags "-X main.version=$(VERSION)"

build:
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/carv/

build-release:
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(GOFLAGS) -ldflags "-s -w -X main.version=$(VERSION)" -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/carv/

test:
	$(GO) test -v ./...

test-cover:
	@mkdir -p $(BUILD_DIR)
	$(GO) test -coverprofile=$(BUILD_DIR)/coverage.out ./...
	$(GO) tool cover -html=$(BUILD_DIR)/coverage.out -o $(BUILD_DIR)/coverage.html

install: build
	install -m 755 $(BUILD_DIR)/$(BINARY_NAME) $(INSTALL_DIR)/$(BINARY_NAME)

uninstall:
	rm -f $(INSTALL_DIR)/$(BINARY_NAME)

clean:
	rm -rf $(BUILD_DIR)
	rm -f examples/*.c examples/hello examples/simple examples/class examples/builtins_test
	$(GO) clean

fmt:
	$(GO) fmt ./...

lint:
	$(GO) vet ./...
	@which staticcheck > /dev/null 2>&1 && staticcheck ./... || echo "staticcheck not installed"

run: build
	./$(BUILD_DIR)/$(BINARY_NAME) run examples/hello.carv

repl: build
	./$(BUILD_DIR)/$(BINARY_NAME) repl

examples: build
	./$(BUILD_DIR)/$(BINARY_NAME) build examples/hello.carv
	./$(BUILD_DIR)/$(BINARY_NAME) build examples/class.carv

help:
	@echo "Carv Programming Language"
	@echo ""
	@echo "Usage:"
	@echo "  make build         Build the carv compiler"
	@echo "  make build-release Build optimized release binary"
	@echo "  make test          Run all tests"
	@echo "  make test-cover    Run tests with coverage report"
	@echo "  make install       Install carv to /usr/local/bin"
	@echo "  make uninstall     Remove carv from /usr/local/bin"
	@echo "  make clean         Remove build artifacts"
	@echo "  make fmt           Format source code"
	@echo "  make lint          Run linters"
	@echo "  make run           Build and run examples/hello.carv"
	@echo "  make repl          Start interactive REPL"
	@echo "  make examples      Compile example programs"
	@echo "  make help          Show this help"
