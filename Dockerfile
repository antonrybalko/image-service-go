FROM golang:1.21-bullseye AS builder

# Install libvips and development dependencies
RUN apt-get update && apt-get install -y \
    libvips-dev \
    pkg-config \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy go.mod and go.sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the application
RUN CGO_ENABLED=1 GOOS=linux go build -a -o image-service ./cmd/server

# Create a minimal runtime image
FROM debian:bullseye-slim

# Install runtime dependencies for libvips
RUN apt-get update && apt-get install -y \
    libvips42 \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/* \
    && apt-get clean

# Create a non-root user to run the application
RUN groupadd -r appuser && useradd -r -g appuser appuser

WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /app/image-service .
COPY --from=builder /app/config ./config

# Set ownership to the non-root user
RUN chown -R appuser:appuser /app

# Switch to non-root user
USER appuser

# Expose the application port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:8080/health || exit 1

# Run the application
CMD ["./image-service"]
