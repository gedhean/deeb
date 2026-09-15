# Multi-stage Dockerfile for Deeb Backend
# Stage 1: Build the Go application
FROM golang:1.25-alpine AS builder

# Install build dependencies (including gcc, musl-dev for CGO)
RUN apk add --no-cache git ca-certificates gcc musl-dev

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the main deeb binary (with Cobra commands)
# CGO is enabled for pg_query_go library
RUN CGO_ENABLED=1 GOOS=linux go build -a -o deeb main.go

# Stage 2: Create minimal runtime image
FROM alpine:latest

# Install ca-certificates for HTTPS, timezone data, libc for CGO binaries, and wget for healthcheck
RUN apk --no-cache add ca-certificates tzdata libc6-compat wget

# Create non-root user
RUN addgroup -g 1000 deeb && \
    adduser -D -u 1000 -G deeb deeb

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/deeb .

# Copy migrations directory
COPY --from=builder /app/migrations ./migrations

# Copy API documentation
COPY --from=builder /app/api ./api

# Copy production configuration
COPY --from=builder /app/config.prd.yaml ./config.yaml

# Copy gh-ost from builder
COPY --from=builder /app/ext-bin/gh-ost /app/gh-ost

# Change ownership to non-root user
RUN chown -R deeb:deeb /app

# Switch to non-root user
USER deeb

# Expose application port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/api/v1/health || exit 1

# Default command runs the server with config file
CMD ["./deeb", "server", "--config", "config.yaml"]
