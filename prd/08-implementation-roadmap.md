# 8. Implementation Roadmap

## 8.1 Overview

The project is broken into 4 phases over approximately 16-20 weeks for a solo developer. Each phase delivers a working, deployable increment.

```
Phase 0: Foundation        (Weeks 1-3)    → Upload + thumbnail pipeline + auth works
Phase 1: Core MVP          (Weeks 4-8)    → Gallery, timeline, map, auth UI, usable product
Phase 2: Polish & Launch   (Weeks 9-12)   → Performance, testing, production ready
Phase 3: Differentiators   (Weeks 13-20)  → AI tagging, mobile apps, albums
```

---

## 8.2 Phase 0 — Foundation (Weeks 1-3)

**Goal:** File upload works end-to-end. Thumbnails are generated and served. S3 storage is operational.

### Week 1: Project Scaffolding & Database

| Task | Details | Deliverable |
|---|---|---|---|
| Initialize Go module | `go mod init github.com/piotrosinski/drive` | `go.mod`, `go.sum` |
| Project structure | `cmd/`, `internal/`, `web/`, `migrations/` | Directory tree |
| SQLite schema | Implement `001_initial.sql` from Data Model | Migration runner |
| FTS5 schema | Implement `002_fts5.sql` for full-text search | Migration runner |
| File model + CRUD | Go structs, insert/query/update/delete | `internal/model/file.go` |
| EXIF model + CRUD | Go structs, insert/query | `internal/model/exif.go` |
| Thumbnail model + CRUD | Go structs, insert/query (4 sizes: sm, md, preview, video_still) | `internal/model/thumbnail.go` |
| User model + CRUD | Go structs, insert/query | `internal/model/user.go` |
| Session model + CRUD | Go structs, insert/delete | `internal/model/session.go` |
| Database layer | Connection pool, migration runner, query helpers | `internal/db/` |

### Week 2: Upload Pipeline

| Task | Details | Deliverable |
|---|---|---|---|
| HTTP server skeleton | Chi router, middleware, health endpoint | `cmd/server/main.go` |
| Auth service + middleware | JWT creation/validation, bcrypt password hashing, user login/register/refresh/logout | `internal/service/auth.go`, `internal/middleware/auth.go` |
| Auth endpoints | `POST /api/v1/auth/register`, `/login`, `/refresh`, `/logout`, `GET /api/v1/auth/me` | `internal/handler/auth.go` |
| Admin CLI command | `drive admin create` — CLI subcommand to bootstrap first admin user with username/password prompt | `cmd/drive/main.go` |
| Upload endpoint | `POST /api/v1/upload` — multipart parsing, validation, `webkitRelativePath` support for recursive folder upload | `internal/handler/upload.go` |
| Name+Size dedup check | For photos folder: query DB for existing file with same `original_name` + `size_bytes`, skip if found (silent, no DB update) | `internal/service/dedup.go` |
| SHA-256 hashing | Stream file to temp, compute hash, detect content-level duplicates (returns 409 Conflict) | `internal/service/dedup.go` |
| Local storage | Write originals to `{storage.local.path}/originals/...` | `internal/storage/local.go` |
| S3 upload | MinIO client, conditional on `s3.enabled` config | `internal/storage/s3.go` |
| EXIF extraction | `goexif` for JPEG/PNG/TIFF, `exiftool` subprocess fallback for HEIC/AVIF | `internal/service/exif.go` |
| Date-based path | Auto-organize: `YYYY/MM/filename` from EXIF date or file mtime | `internal/service/organizer.go` |
| Async job queue | Go channel-based worker pool (4 workers default) for processing | `internal/worker/pool.go` |

### Week 3: Thumbnail Generation

| Task | Details | Deliverable |
|---|---|---|---|
| libvips integration | Go bindings, resize + crop pipeline, autorotate for EXIF orientation | `internal/imaging/vips.go` |
| Thumbnail sizes | Generate `sm` (60px JPEG), `md` (600px JPEG), `preview` (720p WebP), `video_still` (600px JPEG, frame at 5s via ffmpeg) | Worker task |
| Local cache | Write thumbnails to `{cache_dir}/thumbnails/{file_id}/{size}.{format}` | `internal/cache/local.go` |
| S3 thumbnail upload | Upload generated thumbnails to S3 (if `s3.enabled: true`) | Worker task |
| Video proxy generation | 720p H.264/AAC MP4 from uploaded video | `internal/imaging/ffmpeg.go` |
| Thumbnail serve endpoint | `GET /api/v1/thumb/{file_id}/{size}.{format}` — cache-first, S3 fallback | `internal/handler/thumbnail.go` |
| Cache eviction | LRU eviction, scheduled background goroutine (every 5 min) | `internal/cache/eviction.go` |

**Phase 0 Deliverable:** `drive admin create` → register user → login → `curl -F "files=@photo.jpg" localhost:8080/api/v1/upload` (with JWT) → file stored, thumbnails generated, `curl localhost:8080/api/v1/thumb/{id}/sm.jpg` returns a 60px thumbnail.

---

## 8.3 Phase 1 — Core MVP (Weeks 4-8)

**Goal:** Full gallery browsing, timeline, map, and lightbox. The product is usable for daily photo browsing.

### Week 4: Gallery API + Backend + Auth Frontend

| Task | Details | Deliverable |
|---|---|---|---|
| File list endpoint | `GET /api/v1/files` with pagination, sorting, filtering, scoped to user_id | `internal/handler/file.go` |
| File detail endpoint | `GET /api/v1/files/{id}` with joined EXIF + thumbnails | `internal/handler/file.go` |
| Directory tree endpoint | `GET /api/v1/dirs` | `internal/handler/dir.go` |
| Search endpoint | `GET /api/v1/search` — SQLite FTS5 on filename, scoped to user_id | `internal/handler/search.go` |
| Stats endpoint | `GET /api/v1/stats` | `internal/handler/stats.go` |
| Delete endpoints | Soft delete + permanent delete, scoped to user_id | `internal/handler/file.go` |
| Vite + Vue 3 project | `npm create vue@latest`, TypeScript, Vue Router, Pinia, Tailwind CSS | `web/` directory |
| API client + auth store | Axios/fetch wrapper with JWT token management, auto-refresh interceptor | `web/src/api/client.ts`, `web/src/stores/auth.ts` |
| Login/Register views | Auth forms with validation, error states, route guard | `web/src/views/LoginView.vue`, `web/src/views/RegisterView.vue` |

### Week 5: Vue.js Frontend — Gallery & Admin

| Task | Details | Deliverable |
|---|---|---|---|
| Thumbnail grid component | CSS Grid, lazy loading, skeleton states | `web/src/components/GalleryGrid.vue` |
| Infinite scroll | Intersection Observer, cursor-based pagination | Composable |
| Thumbnail card | Image, video overlay, file icon, hover effects | `web/src/components/ThumbnailCard.vue` |
| Sort/filter bar | Media type filter, sort dropdown, search input | `web/src/components/FilterBar.vue` |
| Directory sidebar | Tree component, collapsible, highlights current path | `web/src/components/DirSidebar.vue` |
| Admin panel | User list, role management, registration toggle | `web/src/views/AdminView.vue` |
| User settings/logout | User menu, profile display, logout action | Composable |

### Week 6: Vue.js Frontend — Lightbox & Detail

| Task | Details | Deliverable |
|---|---|---|
| Lightbox component | Full-screen overlay, prev/next, keyboard nav, pinch zoom | `web/src/components/Lightbox.vue` |
| EXIF panel | Slide-up panel with camera, lens, settings, GPS | `web/src/components/ExifPanel.vue` |
| Image zoom | Pinch-to-zoom and scroll-to-zoom | Composable |
| Swipe navigation | Touch swipe left/right for prev/next | Composable |
| Download button | Single file download + batch ZIP | `web/src/components/DownloadButton.vue` |
| Delete with confirmation | Soft delete with undo toast | `web/src/components/DeleteButton.vue` |

### Week 7: Timeline & Map Views

| Task | Details | Deliverable |
|---|---|---|
| Timeline endpoint | `GET /api/v1/timeline` with year/month/day granularity | `internal/handler/timeline.go` |
| Timeline component | Vertical timeline, month strips, date picker | `web/src/views/TimelineView.vue` |
| Geo points endpoint | `GET /api/v1/geo/points` with bbox filter | `internal/handler/geo.go` |
| Geo clusters endpoint | `GET /api/v1/geo/clusters` with H3 hexagons | `internal/handler/geo.go` |
| Map component | Leaflet, cluster layer (Supercluster), marker layer | `web/src/views/MapView.vue` |
| Map bottom sheet | Photo strip when clicking a cluster/marker | `web/src/components/MapBottomSheet.vue` |
| Heatmap layer | Toggle heatmap overlay | `web/src/components/HeatmapLayer.vue` |

### Week 8: Upload UI & Mobile Responsiveness

| Task | Details | Deliverable |
|---|---|---|
| Upload page | Drag & drop zone, file picker, folder picker | `web/src/views/UploadView.vue` |
| Upload progress | Per-file progress bars, WebSocket for live updates | `web/src/components/UploadQueue.vue` |
| WebSocket endpoint | `WS /api/v1/upload/ws` for real-time progress | `internal/handler/upload_ws.go` |
| Folder auto-refresh | Gallery/folder view listens for WS completion events and prepends new files to the current view | `web/src/views/GalleryView.vue`, `web/src/components/FolderTreeView.vue`, `web/src/stores/upload.ts` |
| Responsive layout | Mobile bottom nav, tablet sidebar, desktop full layout | `web/src/App.vue` + CSS |
| PWA manifest | `manifest.json`, service worker for offline caching | `web/public/` |
| Dark/light theme | CSS variables, theme toggle, persisted preference | `web/src/composables/useTheme.ts` |

**Phase 1 Deliverable:** Open `localhost:8080` → browse photos in a responsive gallery, click to see lightbox with EXIF, switch to timeline and map views, upload new photos with progress.

---

## 8.4 Phase 2 — Polish & Launch (Weeks 9-12)

**Goal:** Production-ready performance, error handling, and deployment.

### Week 9: Performance Optimization & Testing

| Task | Details | Deliverable |
|---|---|---|---|
| Virtual scrolling | For galleries with 10,000+ thumbnails | `vue-virtual-scroller` |
| Image lazy loading | Native `loading="lazy"` + Intersection Observer fallback | Gallery grid |
| HTTP caching | ETags, Cache-Control, 304 responses | Middleware |
| SQLite optimization | WAL mode, prepared statements, query plan analysis | `internal/db/` |
| Bundle optimization | Vite code splitting, tree shaking, compression | `vite.config.ts` |
| Backend unit tests | Go tests for handlers, services, stores (target: 70%+ coverage) | `internal/**/*_test.go` |
| Frontend unit tests | Vitest component tests for Pinia stores, composables, key components | `web/src/**/*.test.ts` |
| Lighthouse audit | Target: 95+ Performance, 100 Accessibility | — |

### Week 10: Error Handling & Resilience

| Task | Details | Deliverable |
|---|---|---|
| Graceful degradation | S3 down → serve from cache, show banner | `internal/cache/` |
| Upload retry | Exponential backoff for failed S3 uploads | Worker |
| Corrupt file handling | Detect during processing, quarantine, notify | `internal/service/validator.go` |
| Disk space monitoring | Alert when cache disk <10% free | `internal/monitor/disk.go` |
| Structured logging | `slog` (Go stdlib), JSON format, request IDs | Middleware |
| Health checks | `/health` with DB + S3 connectivity check | `internal/handler/health.go` |

### Week 11: Docker & Deployment

| Task | Details | Deliverable |
|---|---|---|---|
| Multi-stage Dockerfile | Build Go binary + Vue.js frontend, alpine-based final image with libvips, libheif, exiftool, ffmpeg | `Dockerfile` |
| Docker Compose | Drive + optional MinIO + Caddy for HTTPS | `docker-compose.yml` |
| Environment config | `.env` file, env vars, config validation on startup, auto-create defaults | `internal/config/` |
| Caddy reverse proxy | Auto-HTTPS, gzip, static file caching | `Caddyfile` |
| Healthcheck | Docker HEALTHCHECK instruction | `Dockerfile` |
| One-liner deploy | `docker compose up -d` | Install script |

### Week 12: Documentation & Landing Page

| Task | Details | Deliverable |
|---|---|---|
| README | Features, quick start, configuration, screenshots | `README.md` |
| API docs | OpenAPI/Swagger spec generated from code | `docs/api/openapi.yaml` |
| Landing page | Simple marketing page for the project | `web/src/views/Landing.vue` |
| Screenshots | Gallery, timeline, map, lightbox, upload | `docs/screenshots/` |
| Comparison table | Drive vs Immich vs Photoprism vs Google Photos | `README.md` |

**Phase 2 Deliverable:** `docker compose up -d` → fully working Drive instance with HTTPS, production-ready performance, and documentation.

---

## 8.5 Phase 3 — Differentiators (Weeks 13-20)

**Goal:** Features that make Drive stand out from alternatives.

### Weeks 13-14: AI Tagging

| Task | Details | Deliverable |
|---|---|---|
| ONNX runtime setup | Go bindings, model loading | `internal/ai/onnx.go` |
| Object detection model | MobileNet SSD or YOLO-tiny, quantized | Model file (~20MB) |
| Scene classification | EfficientNet-lite, top-5 labels | Model file (~15MB) |
| Inference worker | Background job, processes new uploads | `internal/worker/ai_worker.go` |
| Tag storage | `photo_tags` table with confidence scores | Migration |
| Semantic search | `GET /api/v1/search?q=beach&semantic=true` | `internal/handler/search.go` |
| Tag UI | Tag chips, tag cloud, "People" view | Vue components |

### Weeks 15-16: Mobile Auto-Backup

| Task | Details | Deliverable |
|---|---|---|
| iOS app scaffold | SwiftUI, Photos framework permission | `mobile/ios/` |
| Background upload | BGTaskScheduler, background URLSession | iOS app |
| Android app scaffold | Jetpack Compose, MediaStore permission | `mobile/android/` |
| Background upload | WorkManager, constrained by Wi-Fi + charging | Android app |
| Device registration API | `POST /api/v1/devices` | `internal/handler/device.go` |
| PWA fallback | Web Share Target, Background Sync | `web/src/` |

### Weeks 17-18: Duplicate Detection & Albums

| Task | Details | Deliverable |
|---|---|---|
| pHash generation | Go library, compute during upload | `internal/service/hash.go` |
| Duplicate finder | Hamming distance query, background scan | `internal/service/duplicate.go` |
| Duplicate UI | Side-by-side comparison, batch actions | `web/src/views/DuplicatesView.vue` |
| Album CRUD API | `POST/GET/PUT/DELETE /api/v1/albums` | `internal/handler/album.go` |
| Album items API | `POST/DELETE /api/v1/albums/{id}/items` | `internal/handler/album.go` |
| Album UI | Grid, drag-and-drop, smart album builder | `web/src/views/AlbumsView.vue` |

### Weeks 19-20: Video Streaming & RAW Support

| Task | Details | Deliverable |
|---|---|---|---|
| HLS transcoding | Generate .m3u8 + .ts segments (full adaptive streaming, Phase 2 enhancement) | `internal/imaging/ffmpeg.go` |
| Video player | Plyr component with HLS support | `web/src/components/VideoPlayer.vue` |
| RAW preview extraction | exiftool/dcraw for embedded JPEG | `internal/imaging/raw.go` |
| RAW badge | "RAW" indicator on thumbnails | `web/src/components/ThumbnailCard.vue` |

**Phase 3 Deliverable:** AI-powered search, automatic phone backup, duplicate cleanup, albums, HLS video streaming, and RAW support.

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
Week 3:  ████████████  Phase 0 Complete — Upload + thumbnails + auth work
Week 8:  ████████████████████████████████  Phase 1 Complete — MVP shipped
Week 12: ████████████████████████████████████████████████  Phase 2 Complete — Production ready
Week 20: ████████████████████████████████████████████████████████████████████████████████  Phase 3 Complete — Full platform
```

---

## 8.8 Risk Register

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|---|
| libvips CGo issues on ARM | Medium | High | Test on Raspberry Pi early; fallback to ImageMagick |
| libheif compilation in Docker | Medium | Medium | Use alpine with `libheif-dev` package; verify in CI |
| SQLite concurrency limits | Low | Medium | WAL mode handles reads well; single writer is fine for family workload |
| S3 costs at scale | Low | Medium | Local cache minimizes S3 reads; MinIO self-hosted is free; local-only mode avoids entirely |
| Auth security: JWT secret leakage | Low | High | Auto-generate random secret on first boot; document .env best practices |
| Mobile app store rejection | Medium | Low | PWA fallback works for most use cases |
| ONNX model accuracy | Medium | Low | Tags are additive, not critical path; user can always search by date |
| Browser video codec support | Low | Medium | Transcode to H.264 which is universally supported |