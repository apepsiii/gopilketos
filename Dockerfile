# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o pilketos .

# Final stage
FROM alpine:3.19

WORKDIR /app

# Install CA certificates for HTTPS and runas user
RUN apk --no-cache add ca-certificates su-exec tzdata

# Create non-root user
RUN addgroup -g 1000 appgroup && \
    adduser -u 1000 -G appgroup -s /bin/sh -D appuser

# Copy binary from builder
COPY --from=builder /app/pilketos .
COPY --from=builder /app/config.docker.yaml ./config.yaml

# Create directories
RUN mkdir -p database public/uploads && \
    chown -R appuser:appgroup /app

USER appuser

EXPOSE 8024

CMD ["./pilketos"]
