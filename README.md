# bookmark-worker

Background job worker for processing bulk bookmark imports. Part of the bookmark microservices architecture.

## Overview

**bookmark-worker** is a headless service with no HTTP server. It polls a Redis list for import jobs pushed by `bookmark-service`, then writes the bookmarks to PostgreSQL. Each job carries a batch of CSV-parsed rows for a single user.

## Tech Stack

| Component | Technology | Version |
|---|---|---|
| Language | Go | 1.26 |
| Database | PostgreSQL (GORM) | v1.31.1 / v1.6.0 |
| Queue | Redis (go-redis) | v9.19.0 |
| Logger | Zerolog | v1.35.1 |
| Shared library | bookmark-common | v0.3.0 |
| Testing | Testify + miniredis + SQLite | v1.11.1 |

## How It Works

### Job flow

```
bookmark-service  →  LPUSH "bookmark:import:jobs"  →  Redis list
                                                            │
bookmark-worker  ←  RPOP ────────────────────────────────┘
    │
    ├── unmarshal JSON job  {job_id, user_id, records: [{url, description}, ...]}
    └── for each record:
            INSERT bookmark (code='')  →  get serial code_int
            encode shortcode (prefix i–z + base62(code_int))
            UPDATE bookmark SET code = shortcode
```

### Worker pool

```
Pool
 ├── 1 poller goroutine   → RPOP from Redis, sends payloads to buffered channel
 └── N worker goroutines  → receive from channel, call handler.Handle
       (N = WORKER_COUNT, buffer = JOB_BUFFER_SIZE)
```

The poller blocks when the channel buffer is full (backpressure). If an RPOP returns empty, the poller sleeps for `POLL_INTERVAL` before retrying. Workers are supervised: a panic in one goroutine is recovered and the goroutine is restarted so a single bad job cannot permanently shrink the pool.

On `SIGINT`/`SIGTERM`, the pool stops polling, closes the job channel, and waits for all in-flight jobs to finish before exiting.

## Quick Start

```bash
cd bookmark-worker
cp .env.example .env   # edit DB + Redis config
make run
```

> The binary reads configuration from **shell environment variables** only — it does not auto-load `.env` files. Export variables manually or use `env $(cat .env | xargs) ./bookmark-worker`.

### Environment variables

| Variable | Default | Description |
|---|---|---|
| `SERVICE_NAME` | _(required)_ | Service identifier |
| `DB_HOST` | `localhost` | PostgreSQL host |
| `DB_PORT` | `5432` | PostgreSQL port |
| `DB_USER` | — | PostgreSQL user |
| `DB_PASSWORD` | — | PostgreSQL password |
| `DB_NAME` | `bookmark_db` | PostgreSQL database |
| `DB_SSLMODE` | `disable` | PostgreSQL SSL mode |
| `DB_TIMEZONE` | `UTC` | PostgreSQL timezone |
| `REDIS_ADDR` | `localhost:6379` | Redis address |
| `REDIS_PASSWORD` | — | Redis password |
| `REDIS_DATABASE` | `1` | Redis logical DB index |
| `QUEUE_KEY` | `bookmark:import:jobs` | Redis list key to poll |
| `WORKER_COUNT` | `5` | Number of concurrent worker goroutines |
| `JOB_BUFFER_SIZE` | `100` | In-memory job channel buffer size |
| `POLL_INTERVAL` | `1s` | Sleep duration when queue is empty |

When deployed via Docker Compose, set `REDIS_ADDR=redis:6379` (Docker network hostname) and `DB_HOST=postgres`.

## Bookmark Insert Strategy

Each bookmark is inserted row by row (not as a batch) inside a single transaction:

1. `INSERT` with `code=''` → PostgreSQL assigns the next `code_int` serial value
2. Encode: `shortcode = randomSQLPrefix + base62(code_int)` (prefix `i–z`)
3. `UPDATE SET code = shortcode`

This avoids a unique constraint violation on `code` that would occur if multiple rows with `code=''` were batch-inserted at once.

## Testing

```bash
make test              # unit + integration tests, 80% coverage gate
make test-coverage     # open HTML coverage report
make docker-test       # test inside Docker (CI parity)
```

Integration tests use SQLite and miniredis — no external services required.

**Coverage threshold: 80%** on business logic. Infrastructure packages (`cmd`, `bootstrap`, `dto`, `model`) are excluded from the threshold but still scanned by SonarCloud.

## Make Targets

```
Development:
  make run             Run locally
  make fmt / vet / lint / tidy / vendor

Testing:
  make test
  make test-coverage

Build:
  make build / build-linux / build-macos / build-windows / build-prod / release

Mocks:
  make generate-mocks
  make clean-mocks

Docker / CI:
  make docker-test / docker-sonar / docker-build-push
  make docker-run / docker-stop / docker-logs / docker-shell / docker-clean

Utilities:
  make install-tools / info / clean / clean-all
```

## CI/CD

| Trigger | CI | CD |
|---|---|---|
| PR to `main` | test + SonarCloud (no push) | — |
| Push to `main` | test + SonarCloud + build + push `main`/`<sha7>` tags | deploy via self-hosted runner |
| Git tag `v*.*.*` | test + SonarCloud + build + push `<tag>` + `latest` | deploy via self-hosted runner |

CD runner working directory: `/opt/bookmark-system`. Updates `BOOKMARK_WORKER_TAG` in `.env` and runs `docker compose up -d --force-recreate bookmark-worker`.

## Project Structure

```
bookmark-worker/
├── cmd/
│   └── worker/main.go         # entrypoint
├── internal/
│   ├── bootstrap/             # app.go (wiring), config.go
│   ├── dto/bookmark/          # message.go (job JSON shape)
│   ├── handler/bookmark/      # handler.go, import.go + mocks
│   ├── model/                 # base.go (UUID PK), bookmark.go
│   ├── repository/
│   │   ├── bookmark/          # write.go (insert + shortcode update) + mocks
│   │   ├── cache/             # Redis cache repo + mocks
│   │   └── queue/             # redis.go (RPOP subscriber) + mocks
│   ├── service/bookmark/      # import.go (orchestrates handler→repo) + mocks
│   ├── worker/                # pool.go (Pool), worker.go (Worker)
│   └── test/
│       ├── integration/       # end-to-end worker_test.go
│       └── fixtures/          # SQLite testdb helpers
├── Makefile
├── Dockerfile
├── sonar-project.properties
└── .github/workflows/ci.yaml, cd.yaml
```
