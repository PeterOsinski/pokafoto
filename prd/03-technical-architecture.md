# 3. Technical Architecture

## 3.1 Technology Stack

| Layer | Technology | Rationale |
|---|---|---|---|
| **Backend Language** | Go 1.22+ | Single binary, low memory, fast, great concurrency |
| **Frontend Framework** | Vue.js 3 + TypeScript | Reactive, lightweight, excellent DX |
| **Frontend Build** | Vite | Fast HMR, optimized builds |
| **Database** | SQLite 3 (via `modernc.org/sqlite`) | Zero setup, single file, fast enough for family workload |
| **HTTP Router** | `chi` | Lightweight, no heavy framework |
| **Image Processing** | `libvips` (via `govips`) | 4-5x faster than ImageMagick, low memory. CGO-required. |
| **Video Processing** | `ffmpeg` (subprocess) | Industry standard, hardware acceleration support |
| **EXIF Parsing** | `goexif` + `exiftool` subprocess (HEIC/AVIF fallback) | Pure Go for JPEG, exiftool for HEIC container |
| **S3 Client** | `github.com/minio/minio-go` | S3-compatible, widely used |
| **Map (Frontend)** | Leaflet + OpenStreetMap tiles | No API key, lightweight |
| **Clustering** | Supercluster (client-side) | Fast, works offline |
| **CSS** | Tailwind CSS | Utility-first, responsive, large ecosystem |
| **State Management** | Pinia | Vue 3 official state management |
| **Authentication** | `golang-jwt/jwt/v5` + `golang.org/x/crypto/bcrypt` | JWT access + refresh tokens |
| **Logging** | `log/slog` (Go stdlib) | Structured JSON, zero dependencies |
| **Testing** | `go test` (backend) + `vitest` (frontend) | Tests from Day 1 |
| **Container** | Docker (multi-stage build → alpine) | Small final image, libheif for HEIC support |

## 3.2 Project Structure

```
drive/
├── cmd/
│   └── drive/
│       └── main.go              # Entry point
├── internal/
│   ├── server/
│   │   ├── server.go            # HTTP server setup
│   │   ├── middleware.go        # Logging, CORS, recovery
│   │   └── routes.go            # Route registration
│   ├── handler/
│   │   ├── auth.go              # Login, register, logout, refresh
│   │   ├── admin.go             # Admin user management
│   │   ├── upload.go            # Upload endpoints
│   │   ├── gallery.go           # Gallery/thumbnail endpoints
│   │   ├── file.go              # File CRUD endpoints
│   │   ├── map.go               # Geo-data endpoints
│   │   ├── exif.go              # EXIF data endpoints
│   │   └── download.go          # Download/zip endpoints
│   ├── middleware/
│   │   └── auth.go              # JWT validation middleware
│   ├── service/
│   │   ├── auth.go              # Authentication service
│   │   ├── ingest.go            # Upload processing pipeline
│   │   ├── thumbnail.go         # Thumbnail generation
│   │   ├── exif.go              # EXIF extraction
│   │   ├── storage.go           # S3 + local storage abstraction
│   │   └── organizer.go         # Auto-organization logic
│   ├── store/
│   │   ├── sqlite.go            # DB connection & migrations
│   │   ├── user.go              # User queries
│   │   ├── session.go           # Session/refresh token queries
│   │   ├── file.go              # File metadata queries
│   │   ├── exif.go              # EXIF data queries
│   │   └── geo.go               # Spatial queries
│   ├── model/
│   │   ├── file.go              # File model
│   │   ├── exif.go              # EXIF model
│   │   ├── thumbnail.go         # Thumbnail model
│   │   ├── user.go              # User model
│   │   └── session.go           # Session model
│   └── config/
│       └── config.go            # Configuration loading
├── web/
│   ├── src/
│   │   ├── App.vue
│   │   ├── main.ts
│   │   ├── router/
│   │   │   └── index.ts
│   │   ├── stores/
│   │   │   ├── gallery.ts       # Gallery state
│   │   │   ├── upload.ts        # Upload state
│   │   │   └── map.ts           # Map state
│   │   ├── views/
│   │   │   ├── LoginView.vue
│   │   │   ├── RegisterView.vue
│   │   │   ├── GalleryView.vue
│   │   │   ├── TimelineView.vue
│   │   │   ├── MapView.vue
│   │   │   ├── UploadView.vue
│   │   │   ├── PhotoDetail.vue
│   │   │   └── AdminView.vue
│   │   ├── components/
│   │   │   ├── ThumbnailGrid.vue
│   │   │   ├── ThumbnailCard.vue
│   │   │   ├── Lightbox.vue
│   │   │   ├── UploadZone.vue
│   │   │   ├── UploadProgress.vue
│   │   │   ├── Timeline.vue
│   │   │   ├── PhotoMap.vue
│   │   │   ├── ExifPanel.vue
│   │   │   ├── DirectoryTree.vue
│   │   │   ├── FileIcon.vue
│   │   │   └── VideoPlayer.vue
│   │   ├── composables/
│   │   │   ├── useLazyLoad.ts
│   │   │   ├── useSwipe.ts
│   │   │   ├── useKeyboardNav.ts
│   │   │   └── useAuth.ts
│   │   └── api/
│   │       └── client.ts        # API client with JWT handling
│   ├── index.html
│   ├── vite.config.ts
│   ├── tsconfig.json
│   └── package.json
├── migrations/
│   └── 001_initial.sql
├── Dockerfile
├── docker-compose.yml
├── go.mod
├── go.sum
└── Makefile
```

## 3.3 Data Flow: Upload Pipeline

```
┌──────────┐     ┌─────────────┐     ┌──────────────┐
│  Browser  │────▶│  POST /api/ │────▶│  IngestSvc   │
│  (upload) │     │  upload     │     │  .Process()  │
└──────────┘     │  (Auth req)  │     └──────┬───────┘
                 └─────────────┘              │
                              ┌───────────────┼────────────────────┐
                              ▼               ▼                    ▼
                      ┌──────────────┐ ┌──────────────┐ ┌──────────────────┐
                      │ 1. Name+Size │ │ 2. Hash file │ │ 3. EXIF          │
                      │   Dedup      │ │   (SHA-256)  │ │   Extraction     │
                      └──────┬───────┘ └──────┬───────┘ └──────┬───────────┘
                             │                │                │
                             ▼                ▼                ▼
                      ┌──────────────┐ ┌──────────────┐ ┌──────────────────┐
                      │ Check dup    │ │ Check dup    │ │ Store in         │
                      │ (name+size)  │ │ (content)    │ │ SQLite           │
                      └──────────────┘ └──────────────┘ └──────────────────┘
                                                             │
                                                             ▼
                                                  ┌──────────────────┐
                                                  │ 4. Generate       │
                                                  │   Thumbnails      │
                                                  │   (sm/md/preview  │
                                                  │    + video_still) │
                                                  └────────┬─────────┘
                                                           │
                                                           ▼
                                                  ┌──────────────────┐
                                                  │ 5. Save to        │
                                                  │   Local Cache     │
                                                  └────────┬─────────┘
                                                           │
                              ┌────────────────────────────┘
                              ▼
                      ┌──────────────────────┐
                      │ 6. Upload to S3      │  ← Only if s3.enabled: true
                      │   (originals +       │
                      │    thumbnails)       │
                       └──────────────────────┘
```

### ERD: system_events

| Column | Type | Description |
|---|---|---|
| `id` | TEXT (UUIDv7) | Primary key |
| `event_type` | TEXT | e.g. `backup_success`, `upload_error`, `cache_eviction`, `s3_disconnected` |
| `severity` | TEXT | `info`, `warn`, `err` |
| `message` | TEXT | Human-readable description |
| `metadata` | TEXT (JSON) | Optional structured data (file_id, size_bytes, error details) |
| `created_at` | TEXT | ISO 8601 timestamp |

## 3.4 Data Flow: Gallery Browsing

```
┌──────────┐     ┌─────────────┐     ┌──────────────┐
│  Browser  │────▶│  GET /api/  │────▶│  Auth Middle  │
│ (gallery) │     │  files?...  │     │  (JWT check)  │
└──────────┘     └─────────────┘     └──────┬───────┘
                                            │
                                    ┌───────┴───────┐
                                    │   FileStore    │
                                    │  .List(user_id)│
                                    └───────┬───────┘
                                            │
                                ┌────────────┼────────────┐
                                ▼            ▼            ▼
                         ┌──────────┐ ┌──────────┐ ┌──────────┐
                         │ SQLite   │ │ Return   │ │ Paginate │
                         │ Query    │ │ JSON     │ │ (cursor) │
                         │ WHERE    │ └────┬─────┘ └──────────┘
                         │ user_id=?│      │
                         └──────────┘      │
                                          │
            ┌─────────────────────────────┘
            ▼
   ┌────────────────┐     ┌──────────────┐
   │ Browser renders│     │ GET /thumb/  │
   │ thumbnail grid │────▶│ {id}/sm.jpg  │
   │ with <img> tags│     │              │
   └────────────────┘     └──────┬───────┘
                                 │
                    ┌────────────┼────────────┐
                    ▼            ▼            ▼
             ┌──────────┐ ┌──────────┐ ┌──────────┐
             │ Check    │ │ Serve    │ │ If miss: │
             │ Local    │ │ from     │ │ fetch    │
             │ Cache    │ │ cache    │ │ from S3  │
             └──────────┘ └──────────┘ └──────────┘
```

## 3.5 Key Design Decisions

### Why SQLite instead of PostgreSQL?
- **Zero operational overhead**: No separate database process to manage
- **Single-file backup**: The entire metadata database is one file
- **Sufficient performance**: For a single-user/small-family workload, SQLite handles millions of rows easily
- **WAL mode**: Enables concurrent reads during writes
- **Spatial queries**: SQLite has R-tree indexes for geo-bounding-box queries

### Why Go?
- **Single static binary**: No runtime dependencies, easy Docker image (alpine-based, ~30MB including libvips/libheif)
- **Low memory**: Goroutines are lightweight; idle memory < 50MB without S3 client
- **Fast image processing**: `libvips` bindings are mature in Go
- **Great stdlib**: `net/http` is production-ready; `embed` for bundling the Vue.js SPA

### Why Vue.js over React?
- **Smaller bundle**: Vue 3 + Vite produces smaller bundles than React + Next.js
- **Simpler reactivity**: Pinia stores are more intuitive than Redux/Zustand for this scale
- **Single-page app**: No SSR needed — the Go backend serves the static SPA

### Why Local Cache + S3 (Optional)?
- **Speed**: Local NVMe/SSD serves thumbnails at 3-5 GB/s vs S3 at 100-500 MB/s
- **Resilience**: S3 is optional. In local-only mode, the local disk is the sole storage tier with no external dependencies.
- **Cost**: S3 is cheap for bulk storage; local SSD is limited and expensive per GB. Users choose their balance.
- **Offline resilience**: If S3 is temporarily unreachable, cached thumbnails still work.

## 3.6 Concurrency Model

```
┌────────────────────────────────────────────┐
│              Upload Worker Pool             │
│  ┌──────┐ ┌──────┐ ┌──────┐ ┌──────┐      │
│  │ W-1  │ │ W-2  │ │ W-3  │ │ W-4  │ ...  │
│  └──┬───┘ └──┬───┘ └──┬───┘ └──┬───┘      │
│     │        │        │        │           │
│     └────────┴────────┴────────┘           │
│                    │                        │
│           ┌────────┴────────┐              │
│           │  Poll SQLite    │              │
│           │  upload_jobs    │              │
│           │  (status=queued)│              │
│           └────────┬────────┘              │
└────────────────────┼───────────────────────┘
                     │
              ┌──────┴──────┐
              │ HTTP Handler │
              │ POST /upload │
              └──────┬───────┘
                     │
              ┌──────┴──────┐
              │ INSERT INTO  │
              │ upload_jobs  │
              │ (status:     │
              │  queued)     │
              └──────────────┘
```

- Uploads are persisted directly into the `upload_jobs` SQLite table with `status = 'queued'`
- A **claim** goroutine polls the table every 1 second (or on `NotifyJobsAvailable`) and atomically claims a job via `UPDATE ... WHERE status = 'queued' ORDER BY created_at ASC LIMIT 1`, transitioning it to `processing`
- Claimed jobs are pushed into a buffered channel (`jobCh`, capacity = `concurrent_workers × 2`) consumed by worker goroutines
- **Backpressure**: The claimer checks `len(jobCh) < cap(jobCh)` before claiming from the DB. If the channel is full, no job is claimed — the claimer simply waits for the next tick, leaving jobs safely in `queued` status
- Each worker: dedup check → hashes → extracts EXIF → generates thumbnails → stores to local disk → completes
- S3 uploads (if enabled) are dispatched to a separate pool of 2 dedicated `s3Worker` goroutines via `s3JobCh` (capacity = `concurrent_workers × 4`). This prevents slow S3 operations from blocking the main processing pipeline. If the S3 channel is full, the local copy is retained (non-fatal fallback)
- Progress is stored in the database at each stage and reported via WebSocket to the frontend
- **Crash recovery**: On startup, any jobs stuck in `status = 'processing'` are reset to `queued` and reprocessed
- Temp file existence is validated before processing; missing temp files mark the job as `failed`

## 3.7 Configuration

```yaml
# drive.yaml (mounted into Docker container)
server:
  host: "0.0.0.0"
  port: 8080

storage:
  s3:
    enabled: false                   # Set true to use S3-compatible storage
    endpoint: "https://s3.eu-central-1.amazonaws.com"
    bucket: "drive-media"
    access_key: "${DRIVE_S3_ACCESS_KEY}"
    secret_key: "${DRIVE_S3_SECRET_KEY}"
    region: "eu-central-1"
    use_ssl: true
  local:
    path: "/data"                    # Root for all local storage (originals + cache + DB)

database:
  path: "/data/drive.db"

auth:
  allow_registration: true           # Enable/disable user self-registration
  jwt_secret: "${DRIVE_JWT_SECRET}"  # Auto-generated if not provided
  session_duration_hours: 72

media:
  auto_organize: true
  organization_pattern: "{{year}}/{{month}}"
  thumbnail_sizes:
    small: { width: 60, quality: 60, format: jpeg }
    medium: { width: 600, quality: 75, format: jpeg }
    preview: { max_dimension: 720, quality: 80, format: webp }
    video_still: { max_dimension: 600, quality: 75, format: jpeg }

upload:
  max_file_size_mb: 10240
  concurrent_workers: 4
  allowed_extensions: ["*"]  # all files allowed

map:
  tile_source: "https://tile.openstreetmap.org/{z}/{x}/{y}.png"
  max_cluster_radius: 80

backup:
  enabled: false                    # Set true to enable auto-backup to S3
  interval_h: 24                    # Hours between backups
  retention_days: 0                 # 0 = no expiration, N = expire after N days

---

## 3.8 First-Run Boot Sequence

On startup, Drive follows this sequence:

1. **Load configuration** — Read `drive.yaml` or env vars. If no config file exists, auto-create defaults (listen `:8080`, local-only storage at `./data`, SQLite at `./data/drive.db`, no S3).
2. **Initialize SQLite** — Create database file if missing. Run pending migrations (`001_initial.sql`, `002_fts5.sql`, etc.). Enable WAL mode and foreign keys.
3. **Create storage directories** — Ensure `{storage.local.path}/originals` and `{storage.local.path}/thumbnails` exist.
4. **Check S3 connectivity** (if `s3.enabled: true`) — Validate endpoint, bucket access, and credentials. Log warning on failure but continue (degraded mode).
5. **Start cache eviction goroutine** — Background scheduler that runs LRU eviction every 5 minutes.
5c. **Start backup scheduler goroutine** — If backup.enabled and S3 connected, runs an immediate backup then on configured interval. If disabled or S3 unavailable, logs a warning and skips.
6. **Start HTTP server** — Listen on configured `host:port`. Serve Vue.js SPA (embedded via `embed.FS`) and API.
7. **Admin user setup** — Admin user must be created via CLI: `drive admin create`. First boot without any users shows an error to run the CLI command. Self-registration is available if `auth.allow_registration: true` and at least one admin exists.

---

## 3.9 Development Workflow

**Prerequisites:** Go 1.22+, Node.js 20+, ffmpeg, exiftool, libvips (with libheif for HEIC support).

**Running in development:**
- `make dev` — starts both backend and frontend concurrently with live reload
- **Backend:** `go run ./cmd/drive` on `:8080`, optionally using `air` for file-watch recompilation
- **Frontend:** Vite dev server on `:5173` with HMR; proxies `/api/*` requests to `:8080` via `vite.config.ts`
- **Config:** Copy `drive.example.yaml` → `drive.yaml` and customize
- **Database:** Auto-created at config path on first run
- **Admin setup:** `go run ./cmd/drive admin create` to create the first admin user

**Build commands:**
- `make build` — compile Go binary with embedded SPA
- `make build-web` — build Vue SPA for production (`web/dist/`)
- `make test` — run `go test ./...` and `vitest`
- `make lint` — run `golangci-lint` and `eslint`

**Production:**
- `docker compose up -d` — builds and starts the full stack in a single container