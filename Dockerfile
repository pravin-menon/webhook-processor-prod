# Multi-stage build for production
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache \
    build-base \
    gcc \
    git \
    tzdata

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the application binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o webhook-processor ./cmd/app

# Production stage
FROM alpine:latest

WORKDIR /app

# Install ca-certificates for HTTPS requests and timezone data
RUN apk --no-cache add ca-certificates tzdata

# Create a non-root user
RUN addgroup -g 1001 appgroup && \
    adduser -u 1001 -G appgroup -s /bin/sh -D appuser

# Copy the binary from builder
COPY --from=builder /app/webhook-processor .
COPY --from=builder /app/config/config.yaml ./config/

# Change ownership to the app user
RUN chown -R appuser:appgroup /app

# Switch to non-root user
USER appuser

# Expose port
EXPOSE 8080 9090

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Run the binary
CMD ["./webhook-processor"]
