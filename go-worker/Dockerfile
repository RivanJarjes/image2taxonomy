# syntax=docker/dockerfile:1
# Multi-stage build for Go worker with AI inference support

# Stage 1: Build the Go application
FROM golang:1.24.2-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make cmake build-base

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the rest of the application code
COPY . .

# Build the worker binary
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o worker ./cmd/worker

# Stage 2: Runtime image
FROM alpine:3.20

# Install runtime dependencies
# libc for cgo compatibility, curl for health checks
RUN apk add --no-cache ca-certificates libc6-compat curl postgresql-client

# Create app directory
WORKDIR /app

# Create non-root user for security
RUN addgroup -g 1000 worker && adduser -D -u 1000 -G worker worker

# Copy the built binary from builder
COPY --from=builder /build/worker /app/worker

# Copy config template (can be overridden at runtime)
COPY config.yml /app/config.yml

# Create infra directories structure for the app to find resources
RUN mkdir -p /app/infra/llama /app/infra/models /app/docs

# The actual llama-server and models will be mounted or provided at runtime
# Set ownership
RUN chown -R worker:worker /app

USER 1000:1000

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:8000/health || exit 1

EXPOSE 8000

# Run the worker
CMD ["./worker"]

