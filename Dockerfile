# Build Stage
FROM golang:1.19-alpine AS builder
WORKDIR /app

# Download dependencies first to leverage Docker cache
COPY go.mod go.sum ./
RUN go mod download

# Copy the remaining project files
COPY . .

# Build the Go binary from the cmd/api directory
WORKDIR /app/cmd/api
RUN go build -o main .

# Final stage: a minimal container for running the application
FROM alpine:latest
WORKDIR /app

# Copy the binary from the build stage
COPY --from=builder /app/cmd/api/main .

# Expose the port (adjust this if you configure a different port via config)
EXPOSE 8080

# Start the application
CMD ["./main"] 