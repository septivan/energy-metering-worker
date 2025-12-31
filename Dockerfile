# Build stage
FROM golang:1.23-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o worker ./cmd/worker

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates

# Create non-root user
RUN addgroup -g 1000 worker && \
    adduser -D -u 1000 -G worker worker

WORKDIR /home/worker

# Copy binary from builder
COPY --from=builder /app/worker .

# Change ownership
RUN chown -R worker:worker /home/worker

# Switch to non-root user
USER worker

# Run the worker
CMD ["./worker"]
