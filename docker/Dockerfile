# Build stage
FROM ubuntu:22.04 as builder

# Install build dependencies
RUN apt-get update && apt-get install -y \
    golang-go \
    git \
    && rm -rf /var/lib/apt/lists/*

# Set the working directory
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o rollups-avail-reader .

# Final stage
FROM ubuntu:22.04

# Install ca-certificates for HTTPS connections
RUN apt-get update && apt-get install -y \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Create a non-root user
RUN useradd -m -u 1000 -s /bin/bash appuser

# Set the working directory
WORKDIR /app

# Copy the binary from builder
COPY --from=builder /app/rollups-avail-reader .

# Change ownership to non-root user
RUN chown -R appuser:appuser /app

# Switch to non-root user
USER appuser

# Expose the server port (both HTTP and Rollups ports)
EXPOSE 8080 5004

# Add healthcheck
HEALTHCHECK --interval=30s --timeout=3s \
    CMD curl -f http://localhost:8080/health || exit 1

# Command to run the application
ENTRYPOINT ["/app/rollups-avail-reader"]