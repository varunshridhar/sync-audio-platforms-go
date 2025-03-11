# Sync Audio Platforms Go

A Go application for synchronizing audio platforms.

## Table of Contents
- [Requirements](#requirements)
- [Setup](#setup)
- [Configuration](#configuration)
- [Running Locally](#running-locally)
- [API Endpoints](#api-endpoints)
- [Docker](#docker)
- [Development](#development)

## Requirements
- Go 1.19 or higher
- Docker (optional, for containerized deployment)

## Setup

1. Clone the repository:

```bash
git clone https://github.com/yourusername/sync-audio-platforms-go.git
cd sync-audio-platforms-go
```

2. Install dependencies:

```bash
go mod download
```

## Configuration

The application uses environment variables for configuration, which can be set in the `config.yml` file:

- `APP_PORT`: The port on which the server will run (default: 8080)
- `APP_LOG_LEVEL`: Logging level (default: INFO)

## Running Locally

To run the application locally:

```bash
cd cmd/api
go run main.go
```

The server will start on the configured port (default: 8080). You can access the API at `http://localhost:8080`.

## API Endpoints

- `GET /health`: Health check endpoint that returns the application version and status

## Docker

### Building the Docker Image

```bash
docker build -t sync-audio-platforms-go .
```

### Running with Docker

```bash
docker run -p 8080:8080 sync-audio-platforms-go
```

You can also use Docker Compose:

```bash
docker-compose up
```

## Development

### Project Structure

```
.
├── cmd/
│   └── api/            # Application entry point
├── internal/
│   ├── config/         # Configuration management
│   ├── delivery/       # HTTP handlers and routes
│   ├── domain/         # Domain models and interfaces
│   ├── repository/     # Data access layer
│   └── usecase/        # Business logic
├── Dockerfile          # Docker configuration
├── go.mod             # Go module definition
├── go.sum             # Go module checksums
└── config.yml         # Application configuration
```

### Adding New Features

1. Define domain models in `internal/domain`
2. Implement business logic in `internal/usecase`
3. Create HTTP handlers in `internal/delivery/http/handlers`
4. Register routes in `cmd/api/main.go`
