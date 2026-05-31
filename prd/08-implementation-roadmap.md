# 8. Implementation Roadmap

> **Status legend:** ✅ Complete | ⚠️ Partial | ❌ Not implemented

## 8.1 Overview

The project is broken into 4 phases over approximately 16-20 weeks for a solo developer. Each phase delivers a working, deployable increment.

```
Phase 0: Foundation        (Weeks 1-3)    ✅ Complete — Upload + thumbnail pipeline + auth works
Phase 1: Core MVP          (Weeks 4-8)    ⚠️ ~85% — Gallery, timeline, map, auth UI, usable product
Phase 2: Polish & Launch   (Weeks 9-12)   ⚠️ ~35% — Performance, testing, production ready
Phase 3: Differentiators   (Weeks 13-20)  ❌ 0% — AI tagging, mobile apps, albums
```

---

## 8.2 Phase 0 — Foundation (Weeks 1-3) — ✅ Complete

**Goal:** File upload works end-to-end. Thumbnails are generated and served. S3 storage is operational.

### Week 1: Project Scaffolding & Database

| Task | Details | Deliverable | Status |
|---|---|---|---|
| Initialize Go module | `go mod init github.com/piotrosinski/drive` | `go.mod`, `go.sum` | ✅ |
| Project structure | `cmd/`, `internal/`, `web/`, `migrations/` | Directory tree | ✅ |
| SQLite schema | Implement `001_initial.sql` from Data Model | Migration runner | ✅ |
| FTS5 schema | Implement `002_fts5.sql` for full-text search | Migration runner | ✅ |
| File model + CRUD | Go structs, insert/query/update/delete | `internal/model/file.go` | ✅ |
| EXIF model + CRUD | Go structs, insert/query | `internal/model/exif.go` | ✅ |
| Thumbnail model + CRUD | Go structs, insert/query (4 sizes: sm, md, preview, video_still) | `internal/model/thumbnail.go` | ✅ |
| User model + CRUD | Go structs, insert/query | `internal/model/user.go` | ✅ |
| Session model + CRUD | Go structs, insert/delete | `internal/model/session.go` | ✅ |
| Database layer | Connection pool, migration runner, query helpers | `internal/db/` | ✅ (in `internal/store/sqlite.go`) |

### Week 2: Upload Pipeline

| Task | Details | Deliverable | Status |
|---|---|---|---|
| HTTP server skeleton | Chi router, middleware, health endpoint | `cmd/server/main.go` | ✅ (`internal/server/server.go`) |
| Auth service + middleware | JWT creation/validation, bcrypt password hashing | `internal/service/auth.go`, `internal/middleware/auth.go` | ✅ (in `internal/server/auth.go`, `middleware.go`) |
| Auth endpoints | `POST /api/v1/auth/register`, `/login`, `/refresh`, `/logout`, `GET /api/v1/auth/me`, `GET /api/v1/auth/config` | `internal/handler/auth.go` | ✅ (in `internal/server/auth.go`) |
| Admin CLI command | `drive admin create` — CLI subcommand to bootstrap first admin user | `cmd/drive/main.go` | ✅ |
| Admin create user | `POST /api/v1/admin/users` — admin creates users from admin panel | `internal/server/handlers.go` | ✅ |
| Registration toggle | `GET/PUT /api/v1/admin/registration` — runtime toggle persisted in SQLite settings table; `GET /api/v1/auth/config` — public endpoint for UI | `internal/server/handlers.go`, `migration_008_settings.sql` | ✅ |
| Upload endpoint | `POST /api/v1/upload` — multipart parsing, validation | `internal/handler/upload.go` | ✅ (`internal/server/upload.go`) |
| Name+Size dedup check | Query DB for existing file with same `original_name` + `size_bytes` | `internal/service/dedup.go` | ✅ (in worker pipeline) |
| SHA-256 hashing | Stream file to temp, compute hash, detect content-level duplicates | `internal/service/dedup.go` | ✅ (in worker pipeline) |
| Local storage | Write originals to `{storage.local.path}/originals/...` | `internal/storage/local.go` | ✅ (in worker pipeline) |
| S3 upload | MinIO client, conditional on `s3.enabled` config | `internal/storage/s3.go` | ✅ (`internal/service/storage.go`) |
| EXIF extraction | `goexif` for JPEG/PNG/TIFF, `exiftool` subprocess fallback for HEIC/AVIF | `internal/service/exif.go` | ✅ |
| Date-based path | Auto-organize: `YYYY/MM/filename` from EXIF date or file mtime | `internal/service/organizer.go` | ✅ (in worker pipeline) |
| Async job queue | Go channel-based worker pool (4 workers default) for processing | `internal/worker/pool.go` | ✅ |

### Week 3: Thumbnail Generation

| Task | Details | Deliverable | Status |
|---|---|---|---|
| libvips integration | Go bindings, resize + crop pipeline, autorotate for EXIF orientation | `internal/imaging/vips.go` | ❌ Using `github.com/disintegration/imaging` (pure Go) instead of libvips |
| Thumbnail sizes | Generate `sm` (60px JPEG), `md` (600px JPEG), `preview` (720p WebP), `video_still` (600px JPEG, frame at 5s via ffmpeg) | Worker task | ✅ (plus extra `lg` 300px size) |
| Local cache | Write thumbnails to `{cache_dir}/thumbnails/{file_id}/{size}.{format}` | `internal/cache/local.go` | ✅ (in `internal/service/thumbnail.go`) |
| S3 thumbnail upload | Upload generated thumbnails to S3 (if `s3.enabled: true`) | Worker task | ✅ |
| Video proxy generation | 720p H.264/AAC MP4 from uploaded video | `internal/imaging/ffmpeg.go` | ❌ Only video still frames generated, no full video transcoding |
| Thumbnail serve endpoint | `GET /api/v1/thumb/{file_id}/{size}.{format}` — cache-first, S3 fallback | `internal/handler/thumbnail.go` | ✅ (in `internal/server/handlers.go`) |
| Cache eviction | LRU eviction, scheduled background goroutine (every 5 min) | `internal/cache/eviction.go` | ✅ (`internal/server/cache.go`) |

---

## 8.3 Phase 1 — Core MVP (Weeks 4-8) — ⚠️ ~85% Complete

**Goal:** Full gallery browsing, timeline, map, and lightbox. The product is usable for daily photo browsing.

### Week 4: Gallery API + Backend + Auth Frontend — ✅ Complete

| Task | Details | Deliverable | Status |
|---|---|---|---|
| File list endpoint | `GET /api/v1/files` with pagination, sorting, filtering, scoped to user_id | `internal/handler/file.go` | ✅ |
| File detail endpoint | `GET /api/v1/files/{id}` with joined EXIF + thumbnails | `internal/handler/file.go` | ✅ |
| Directory tree endpoint | `GET /api/v1/dirs` | `internal/handler/dir.go` | ✅ |
| Search endpoint | `GET /api/v1/search` — SQLite FTS5 on filename, scoped to user_id | `internal/handler/search.go` | ✅ |
| Stats endpoint | `GET /api/v1/stats` | `internal/handler/stats.go` | ✅ |
| Delete endpoints | Soft delete + permanent delete, scoped to user_id | `internal/handler/file.go` | ✅ |
| Batch operations | Batch delete, move, copy | — | ✅ |
| Vite + Vue 3 project | `npm create vue@latest`, TypeScript, Vue Router, Pinia, Tailwind CSS | `web/` directory | ✅ |
| API client + auth store | Axios/fetch wrapper with JWT token management, auto-refresh interceptor | `web/src/api/client.ts`, `web/src/stores/auth.ts` | ✅ |
| Login/Register views | Auth forms with validation, error states, route guard | `web/src/views/LoginView.vue`, `web/src/views/RegisterView.vue` | ✅ |
| Folders CRUD | Create, rename, delete folders with hierarchy | `internal/handler/` | ✅ |

### Week 5: Vue.js Frontend — Gallery & Admin — ⚠️ Mostly Complete

| Task | Details | Deliverable | Status |
|---|---|---|---|
| Thumbnail grid component | CSS Grid, lazy loading, skeleton states | `web/src/components/GalleryGrid.vue` | ⚠️ Tiles/List/Grouped views exist, no skeleton states |
| Infinite scroll | Intersection Observer, cursor-based pagination | Composable | ✅ |
| Deep-linkable URLs | `useRouteQuery` composable, URL-synced gallery state | `web/src/composables/useRouteQuery.ts`, GalleryView | ✅ |
| Thumbnail card | Image, video overlay, file icon, hover effects | `web/src/components/ThumbnailCard.vue` | ✅ |
| Sort/filter bar | Media type filter, sort dropdown, search input | `web/src/components/FilterBar.vue` | ✅ |
| Directory sidebar | Tree component, collapsible, highlights current path | `web/src/components/DirectoryTree.vue` | ✅ |
| Admin panel | User list, role management, registration toggle, per-user space quota, per-user thumbnail breakdown, quota progress bar in top nav, per-user file/thumbnail breakdown selectors, quota enforcement on upload | `web/src/views/AdminView.vue`, `web/src/App.vue`, `internal/server/handlers.go`, `internal/store/user.go`, `internal/server/upload.go`, `internal/store/file.go`, `internal/store/thumbnail.go` | ⚠️ User management works; registration toggle is read-only; dashboard added; quota management completed; per-user breakdowns completed; quota enforcement completed |
| Dedup per-user scoping | FindBySHA256/FindByNameAndSize filtered by user_id | `internal/store/file.go`, `internal/worker/pool.go`, `internal/server/upload.go` | ✅ |
| User settings/logout | User menu, profile display, logout action | Composable | ✅ |
| Folders tab | Dedicated `/folders` route with tiles/list/table layouts | `web/src/views/FoldersView.vue` | ✅ |

### Week 6: Vue.js Frontend — Lightbox & Detail — ⚠️ Partial

| Task | Details | Deliverable | Status |
|---|---|---|---|
| Lightbox component | Full-screen overlay, prev/next, keyboard nav, pinch zoom | `web/src/components/Lightbox.vue` | ⚠️ Lightbox works but no pinch-to-zoom |
| EXIF panel | Slide-up panel with camera, lens, settings, GPS | `web/src/components/ExifPanel.vue` | ❌ EXIF shown inline in Lightbox footer, no dedicated slide-up panel |
| Image zoom | Pinch-to-zoom and scroll-to-zoom | Composable | ❌ Not implemented |
| Swipe navigation | Touch swipe left/right for prev/next | Composable | ❌ Basic touch handlers exist but no swipe gestures |
| Download button | Single file download + batch ZIP | `web/src/components/DownloadButton.vue` | ❌ Download button is inline in Lightbox, no dedicated component |
| Delete with confirmation | Soft delete with undo toast | `web/src/components/DeleteButton.vue` | ❌ No dedicated component; batch delete via ActionBar only |

### Week 7: Timeline & Map Views — ⚠️ Partial

| Task | Details | Deliverable | Status |
|---|---|---|---|
| Timeline endpoint | `GET /api/v1/timeline` with year/month/day granularity | `internal/handler/timeline.go` | ✅ |
| Timeline component | Vertical timeline, month strips, date picker | `web/src/views/TimelineView.vue` | ⚠️ Timeline works; no date picker/scrubber |
| Geo points endpoint | `GET /api/v1/geo/points` with bbox filter | `internal/handler/geo.go` | ✅ |
| Geo clusters endpoint | `GET /api/v1/geo/clusters` with H3 hexagons | `internal/handler/geo.go` | ⚠️ Uses grid-based clustering, not H3 hexagons |
| Map component | Leaflet, cluster layer (Supercluster), marker layer | `web/src/views/MapView.vue` | ✅ (via PhotoMap component) |
| Map bottom sheet | Photo strip when clicking a cluster/marker | `web/src/components/MapBottomSheet.vue` | ❌ Simple inline info panel only, no photo strip |
| Heatmap layer | Toggle heatmap overlay | `web/src/components/HeatmapLayer.vue` | ❌ Not implemented; `geo/heatmap` endpoint also missing |

### Week 8: Upload UI & Mobile Responsiveness — ⚠️ Partial

| Task | Details | Deliverable | Status |
|---|---|---|---|
| Upload page | Drag & drop zone, file picker, folder picker | `web/src/views/UploadView.vue` | ✅ Replaced by InlineUpload component in gallery/folder views |
| Upload progress | Per-file progress bars, WebSocket for live updates | `web/src/components/GlobalUploadTracker.vue` | ✅ |
| WebSocket endpoint | `WS /api/v1/upload/ws` for real-time progress | `internal/handler/upload_ws.go` | ✅ (`internal/server/upload.go`) |
| Folder auto-refresh | Gallery/folder view listens for WS completion events | GalleryView, FolderTreeView, upload store | ✅ |
| Responsive layout | Mobile bottom nav, tablet sidebar, desktop full layout | `web/src/App.vue` + CSS | ✅ |
| Cross-folder gallery filter | "All folders" toggle includes files from all folders (not just root) | FilterBar, GalleryView, backend List API | ✅ |
| Gallery upload restriction | Gallery InlineUpload restricted to image/video MIME types | InlineUpload, GalleryView | ✅ |
| PWA manifest | `manifest.json`, service worker for offline caching | `web/public/` | ❌ No manifest.json or service worker |
| Dark/light theme | CSS variables, theme toggle, persisted preference | `web/src/composables/useTheme.ts` | ❌ CSS variables exist but no toggle; dark-only |

---

## 8.4 Phase 2 — Polish & Launch (Weeks 9-12) — ⚠️ ~35% Complete

### Week 9: Performance Optimization & Testing — ⚠️ Partial

| Task | Details | Deliverable | Status |
|---|---|---|---|
| Virtual scrolling | For galleries with 10,000+ thumbnails | `vue-virtual-scroller` | ❌ Not implemented |
| Image lazy loading | Native `loading="lazy"` + Intersection Observer fallback | Gallery grid | ✅ Basic lazy loading via Intersection Observer |
| HTTP caching | ETags, Cache-Control, 304 responses | Middleware | ❌ Only immutable cache header on thumbnails |
| SQLite optimization | WAL mode, prepared statements, query plan analysis | `internal/db/` | ⚠️ WAL mode enabled; no statement caching |
| Bundle optimization | Vite code splitting, tree shaking, compression | `vite.config.ts` | ❌ Basic Vite config, no explicit optimization |
| Backend unit tests | Go tests for handlers, services, stores (target: 70%+ coverage) | `internal/**/*_test.go` | ✅ Tests exist for all layers |
| Frontend unit tests | Vitest component tests for Pinia stores, composables, key components | `web/src/**/*.test.ts` | ✅ 122 tests across 18 test files |
| Lighthouse audit | Target: 95+ Performance, 100 Accessibility | — | ❌ Not performed |

### Week 10: Error Handling & Resilience — ⚠️ Partial

| Task | Details | Deliverable | Status |
|---|---|---|---|
| Durable upload queue | Persist upload jobs in SQLite `upload_jobs` table. Workers poll DB instead of in-memory Go channel. Crash recovery resets stuck `processing` jobs to `queued`. | `migration_005_upload_jobs.sql`, `internal/store/uploadjob.go`, `internal/worker/pool.go` refactor | ✅ |
| Graceful degradation | S3 down → serve from cache, show banner | `internal/cache/` | ✅ S3 banner in App.vue |
| Upload retry | Exponential backoff for failed S3 uploads | Worker | ❌ Not implemented |
| Upload job history in Admin UI | Paginated job list with status filter, retry capability, summary counts | `GET /admin/jobs`, AdminView section | ✅ |
| Thumbnail reconciliation | Scan for missing thumbnails, create repair jobs, periodic background reconciler (30min) | `POST /admin/jobs/reconcile`, `RunReconciliation()`, Pool reconciler goroutine | ✅ |
| On-demand thumbnail fallback | Serve next smaller thumbnail size when requested size is unavailable on disk and S3 | `handleServeThumbnail` fallback chain | ✅ |
| Thumbnail file write sync | `f.Sync()` + explicit close before `os.Stat` in thumbnail generation; pre-S3 file verification in worker | `internal/service/thumbnail.go`, `internal/worker/pool.go` | ✅ |
| Corrupt file handling | Detect during processing, quarantine, notify | `internal/service/validator.go` | ❌ Not implemented |
| Disk space monitoring | Alert when cache disk <10% free | `internal/monitor/disk.go` | ✅ Admin dashboard shows disk utilization |
| Structured logging | `slog` (Go stdlib), JSON format, request IDs | Middleware | ⚠️ Uses slog; no request ID propagation |
| Health checks | `/health` with DB + S3 connectivity check | `internal/handler/health.go` | ✅ |

### Week 10b: File Viewer (Non-Media Documents) — Frontend-Only Feature

| Task | Details | Deliverable | Status |
|---|---|---|---|
| FileViewer modal shell | Modal overlay with header (close + download), footer (file info), content slot | `web/src/components/FileViewer.vue` | ✅ |
| PdfViewer sub-component | Fetch file blob, create blob URL, render in `<iframe>` | `web/src/components/viewers/PdfViewer.vue` | ✅ |
| JsonViewer sub-component | Pretty-print JSON with syntax highlighting (keys, strings, numbers, booleans) | `web/src/components/viewers/JsonViewer.vue` | ✅ |
| MarkdownViewer sub-component | Render markdown to HTML via `marked` library, dark theme CSS | `web/src/components/viewers/MarkdownViewer.vue` | ✅ |
| CsvViewer sub-component | Parse CSV, render as HTML table with sticky header and scrollable body | `web/src/components/viewers/CsvViewer.vue` | ✅ |
| TextViewer sub-component | Plain text display in monospace `<pre>` block | `web/src/components/viewers/TextViewer.vue` | ✅ |
| GalleryView integration | Route `mediaType === 'file'` clicks to FileViewer instead of Lightbox | `web/src/views/GalleryView.vue` | ✅ |
| Component tests | Vitest tests for FileViewer routing and each sub-viewer | `web/src/components/FileViewer.test.ts` + viewer tests | ✅ |

### Week 11: Docker & Deployment — ⚠️ Mostly Complete

| Task | Details | Deliverable | Status |
|---|---|---|---|
| Multi-stage Dockerfile | Build Go binary + Vue.js frontend, alpine final image | `Dockerfile` | ✅ |
| Docker Compose | Drive + optional MinIO + Caddy for HTTPS | `docker-compose.yml` | ⚠️ No MinIO or Caddy integration |
| Environment config | `.env` file, env vars, config validation on startup | `internal/config/` | ✅ |
| Caddy reverse proxy | Auto-HTTPS, gzip, static file caching | `Caddyfile` | ❌ Not included |
| Healthcheck | Docker HEALTHCHECK instruction | `Dockerfile` | ✅ |
| One-liner deploy | `docker compose up -d` | Install script | ✅ |

### Week 12: Documentation & Landing Page — ⚠️ Partial

| Task | Details | Deliverable | Status |
|---|---|---|---|
| README | Features, quick start, configuration, screenshots | `README.md` | ✅ |
| API docs | OpenAPI/Swagger spec generated from code | `docs/api/openapi.yaml` | ❌ Not generated |
| Landing page | Simple marketing page for the project | `web/src/views/Landing.vue` | ❌ Not created |
| Screenshots | Gallery, timeline, map, lightbox, upload | `docs/screenshots/` | ❌ Not created |
| Comparison table | Drive vs Immich vs Photoprism vs Google Photos | `README.md` | ❌ Not included |

---

## 8.5 Phase 3 — Differentiators (Weeks 13-20) — ❌ 0% Complete

### Weeks 13-14: AI Tagging — ❌

| Task | Details | Deliverable | Status |
|---|---|---|---|
| ONNX runtime setup | Go bindings, model loading | `internal/ai/onnx.go` | ❌ |
| Object detection model | MobileNet SSD or YOLO-tiny, quantized | Model file (~20MB) | ❌ |
| Scene classification | EfficientNet-lite, top-5 labels | Model file (~15MB) | ❌ |
| Inference worker | Background job, processes new uploads | `internal/worker/ai_worker.go` | ❌ |
| Tag storage | `photo_tags` table with confidence scores | Migration | ❌ |
| Semantic search | `GET /api/v1/search?q=beach&semantic=true` | `internal/handler/search.go` | ❌ |
| Tag UI | Tag chips, tag cloud, "People" view | Vue components | ❌ |

### Weeks 15-16: Mobile Auto-Backup — ❌

| Task | Details | Deliverable | Status |
|---|---|---|---|
| iOS app scaffold | SwiftUI, Photos framework permission | `mobile/ios/` | ❌ |
| Background upload | BGTaskScheduler, background URLSession | iOS app | ❌ |
| Android app scaffold | Jetpack Compose, MediaStore permission | `mobile/android/` | ❌ |
| Background upload | WorkManager, constrained by Wi-Fi + charging | Android app | ❌ |
| Device registration API | `POST /api/v1/devices` | `internal/handler/device.go` | ❌ |
| PWA fallback | Web Share Target, Background Sync | `web/src/` | ❌ |

### Weeks 17-18: Duplicate Detection & Albums — ❌

| Task | Details | Deliverable | Status |
|---|---|---|---|
| pHash generation | Go library, compute during upload | `internal/service/hash.go` | ❌ |
| Duplicate finder | Hamming distance query, background scan | `internal/service/duplicate.go` | ❌ |
| Duplicate UI | Side-by-side comparison, batch actions | `web/src/views/DuplicatesView.vue` | ❌ |
| Album CRUD API | `POST/GET/PUT/DELETE /api/v1/albums` | `internal/handler/album.go` | ❌ |
| Album items API | `POST/DELETE /api/v1/albums/{id}/items` | `internal/handler/album.go` | ❌ |
| Album UI | Grid, drag-and-drop, smart album builder | `web/src/views/AlbumsView.vue` | ❌ |

### Weeks 19-20: Video Streaming & RAW Support — ❌

| Task | Details | Deliverable | Status |
|---|---|---|---|
| HLS transcoding | Generate .m3u8 + .ts segments | `internal/imaging/ffmpeg.go` | ❌ |
| Video player | Plyr component with HLS support | `web/src/components/VideoPlayer.vue` | ⚠️ Basic VideoPlayer exists, no HLS |
| RAW preview extraction | exiftool/dcraw for embedded JPEG | `internal/imaging/raw.go` | ❌ |
| RAW badge | "RAW" indicator on thumbnails | `web/src/components/ThumbnailCard.vue` | ❌ |

---

## 8.6 Future (Beyond Phase 3)

These are not scheduled but represent the long-term vision:

- Plugin system for storage/processing/export
- Print product integration
- Ecosystem migration tools (Google Takeout, iCloud, Immich)
- Advanced map features (travel routes, 3D terrain)
- Collaborative shared albums
- OIDC / SSO integration
- ActivityPub federation for decentralized sharing

---

## 8.7 Milestone Summary

```
Week 3:  ████████████  Phase 0 Complete ✅ — Upload + thumbnails + auth work
Week 8:  █████████████████████████████░░░  Phase 1 ~85% ⚠️ — MVP shipped (missing: pinch-zoom, ExifPanel, PWA, theme toggle, heatmap, map bottom sheet)
Week 12: ████████████████░░░░░░░░░░░░░░░░  Phase 2 ~35% ⚠️ — Production ready (missing: virtual scroll, HTTP caching, Caddy, API docs, OpenAPI, landing page)
Week 20: ░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░  Phase 3 0% ❌ — Full platform (nothing started)
```

---

## 8.8 Risk Register

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| libvips CGo issues on ARM | Medium | High | Test on Raspberry Pi early; fallback to ImageMagick |
| libheif compilation in Docker | Medium | Medium | Use alpine with `libheif-dev` package; verify in CI |
| SQLite concurrency limits | Low | Medium | WAL mode handles reads well; single writer is fine for family workload |
| S3 costs at scale | Low | Medium | Local cache minimizes S3 reads; MinIO self-hosted is free; local-only mode avoids entirely |
| Auth security: JWT secret leakage | Low | High | Auto-generate random secret on first boot; document .env best practices |
| Mobile app store rejection | Medium | Low | PWA fallback works for most use cases |
| ONNX model accuracy | Medium | Low | Tags are additive, not critical path; user can always search by date |
| Browser video codec support | Low | Medium | Transcode to H.264 which is universally supported |
