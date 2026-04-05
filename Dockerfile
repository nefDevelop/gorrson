# Build stage
FROM golang:1.25.8-alpine AS builder

WORKDIR /app

# Copy dependency files and download
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -o gorrson main.go

# Final stage
FROM alpine:latest

WORKDIR /app

# Install certificates for HTTPS (needed to fetch RSS feeds)
RUN apk --no-cache add ca-certificates tzdata

# Copy binary and docs from builder
COPY --from=builder /app/gorrson .
COPY --from=builder /app/docs ./docs

# Default environment variables
ENV GORRSON_PORT=7077
ENV GORRSON_DATA_DIR=/app/data
ENV GORRSON_JWT_SECRET=change-me-in-production

# Expose the configured port
EXPOSE 7077

# Run the server
CMD ["./gorrson", "serve"]
