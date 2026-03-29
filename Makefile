FRONTEND_VERSION ?= 1.28.0
FRONTEND_URL = https://github.com/buggregator/frontend/releases/download/$(FRONTEND_VERSION)/frontend-$(FRONTEND_VERSION).zip
FRONTEND_DIR = internal/frontend/dist
BINARY = buggregator

.PHONY: build run clean frontend deps

# Download and extract frontend assets
frontend:
	@echo "Downloading frontend v$(FRONTEND_VERSION)..."
	@rm -rf $(FRONTEND_DIR)
	@mkdir -p $(FRONTEND_DIR)
	@curl -sL $(FRONTEND_URL) -o /tmp/frontend.zip
	@unzip -qo /tmp/frontend.zip -d $(FRONTEND_DIR)
	@rm /tmp/frontend.zip
	@echo "Frontend v$(FRONTEND_VERSION) installed."

# Install Go dependencies
deps:
	go mod tidy

# Build the binary (downloads frontend if not present)
build: deps
	@if [ ! -f $(FRONTEND_DIR)/index.html ]; then $(MAKE) frontend; fi
	go build -o $(BINARY) ./cmd/buggregator
	@echo "Built $(BINARY) ($$(du -h $(BINARY) | cut -f1))"

# Build and run
run: build
	./$(BINARY)

# Clean build artifacts
clean:
	rm -f $(BINARY)
	rm -rf $(FRONTEND_DIR)/*
	@echo "<!DOCTYPE html><html><body><h1>Buggregator</h1><p>Frontend not built.</p></body></html>" > $(FRONTEND_DIR)/index.html
