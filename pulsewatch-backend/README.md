# pulsewatch-backend

Go backend for PulseWatch. This is the step-1 scaffold: REST CRUD for monitors backed by Postgres, no concurrency/scheduling, no WebSockets, no alerting yet.

## Setup

1. Install Go 1.22+ and Postgres.
2. Copy `.env.example` to `.env` and adjust `DATABASE_URL` if needed.
3. Create the database and run the migration:
   ```
   createdb pulsewatch
   psql "$DATABASE_URL" -f migrations/0001_create_monitors.sql
   ```
4. Fetch dependencies (this also generates `go.sum`, which isn't checked in yet since this scaffold was written without a local Go toolchain):
   ```
   go mod tidy
   ```
5. Run the server:
   ```
   go run ./cmd/server
   ```

## API

| Method | Path                | Description        |
|--------|---------------------|---------------------|
| GET    | /healthz             | Health check        |
| GET    | /api/monitors         | List monitors        |
| POST   | /api/monitors         | Create a monitor      |
| GET    | /api/monitors/{id}     | Get a monitor        |
| PUT    | /api/monitors/{id}     | Update a monitor      |
| DELETE | /api/monitors/{id}     | Delete a monitor      |

## Next steps (per build order)

2. Concurrency layer: goroutine-based checker + ticker-based scheduler (`internal/monitor/`), test against httpstat.us.
3. WebSocket layer (`internal/websocket/`): broadcast status changes live.
4+. Incidents (`internal/models/incident.go`, `internal/store/incident_repo.go`, `internal/api/handlers_incident.go`), alerting (`internal/alert/`), frontend.
