# Media Service

A microservice for managing educational and informational videos in the Baby Kliniek system. Built in Go, following hexagonal (ports and adapters) architecture, and using MongoDB for persistence.

---

## Overview

The Media Service provides:

- **Video Management**: CRUD operations for video resources (URL, content type, description, etc.)
- **Role-Based Access Control**: Only users with the `ADMIN` role can create or delete videos (enforced via JWT middleware)
- **MongoDB Integration**: Stores video metadata in a MongoDB collection
- **JWT Authentication**: Validates JWTs signed by the Identity Access Service using a public RSA key
- **RESTful API**: Exposes endpoints for listing, retrieving, creating, and deleting videos

---

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

The service uses a custom middleware for JWT authentication and role-based authorization. This middleware performs the following:

- **JWT Extraction & Validation:** Extracts the JWT from the `Authorization` header and validates it using the configured RSA public key.
- **Role Enforcement:** Checks the `role` claim in the JWT to ensure the user has the required permissions.
- **User Context:** On successful validation, injects the user ID and role into the request context for downstream handlers.
- **In-Memory Caching:** To optimize performance, the middleware caches the results of JWT validation in memory. When a JWT is first seen, it is parsed and validated; subsequent requests with the same token are served from the cache until the token expires. This reduces cryptographic overhead and improves response times, especially under high load.

**Production Note:**  
The in-memory cache is thread-safe and suitable for most deployments. However, in a multi-replica environment, each instance maintains its own cache. If shared caching across replicas is required, an option is to integrate a distributed cache like Redis.

---
## API Endpoints

| Method | Endpoint                | Description                | Auth Required | Role  |
|--------|------------------------ |----------------------------|--------------|--------|
| GET    | `/media/videos`         | List all videos            | No           | Any    |
| GET    | `/media/videos/{id}`    | Get video by ID            | No           | Any    |
| POST   | `/media/videos`         | Create a new video         | Yes          | ADMIN  |
| DELETE | `/media/videos/{id}`    | Delete a video by ID       | Yes          | ADMIN  |

## Project Structure

```
media-service/
├── cmd/
│   └── api/
│       └── main.go              # Application entry point
├── internal/
│   ├── adapters/
│   │   ├── handler/             # HTTP handlers
│   │   │   └── media_handler.go
│   │   ├── repository/          # Database implementation
│   │   │    └── mongo_repository.go
│   │   └── middleware/          # Middleware implementation
│   │       └── auth_middleware.go
│   ├── core/
│   │   ├── domain/              # Domain models
│   │   │   └── video.go
│   │   ├── ports/               # Interfaces
│   │   │   └── repository.go
│   │   │   └── service.go
│   │   └── services/            # Business logic
│   │       └── video_service.go
│   └── config/
│       └── config.go            # Configuration loading
├── openshift/                   # OKD/OpenShift deployment
│   ├── database.yaml            # MOngoDB resources
│   └── application.yaml         # Application resources
├── Dockerfile
├── go.mod
├── go.sum
└── .gitignore
```

## License
MIT