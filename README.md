# Media Service

A microservice for managing educational and informational videos in the Baby Kliniek system. Built in Go, following hexagonal (ports and adapters) architecture, and using MongoDB for persistence.

## Overview

The Media Service provides:

- **Video Management**: CRUD operations for video resources (URL, content type, description, etc.)
- **Role-Based Access Control**: Only users with the `ADMIN` role can create or delete videos (enforced via JWT middleware)
- **MongoDB Integration**: Stores video metadata in a MongoDB collection
- **JWT Authentication**: Validates JWTs signed by the Identity Access Service using a public RSA key
- **Health Checks**: Liveness and readiness probes for container orchestration
- **Circuit Breaker**: Resilience patterns for MongoDB and Redis connections
- **RESTful API**: Exposes endpoints for listing, retrieving, creating, and deleting videos

## Architecture

```
┌─────────────────────────────────────────────────────────────────────────┐
│                         Hexagonal Architecture                          │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│   ┌─────────────┐     ┌─────────────────┐     ┌─────────────────┐       │
│   │  Handlers   │───▶│    Services     │────▶│   Repository    │       │
│   │  (HTTP)     │     │  (Business)     │     │  (MongoDB)      │       │
│   └─────────────┘     └─────────────────┘     └─────────────────┘       │
│        Adapters              Core                   Adapters            │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

## Middleware (JWT Authentication & Caching)

The service uses a custom middleware for JWT authentication and role-based authorization:

```
┌─────────────────────────────────────────────────────────────────────────┐
│                    JWT Authentication Flow                              │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  1. Extract JWT from Authorization header                               │
│                                                                         │
│  2. Check L1 cache (in-memory) for parsed claims                        │
│     → Cache hit: Skip parsing, proceed to step 4                        │
│     → Cache miss: Continue to step 3                                    │
│                                                                         │
│  3. Parse and validate JWT signature with RSA public key                │
│     → Cache parsed claims in L1 for future requests                     │
│                                                                         │
│  4. Check L2 cache (Redis) for token blacklist                          │
│     → If JTI is blacklisted: Reject request (401)                       │
│     → If Redis unavailable: Fail-closed (reject)                        │
│                                                                         │
│  5. Verify role claim matches required roles for endpoint               │
│     → Inject userID and role into request context                       │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

### Caching Strategy

| Cache Layer | Storage | Purpose | TTL |
|-------------|---------|---------|-----|
| L1 (Hot) | In-memory | Parsed JWT claims | Token expiration |
| L2 (Warm) | Redis | Token blacklist | Token expiration |

**Production Note:**  
The in-memory cache is thread-safe and suitable for most deployments. In a multi-replica environment, each instance maintains its own L1 cache, but the L2 Redis blacklist is shared across all replicas.

---

## API Endpoints

| Method | Endpoint                | Description                | Auth Required | Role          |
|--------|-------------------------|----------------------------|---------------|---------------|
| GET    | `/media/videos`         | List all videos            | Yes           | ADMIN, PARENT |
| GET    | `/media/videos/{id}`    | Get video by ID            | Yes           | ADMIN, PARENT |
| POST   | `/media/videos`         | Create a new video         | Yes           | ADMIN         |
| DELETE | `/media/videos/{id}`    | Delete a video by ID       | Yes           | ADMIN         |
| GET    | `/health`               | Detailed health status     | No            | -             |
| GET    | `/health/live`          | Liveness probe             | No            | -             |
| GET    | `/health/ready`         | Readiness probe            | No            | -             |

---

## Project Structure

```
media-service/
├── cmd/
│   └── api/
│       └── main.go              # Application entry point
├── internal/
│   ├── adapters/
│   │   ├── handler/             # HTTP handlers
│   │   │   ├── media_handler.go
│   │   │   └── health_handler.go
│   │   ├── repository/          # Database implementation
│   │   │   └── mongo_repository.go
│   │   └── middleware/          # Middleware implementation
│   │       └── auth_middleware.go
│   ├── core/
│   │   ├── domain/              # Domain models
│   │   │   └── video.go
│   │   ├── ports/               # Interfaces
│   │   │   ├── repository.go
│   │   │   └── service.go
│   │   └── services/            # Business logic
│   │       └── video_service.go
│   └── config/
│       └── config.go            # Configuration loading
├── openshift/                   # OKD/OpenShift deployment
│   ├── application.yaml         # Application resources
│   └── database.yaml            # MongoDB resources
├── Dockerfile
├── go.mod
├── go.sum
└── README.md
```

## Security

- **JWT Verification** - Validates tokens signed by Identity Access Service using RSA public key
- **Token Blacklist** - Redis-backed blacklist for revoked tokens (logout/discharge)
- **Role-Based Access** - Admin-only create/delete endpoints
- **Non-Root Container** - Runs as unprivileged user (UID 1001)
- **Circuit Breaker** - Fail-closed strategy for Redis authentication failures
- **HTTPS** - TLS termination at OKD Route level

## CI/CD Pipeline

The GitHub Actions workflow (`.github/workflows/media-service.yaml`) runs:

1. **Lint** - golangci-lint static analysis
2. **Unit Tests** - Fast tests with mocks
3. **Integration Tests** - Tests with real MongoDB
4. **Build** - Docker image pushed to GHCR
5. **Deploy** - OKD webhook trigger

## License

MIT