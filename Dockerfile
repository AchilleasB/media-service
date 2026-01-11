# =============================================================================
# Dockerfile for Media Service
# =============================================================================
# Builds a minimal, secure image for the Media Service API.
# The Kubernetes deployment runs this container.
# =============================================================================

ARG APP_VERSION=unknown

# Build stage: compile the Go binary
FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG APP_VERSION
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-X main.Version=${APP_VERSION}" \
    -o media-api ./cmd/api/main.go

# Final stage: minimal runtime image
FROM alpine:latest

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata

# Security: Run as non-root user
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

COPY --from=builder /app/media-api .
RUN chmod 755 ./media-api

USER appuser

EXPOSE 8080

CMD ["./media-api"]

