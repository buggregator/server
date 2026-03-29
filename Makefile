FRONTEND_VERSION ?= 1.28.0
FRONTEND_URL = https://github.com/buggregator/frontend/releases/download/$(FRONTEND_VERSION)/frontend-$(FRONTEND_VERSION).zip
FRONTEND_DIR = internal/frontend/dist
BINARY = buggregator

# PHP VarDumper binary build
PHP_DIR = php/vardumper
PHP_BIN_DIR = modules/vardumper/bin
PHP_VERSION = 8.4.19
BOX_VERSION = 4.6.6
MICRO_URL_BASE = https://dl.static-php.dev/static-php-cli/common

.PHONY: build run clean frontend deps vardumper-php vardumper-deps vardumper-phar \
        vardumper-all vardumper-linux-amd64 vardumper-linux-arm64 vardumper-darwin-amd64 vardumper-darwin-arm64

# ============================================================
# Frontend
# ============================================================

frontend:
	@echo "Downloading frontend v$(FRONTEND_VERSION)..."
	@rm -rf $(FRONTEND_DIR)
	@mkdir -p $(FRONTEND_DIR)
	@curl -sL $(FRONTEND_URL) -o /tmp/frontend.zip
	@unzip -qo /tmp/frontend.zip -d $(FRONTEND_DIR)
	@rm /tmp/frontend.zip
	@echo "Frontend v$(FRONTEND_VERSION) installed."

# ============================================================
# PHP VarDumper binary (cross-platform)
# ============================================================

# Install composer dependencies
vardumper-deps:
	@echo "Installing VarDumper PHP dependencies..."
	@cd $(PHP_DIR) && composer install --no-dev --optimize-autoloader --quiet

# Download box.phar
$(PHP_DIR)/box.phar:
	@echo "Downloading box v$(BOX_VERSION)..."
	@curl -sL https://github.com/box-project/box/releases/download/$(BOX_VERSION)/box.phar -o $(PHP_DIR)/box.phar

# Build platform-independent .phar
vardumper-phar: vardumper-deps $(PHP_DIR)/box.phar
	@echo "Building VarDumper parser .phar..."
	@cd $(PHP_DIR) && php box.phar compile --quiet
	@echo "Built $(PHP_DIR)/vardumper-parser.phar"

# Download micro.sfx for a specific platform: make micro-sfx SPC_OS=linux SPC_ARCH=x86_64
$(PHP_DIR)/micro-%-%.sfx:
	$(eval SPC_PARTS = $(subst -, ,$*))
	@echo "Target: $@"

# Helper to download micro.sfx for given OS/ARCH
define download_micro
	@echo "Downloading micro.sfx for $(1)-$(2)..."
	@curl -sL "$(MICRO_URL_BASE)/php-$(PHP_VERSION)-micro-$(1)-$(2).tar.gz" -o /tmp/micro-$(1)-$(2).tar.gz
	@mkdir -p /tmp/micro-extract-$(1)-$(2)
	@tar -xzf /tmp/micro-$(1)-$(2).tar.gz -C /tmp/micro-extract-$(1)-$(2)/
	@cp /tmp/micro-extract-$(1)-$(2)/micro.sfx $(PHP_DIR)/micro-$(1)-$(2).sfx
	@rm -rf /tmp/micro-$(1)-$(2).tar.gz /tmp/micro-extract-$(1)-$(2)
	@chmod +x $(PHP_DIR)/micro-$(1)-$(2).sfx
endef

# Helper to build final binary for a platform
define build_vardumper_binary
	$(call download_micro,$(1),$(2))
	@mkdir -p $(PHP_BIN_DIR)
	@cat $(PHP_DIR)/micro-$(1)-$(2).sfx $(PHP_DIR)/vardumper-parser.phar > $(PHP_BIN_DIR)/vardumper-parser-$(3)-$(4)
	@chmod +x $(PHP_BIN_DIR)/vardumper-parser-$(3)-$(4)
	@echo "Built $(PHP_BIN_DIR)/vardumper-parser-$(3)-$(4) ($$(du -h $(PHP_BIN_DIR)/vardumper-parser-$(3)-$(4) | cut -f1))"
endef

# Per-platform targets (Go GOOS-GOARCH → static-php OS-ARCH)
vardumper-linux-amd64: vardumper-phar
	$(call build_vardumper_binary,linux,x86_64,linux,amd64)

vardumper-linux-arm64: vardumper-phar
	$(call build_vardumper_binary,linux,aarch64,linux,arm64)

vardumper-darwin-amd64: vardumper-phar
	$(call build_vardumper_binary,macos,x86_64,darwin,amd64)

vardumper-darwin-arm64: vardumper-phar
	$(call build_vardumper_binary,macos,aarch64,darwin,arm64)

# Build all platforms
vardumper-all: vardumper-linux-amd64 vardumper-linux-arm64 vardumper-darwin-amd64 vardumper-darwin-arm64

# Build for current platform only
vardumper-php: vardumper-phar
	$(eval CURRENT_OS := $(shell go env GOOS))
	$(eval CURRENT_ARCH := $(shell go env GOARCH))
	@$(MAKE) vardumper-$(CURRENT_OS)-$(CURRENT_ARCH)

# ============================================================
# Go build
# ============================================================

deps:
	go mod tidy

# Build the Go binary (downloads frontend + PHP binary if not present)
build: deps
	@if [ ! -f $(FRONTEND_DIR)/index.html ]; then $(MAKE) frontend; fi
	@if [ -z "$$(ls $(PHP_BIN_DIR)/vardumper-parser-* 2>/dev/null)" ]; then $(MAKE) vardumper-php; fi
	go build -o $(BINARY) ./cmd/buggregator
	@echo "Built $(BINARY) ($$(du -h $(BINARY) | cut -f1))"

# Cross-compile for a specific platform
# Usage: make build-cross GOOS=linux GOARCH=amd64
build-cross: deps
	@if [ ! -f $(FRONTEND_DIR)/index.html ]; then $(MAKE) frontend; fi
	@if [ ! -f $(PHP_BIN_DIR)/vardumper-parser-$(GOOS)-$(GOARCH) ]; then $(MAKE) vardumper-$(GOOS)-$(GOARCH); fi
	GOOS=$(GOOS) GOARCH=$(GOARCH) go build -o $(BINARY)-$(GOOS)-$(GOARCH) ./cmd/buggregator
	@echo "Built $(BINARY)-$(GOOS)-$(GOARCH) ($$(du -h $(BINARY)-$(GOOS)-$(GOARCH) | cut -f1))"

# Build all platforms
release: deps frontend vardumper-all
	GOOS=linux GOARCH=amd64 go build -o $(BINARY)-linux-amd64 ./cmd/buggregator
	GOOS=linux GOARCH=arm64 go build -o $(BINARY)-linux-arm64 ./cmd/buggregator
	GOOS=darwin GOARCH=amd64 go build -o $(BINARY)-darwin-amd64 ./cmd/buggregator
	GOOS=darwin GOARCH=arm64 go build -o $(BINARY)-darwin-arm64 ./cmd/buggregator
	@echo "Release binaries built."

run: build
	./$(BINARY)

clean:
	rm -f $(BINARY) $(BINARY)-*
	rm -rf $(FRONTEND_DIR)/*
	rm -f $(PHP_BIN_DIR)/vardumper-parser-*
	rm -f $(PHP_DIR)/vardumper-parser.phar $(PHP_DIR)/box.phar $(PHP_DIR)/micro-*.sfx
	@mkdir -p $(FRONTEND_DIR)
	@echo "<!DOCTYPE html><html><body><h1>Buggregator</h1><p>Frontend not built.</p></body></html>" > $(FRONTEND_DIR)/index.html
