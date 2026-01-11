# Testing Guide for Media Service

This document explains the testing architecture and how to run tests for the Media Service.

## Table of Contents

1. [Testing Philosophy](#testing-philosophy)
2. [Hexagonal Architecture & Testing](#hexagonal-architecture--testing)
3. [Test Structure](#test-structure)
4. [Running Tests](#running-tests)
5. [Mock Implementations](#mock-implementations)
6. [CI/CD Pipeline](#cicd-pipeline)

---

## Testing Philosophy

The Media Service follows the **Testing Pyramid** approach:

```
                    ▲
                   ╱ ╲
                  ╱ E2E ╲        Few, slow, expensive
                 ╱───────╲
                ╱         ╲
               ╱Integration╲    Medium amount, moderate speed
              ╱─────────────╲
             ╱               ╲
            ╱   Unit Tests    ╲  Many, fast, cheap
           ╱───────────────────╲
```

- **Unit Tests**: Test individual components in isolation using mocks
- **Integration Tests**: Test components working together with real MongoDB
- **E2E Tests**: Test the complete system (handled at deployment level)

---

## Hexagonal Architecture & Testing

The service follows **Hexagonal (Ports & Adapters) Architecture**, which makes testing easy:

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              APPLICATION                                    │
│                                                                             │
│     DRIVING ADAPTERS           CORE               DRIVEN ADAPTERS           │
│    (HTTP Handlers)           (Domain)            (Infrastructure)           │
│                                                                             │
│    ┌───────────┐          ┌───────────┐          ┌───────────┐              │
│    │   Media   │          │  Domain   │          │   Mongo   │              │
│    │  Handler  │──────────│  Models   │──────────│Repository │              │
│    └───────────┘   PORT   │  (Video)  │   PORT   └───────────┘              │
│                    │      └───────────┘    │                                │
│    ┌───────────┐   │                       │     ┌───────────┐              │
│    │  Health   │   │      ┌───────────┐    │     │   Redis   │              │
│    │  Handler  │──┼──────│ Services  │────┼─────│   Client  │               │
│    └───────────┘  │      │  (Video)  │    │     └───────────┘               │
│                   │      └───────────┘    │                                 │
│    ────────────────                       ────────────────────              │
│       PORTS                                     PORTS                       │
│    (Interfaces)                              (Interfaces)                   │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Why This Matters for Testing

**Ports are interfaces** - they define contracts between components:
- `ports.VideoRepository` - data access contract
- `ports.VideoService` - business logic contract

**Adapters implement ports** - we can swap implementations:
- Real: `repository.MongoRepository`
- Mock: `mocks.MockVideoRepository`

---

## Test Structure

```
media-service/
└── test/
    ├── mocks/                      # Mock implementations
    │   ├── repository_mock.go      # MockVideoRepository
    │   └── test_helpers.go         # Helper functions
    │
    └── api/
        ├── unit/                   # Unit tests
        │   ├── video_service_test.go
        │   ├── media_handler_test.go
        │   └── health_handler_test.go
        │
        └── integration/            # Integration tests
            └── api_integration_test.go
```

