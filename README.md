# Xynenyx Gateway

Go API Gateway for Xynenyx - handles authentication, rate limiting, and routing to backend services.

## Overview

The gateway is the single entry point for all API requests. It provides:

- JWT validation (Supabase tokens)
- Rate limiting (token bucket algorithm)
- Request routing to services
- Circuit breakers
- Structured logging
- CORS handling

## Quick Start

### Local Development

```bash
# Install dependencies
go mod download

# Run locally
go run main.go

# Or with environment variables
export SUPABASE_JWT_SECRET=your-secret
export AGENT_SERVICE_URL=http://localhost:8001
export RAG_SERVICE_URL=http://localhost:8002
export LLM_SERVICE_URL=http://localhost:8003
go run main.go
```

### Docker

```bash
docker build -t xynenyx-gateway .
docker run -p 8080:8080 --env-file .env xynenyx-gateway
```

## API Endpoints

- `GET /health` - Health check
- `GET /ready` - Readiness check
- `POST /api/agent/chat` - Chat endpoint (proxied to agent service)
- `POST /api/agent/chat/stream` - Streaming chat
- `GET /api/agent/conversations` - List conversations
- `POST /api/rag/query` - RAG query
- `POST /api/llm/complete` - LLM completion

## Configuration

See `.env.example` for all configuration options.

## Testing

```bash
go test -v ./...
go test -race -coverprofile=coverage.out ./...
```

## License

MIT License - see [LICENSE](LICENSE) file

