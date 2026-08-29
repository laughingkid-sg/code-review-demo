# Multi-stage Dockerfile for Medium API
FROM golang:1.25-alpine AS builder

WORKDIR /app

RUN apk add --no-cache ca-certificates git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
	-trimpath \
	-ldflags="-s -w" \
	-o /app/bin/server ./cmd/server

FROM alpine:3.20

WORKDIR /app

RUN apk --no-cache add ca-certificates tzdata curl

COPY --from=builder /app/bin/server /app/server

EXPOSE 8081

ENV PORT=8081 \
	DATABASE_URL=postgres://postgres:postgres@postgres:5432/orderdb?sslmode=disable \
	REDIS_URL=redis:6379 \
	REDIS_PASSWORD= \
	REDIS_DB=0 \
	JWT_SECRET=medium-api-jwt-secret-key-98765 \
	RATE_LIMIT_RPM=100 \
	ENV=production

CMD ["/app/server"]
