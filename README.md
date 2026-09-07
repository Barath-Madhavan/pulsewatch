# PulseWatch

A self-hosted uptime monitoring dashboard that tracks HTTP/HTTPS endpoints, detects outages in real time, and alerts you by email or webhook. Built with Go (chi, pgx, goroutine-based scheduling) on the backend and React + TypeScript on the frontend, with a live WebSocket feed powering the dashboard.

![Dashboard overview](docs/screenshots/dashboard.png)

## Live demo

- **App:** [pulsewatch-2p5.pages.dev](https://pulsewatch-2p5.pages.dev)
- **API:** [pulsewatch-it4r.onrender.com](https://pulsewatch-it4r.onrender.com)

> The demo runs in `DEMO_MODE` (email/webhook alerts are logged, not actually sent) with a cap on how many monitors can exist at once.

## Features

- **HTTP/HTTPS monitoring** with configurable method, interval, timeout, and expected status code per monitor
- **Up / degraded / down status detection**: degraded means "responded correctly but slowly," down means an actual failure (wrong status, timeout, connection error)
- **Automatic incident tracking**: an incident opens the moment a monitor goes down and resolves itself the moment it recovers, with duration tracked automatically
- **Email and webhook alerts** on down/recovery transitions, with per-channel enable toggles
- **Live dashboard**: every check result streams to connected clients over WebSocket, no polling
- **Response-time history** with 24h/7d/30d ranges, bucketed and anomaly-flagged server-side
- **SSRF-hardened checker**: refuses to dial loopback, private, link-local, or CGNAT addresses, enforced at actual dial time (not just URL-string validation)

## Screenshots

| Monitor detail | Incidents |
|---|---|
| ![Monitor detail](docs/screenshots/monitor-detail.png) | ![Incidents](docs/screenshots/incidents.png) |

![Settings](docs/screenshots/settings.png)

## Tech stack

**Backend:** Go, [chi](https://github.com/go-chi/chi) router, [pgx](https://github.com/jackc/pgx)/pgxpool (PostgreSQL), [gorilla/websocket](https://github.com/gorilla/websocket), [go.uber.org/mock](https://github.com/uber-go/mock) + [testify](https://github.com/stretchr/testify) for testing

**Frontend:** React 19, TypeScript, Vite, Tailwind CSS, TanStack Query, Recharts, [Motion](https://motion.dev/)

**Database:** PostgreSQL, plain SQL migrations (no ORM)

## Architecture

The backend runs three concurrent pieces against one process: a **scheduler** that keeps one ticker goroutine per active monitor and reconciles against the database on a timer (so creating/editing/deleting a monitor via the API takes effect without a restart); a **worker pool** that actually performs the HTTP checks; and a **WebSocket hub** that fans every check result out to connected dashboards the instant it happens. Each check result also passes through an **alert dispatcher**, which tracks each monitor's healthy/unhealthy state and opens or resolves incidents (and fires email/webhook alerts) only on an actual transition, not on every check.

## Running locally

### Prerequisites

- Go 1.23+
- Node 20+
- PostgreSQL (14+)

### Backend

```bash
cd pulsewatch-backend

# Create the database, then apply migrations in order:
for f in migrations/*.sql; do psql "$DATABASE_URL" -f "$f"; done

# Configure (a .env file is loaded automatically if present):
cp .env.example .env
# then edit .env (defaults work as-is for a local Postgres on 5432)

go run ./cmd/server
```

> Run it as `go run ./cmd/server` (the package/directory), not `go run cmd/server/main.go`: the package is split across multiple files (`main.go`, `interfaces.go`), and naming a single file excludes the rest.

### Frontend

```bash
cd pulsewatch-frontend
npm install

# Optional (defaults to http://localhost:8080 if unset)
cp .env.example .env

npm run dev
```

## Testing

The backend has a full unit and integration test suite (**92.9% coverage, 157 tests, all passing**), using real `httptest` servers for anything HTTP-based (checker, email/webhook senders, WebSocket client), [go.uber.org/mock](https://github.com/uber-go/mock) for repository/business-logic interfaces, and a real migrated PostgreSQL database for the storage layer's integration tests (not a mock: the actual SQL runs against actual Postgres).

```bash
cd pulsewatch-backend

# Integration tests need a reachable test database (defaults to
# postgres://postgres:postgres@localhost:5433/pulsewatch_test, overridable
# via TEST_DATABASE_URL). Create it and apply migrations the same way as above.

go test ./...

# Coverage report:
pkgs=$(go list ./... | grep -v /mocks)
go test $pkgs -coverprofile=coverage.out -coverpkg=$(echo "$pkgs" | tr '\n' ',' | sed 's/,$//')
go tool cover -html=coverage.out -o coverage.html
```

## Interesting decisions

**Scheduler tick-vs-cancel race.** The scheduler reconciles against the database on a timer: if a monitor is edited, its old per-monitor goroutine is cancelled and a new one started with the updated config, in the same lock. But if that old goroutine's ticker fired at the *exact* instant its context was cancelled, Go's `select` doesn't prioritize between two simultaneously-ready cases: it could pick the ticker case over `ctx.Done()` and submit one stale check under the old (pre-edit) config. The fix is a second check inside the ticker branch: `ctx.Err() != nil` is guaranteed non-nil the instant `ctx.Done()` is observably closed, so re-checking it there reliably catches the race that `select` alone can't. See [`scheduler.go`](pulsewatch-backend/internal/monitor/scheduler.go).

**Anomaly-ratio history bucketing.** Response-time history is bucketed server-side (e.g. 6h-wide buckets for a 30-day view, each holding 100+ individual checks). Flagging a bucket as "down" because *any single check* in it failed would paint almost every bucket red from one-off network blips, which is noise, not signal. Instead a bucket is only marked down/degraded once at least 10% of its checks were actually down/degraded, turning the marker into "this window had a real problem" rather than "one request out of hundreds hiccupped." See [`check_result_repo.go`](pulsewatch-backend/internal/store/check_result_repo.go).

**`DEMO_MODE` and `MAX_MONITORS`.** Two guards exist specifically so this can be safely hosted somewhere publicly reachable, letting a stranger explore the whole app without being able to abuse it: `DEMO_MODE` short-circuits real email/webhook sends into a log line instead of actually delivering them (so the whole alerting feature, including the form, validation, and saving, is fully real and testable, but nobody's inbox or arbitrary URL can actually be hit through it), and `MAX_MONITORS` caps how many monitors can exist at once so a stranger can't mass-create monitors and run up hosting costs. Both default off/unlimited, since they only matter for a public deployment: a local/private instance doesn't need either. See [`config.go`](pulsewatch-backend/internal/config/config.go).

**SSRF guard at dial time, not just creation time.** A monitor URL is validated when it's created (rejecting `localhost`, literal private/loopback IPs, etc.), but a hostname's DNS can resolve to a different, private address later, so the real enforcement is a `net.Dialer.Control` hook that checks the *actually resolved* IP against private/loopback/link-local/CGNAT/multicast ranges at the moment of every dial, refusing the connection regardless of what the original URL string looked like. See [`netguard.go`](pulsewatch-backend/internal/netguard/netguard.go).

## License

[MIT](LICENSE) © Barath Madhavan
