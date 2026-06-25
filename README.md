# URL Shortener

A production-oriented URL shortener written in Go. It shortenes long URLs, redirects via short codes, tracks click counts and exposes basic analytics — backed by PostgreSQL and Redis.

## Features
- **Shorten URLs** — auto-generated base62 short codes or optional custom aliases
- **Redirect** — HTTP 302 redirects with cache-aside lookup (Redis → PostgreSQL)
- **Click tracking** — async click counter increments on each redirect
- **Analytics** — total clicks, top countries, device breakdown, and daily clicks (last 30 days)
- **Expiry** — optional per-URL expiration
- **Negative caching** — Redis caches unknown short codes to reduce DB load
- **Graceful shutdown** — SIGINT/SIGTERM handling with a 10s drain timeout

## Tech Stack
| Layer        | Technology                          |
| ------------ | ----------------------------------- |
| Language     | Go 1.26                             |
| HTTP router  | [chi](https://github.com/go-chi/chi) |
| Database     | PostgreSQL (pgx/v5 connection pool) |
| Cache        | Redis (go-redis/v9)                 |
| Config       | Viper + godotenv                    |
| Logging      | zerolog                             |


## Architecture

```
HTTP Request
     │
     ▼
┌─────────────┐
│   Handler   │  ← JSON validation, HTTP status codes
└──────┬──────┘
       │
       ▼
┌─────────────┐
│   Service   │  ← business logic, cache-aside, code generation
└──────┬──────┘
       │
   ┌───┴───┐
   ▼       ▼
┌──────┐ ┌───────┐
│ Repo │ │ Redis │  ← PostgreSQL persistence + URL caching
└──────┘ └───────┘
```
Request flow:

1. **Handler** (`internal/handler`) — parses HTTP requests, validates input, and writes JSON or redirect responses.
2. **Service** (`internal/service`) — generates short codes, resolves redirects using a cache-aside pattern, and increments click counts asynchronously.
3. **Repository** (`internal/repository`) — persists URLs and analytics data in PostgreSQL.
4. **Cache** (`internal/cache`) — stores resolved URLs in Redis and negative-caches unknown short codes.

The entry point in `cmd/api/main.go` wires configuration, database, Redis, and the HTTP server together.

## Project Structure

```
URL-Shortener/
├── cmd/
│   └── api/
│       └── main.go              # Application entry point
├── internal/
│   ├── cache/
│   │   └── redis.go             # Redis cache-aside + negative caching
│   ├── config/
│   │   └── config.go            # Environment-based configuration
│   ├── database/
│   │   └── postgres.go          # PostgreSQL connection pool
│   ├── handler/
│   │   └── url.go               # HTTP handlers
│   ├── model/
│   │   └── url.go               # Domain models and DTOs
│   ├── repository/
│   │   └── url.go               # PostgreSQL data access
│   ├── server/
│   │   └── server.go            # Router, middleware, server lifecycle
│   └── service/
│       └── url.go               # Core business logic
├── migrations/
│   ├── 000001_init.up.sql       # Schema: urls, click_events
│   └── 000001_init.down.sql     # Rollback
├── go.mod
└── .env                         # Local env vars (gitignored)
```


## Environment Variables
Create a `.env` file in the project root (or set these in your environment):
| Variable               | Default          | Description                          |
| ---------------------- | ---------------- | ------------------------------------ |
| `SERVER_PORT`          | `8080`           | HTTP server port                     |
| `SERVER_READ_TIMEOUT`  | `10`             | Read timeout (seconds)               |
| `SERVER_WRITE_TIMEOUT` | `30`             | Write timeout (seconds)              |
| `DATABASE_URL`         | —                | PostgreSQL connection string         |
| `DB_MAX_CONNS`         | —                | Max DB pool connections              |
| `DB_MIN_CONNS`         | —                | Min DB pool connections              |
| `REDIS_ADDR`           | `localhost:6379` | Redis address                        |
| `REDIS_PASSWORD`       | —                | Redis password                       |
| `REDIS_DB`             | `0`              | Redis database index                 |
| `REDIS_CACHE_TTL`      | `3600`           | Default URL cache TTL (seconds)      |
| `APP_BASE_URL`         | —                | Base URL for generated short links   |
| `APP_SHORT_CODE_LEN`   | `7`              | Length of auto-generated short codes |
| `APP_DEFAULT_EXPIRY`   | `0`              | Default expiry in days (0 = none)    |
| `GEOIP_DB_PATH`        | —                | GeoIP database path (reserved)       |

Example `.env`:
```env
SERVER_PORT=8080
DATABASE_URL=postgres://user:password@localhost:5432/urlshortener?sslmode=disable
REDIS_ADDR=localhost:6379
APP_BASE_URL=http://localhost:8080