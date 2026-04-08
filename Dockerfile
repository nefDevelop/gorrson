# Stage 1: Build the UI (Svelte/Vite)
FROM node:22-alpine AS ui-builder

WORKDIR /app

# Copy only package files first to leverage Docker cache
COPY internal/api/web/package*.json ./internal/api/web/
RUN cd internal/api/web && npm install

# Copy the rest of the frontend source and build
COPY internal/api/web/ ./internal/api/web/
RUN cd internal/api/web && npm run build

# Stage 2: Build the Go Backend
FROM golang:1.25.8-alpine AS builder

WORKDIR /app

# Copy dependency files and download
COPY go.mod go.sum ./
RUN go mod download

# Copy the entire source code
COPY . .

# Copy the built UI from the previous stage
COPY --from=ui-builder /app/internal/api/web/dist ./internal/api/web/dist

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -o gorrson main.go

# Stage 3: Final Runtime Image
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
