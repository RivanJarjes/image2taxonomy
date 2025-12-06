# syntax=docker/dockerfile:1
# Multi-stage build for Go worker with AI inference support

# Stage 1: Build llama-server for Linux
# Use Debian for better compatibility with llama.cpp
FROM debian:bookworm-slim AS llama-builder

# Install build dependencies for llama.cpp
RUN apt-get update && apt-get install -y --no-install-recommends \
    git \
    cmake \
    build-essential \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /build

# Copy config to determine build acceleration settings
COPY config.yml /tmp/config.yml

# Clone and build llama.cpp with acceleration based on docker_acceleration config
# Reads docker_acceleration from config.yml and applies appropriate cmake flags
RUN git clone --depth 1 https://github.com/ggml-org/llama.cpp && \
    cd llama.cpp && \
    mkdir build && \
    cd build && \
    DOCKER_ACCEL=$(grep 'docker_acceleration:' /tmp/config.yml | awk '{print $2}') && \
    if [ "$DOCKER_ACCEL" = "metal" ]; then \
        echo "Building with Metal acceleration (not applicable on Linux, using CPU)"; \
        cmake .. -DCMAKE_BUILD_TYPE=Release -DLLAMA_CURL=OFF -DGGML_CCACHE=OFF -DGGML_NATIVE=OFF; \
    elif [ "$DOCKER_ACCEL" = "gpu" ]; then \
        echo "Building with CUDA GPU acceleration"; \
        cmake .. -DCMAKE_BUILD_TYPE=Release -DLLAMA_CUDA=ON; \
    elif [ "$DOCKER_ACCEL" = "arm" ]; then \
        echo "Building with ARM NEON acceleration"; \
        cmake .. -DCMAKE_BUILD_TYPE=Release -DLLAMA_NEON=ON -DGGML_CCACHE=OFF; \
    else \
        echo "Building with CPU-only (no GPU acceleration)"; \
        cmake .. -DCMAKE_BUILD_TYPE=Release -DLLAMA_CURL=OFF -DGGML_CCACHE=OFF -DGGML_NATIVE=OFF; \
    fi && \
    cmake --build . --config Release -j$(nproc)

# Stage 2: Build the Go application
FROM golang:1.24.2-bookworm AS builder

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the rest of the application code
COPY . .

# Build the worker binary
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o worker ./cmd/worker

# Stage 3: Runtime image
# Use Debian for glibc compatibility with llama-server
FROM debian:bookworm-slim

# Install runtime dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    curl \
    postgresql-client \
    libstdc++6 \
    libgomp1 \
    && rm -rf /var/lib/apt/lists/*

# Create app directory
WORKDIR /app

# Create non-root user for security
RUN groupadd -g 1000 worker && useradd -u 1000 -g worker -m worker

# Create directories first
RUN mkdir -p /app/infra/llama /app/infra/models /app/docs

# Copy the built binary from builder
COPY --from=builder /build/worker /app/worker

# Copy llama-server and all shared libraries from llama-builder stage
COPY --from=llama-builder /build/llama.cpp/build/bin/llama-server /app/infra/llama/llama-server
COPY --from=llama-builder /build/llama.cpp/build/bin/*.so* /app/infra/llama/

# Set permissions and library path
RUN chmod +x /app/infra/llama/llama-server && \
    chown -R worker:worker /app

# Set LD_LIBRARY_PATH so llama-server can find its shared libraries
ENV LD_LIBRARY_PATH=/app/infra/llama:$LD_LIBRARY_PATH

# The config file and models are mounted at runtime via docker-compose volumes
# Set ownership
RUN chown -R worker:worker /app

USER 1000:1000

# Health check
# Must match the port configured in service.go (--port 8080)
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:8080/health || exit 1

EXPOSE 8080

# Run the worker
CMD ["./worker"]
