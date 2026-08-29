# Multi-stage Dockerfile for Simple API
FROM golang:alpine AS builder

WORKDIR /app

# Install build dependencies and SSL certificates
RUN apk add --no-cache ca-certificates git

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build statically linked binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w" \
    -o /app/bin/server ./cmd/server

# Final minimal runtime image
FROM alpine:3.20

WORKDIR /app

# Install ca-certificates and tzdata
RUN apk --no-cache add ca-certificates tzdata

# Create data directory
RUN mkdir -p /data

# Copy binary from builder
COPY --from=builder /app/bin/server /app/server

EXPOSE 8080

ENV PORT=8080 \
    DB_PATH=/data/catalog.db \
    API_KEY=dev-api-key-12345

VOLUME ["/data"]

CMD ["/app/server"]
