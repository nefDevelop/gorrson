.PHONY: build test clean serve register add

# gorrson - Expert RSS Engine

# Build the binary
build:
	go build -o rss-cli main.go

# Run all internal tests
test:
	go test -v ./internal/...

# Clean binaries and test data
clean:
	rm -f rss-cli
	rm -rf test_data test_rss_data data/*

# Convenience command to register a new user locally
register: build
	./rss-cli register --user $(USER_NAME) --pass $(USER_PASS)

# Convenience command to add a feed
add: build
	./rss-cli add --url $(URL) --user $(USER_NAME)

# Start the API and Worker engine
serve: build
	./rss-cli serve --port 7077

# Show help
help:
	@echo "Available commands:"
	@echo "  make build      - Build binary"
	@echo "  make test       - Run internal tests"
	@echo "  make clean      - Remove binaries and databases"
	@echo "  make serve      - Run the API server"
	@echo "  make register   - Register USER_NAME/USER_PASS"
	@echo "  make add        - Add feed URL to USER_NAME"
