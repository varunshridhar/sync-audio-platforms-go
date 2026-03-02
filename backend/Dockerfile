# Build stage compiles static Go binary for small/runtime-safe final image.
# Go version must be >= version declared in backend/go.mod.
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /bin/api ./cmd/api

# Runtime stage uses distroless to reduce attack surface and image size.
FROM gcr.io/distroless/static-debian12
WORKDIR /
COPY --from=builder /bin/api /api
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/api"]

