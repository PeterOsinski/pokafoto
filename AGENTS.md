# Drive — Development Rules

## PRD-First Workflow

All feature requests and significant changes MUST go through the `prd/` folder first. Before writing any code:

1. **Identify or create the relevant PRD file** — `prd/` contains numbered specification documents (e.g. `02-product-requirements.md`, `04-data-model.md`, `08-implementation-roadmap.md`).
2. **Update the PRD** — Add the requirement, change, or design decision to the appropriate file. If no suitable file exists, create a new one following the `NN-short-name.md` naming pattern.
3. **Plan from the PRD** — Once the PRD is updated, proceed to plan and implement the change based on what was specified.

## Test-Driven Development

Every change starts with a failing test. No feature or fix is accepted without a corresponding test diff. Tests must run as a single command and complete in under 10 seconds.

## Test Layering

### Backend (Go)
1. **Store tests** — Real `:memory:` SQLite via `testutil.OpenTestDB()`. Migrations run, every CRUD method exercised. No mocking.
2. **Handler tests** — `httptest.NewServer` with real chi router and `:memory:` DB. Assert status codes and JSON response shapes.
3. **Worker tests** — Unit-test the worker pool with `:memory:` store. Inject jobs, verify state transitions. Use `-race` flag.
4. **Integration tests** — Full roundtrip: create user -> login -> upload -> list -> download.

### Frontend (TypeScript/Vue)
1. **Store tests** — Pinia stores with mock API client. Assert state transitions.
2. **Component tests** — `@vue/test-utils` to mount, trigger events, assert DOM. Mock API calls.
3. **Router tests** — Navigation guards with auth state permutations.

## Test Naming

- File: `{source}_test.go` or `{source}.test.ts` (colocated)
- Function: `Test{Pkg}_{Unit}_{scenario}` e.g. `TestUserStore_Create_shouldReturnUser`
- Table-driven tests where multiple inputs share logic

## Coverage Targets

| Layer | Target |
|---|---|
| Store | 90%+ |
| Handler/Middleware | 70%+ |
| Worker | 80%+ |
| Config | 80%+ |
| Frontend stores | 90%+ |
| Frontend components | 60%+ |

## Required Before Finishing

Before any piece of work is considered complete:

1. **Run `make test-all`** and ensure all existing tests pass. Any new feature or fix MUST include one or more new tests that fail without the change and pass with it. Never skip running the test suite — even for small or "obvious" changes.
2. **Run `go build ./...`** to verify the backend compiles without errors.
3. **Run `npm run build` (in `web/`)** to verify the frontend compiles without TypeScript or Vite errors.
4. **Commit and push your work** to the remote repository. All tests and builds must pass before committing.

## Running Tests

```bash
make test          # go test -count=1 ./...
make test-cover    # go test -race -count=1 -coverprofile=coverage.out ./...
make test-web      # cd web && npx vitest run
make test-all      # both (test + test-web)
```

## Running Builds

```bash
go build ./...     # Backend Go compilation
cd web && npm run build  # Frontend TypeScript + Vite build
```

Backend tests run against `:memory:` SQLite with real migrations. Frontend tests run in jsdom with mocked API calls. The full suite completes in under 15 seconds.

## Code Style

- No comments unless explaining a non-obvious design decision
- Follow existing patterns in the codebase
- Imports grouped: stdlib, third-party, project
- Errors always wrapped with `fmt.Errorf("context: %w", err)`
- Use `interface{}` only for JSON map literals

## Commit Conventions

Follow [Conventional Commits](https://www.conventionalcommits.org): `type: description`

| Type | Usage |
|---|---|
| `feat` | New feature |
| `fix` | Bug fix |
| `docs` | Documentation changes |
| `style` | Formatting, missing semicolons, etc. (no code change) |
| `refactor` | Code restructuring without feature/fix |
| `perf` | Performance improvement |
| `test` | Adding or updating tests |
| `chore` | Build, CI, dependency, or tooling changes |

Keep the description lowercase, imperative mood, and under 72 characters.

## Debugging with System Events

The `system_events` table records every significant system operation — backup successes/failures, upload errors, S3 connectivity changes, cache evictions, reconciliation runs, and server lifecycle events. It is the first place to look when diagnosing failures.

### Querying events directly

```bash
# just the errors (fast scan for problems)
sqlite3 data/drive.db \
  "SELECT event_type, substr(message,1,120), created_at FROM system_events \
   WHERE severity='error' ORDER BY created_at DESC LIMIT 20"

# recent backup outcomes
sqlite3 data/drive.db \
  "SELECT severity, event_type, substr(message,1,80), created_at FROM system_events \
   WHERE event_type LIKE 'backup_%' ORDER BY created_at DESC LIMIT 10"

# upload activity in last hour
sqlite3 data/drive.db \
  "SELECT event_type, severity, substr(message,1,120), created_at FROM system_events \
   WHERE created_at >= datetime('now','-1 hour') ORDER BY created_at DESC"

# check for S3 connectivity issues
sqlite3 data/drive.db \
  "SELECT substr(message,1,100), created_at FROM system_events \
   WHERE event_type IN ('s3_disconnect','s3_upload_error') ORDER BY created_at DESC"
```

### Useful diagnostic queries

```bash
# count events by type (overview of system health)
sqlite3 data/drive.db \
  "SELECT event_type, COUNT(*) as cnt FROM system_events \
   WHERE created_at >= datetime('now','-7 days') GROUP BY event_type ORDER BY cnt DESC"

# find specific failed uploads (cross-reference with upload_jobs)
sqlite3 data/drive.db \
  "SELECT e.created_at, e.message, json_extract(e.metadata,'$.filename') as file \
   FROM system_events e WHERE e.event_type='upload_error' \
   AND e.created_at >= datetime('now','-1 day') ORDER BY e.created_at DESC"

# check cache eviction health
sqlite3 data/drive.db \
  "SELECT severity, substr(message,1,100), created_at FROM system_events \
   WHERE event_type IN ('cache_eviction_run','cache_over_limit') ORDER BY created_at DESC LIMIT 5"
```

### Event metadata

The `metadata` column contains JSON with event-specific context. Use `json_extract` to access fields:

```bash
# extract backup file size from metadata
sqlite3 data/drive.db \
  "SELECT created_at, json_extract(metadata,'$.size_bytes') as bytes, \
   json_extract(metadata,'$.s3_key') as s3_key FROM system_events \
   WHERE event_type='backup_success' ORDER BY created_at DESC LIMIT 5"

# extract upload error job IDs
sqlite3 data/drive.db \
  "SELECT created_at, json_extract(metadata,'$.job_id') as job_id, \
   json_extract(metadata,'$.filename') as file FROM system_events \
   WHERE event_type='upload_error' ORDER BY created_at DESC LIMIT 10"
```

### Where events are emitted

| Component | Event types |
|---|---|
| `internal/backup/backup.go` | `backup_success`, `backup_failure`, `backup_pruned` |
| `internal/worker/pool.go` | `upload_error`, `upload_skipped`, `s3_upload_error`, `reconciliation_run`, `reconciliation_error` |
| `internal/server/cache.go` | `cache_eviction_run`, `cache_over_limit` |
| `internal/server/server.go` | `server_start`, `server_shutdown`, `s3_disconnect` |

When adding new features, record significant outcomes via `eventRecorder.Info/Warn/Error(...)`. No schema migration needed — just pick a new `event_type` string and call `Record()`.

### Retention

Events are retained for 90 days. A background goroutine in `server.go` purges entries older than 90 days once every 24 hours.
