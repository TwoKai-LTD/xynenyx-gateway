# Build stage
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Copy source code
COPY . .

# Download dependencies and generate go.sum if needed
RUN go mod download && go mod tidy

# Build binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o gateway main.go

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates wget

WORKDIR /root/

# Copy binary from builder
COPY --from=builder /app/gateway .

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:${PORT:-8080}/health || exit 1

# Run gateway
CMD ["./gateway"]
