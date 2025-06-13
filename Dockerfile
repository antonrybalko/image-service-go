FROM golang:1.21-bullseye AS builder

WORKDIR /app

# Install libvips-dev for CGO compilation
RUN apt-get update && apt-get install -y \
    libvips-dev \
    && rm -rf /var/lib/apt/lists/*

# Download dependencies first (better caching)
# Copy project files (includes go.mod; go.sum may not yet exist)
COPY . .
# Ensure go.sum is generated *after* all code is available, so
# that `go mod tidy` can discover every imported package.
RUN go mod tidy && go mod download

# Build the application
# CGO is required for libvips
ENV CGO_ENABLED=1
RUN go build -o /app/bin/server ./cmd/server

# Runtime stage
FROM debian:bullseye-slim

# Install runtime dependencies
RUN apt-get update && apt-get install -y \
    ca-certificates \
    libvips \
    && rm -rf /var/lib/apt/lists/*

# Create a non-root user
RUN groupadd -r appuser && useradd -r -g appuser appuser

# Create app directories
WORKDIR /app
RUN mkdir -p /app/config && chown -R appuser:appuser /app

# Copy the binary from builder
COPY --from=builder /app/bin/server /app/server
# Copy config files
COPY --from=builder /app/config/images.yaml /app/config/

# Use non-root user
USER appuser

# Set environment variables
ENV PORT=8080
ENV ENVIRONMENT=production

# Expose the port
EXPOSE 8080

# Run the application
CMD ["/app/server"]
