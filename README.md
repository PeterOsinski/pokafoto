# Drive — Self-hosted photo & file backup

Drive lets you back up photos and files to your own server. Browse by timeline or map, search by EXIF metadata, and keep everything under your control.

## Prerequisites

- **Go 1.25**, **Node 22**, `ffmpeg`, `exiftool` — or just **Docker**

## Quick Start

```bash
docker compose up -d
docker compose exec drive drive admin create
# Open http://localhost:8080
```

## Running Locally

```bash
make dev          # backend on :8080
make dev-web      # Vite dev server on :5173
make test-all     # 185 tests (137 Go + 48 Vue)
make build        # compile binary to bin/drive
make build-web    # build SPA to web/dist/
```

## Running with Docker

```bash
make docker       # docker compose build
make docker-up    # docker compose up -d
make docker-down  # docker compose down
```

After starting, create your admin user:

```bash
docker compose exec drive drive admin create
```

Open `http://localhost:8080` and log in.

### Data Persistence

The `docker-compose.yml` mounts `./data` to `/data` inside the container. All uploaded files, thumbnails, and the SQLite database live there.

## Configuration

All configuration is via environment variables.

| Variable | Default | Description |
|---|---|---|
| `DRIVE_STORAGE_PATH` | `./data` | Root directory for originals and thumbnails |
| `DRIVE_DB_PATH` | `./data/drive.db` | SQLite database path |
| `DRIVE_PORT` | `8080` | Server listen port |
| `DRIVE_JWT_SECRET` | auto-generated | JWT signing secret |
| `DRIVE_ALLOW_REGISTRATION` | `true` | Allow new user registration |
| `DRIVE_S3_ENABLED` | `false` | Enable S3-compatible storage |
| `DRIVE_S3_ENDPOINT` | — | S3 endpoint URL |
| `DRIVE_S3_BUCKET` | — | S3 bucket name |
| `DRIVE_S3_ACCESS_KEY` | — | S3 access key |
| `DRIVE_S3_SECRET_KEY` | — | S3 secret key |
| `DRIVE_S3_REGION` | — | S3 region |

**Production note**: Set a fixed `DRIVE_JWT_SECRET`. When empty, the server auto-generates one on every restart — this invalidates all existing sessions.

## Architecture

```
cmd/drive/           — entry point, CLI (serve + admin create)
internal/
├── config/          — env var parsing, defaults
├── model/           — domain types (File, User, Exif, Thumbnail)
├── store/           — SQLite via :memory: in tests, embedded migrations
├── server/          — chi router, auth, handlers, SPA serving
├── service/         — EXIF extraction, thumbnail generation, S3 storage
└── worker/          — background upload pipeline
web/                 — Vue 3 SPA (Pinia, Leaflet, vitest)
migrations/          — empty; real migrations are embedded in store/ as SQL files
```

- **SQLite** with WAL mode, embedded migrations via `//go:embed`
- **Pure Go** background workers with configurable pool size
- **Vue 3 SPA** with Pinia stores, Leaflet maps, and real-time upload progress

## Background Processing

Uploads go through a 5-stage pipeline in `internal/worker/pool.go`:

| Stage | Progress | Description |
|---|---|---|
| Hashing | 10% | SHA-256 content hash of the temp file |
| Dedup | 20% | Content-hash dedup (always); name+size dedup for photos |
| EXIF | 30% | Extract metadata via goexif (exiftool fallback for HEIC/AVIF) |
| Storing | 50% | Copy to organized path, create DB records |
| Thumbnails | 80% | Generate sm(60px)/md(600px)/preview(720px WebP) |

**Storage organization**: `{userID}/{year}/{month}/` for media, `{userID}/files/{year}/{month}/` for non-media files.

**Video support**: Detects mp4/mov/mkv, extracts stills via ffmpeg at 5 seconds, generates preview WebP.

## API Endpoints

### Public

| Method | Path | Auth | Description |
|---|---|---|---|
| `POST` | `/api/v1/auth/register` | No | Register new user |
| `POST` | `/api/v1/auth/login` | No | Login, returns JWT + refresh token |
| `POST` | `/api/v1/auth/refresh` | No | Refresh JWT |
| `POST` | `/api/v1/auth/logout` | No | Invalidate session |
| `GET` | `/api/v1/health` | No | Health check (DB + S3 status) |
| `GET` | `/api/v1/thumb/{fileID}/{size}` | No | Serve thumbnail by UUID (sm/md/preview/video_still) |
| `GET` | `/api/v1/upload/ws` | No | WebSocket for live upload progress |

### Authenticated

| Method | Path | Description |
|---|---|---|
| `GET` | `/api/v1/auth/me` | Current user info |
| `POST` | `/api/v1/upload` | Upload file (multipart) |
| `GET` | `/api/v1/upload/{batchID}/status` | Upload batch progress |
| `GET` | `/api/v1/files` | List files (cursor pagination, filters) |
| `GET` | `/api/v1/files/{id}` | File metadata |
| `DELETE` | `/api/v1/files/{id}` | Soft-delete file |
| `DELETE` | `/api/v1/files/{id}/permanent` | Permanent delete |
| `GET` | `/api/v1/dirs` | List directories |
| `GET` | `/api/v1/search` | Search by filename |
| `GET` | `/api/v1/timeline` | Files grouped by date |
| `GET` | `/api/v1/geo/points` | Geo-tagged file points |
| `GET` | `/api/v1/geo/clusters` | Geo-clustered points for map |
| `GET` | `/api/v1/stats` | User storage statistics |
| `GET` | `/api/v1/download/{id}` | Download single file |
| `POST` | `/api/v1/download/batch` | Download batch (zip) |

### Admin

| Method | Path | Description |
|---|---|---|
| `GET` | `/api/v1/admin/users` | List all users |
| `DELETE` | `/api/v1/admin/users/{id}` | Delete user |
| `PUT` | `/api/v1/admin/users/{id}/role` | Change user role |

## S3 / MinIO Setup

Drive supports S3-compatible storage for originals and thumbnails. When S3 is enabled, files are stored both locally and in S3. Example MinIO configuration:

```yaml
environment:
  - DRIVE_S3_ENABLED=true
  - DRIVE_S3_ENDPOINT=https://minio.example.com
  - DRIVE_S3_BUCKET=drive
  - DRIVE_S3_ACCESS_KEY=your-key
  - DRIVE_S3_SECRET_KEY=your-secret
  - DRIVE_S3_REGION=us-east-1
```

## Testing

```bash
make test        # go test -count=1 ./...        (137 tests)
make test-cover  # go test -race -count=1 ...     (with race detection)
make test-web    # npx vitest run                 (48 tests)
make test-all    # both combined                  (185 tests)
```

Backend tests use real `:memory:` SQLite with embedded migrations — no mocking for data access. Frontend tests use `@vue/test-utils` with mocked API clients.

## Production Notes

- **JWT secret**: Set `DRIVE_JWT_SECRET` to a fixed, strong value. Auto-generated secrets change on restart, breaking all sessions.
- **TLS termination**: Run behind nginx or Caddy for HTTPS.
- **Backups**: The `./data` directory contains everything — SQLite DB, original files, and thumbnails. Back it up regularly.
- **Registration**: Disable after creating users with `DRIVE_ALLOW_REGISTRATION=false`.
