# Docker Support for Rollups Avail Reader

This directory contains Docker-related files for building and running dockerized Rollups Avail Reader application.

## Building the Docker Image

From the root directory of the project, run:

```bash
docker build --no-cache -t rollups-avail-reader:latest -f docker/Dockerfile .
```

PS: `--no-chage` is optional

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

Using .dockerenv file with environment variables set with default values:

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

## Port Mapping

The container exposes two ports:

- 8080: HTTP server port
- 5004: Rollups service port

Make sure to map these ports when running the container if you need to access these services from the host machine.
