# Sahiy Stream

Ideal live streaming platform — Go backend, Next.js frontend.

## Hujjatlar

| Hujjat | Tavsif |
|--------|--------|
| [ARCHITECTURE.md](docs/ARCHITECTURE.md) | To'liq system design, microservices, media pipeline |
| [DATABASE.md](docs/DATABASE.md) | Database schema, indexing, cache patterns |
| [API.md](docs/API.md) | API reference, auth flow, error codes |

## Stack

- **Backend:** Go 1.22+ (microservices, gRPC)
- **Frontend:** Next.js 15, TypeScript, Tailwind
- **Database:** PostgreSQL, Redis, ClickHouse, Elasticsearch
- **Media:** FFmpeg, LL-HLS, WebRTC
- **Infra:** Docker, Kubernetes, Terraform

## Tez boshlash

```bash
# 1. Environment
cp .env.example .env

# 2. Infrastructure (PostgreSQL, Redis, NATS, MinIO)
make up

# 3. Database migrations
make migrate

# 4. Servislarni ishga tushirish (4 ta terminal)
go run ./services/auth-service/cmd/server
go run ./services/user-service/cmd/server
go run ./services/stream-service/cmd/server
go run ./services/api-gateway/cmd/server
```

### API test

```bash
# Register
curl -X POST http://localhost:8080/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"test@sahiy.stream","username":"testuser","display_name":"Test User","password":"password123"}'

# Login
curl -X POST http://localhost:8080/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@sahiy.stream","password":"password123"}'

# Me (access_token bilan)
curl http://localhost:8080/v1/auth/me \
  -H "Authorization: Bearer <access_token>"
```

## Loyiha strukturasi

```
sahiy-stream/
├── pkg/                    # Shared Go packages
├── proto/                  # gRPC definitions + generated code
├── services/
│   ├── auth-service/       # Auth microservice (gRPC + health)
│   └── api-gateway/        # REST API gateway
├── infra/docker/           # Docker Compose
├── scripts/                # migrate, proto-gen
└── docs/                   # Architecture docs
```

## Implementation tartibi

Reja bo'yicha 9 bosqich — batafsil: [ARCHITECTURE.md §16](docs/ARCHITECTURE.md#16-implementation-tartibi)

1. **Foundation** — monorepo, auth, API gateway ✅
2. **Core Platform** — user-service, stream-service, channels, streams ✅
3. Media Pipeline — ingest, transcode, playback
4. Real-time — chat, notifications
5. Discovery — search, feed, social
6. Quality — LL-HLS, WebRTC, 4K
7. Moderation — safety, admin
8. Analytics & Monetization
9. Production Hardening
