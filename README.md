# Digital Square

Real-time chat application built with Go and Next.js - a learning project exploring production-level concepts.

## Overview

A functional messaging platform implementing WebSocket communication, custom Write-Ahead Logging for crash recovery, Redis caching, and role-based admin moderation. Built to learn and demonstrate real-time system architecture, clean code patterns, and modern web development practices.

**Status:** Educational/Portfolio Project - Implements core features with some production concepts (see [Known Limitations](#known-limitations))

## Key Features

- **Real-time messaging** via WebSocket (bidirectional communication)
- **JWT authentication** with Argon2 password hashing
- **Admin moderation** (message deletion, user banning, audit logs)
- **Crash recovery** using custom Write-Ahead Log (WAL)
- **Rate limiting** and IP-based ban system
- **Soft delete** architecture for data retention
- **Infinite scroll** pagination for message history

## Tech Stack

**Backend:**
- Go (Gin framework) - HTTP + WebSocket server
- PostgreSQL (GORM) - Primary database
- Redis - Caching and rate limiting
- Custom WAL - Write-Ahead Log for durability

**Frontend:**
- Next.js 14 (App Router)
- shadcn/ui + TailwindCSS
- WebSocket client with auto-reconnect

## Architecture

```
Client (WebSocket) → Go Server (JWT Auth)
                          ↓
                     1. Write to WAL (fsync)
                     2. Broadcast to clients
                     3. Send ACK
                          ↓
                    Async operations:
                    → Redis Cache (last 100 messages)
                    → PostgreSQL (batch insert every 1 min)
```

**Design principles:**
- **WAL-first writes** ensure zero message loss on crashes
- **Direct in-memory broadcast** (no Redis Pub/Sub overhead for single-node)
- **Batch inserts** to PostgreSQL for write optimization
- **Redis caching** for fast message retrieval on new connections

## Project Structure

```
backend/
├── cmd/server/          # Application entry point
├── internal/
│   ├── handlers/        # HTTP + WebSocket handlers
│   ├── services/        # Business logic layer
│   ├── repository/      # Data access layer
│   ├── models/          # Database models (GORM)
│   ├── middleware/      # Auth, rate limiting, CORS
│   └── wal/            # Write-Ahead Log implementation
└── pkg/cache/          # Redis cache interface

frontend/
└── src/
    ├── app/            # Next.js App Router pages
    ├── components/     # React components
    ├── hooks/          # Custom hooks (useAuth, useWebSocket)
    └── lib/            # Utilities
```

## Quick Start

**Prerequisites:** Go 1.21+, Node.js 18+, PostgreSQL, Redis

### Backend

```bash
cd backend

# Create .env file
DATABASE_URL=postgresql://user:pass@host/db
REDIS_URL=redis://host:port
JWT_SECRET=your-secret-key

# Run with hot reload
air
```

### Frontend

```bash
cd frontend

# Create .env.local
NEXT_PUBLIC_API_URL=http://localhost:8080
NEXT_PUBLIC_WS_URL=ws://localhost:8080

# Install and run
bun install && bun dev
```

### Create Admin User

```bash
cd backend && go run cmd/seed/main.go
# Credentials: admin@example.com / admin123
```

## Features & Implementation

### Core Functionality
- ✅ **Real-time messaging** via WebSocket with automatic reconnection
- ✅ **User authentication** with JWT and Argon2 password hashing
- ✅ **Admin moderation panel** for message deletion and user banning
- ✅ **Crash recovery** using custom Write-Ahead Log (~100 lines)
- ✅ **Message persistence** with batch writes to PostgreSQL
- ✅ **Redis caching** for fast message retrieval
- ✅ **IP-based rate limiting** with ban system
- ✅ **Infinite scroll** pagination
- ✅ **Soft delete** with role-based visibility

### Technical Implementation

**Backend (Go):**
- Clean Architecture: Handler → Service → Repository layers
- Custom WAL for durability (fsync-based crash recovery)
- Redis integration for caching and rate limiting
- Structured logging with performance metrics (Zap)
- Security: XSS prevention, SQL injection protection, CSRF headers
- GORM for database operations with batch insert optimization

**Frontend (Next.js 14):**
- App Router with client-side rendering
- WebSocket client with auto-reconnect
- shadcn/ui component library
- Role-based UI (admin/user views)
- Real-time message updates

**Testing:**
- Unit tests for utils (JWT, hashing): 81% coverage
- WAL implementation: 70% coverage
- Rate limiter: 35% coverage with benchmark tests
- Integration tests for auth flow

## Key Implementation Details

**Write-Ahead Log (WAL):**
- Custom implementation (~100 lines) in `backend/internal/wal/`
- fsync-based durability guarantees
- Auto-recovery on server restart

**WebSocket Management:**
- In-memory client registry with concurrent access control
- Ping/Pong keepalive (54s interval)
- Session expiry after 15 minutes of inactivity

**Security:**
- IP-based rate limiting (100 req/min per IP)
- CSRF protection with security headers
- XSS prevention via HTML escaping
- SQL injection protection (GORM parameterized queries)

**Performance:**
- Batch PostgreSQL writes (1 min interval)
- Redis caching for last 100 messages
- Denormalized username field for query optimization

## Testing

```bash
# Backend unit tests
cd backend && go test ./... -v -cover

# Current coverage: 81.1%
```

## License

GNU General Public License v3.0 - see [LICENSE](LICENSE) for details.

---

## Known Limitations

As a learning project, some production concerns are not fully addressed:

**Architecture & Scalability:**
- Single-node design (horizontal scaling would require Redis Pub/Sub)
- No graceful shutdown implementation
- Manual deployment (Docker/Kubernetes not included)

**Testing:**
- Integration and E2E tests not yet implemented
- Test coverage varies by package (12-81%)
- Frontend components not tested

**Production Readiness:**
- WebSocket origin checking disabled for development (needs hardening)
- No monitoring/alerting infrastructure
- Health check endpoint not implemented
- CI/CD pipeline not configured

See [Future Improvements](#future-improvements) for planned enhancements.

## What I Learned

Building this project provided hands-on experience with:

- **Real-time architecture:** WebSocket lifecycle management, message broadcasting
- **Concurrency patterns:** Goroutines, mutexes, channel-based communication
- **Data durability:** Custom WAL implementation with crash recovery
- **Clean architecture:** Separation of concerns across layers
- **Security fundamentals:** Authentication, authorization, input validation
- **Performance optimization:** Batch writes, caching strategies, denormalization
- **Testing practices:** Unit tests, benchmarks, test fixtures

## Future Improvements

Potential enhancements for learning:

- [ ] Add comprehensive integration and E2E test suites
- [ ] Implement Docker Compose for easier local setup
- [ ] Add Prometheus metrics and Grafana dashboards
- [ ] Create proper graceful shutdown handling
- [ ] Implement WebSocket origin validation
- [ ] Add frontend message virtualization (TanStack Virtual)
- [ ] Set up CI/CD pipeline with GitHub Actions
- [ ] Prepare for horizontal scaling (Redis Pub/Sub)
