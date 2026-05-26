.PHONY: all build build-core build-legacy test install clean

# Detect Go binary and flags
GO        := $(shell command -v go 2>/dev/null)
GOFLAGS   ?=
LDFLAGS   ?=

# Output directories
BIN_DIR   := bin
PREFIX    ?= /usr/local/bin

# Binary names
CORE_BIN  := maintainability-sensors
LEGACY_BIN:= legacy-plugin

# Source paths
CORE_SRC  := ./cmd/maintainability-sensors
LEGACY_SRC:= ./cmd/legacy-plugin

## Default: build both binaries
all: build test

## Build all binaries into bin/
build: build-core build-legacy
	@echo "Build complete: $(BIN_DIR)/$(CORE_BIN) $(BIN_DIR)/$(LEGACY_BIN)"

## Build the core CLI
build-core: $(BIN_DIR)
	@if [ -z "$(GO)" ]; then echo "Error: Go is not installed or not in PATH" >&2; exit 1; fi
	$(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(CORE_BIN) $(CORE_SRC)

## Build the legacy plugin subprocess
build-legacy: $(BIN_DIR)
	@if [ -z "$(GO)" ]; then echo "Error: Go is not installed or not in PATH" >&2; exit 1; fi
	$(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(LEGACY_BIN) $(LEGACY_SRC)

## Run the test suite
test:
	@if [ -z "$(GO)" ]; then echo "Error: Go is not installed or not in PATH" >&2; exit 1; fi
	$(GO) test $(GOFLAGS) ./...

## Install both binaries to PREFIX (default: /usr/local/bin)
install: build
	@if [ ! -w "$(PREFIX)" ] && [ "$${EUID}" -ne 0 ]; then \
		echo "Warning: $(PREFIX) is not writable. Try: sudo make install" >&2; \
	fi
	cp $(BIN_DIR)/$(CORE_BIN) $(PREFIX)/
	cp $(BIN_DIR)/$(LEGACY_BIN) $(PREFIX)/
	@echo "Installed to $(PREFIX): $(CORE_BIN) $(LEGACY_BIN)"

## Remove built artifacts
clean:
	rm -rf $(BIN_DIR)
	@echo "Cleaned $(BIN_DIR)/"

$(BIN_DIR):
	@mkdir -p $(BIN_DIR)
