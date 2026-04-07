.PHONY: all build-ui build-go build clean run install test bench help

# Config
BINARY_NAME=rss-cli
PORT?=7077
DATA_DIR?=./data

# Default target: Build everything
all: install build

# Install dependencies
install:
	@echo "Installing frontend dependencies..."
	cd internal/api/web && npm install
	@echo "Installing backend dependencies..."
	go mod download

# Build only the frontend
build-ui:
	@echo "Building Svelte frontend..."
	cd internal/api/web && npm run build

# Build only the Go binary
build-go:
	@echo "Building Go backend..."
	go build -o $(BINARY_NAME) main.go

# Build both (Embeds UI in binary)
build: build-ui build-go

# Build and run the server immediately
run: build
	@echo "Starting Gorrson on http://localhost:$(PORT)"
	./$(BINARY_NAME) serve --port $(PORT) --data-dir $(DATA_DIR)

# Run development mode (Frontend)
dev-ui:
	cd internal/api/web && npm run dev

# Run development mode (Backend - needs existing UI build in web/dist)
dev-go:
	go run main.go serve --port $(PORT) --data-dir $(DATA_DIR)

# Clean binaries and data
clean:
	@echo "Cleaning up..."
	rm -f $(BINARY_NAME)
	rm -rf internal/api/web/dist
	rm -rf $(DATA_DIR)/*

# Tests & Benchmarks
test: build-ui
	go test -v ./internal/...

bench:
	go test -bench=. ./internal/db -benchmem

register: build
	./$(BINARY_NAME) register --user $(USER_NAME) --pass $(USER_PASS) --data-dir $(DATA_DIR)

# Reset DB and Seed
reset-db:
	@echo "Resetting and seeding database for user 'user'..."
	@mkdir -p $(DATA_DIR)
	go run scripts/seed.go

# Help
help:
	@echo "Gorrson Management:"
	@echo "  make install    - Install all dependencies (npm & go)"
	@echo "  make build      - Build full app (Frontend + Backend)"
	@echo "  make run        - Build and start the production server"
	@echo "  make dev-ui     - Start Svelte/Vite dev server"
	@echo "  make dev-go     - Start Go backend in dev mode"
	@echo "  make clean      - Delete binary and data"
	@echo "  make test       - Run tests"
