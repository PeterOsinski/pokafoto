# Drive — Development Rules

## Architecture

| package   | role                                  | examples |
|-----------|---------------------------------------|----------|
| `model/`  | pure data structs, zero dependencies  | `File`, `UploadJob`, `User` |
| `store/`  | data access, SQL, migrations          | `FileStore`, `UserStore`, `DB` |
| `service/`| business logic, external integrations | `StorageService`, `EventRecorder`, `ExifService`, `FileSystem` |
| `worker/` | async job processing                  | `Pool` (upload pipeline) |
| `server/` | HTTP handlers, middleware, routing    | `handlers_*.go`, `auth.go`, `chunk_upload.go` |
| `backup/` | database backup scheduling            | `Scheduler` |
| `config/` | config struct + env loading           | `Config`, `Load()` |

### Interface contracts

Every store and external dependency defines an interface in `store/interfaces.go` or `service/interfaces.go`. All 22 store types (`FileRepository`, `UserRepository`, etc.) and 2 service types (`StorageProvider`, `FileSystem`) have explicit contracts. New dependencies must follow this pattern: define the interface where it's consumed, not where it's implemented.

### Handler organization

One file per domain, using `handlers_*.go` naming. Admin handlers are split to `handlers_admin.go`. Shared response types live in `response_types.go`. Do NOT add to a single monolithic file.

### Filesystem abstraction

All `os`, `filepath`, and `unix` calls go through `service.FileSystem`. Use `service.NewRealFS()` in production, `service.NewMockFS()` in tests. The mock supports `AddFile`, `ReadDir`, `Walk`, `Stat`, `Remove`, `RemoveAll`, `MkdirAll`, `ReadFile`, and `Statfs` with an in-memory store.

### Store access

Handlers call store methods exclusively — never embed raw SQL. The `db.Query()` pattern seen in legacy trash handlers is deprecated.

### Background services

`CacheEvictor`, `S3DeletionPool`, `trashCleanup`, and `chunkCleanup` should accept interfaces (`FileSystem`, `StorageProvider`) rather than calling `os`/`unix` directly. This avoids coupling background processes to global state and enables isolated testing.

### Future direction

- **Domain controllers** — The `Server` struct should shrink from ~49 fields to ~15 by extracting domain controllers (`FileController`, `AuthController`, etc.), each owning only its required dependencies.
- **Service extraction** — Move `CacheEvictor`, `S3DeletionPool`, `trashCleanup`, and `chunkCleanup` from `server/` into `service/` with their own interface contracts and tests.
- **Coverage targets** — Current: backup 17%, server 45%, service 31%, store 66%, worker 54%. Target per this document below: store 90%+, handler 70%+, worker 80%+.

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

> **⚠️ If system_events has no entries for a failure that the code should
> have logged:** the event recorder may be receiving a nil pointer. Check
> `internal/server/server.go` — `s.eventRecorder` must be initialized
> *before* it is passed to `worker.NewPool()`. The nil guard in `Record()`
> silently drops all events from the worker pool (`upload_error`,
> `s3_upload_error`, `reconciliation_error`) without a panic or log line.

### Checking Docker logs

**Docker logs are the most reliable debugging source.** Raw `slog` output
bypasses the event recorder entirely — it works even when `system_events`
is empty or broken. Always check Docker logs first before querying the DB.

```bash
# Docker logs for a specific time window
docker compose logs --since="2026-06-03T11:28:00" --until="2026-06-03T11:29:00"
```

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

## Incident Triage

### Schema Quick Reference

Three tables are essential for upload diagnostics. Full schema at `prd/04-data-model.md`.

| Table | Purpose | Key columns |
|---|---|---|
| `upload_jobs` | Upload lifecycle tracking | `id`, `filename`, `size_bytes`, `status`, `upload_mode`, `total_chunks`, `chunk_size`, `error`, `reason`, `resume_token`, `created_at`, `updated_at` |
| `upload_chunks` | Per-chunk storage records | `upload_id`, `chunk_index`, `chunk_size`, `offset`, `status`, `chunk_sha256`, `temp_path`, `created_at` |
| `system_events` | System-wide events/errors | `event_type`, `severity`, `message`, `metadata` (JSON), `created_at` |

### Upload Error Triage Workflow

1. **Identify the file/job** — query `upload_jobs` by filename:
   ```bash
   sqlite3 data/drive.db "SELECT id, filename, size_bytes, status, upload_mode, total_chunks, error, created_at FROM upload_jobs WHERE filename LIKE '%FILENAME%' ORDER BY created_at DESC"
   ```

2. **Check system events** for server-side errors related to the upload:
   ```bash
   sqlite3 data/drive.db "SELECT event_type, severity, substr(message,1,120), created_at FROM system_events WHERE (message LIKE '%FILENAME%' OR json_extract(metadata,'$.filename') LIKE '%FILENAME%') ORDER BY created_at DESC"
   ```

3. **Check chunk records** (for chunked uploads) — find missing chunks:
   ```bash
   sqlite3 data/drive.db "SELECT uc.chunk_index, uc.status, uc.chunk_size, uc.created_at FROM upload_chunks uc WHERE uc.upload_id = 'JOB_ID' ORDER BY uc.chunk_index"
   ```
   Compare against `total_chunks` from step 1 to identify gaps.

4. **Identify the error origin** — trace the error message back to source code by function name:
   - `"Network error during chunk upload"` → `web/src/stores/chunkedUpload.ts`, function `startChunkedUpload` (catch block inside the concurrent worker loop where `uploadChunk` throws on non-422 errors)
   - `"Chunk hash mismatch"` → `internal/server/chunk_upload.go`, function `handleChunkUpload` (server returns 422 when SHA-256 of received body doesn't match `X-Chunk-SHA256` header)
   - `"assembly_error"` → `internal/worker/pool.go`, function `processChunkedJob` (worker calls `chunkStore.AssembleFile` which fails if any chunk file is missing from disk)
   - `"upload_expired"` → `internal/store/chunk.go`, function `CleanupOldUploads` (background cleanup marks queued/processing chunked jobs older than `max_chunk_upload_age_hours` as failed)

5. **Cross-reference timing** — overlap timestamps across `upload_jobs`, `upload_chunks`, and `system_events` to reconstruct the event sequence.

6. **Check infrastructure** — disk space (`df -h`), S3 connectivity, and server restarts.

### Chunked Upload Data Flow

```
Browser (chunkedUpload.ts)
  → POST /api/v1/upload/chunk (with X-Filename, X-Total-Size, X-Total-Chunks headers)
  → handleChunkUpload (chunk_upload.go) writes chunk file to disk + upload_chunks row
  → POST /api/v1/upload/chunk/{token}/complete
  → handleChunkUploadComplete (chunk_upload.go) validates all chunks present, notifies worker pool
  → worker pool (pool.go) → processChunkedJob assembles chunks → standard processing pipeline
```

With `MAX_CONCURRENT_CHUNKS=3`, chunks are uploaded in parallel batches of 3. If any chunk fails with a non-422 error, the frontend marks the entire upload as failed with `"Network error during chunk upload"`.

### Common Upload Error Categories

| Symptom | Likely Cause | Check |
|---|---|---|
| Job `status=queued` with partial chunks | Client network drop during transfer | Check `upload_chunks` for gaps vs `total_chunks` |
| Multiple queued jobs for same file | Retry after failure, both abandoned | Count `queued` jobs for filename |
| `error="upload_expired"` | Chunks not completed within `max_chunk_upload_age_hours` | Check config `upload.max_chunk_upload_age_hours` |
| All chunked uploads failing | S3 or storage path issue | Check `system_events` for s3_disconnect, disk space |
| Chunk hash mismatches | Corrupted transfer or client-side mutation | Server returns 422 with expected/actual hashes |
| No events in system_events | Client-side error OR event recorder nil (check server.go init order) | Check Docker logs for slog output AND browser console |

### Code Trace Map

| Error/Endpoint | Frontend | Server | Worker/Store |
|---|---|---|---|
| `"Network error during chunk upload"` | `chunkedUpload.ts` `startChunkedUpload` (catch block in concurrent worker loop) | N/A (client-side) | N/A |
| Chunk upload failure | `chunkedUpload.ts` `uploadChunk` (throws on error, returns false on 422) | `chunk_upload.go` `handleChunkUpload` | `chunk.go` `CreateChunkRecord` |
| Resume/recovery | `chunkedUpload.ts` `checkResume` / `resumeUpload` | `chunk_upload.go` `handleChunkUploadResume` | — |
| Assembly + processing | `chunkedUpload.ts` `pollForCompletion` (WebSocket polling) | — | `pool.go` `processChunkedJob` |
| Standard (non-chunked) upload | `upload.ts` | `upload.go` `handleUpload` | `pool.go` worker loop |
