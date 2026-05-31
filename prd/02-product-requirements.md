# 2. Product Requirements

## 2.1 Core User Stories

### Upload & Ingest
| ID | User Story | Priority |
|---|---|---|
| U-01 | As a user, I can upload photos and files via a drag-and-drop web interface | P0 |
| U-02 | As a user, I can select an entire folder for upload (browser File API / directory picker) | P0 |
| U-03 | As a user, photos are automatically organized into monthly folders (`YYYY/MM`) upon upload | P0 |
| U-04 | As a user, I can see upload progress with per-file status (queued, uploading, processing, done, error) | P1 |

**U-04 Implementation note:** Upload progress has two phases: (1) HTTP transfer progress (tracked via axios `onUploadProgress` — bytes sent / total), shown as `uploading` status with percentage; (2) server-side processing progress (WebSocket-driven — `hashing` → `dedup` → `exif` → `storing` → `thumbnails`), shown as `processing` status with stage name. Files appear in the queue immediately upon selection with `uploading` status. After the HTTP POST completes, they transition to `queued` and then `processing` as the worker pool handles them.
| U-05 | As a user, duplicate uploads are detected by content hash (SHA-256) and skipped — applies only to root uploads (no folder context). Folder-scoped uploads skip both name+size and content hash dedup. | P1 |
| U-05a | As a user, both name+size dedup and content hash (SHA-256) dedup are applied only when uploading to the root (no folder context — gallery view or Upload tab without a target folder). Uploads targeting a specific folder skip both dedup checks entirely, allowing the same file to exist in multiple folders. | P0 |
| U-06 | As a user, I can upload from mobile devices with the same responsive UI | P1 |

### Gallery & Browsing
| ID | User Story | Priority |
|---|---|---|
| G-01 | As a user, I can browse my photos in a responsive thumbnail grid | P0 |
| G-02 | As a user, thumbnails load instantly from local cache, with lazy loading for off-screen images | P0 |
| G-03 | As a user, I can navigate nested directory structures in the gallery | P0 |
| G-04 | As a user, I can view a photo in full resolution with zoom and pan | P1 |
| G-05 | As a user, I can navigate between photos with keyboard arrows and swipe gestures (mobile) | P1 |
| G-06 | As a user, I can sort gallery by date taken, date uploaded, or file name | P2 |
| G-07 | As a user, I can filter gallery by media type (photos, videos, all) | P2 |
| G-08 | As a user, I can switch between gallery layouts — tiles (grid), list (table), and grouped by day | P2 |
| G-09 | As a user, I can choose thumbnail sizes — small, medium, and large — in the gallery view | P2 |
| G-10 | As a user, the URL updates when I navigate folders, open photo previews, or change view state — enabling browser back/forward, refresh-without-context-loss, and shareable links | P0 |
| G-11 | As a user, I can toggle a filter to include files from all folders in the gallery view, enabling cross-folder sorting and browsing | P1 |

### Timeline View
| ID | User Story | Priority |
|---|---|---|
| T-01 | As a user, I can scroll through a timeline of my photos grouped by month/year | P0 |
| T-02 | As a user, the timeline shows a representative thumbnail for each time group | P1 |
| T-03 | As a user, I can jump to a specific month/year via a date picker or scrubber | P1 |

### Map View
| ID | User Story | Priority |
|---|---|---|
| M-01 | As a user, I can see my geo-tagged photos plotted on an interactive map (Leaflet) | P0 |
| M-02 | As a user, photo markers cluster automatically when zoomed out | P0 |
| M-03 | As a user, clicking a cluster zooms in; clicking a single marker shows the photo thumbnail | P1 |
| M-04 | As a user, I can see a heatmap overlay of photo density | P2 |
| M-05 | As a user, photos without GPS data are not shown on the map (no fake locations) | P0 |

### EXIF & Metadata
| ID | User Story | Priority |
|---|---|---|
| E-01 | As a user, I can view EXIF data for any photo (camera, lens, aperture, shutter, ISO, GPS, date) | P0 |
| E-02 | As a user, GPS coordinates are extracted and stored for map rendering | P0 |
| E-03 | As a user, I can search/filter photos by EXIF metadata (camera model, lens, date range) | P2 |
| E-04 | As a user, EXIF data is preserved in original files and never modified | P0 |

### File Management
| ID | User Story | Priority |
|---|---|---|
| FM-01 | As a user, I can delete individual files with a confirmation step (soft-delete with trash) | P0 |
| FM-02 | As a user, I can select multiple files using checkboxes with Shift+click range selection | P0 |
| FM-03 | As a user, I can batch delete multiple selected files at once | P0 |
| FM-04 | As a user, I can move files between folders (including back to root) | P1 |
| FM-05 | As a user, I can copy files to other folders | P1 |
| FM-06 | As a user, I can press the Delete key to delete selected files | P1 |
| FM-07 | As a user, I can view PDF files directly in the browser without downloading | P1 |
| FM-08 | As a user, I can view text files with proper formatting in a monospace viewer | P1 |
| FM-09 | As a user, I can view JSON files with syntax highlighting and pretty-print formatting | P1 |
| FM-10 | As a user, I can view Markdown files rendered as HTML | P1 |
| FM-11 | As a user, I can view CSV files as a sortable data table | P1 |
| FM-12 | As a user, I can always download the original raw file from any viewer | P0 |

### Folder Organization
| ID | User Story | Priority |
|---|---|---|
| FO-01 | As a user, I can create custom folders with a name | P0 |
| FO-02 | As a user, I can create nested folders (hierarchical, arbitrary depth) | P1 |
| FO-03 | As a user, I can rename folders | P1 |
| FO-04 | As a user, I can delete folders (files inside revert to root) | P1 |
| FO-05 | As a user, I can switch to the Folders tab to browse files by folder with nested navigation | P0 |
| FO-06 | As a user, I can navigate into a folder to see its files and subfolders | P0 |
| FO-07 | As a user, I can upload files directly into a chosen folder | P1 |
| FO-08 | As a user, I can upload files directly into the folder I'm currently browsing, without leaving the gallery/folder view | P1 |
| FO-09 | As a user, I can see a floating upload tracker showing progress of all current uploads, regardless of which page I'm on | P1 |
| FO-10 | As a user, uploads run asynchronously — I can start uploads in Folder A, then navigate to Folder B and start more uploads while Folder A's uploads continue | P1 |
| FO-11 | As a user, when uploads complete in the folder I'm currently browsing, the file listing updates automatically with the new files (no manual refresh) | P1 |

### File Backup
| ID | User Story | Priority |
|---|---|---|
| F-01 | As a user, I can upload any file type (documents, archives, etc.) — not just media | P1 |
| F-02 | As a user, non-media files show appropriate file type icons in the gallery | P1 |
| F-03 | As a user, I can download original files individually or as a zip bundle | P1 |

### Authentication & User Management
| ID | User Story | Priority |
|---|---|---|
| A-01 | As a user, I can register an account with a username and password | P0 |
| A-02 | As a user, I can log in and log out of my account | P0 |
| A-03 | As a user, my files are private and only accessible to me | P0 |
| A-04 | As an admin, I can create the first admin user via the CLI (`drive admin create`) | P0 |
| A-05 | As an admin, I can manage users (list, delete, change roles, create) from the admin panel | P0 |
| A-06 | As an admin, I can enable or disable public registration via a toggle in the admin panel (runtime setting persisted in SQLite) | P1 |
| A-07 | As an admin, I can create new users from the admin panel (no CLI required after initial admin setup) | P0 |
| A-08 | As an admin, I can set a space quota per user (original file sizes only, not thumbnails) | P1 |
| A-09 | As an admin, I can view per-user storage breakdowns (files + thumbnails) and total usage | P1 |
| A-10 | As a user, I see a quota progress bar in the top bar showing my storage usage vs limit | P1 |
| A-11 | Space quota decreases below the user's current usage are rejected with a validation error | P1 |

### Upload Reliability & Self-Healing
| ID | User Story | Priority |
|---|---|---|
| R-01 | As an admin, I can view a paginated history of all upload jobs with status, errors, and timestamps in the Admin Panel | P1 |
| R-02 | As an admin, I can retry failed upload jobs from the job history view | P1 |
| R-03 | As an admin, I can trigger thumbnail reconciliation to scan for and regenerate missing thumbnails across all photos | P1 |
| R-04 | The system periodically self-heals by running thumbnail reconciliation every 30 minutes | P1 |
| R-05 | When a thumbnail is missing both locally and on S3, the handler falls back to serving the next smaller available size instead of returning 404 | P1 |
| R-06 | Thumbnail files are explicitly synced to disk after generation to prevent data loss on Docker overlayfs | P0 |
| R-07 | The worker verifies thumbnail files still exist on disk before attempting S3 upload, logging a warning and skipping if missing | P0 |

### Sharing (Future / V2)
| ID | User Story | Priority |
|---|---|---|
| S-01 | As a user, I can generate a shareable link to an album or individual photo | P3 |
| S-02 | As a user, shared links can be password-protected and time-limited | P3 |

---

## 2.2 Functional Requirements

### FR-01: Media Processing Pipeline
Upon upload, every image and video goes through:
1. **Name+Size dedup check (root uploads only)** — When uploading without a target folder (root/gallery context), if a file with the same `original_name` AND `size_bytes` already exists in the root (`folder_id IS NULL`), skip the upload entirely (silently ignored, no DB update). This is a fast pre-check before any processing. Folder-scoped uploads skip this check entirely.
2. **Hash computation** — SHA-256 of file content for content-level deduplication. Content-hash dedup is only applied for root uploads (no folder context). Folder-scoped uploads skip this check.
3. **EXIF extraction** — Parse all EXIF/XMP tags using `goexif` (JPEG/PNG/TIFF) with `exiftool` subprocess fallback (HEIC/AVIF). Non-media files skip EXIF entirely.
4. **Thumbnail generation** — Four sizes per image:
   - `thumb_sm`: 60px wide (JPEG, quality 60%) — for grid thumbnails
   - `thumb_md`: 600px wide (JPEG, quality 75%) — for preview/lightbox
   - `preview`: 720p max dimension (WebP, quality 80%) — for full preview
   - `video_still`: frame at 5s (JPEG, quality 75%) — videos only
5. **Video proxy** — Generate a 720p H.264/AAC MP4 proxy for browser playback. Stored as original + proxy in storage.
6. **Storage** — Originals go to local disk (and S3 if enabled). Thumbnails go to local cache (and S3 if enabled).

### FR-02: Storage Tiers

Two modes of operation determined by `s3.enabled` configuration:

**Local-Only Mode (default, no S3 config needed):**
```
Storage Root (local disk, e.g., /data)
├── originals/    (full-resolution uploads)
├── thumbnails/   (60px, 600px, 720p previews, video stills)
└── sqlite.db     (metadata, EXIF, file index)
```

**S3 Mode (s3.enabled: true):**
```
Tier 1 — Local Cache (SSD/NVMe, fast)
  ├── thumbnails/  (60px, 600px, 720p previews, video stills)
  └── sqlite.db    (metadata, EXIF, file index)

Tier 2 — S3-Compatible Object Storage (durable, scalable)
  ├── originals/   (full-resolution uploads)
  └── thumbnails/  (backup copy of all thumbnails)
```

Cache eviction policy: LRU (least recently used), configurable max cache size (default: 50GB). Eviction runs as a scheduled background goroutine every 5 minutes. Thumbnails are regenerated on cache miss from stored originals (or S3 if enabled).

### FR-03: File Actions & Batch Operations
- Files can be selected individually or in ranges (Shift+click)
- Batch actions: delete (soft-delete with trash), move (to folder or root), copy (to folder or root)
- Delete key shortcut triggers batch delete when files are selected
- Action bar appears when files are selected, showing selected count and action buttons
- All batch operations are user-scoped (row-level security enforced per query)

### FR-03b: Folder Organization
- Folders are user-created, user-scoped, and support arbitrary nesting via `parent_id` self-reference
- Folder tree displayed in sidebar and as a dedicated folder browser layout
- Inline folder creation within the folder tree and folder picker dialogs
- Moving a file to a folder sets `folder_id`; moving to root sets `folder_id = NULL`
- Copying a file creates a new file record (new UUID, same storage path) with the target `folder_id`
- Deleting a folder cascades: files inside revert to root via `ON DELETE SET NULL` FK
- Upload destination can target a specific folder via `folder_id` multipart field; default is root (auto-organized by date)

### FR-04: Auto-Organization
- Photos are organized by **date taken** (from EXIF `DateTimeOriginal`), falling back to file modification date
- Directory structure: `{root}/{YYYY}/{MM}/{filename}`
- Videos follow the same structure using `CreateDate` or file date
- Non-media files are organized by upload date: `{root}/files/{YYYY}/{MM}/{filename}`

### FR-04: Map Rendering
- Uses **Leaflet** with OpenStreetMap tiles (no API key required)
- Optional: MapLibre GL JS with self-hosted vector tiles for offline use
- Photo markers clustered with **Supercluster** (client-side) or **H3** (server-side pre-computed)
- GPS coordinates stored as `latitude`, `longitude` (float64) in SQLite
- Spatial index on (lat, lon) for fast bounding-box queries

### FR-05: Responsive Design
- Mobile-first CSS with CSS Grid for thumbnail layout
- Touch gestures: swipe left/right for photo navigation, pinch-to-zoom
- Progressive Web App (PWA) capable: service worker for offline thumbnail cache
- Adaptive thumbnail resolution based on device pixel ratio and viewport

### FR-07: Deep-Linkable UI
All primary UI states are reflected in the URL as query parameters on the gallery route `/`:
- Navigating into a folder → `?folder_id=<uuid>` or `?path=<YYYY/MM>`
- Opening photo preview (lightbox) → `?photo=<fileId>`
- Gallery layout, sort order, media type filter, thumbnail size → `?layout=`, `?sort=`, `?media=`, `?thumb=`
- Browser back/forward navigates between these states correctly
- Copying the URL and opening it in another tab restores the same view (auth redirect preserves query params via vue-router)

### FR-06: Authentication & User Management
- User registration with username + password (bcrypt hashed)
- JWT-based sessions with access + refresh tokens
- Roles: `admin` and `member`
- Admin user created via CLI: `drive admin create`
- Additional users created from admin panel by admins (`POST /api/v1/admin/users`)
- Self-registration disabled by default (`auth.allow_registration: false`)
- Self-registration can be enabled/disabled at runtime via admin panel toggle (persisted in SQLite `settings` table, overriding config file)
- All API endpoints require `Authorization: Bearer <token>` header (exceptions: health, login, register, auth config)
- Row-level filtering: all file/resource queries scoped to `user_id`
- Admins can list, delete, change roles, and create all users

---

## 2.3 Non-Functional Requirements

| ID | Requirement | Target |
|---|---|---|
| NFR-01 | Gallery page load (100 thumbnails) | < 500ms |
| NFR-02 | Single photo upload + processing | < 2s |
| NFR-03 | Map render with 10k markers | < 1s |
| NFR-04 | Idle memory usage | < 200MB |
| NFR-05 | Concurrent upload support | 10 simultaneous, 4 processing workers |
| NFR-06 | Supported image formats | JPEG, PNG, HEIC, WebP, AVIF, TIFF, RAW (CR2, NEF, ARW, DNG) |
| NFR-07 | Supported video formats | MP4, MOV, AVI, MKV, HEVC |
| NFR-08 | Max individual file size | 10GB (configurable) |
| NFR-09 | Storage backends | Local disk, AWS S3, Cloudflare R2, Backblaze B2, MinIO, Wasabi |
| NFR-10 | Browser support | Chrome 100+, Firefox 100+, Safari 16+, Mobile Safari, Chrome Android |

---

## 2.4 User Flows

### Primary Flow: Upload & View Photos
```
1. User opens Drive web UI → redirected to login page
2. User logs in with username + password (or registers if registration enabled)
3. Lands on Gallery view (most recent first)
4. Clicks the "Upload" button above the thumbnail grid (file picker restricted to images/videos)
5. Upload progress shown in the global upload tracker
6. Photos appear in gallery as thumbnails are generated — the file listing auto-refreshes in the current folder view when uploads complete
7. User switches to Timeline view → scrolls through months
8. User switches to Map view → sees photo clusters on world map
9. User clicks a photo → lightbox with full preview, EXIF panel, map pin
```

### Secondary Flow: Browse Existing Library
```
1. User opens Drive web UI → logs in
2. Lands on Gallery view (most recent first)
3. Scrolls down → lazy-loaded thumbnails appear instantly
4. Uses directory tree sidebar to navigate to a specific folder
5. Clicks Timeline tab → jumps to a specific month
6. Clicks Map tab → explores photos by location
```

### Secondary Flow: Browse Existing Library
```
1. User opens Drive → lands on Gallery view (most recent first)
2. Scrolls down → lazy-loaded thumbnails appear instantly
3. Uses directory tree sidebar to navigate to a specific folder
4. Clicks Timeline tab → jumps to a specific month
5. Clicks Map tab → explores photos by location