# 7. Additional Features Proposal

This section proposes features beyond the original readme scope. They are organized by priority and implementation phase.

---

## 7.1 Phase 1 — Differentiators (Post-MVP, High Impact)

### 7.1.1 AI-Powered Photo Tagging & Search

**Problem:** Users with 50,000+ photos can't find specific images by scrolling. They need semantic search.

**Solution:** Run a lightweight ONNX vision model locally (no cloud dependency) to generate:
- **Object detection labels**: "dog", "beach", "car", "mountain", "sunset", "food"
- **Scene classification**: "indoor", "outdoor", "night", "cityscape", "nature"
- **Text recognition (OCR)**: Extract text from signs, menus, documents in photos
- **Face detection + clustering**: Group photos by person (no identification, just "Person A", "Person B")

**Implementation:**
- Use `onnxruntime` Go bindings with quantized MobileNet/EfficientNet models (~20MB)
- Run inference as a background job after upload (non-blocking)
- Store tags in a new `photo_tags` table with confidence scores
- Search: `GET /api/v1/search?q=beach+sunset&semantic=true`
- All processing is local — no data leaves the server

**UI Impact:**
- Search bar becomes the primary navigation: "Show me beach photos from 2023"
- Tag chips below each photo in lightbox
- "People" view showing face clusters
- Auto-generated tag cloud on the stats page

---

### 7.1.2 Automatic Backup from Mobile Devices

**Problem:** Users need to manually upload photos. True backup should be automatic.

**Solution:** A companion mobile app (or PWA with Background Sync) that:
- Monitors the camera roll for new photos
- Auto-uploads in the background when on Wi-Fi
- Respects battery optimization (only uploads when charging + Wi-Fi)
- Selective backup: choose which albums to sync

**Implementation:**
- **iOS**: Swift app using BGTaskScheduler + Photos framework
- **Android**: Kotlin app using WorkManager + MediaStore
- **PWA fallback**: Web Share Target API + Background Sync (limited but zero-install)
- Endpoint: Same `POST /api/v1/upload` with an `X-Client: mobile` header
- Device registration: `POST /api/v1/devices` to track which device uploaded what

**UI Impact:**
- "Devices" settings page showing connected phones and last sync time
- "Backup status" widget on dashboard: "3 new photos pending backup"

---

### 7.1.3 Duplicate & Near-Duplicate Detection

**Problem:** Users accumulate duplicates from multiple backups, edits, and re-exports.

**Solution:** Beyond SHA-256 exact dedup, add perceptual hashing:
- **pHash (perceptual hash)**: Detect visually identical images even if resized/recompressed
- **dHash (difference hash)**: Faster, good for near-duplicates
- **SSIM comparison**: For high-confidence matches, compare structural similarity

**Implementation:**
- Generate pHash during upload processing (alongside thumbnails)
- Store in `file_hashes` table: `file_id`, `hash_type` ('phash', 'dhash'), `hash_value`
- Hamming distance query to find duplicates within threshold
- Background job periodically scans for new duplicates
- User-facing: "We found 23 potential duplicates. Review and clean up?"

**UI Impact:**
- "Duplicates" page with side-by-side comparison
- Batch select: "Keep newest", "Keep highest resolution", "Keep all from iPhone"
- Stats: "You could free 15.3 GB by removing duplicates"

---

### 7.1.4 Album & Collection Management

**Problem:** Auto-organization by date is great, but users also want thematic grouping.

**Solution:** Virtual albums (no file duplication):
- **Manual albums**: Drag & drop photos into named collections
- **Smart albums**: Rule-based auto-collections ("All photos with GPS in Italy", "Photos taken with Sony A7III")
- **Shared albums**: Generate shareable links (read-only or with upload capability)
- **Album cover**: Auto-selected or manually chosen

**Implementation:**
- New tables: `albums`, `album_items` (album_id, file_id, sort_order)
- Smart album rules stored as JSON query definition
- Album API: `CRUD /api/v1/albums`, `POST /api/v1/albums/{id}/items`

**UI Impact:**
- "Albums" tab in navigation
- Album grid view with cover thumbnails
- Drag-and-drop to add photos to albums
- Smart album rule builder UI

---

## 7.2 Phase 2 — Power User Features

### 7.2.1 RAW File Processing Pipeline

**Problem:** Photographers shoot RAW but want to browse and share JPEG previews.

**Solution:** Enhanced RAW support:
- Extract embedded JPEG preview from RAW files (most cameras include one)
- For RAW files without embedded preview, use libvips to develop with default settings
- Store the developed preview as the "original" for gallery purposes
- Keep the RAW file as downloadable original
- Show "RAW" badge on thumbnails

**Implementation:**
- Detect RAW MIME types during upload
- Extract embedded preview using `exiftool` or `dcraw` subprocess
- Fall back to `libvips` for basic demosaicing
- Store both: RAW original in S3, developed preview as `preview` thumbnail

---

### 7.2.2 Video Transcoding & Streaming

**Problem:** Original videos may be in formats not playable in browsers (HEVC, AV1, MKV).

**Solution:**
- Generate H.264/AAC MP4 proxy for all videos (720p, 4Mbps)
- Generate HLS (HTTP Live Streaming) segments for adaptive bitrate playback
- Extract video metadata: codec, bitrate, framerate, color profile
- Generate animated thumbnails (GIF/WebP) on hover

**Implementation:**
- `ffmpeg` pipeline: original → 720p proxy + HLS segments
- Store proxy and segments in S3 + local cache
- Serve HLS via `GET /api/v1/stream/{id}/master.m3u8`
- Video player component using Video.js or Plyr

---

### 7.2.3 Advanced Search & Filtering

**Problem:** Basic search is not enough for power users.

**Solution:**
- **EXIF range filters**: "Aperture between f/1.4 and f/2.8", "Focal length > 85mm"
- **Location search**: "Photos within 10km of Warsaw" (geocoding + radius)
- **Date range with precision**: "Photos taken in summer 2023"
- **Combined filters**: "Portrait photos taken with 85mm lens in Paris in 2023"
- **Saved searches**: Named, reusable filter presets

**Implementation:**
- Query builder in the backend that constructs SQL from filter JSON
- Geocoding via OpenStreetMap Nominatim (self-hosted or public API)
- Full-text search via SQLite FTS5 extension on filename + EXIF fields

---

### 7.2.4 Multi-User Support (Now in Phase 0/1 Core)

**Problem:** Families want separate libraries with optional sharing.

**Solution:**
- **User accounts**: Simple username/password
- **Per-user library**: Each user has their own file namespace
- **Roles**: Admin, Member

*Multi-user support was moved from Phase 2 additional features to Phase 0/1 core. See Section 2.2 FR-06 for current spec. This section is kept for reference on future enhancements (OIDC, sharing, quotas).*

---

## 7.3 Phase 3 — Ecosystem & Polish

### 7.3.1 Plugin System

**Problem:** Users have diverse needs that can't all be built into core.

**Solution:** A plugin system using Go's `plugin` package or a sidecar/WebAssembly model:
- **Storage plugins**: Support for Backblaze B2, Google Drive, Dropbox as backends
- **Processing plugins**: Custom image transformations, watermarking, AI models
- **Export plugins**: Export to Flickr, SmugMug, Instagram
- **Notification plugins**: Webhook, Email, Pushover, Discord

**Implementation:**
- Define a Go interface for each plugin type
- Plugins are separate binaries that communicate via gRPC or stdin/stdout
- Plugin registry in the database
- UI for installing/configuring plugins

---

### 7.3.2 Print & Physical Products

**Problem:** Digital photos stay digital. Users want physical keepsakes.

**Solution:** Integration with print APIs:
- **Photo books**: Select photos, arrange layouts, order print
- **Prints**: Individual photo prints in various sizes
- **Calendars**: Auto-generate from monthly highlights
- **Canvas/wall art**: Large format prints

**Implementation:**
- Integrate with print-on-demand APIs (Printful, Gelato, or local print shops)
- Layout editor in the browser (or generate automatically)
- Export to PDF for local printing

---

### 7.3.3 Integration with Existing Ecosystems

- **Immich migration tool**: One-click import from Immich
- **Google Takeout importer**: Parse Takeout ZIP, extract metadata from JSON sidecars, merge albums
- **iCloud/Photos export**: Import from macOS Photos library
- **Synology Photos migration**: Direct import from Synology NAS
- **Nextcloud integration**: Use Nextcloud as a storage backend

---

### 7.3.4 Advanced Map Features

- **Travel routes**: Connect GPS points chronologically to show travel paths
- **Country/region stats**: "You've taken photos in 23 countries"
- **Photo heatmap by time**: Animate photo density over time on the map
- **3D terrain**: Use MapLibre GL JS with terrain exaggeration for landscape photos
- **Street View integration**: See where a photo was taken in Google Street View (if API key provided)

---

## 7.4 Feature Priority Matrix

| Feature | Impact | Effort | Phase |
|---|---|---|---|
| AI Tagging & Semantic Search | 🔴 High | 🟡 Medium | 1 |
| Mobile Auto-Backup | 🔴 High | 🔴 High | 1 |
| Duplicate Detection (pHash) | 🟡 Medium | 🟢 Low | 1 |
| Album Management | 🟡 Medium | 🟡 Medium | 1 |
| RAW Processing Pipeline | 🟡 Medium | 🟡 Medium | 2 |
| Video Transcoding & HLS | 🟡 Medium | 🟡 Medium | 2 |
| Advanced Search & Filtering | 🟢 Low | 🟢 Low | 2 |
| Multi-User Support | 🔴 High | 🔴 High | 0 |
| Plugin System | 🟢 Low | 🔴 High | 3 |
| Print Integration | 🟢 Low | 🟡 Medium | 3 |
| Ecosystem Migration Tools | 🟡 Medium | 🟡 Medium | 3 |
| Advanced Map Features | 🟢 Low | 🟡 Medium | 3 |

---

## 7.5 Feature Summary

The core readme vision is solid: a fast, self-hosted photo backup with gallery, timeline, and map views. The additional features proposed here transform Drive from a "photo viewer" into a **complete photo management platform** that can genuinely replace Google Photos for self-hosters.

Multi-user support (user accounts, JWT auth, per-user libraries) is part of the **Phase 0/1 core** — not an additional feature.

The most impactful additional features are:
1. **AI tagging** — makes large libraries searchable without manual organization
2. **Mobile auto-backup** — removes the friction of manual uploads
3. **Duplicate detection** — saves storage and cleans up messy libraries
4. **Album management** — gives users organizational flexibility beyond date-based folders

These four features would make Drive competitive with Immich and Photoprism while maintaining its key differentiators: SQLite simplicity, S3-native storage, and Go/Vue.js performance.