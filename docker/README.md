# Docker Support for Rollups Avail Reader

This directory contains Docker-related files for building and running the Rollups Avail Reader application.

## Building the Docker Image

From the root directory of the project, run:

```bash
docker build -t rollups-avail-reader:latest -f docker/Dockerfile .
```

## Running the Container

There are multiple ways to run the container with environment variables:

### 1. Using command-line environment variables:

```bash
docker run -d \
    -p 8080:8080 \
    -p 5004:5004 \
    -e CARTESI_BLOCKCHAIN_WS_ENDPOINT="your-endpoint" \
    -e CARTESI_DATABASE_CONNECTION="your-connection-string" \
    rollups-avail-reader:latest
```

### 2. Using an environment file:

Create a .dockerenv file with your environment variables:

```env
CARTESI_BLOCKCHAIN_WS_ENDPOINT=your-endpoint
CARTESI_DATABASE_CONNECTION=your-connection-string
# Add other variables as needed
```

Then run:

```bash
docker run -d \
    -p 8080:8080 \
    -p 5004:5004 \
    --env-file .dockerenv \
    rollups-avail-reader:latest
```

## Required Environment Variables

The following environment variables are required:

- `CARTESI_BLOCKCHAIN_WS_ENDPOINT`: WebSocket endpoint for the blockchain connection
- `CARTESI_DATABASE_CONNECTION`: PostgreSQL connection string

## Optional Environment Variables

All other environment variables are optional and will use default values if not provided. See `.env.example` in the root directory for the complete list of available variables and their default values.

## Port Mapping

The container exposes two ports:

- 8080: HTTP server port
- 5004: Rollups service port

Make sure to map these ports when running the container if you need to access these services from the host machine.

## Health Check

The container includes a health check that makes a request to `/health` endpoint every 30 seconds. You can monitor the container's health status using:

```bash
docker ps
```

## Security Notes

- The application runs as a non-root user inside the container
- The container uses the official Ubuntu 22.04 base image
- All unnecessary build dependencies are removed in the final image