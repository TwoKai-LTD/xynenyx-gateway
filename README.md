# Xynenyx Gateway

Go API Gateway for Xynenyx - handles authentication, rate limiting, and routing to backend services.

## Overview

The gateway is the single entry point for all API requests. It provides:

- **JWT Validation**: Validates Supabase JWT tokens
- **Rate Limiting**: Token bucket algorithm per user (100 requests/minute default)
- **Request Routing**: Reverse proxy to agent, RAG, and LLM services
- **Circuit Breakers**: Protects against cascading failures
- **Structured Logging**: JSON logs with correlation IDs
- **CORS Handling**: Supports Vercel frontend and localhost development
- **Panic Recovery**: Prevents service crashes

## Architecture

```
Frontend (Vercel)
    ↓
Gateway (Port 8080)
    ├── JWT Validation
    ├── Rate Limiting
    ├── Circuit Breaker
    └── Reverse Proxy
        ├── Agent Service (8001)
        ├── RAG Service (8002)
        └── LLM Service (8003)
```

## Quick Start

### Local Development

```bash
# Install dependencies
go mod download

# Set environment variables (see .env.example)
export SUPABASE_JWT_SECRET=your-secret
export AGENT_SERVICE_URL=http://localhost:8001
export RAG_SERVICE_URL=http://localhost:8002
export LLM_SERVICE_URL=http://localhost:8003

# Run locally
go run main.go
```

### Docker

```bash
# Build image
docker build -t xynenyx-gateway .

# Run container
docker run -p 8080:8080 --env-file .env xynenyx-gateway
```

## Configuration

All configuration is done via environment variables. See `.env.example` for all options.

### Required Variables

- `SUPABASE_JWT_SECRET` - JWT secret from Supabase project settings

### Optional Variables

- `PORT` - Gateway port (default: 8080)
- `AGENT_SERVICE_URL` - Agent service URL (default: http://localhost:8001)
- `RAG_SERVICE_URL` - RAG service URL (default: http://localhost:8002)
- `LLM_SERVICE_URL` - LLM service URL (default: http://localhost:8003)
- `RATE_LIMIT_REQUESTS` - Requests per minute (default: 100)
- `RATE_LIMIT_BURST` - Burst size (default: 10)
- `CIRCUIT_BREAKER_FAILURES` - Failures before opening (default: 5)
- `CIRCUIT_BREAKER_TIMEOUT` - Timeout in seconds (default: 30)
- `REQUEST_TIMEOUT` - Request timeout in seconds (default: 30)
- `CORS_ORIGINS` - Allowed origins (comma-separated, default: http://localhost:3000,https://xynenyx.com)
- `LOG_LEVEL` - Log level (default: info)

## API Endpoints

### Health Checks

- `GET /health` - Basic health check (no auth required)
- `GET /ready` - Readiness check (checks downstream services, no auth required)

### API Routes

All `/api/*` routes require JWT authentication via `Authorization: Bearer <token>` header.

- `POST /api/agent/chat` - Chat endpoint (proxied to agent service)
- `POST /api/agent/chat/stream` - Streaming chat
- `GET /api/agent/conversations` - List conversations
- `POST /api/rag/query` - RAG query
- `POST /api/llm/complete` - LLM completion

## Features

### JWT Authentication

The gateway validates JWT tokens from Supabase:

1. Extracts token from `Authorization: Bearer <token>` header
2. Verifies signature using Supabase JWT secret
3. Validates expiration and issued-at claims
4. Extracts user ID from `sub` claim
5. Sets `X-User-ID` header for downstream services

### Rate Limiting

Token bucket algorithm with per-user limits:

- Default: 100 requests per minute
- Burst: 10 requests immediately
- Returns `429 Too Many Requests` with `Retry-After` header when exceeded
- Health check endpoints are exempt

### Circuit Breakers

Protects against cascading failures:

- Tracks failures per service
- Opens circuit after 5 consecutive failures (configurable)
- Half-open state for recovery testing
- Returns `503 Service Unavailable` when circuit is open

### Structured Logging

JSON logs with correlation IDs:

```json
{
  "request_id": "550e8400-e29b-41d4-a716-446655440000",
  "user_id": "user-123",
  "method": "POST",
  "path": "/api/agent/chat",
  "status_code": 200,
  "duration_ms": 150,
  "timestamp": "2024-01-01T00:00:00Z"
}
```

Correlation IDs are passed to downstream services via `X-Request-ID` header.

### CORS

Supports multiple origins:

- Vercel frontend (https://xynenyx.com)
- Local development (http://localhost:3000)
- Configurable via `CORS_ORIGINS` environment variable

## Testing

### Unit Tests

```bash
go test -v ./...
```

### With Coverage

```bash
go test -race -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Integration Tests

Integration tests require all services to be running:

```bash
# Start services
docker-compose up -d

# Run integration tests
go test -v ./integration/...
```

## Deployment

### Railway

1. Connect GitHub repository
2. Set environment variables in Railway dashboard
3. Deploy automatically on push

### Manual

1. Build Docker image: `docker build -t xynenyx-gateway .`
2. Run container: `docker run -p 8080:8080 --env-file .env xynenyx-gateway`

## Troubleshooting

### 401 Unauthorized

- Check that `SUPABASE_JWT_SECRET` is set correctly
- Verify token is valid and not expired
- Ensure `Authorization: Bearer <token>` header is present

### 429 Too Many Requests

- Rate limit exceeded
- Check `Retry-After` header for wait time
- Adjust `RATE_LIMIT_REQUESTS` if needed

### 503 Service Unavailable

- Circuit breaker is open (service is down)
- Check downstream service health
- Circuit will automatically close when service recovers

### 504 Gateway Timeout

- Request to downstream service timed out
- Check service is running and responsive
- Adjust `REQUEST_TIMEOUT` if needed

## License

MIT License - see [LICENSE](LICENSE) file
