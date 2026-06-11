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

**U-07 Implementation note:** The `drive import` CLI subcommand authenticates against the running server via username/password, walks the local directory tree, creates matching folders on the server, checks if each file already exists at the target location via `GET /api/v1/files?folder_id=...` (matching `originalName` + `sizeBytes`), and uploads new files using the chunked upload API. Supports `--target` (drive folder path), `--concurrency` (parallel uploads), `--dry-run` (preview without uploading), and retries on transient failures.

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
| A-12 | As a user, I am prevented from uploading files that would exceed my space quota, with a clear error message | P1 |
| A-13 | Deduplication only matches files within the same user account, not across all users | P1 |
| A-14 | As an admin, I can view per-user file and thumbnail breakdowns alongside the global aggregate | P2 |

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

### System Backup & Monitoring
| ID | User Story | Priority |
|---|---|---|
| B-01 | As an admin, the database is automatically backed up to S3 on a configurable schedule | P1 |
| B-02 | As an admin, old backups are automatically cleaned up based on retention policy | P1 |
| B-03 | As an admin, I can view system events (backups, upload errors, cache evictions) in the Admin Panel with filtering by type and severity | P1 |
| B-04 | As an admin, I can trigger an immediate backup from the Admin Panel | P2 |

### Sharing & Social Features (v3)
| ID | User Story | Priority |
|---|---|---|
| AL-01 | As a user, I can create an album with a name and optional description | P0 |
| AL-02 | As a user, I can add images to an album from my library | P0 |
| AL-03 | As a user, I can share an album with other users by username with view/comment/edit permission levels | P0 |
| AL-04 | As a shared album recipient, I can view all images in albums shared with me | P0 |
| AL-05 | As a shared album recipient with edit permission, I can add images to the album | P1 |
| AL-06 | As an album owner, I can remove users' access to the album | P0 |
| AL-07 | As a user, I can comment on images in shared albums (all album members can see comments; view-only members can read but not add) | P0 |
| AL-08 | As a user, I can add emoji reactions (👍❤️😂😮😢🙏) to comments via toggle | P0 |
| AL-09 | As a user, I can add comments to my own files in private space (visible only to me) | P0 |
| AL-10 | As a user, I can tag files with text labels; autocomplete suggests existing tags | P0 |

### Advanced Search (v3)
| ID | User Story | Priority |
|---|---|---|
| S-01 | As a user, I can search by filename, size range (slider), date added, date created, tags, and comment text | P0 |
| S-02 | As a user, search results show a flat list with folder path breadcrumb for files in folders | P0 |
| S-03 | As a user, I can preview and download files directly from search results | P0 |
| S-04 | As a user, tag input shows autocomplete from existing tags in the system | P0 |

### Sharing (Legacy — future v2)
| ID | User Story | Priority |
|---|---|---|
| S-01-legacy | As a user, I can generate a shareable link to an album or individual photo | P3 |
| S-02-legacy | As a user, shared links can be password-protected and time-limited | P3 |

### Folder Password Protection (v2.5)
| ID | User Story | Priority |
|---|---|---|
| FP-01 | As a user, I can set a password on a folder that protects all its contents (files, thumbnails, downloads) | P0 |
| FP-02 | As a user, I must provide the folder password to access its contents — even for myself as the folder owner | P0 |
| FP-03 | As a user, the folder password expires after 30 minutes of inactivity (configurable), requiring re-entry | P0 |
| FP-04 | As a user, I can remove the folder password at any time | P1 |
| FP-05 | As a user, thumbnails from password-protected folders are also protected — no unauthenticated access | P0 |
| FP-06 | As a user, downloads and file previews from password-protected folders require the unlock token | P0 |

### Folder Public Sharing (v2.5)
| ID | User Story | Priority |
|---|---|---|
| PS-01 | As a user, I can share a folder publicly without requiring account registration from recipients | P0 |
| PS-02 | As a user, I can set share permissions: read-only, read+upload, or read+upload+delete | P0 |
| PS-03 | As a user, I can set an expiration date when the share link stops working | P0 |
| PS-04 | As a user, I can set an upload size limit (e.g., 100MB total) for shared folders with upload permission | P0 |
| PS-05 | As a user, the upload limit is enforced server-side — uploads exceeding the quota are rejected immediately | P0 |
| PS-06 | As a user, I can password-protect the share link for an extra layer of security | P1 |
| PS-07 | As a user, I can list, update, and revoke share links for a folder | P0 |
| PS-08 | As a public recipient, I can browse, preview, and download files from a shared folder (with a session token) | P0 |
| PS-09 | As a public recipient, I can upload files to a shared folder if the share permits it, up to the quota limit | P0 |
| PS-10 | As a public recipient, I can delete files from a shared folder if the share permits write access | P0 |
| PS-11 | As a user, I can optionally include subdirectories when creating a share, allowing recipients to navigate the full folder tree | P0 |
| PS-12 | As a public recipient, I can create and delete subfolders within a shared folder when the share permits write access and subdirectories are included | P0 |

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
    - `video_proxy`: 720p H.264/AAC MP4 transcode (videos only, skipped if source ≤ 720p) — for browser streaming with byte range support; uploaded to S3 and streamed from there, local file deleted after S3 upload succeeds
5. **Video proxy** — Generate a 720p H.264/AAC MP4 proxy for browser playback during upload processing. Only generated if the source video exceeds 720p resolution. The proxy is uploaded to S3 (if enabled), the s3_key is persisted in the database, and the local file is removed — causing the streaming endpoint to serve from S3 via HTTP range requests. Video still thumbnails (video_still) remain on local disk for instant preview.
6. **Storage** — Originals go to local disk (and S3 if enabled, local removed after S3 upload). Image thumbnails and video stills stay on local cache (with optional S3 backup). Video proxy is S3-only after upload (local file deleted).

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
  ├── thumbnails/  (60px, 600px, 720p previews, video stills; video proxy removed after S3 upload)
  └── sqlite.db    (metadata, EXIF, file index)

Tier 2 — S3-Compatible Object Storage (durable, scalable)
  ├── originals/   (full-resolution uploads)
  └── thumbnails/  (backup copy of image thumbnails + primary store for video proxy)
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

### FR-08: Automated Database Backup
Upon startup, if `backup.enabled: true` and S3 is available:
1. The system creates a consistent SQLite snapshot via `VACUUM INTO`
2. The snapshot is uploaded to S3 under `backups/database/drive-backup-<timestamp>.db`
3. Each backup run is recorded as a system_events row (backup_success/backup_failure/pruned)
4. Old backups are pruned according to `backup.retention_days` (0 = keep all backups forever)
5. Backups run automatically on the configured `backup.interval_h` schedule
6. Admins can view backup status and trigger manual backups from the Admin Panel

### FR-09: System Event Logging
All key system operations are recorded in a single `system_events` table:
1. Backup successes, failures, and prunes
2. Upload errors (worker failures, S3 upload failures) 
3. Upload skips (dedup hits)
4. Reconciliation results
5. Cache eviction outcomes
6. S3 connectivity changes
7. Server lifecycle events (start, shutdown)

Admins can filter the event log by type and severity in the Admin Panel.
90-day automatic retention with daily purge.

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