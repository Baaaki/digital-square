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


## Quick Start

**Prerequisites:** Docker and Docker Compose

### Using Docker (Recommended)

```bash
# Clone the repository
git clone https://github.com/yourusername/digital-square.git
cd digital-square

# Start all services (PostgreSQL, Redis, Backend, Frontend)
make up

# Create admin user
make seed

# View logs
make logs

# Stop all services
make down
```

**Access the application:**
- Frontend: http://localhost:3001
- Backend API: http://localhost:9090

**Default admin credentials:**
- Email: `admin@digitalsquare.com`
- Password: `Admin123SecurePassword`

### Available Make Commands

```bash
make help          # Show all available commands
make up            # Start all services
make down          # Stop all services
make ps            # Show container status
make logs          # View all logs (live)
make logs-backend  # View backend logs only
make logs-frontend # View frontend logs only
make rebuild       # Rebuild and restart all services
make seed          # Create admin user
make clean         # Stop and remove all data (WARNING: deletes database)
```

### Local Development (Without Docker)

If you prefer to run services locally:

**Backend:**
```bash
cd backend
# Requires: Go 1.21+, PostgreSQL, Redis
air  # Hot reload
```

**Frontend:**
```bash
cd frontend
# Requires: Node.js 18+ or Bun
bun run dev
```

### Port Configuration

| Service | Host Port | Container Port |
|---------|-----------|----------------|
| Frontend | 3001 | 3000 |
| Backend | 9090 | 8080 |
| PostgreSQL | 15432 | 5432 |
| Redis | 16379 | 6379 |

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
- Production deployment guide not included

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
