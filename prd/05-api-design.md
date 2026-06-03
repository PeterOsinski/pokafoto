# 5. API Design

## 5.1 REST API Endpoints

### Base URL: `/api/v1`

All endpoints return JSON. Timestamps are ISO 8601. IDs are UUID v7 strings.

**Authentication:** All endpoints require `Authorization: Bearer <token>` header, except for:
- `GET /api/v1/auth/config`
- `POST /api/v1/auth/register` (if registration enabled)
- `POST /api/v1/auth/login`
- `GET /api/v1/health`

---

### 5.1.0 Authentication

#### `POST /api/v1/auth/register`
Create a new user account. Requires `auth.allow_registration: true` in config.

**Request:**
```json
{
  "username": "johndoe",
  "password": "securepassword123",
  "display_name": "John Doe"
}
```

**Response:** `201 Created`
```json
{
  "user": {
    "id": "uuid-v7",
    "username": "johndoe",
    "display_name": "John Doe",
    "role": "member",
    "created_at": "2024-07-15T14:30:00Z"
  }
}
```

#### `POST /api/v1/auth/login`
Authenticate and receive JWT tokens.

**Request:**
```json
{
  "username": "johndoe",
  "password": "securepassword123"
}
```

**Response:** `200 OK`
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIs...",
  "refresh_token": "uuid-v7",
  "expires_in": 259200,
  "user": {
    "id": "uuid-v7",
    "username": "johndoe",
    "display_name": "John Doe",
    "role": "member"
  }
}
```

**Error:** `401 Unauthorized` for invalid credentials.

#### `POST /api/v1/auth/refresh`
Get a new access token using a refresh token.

**Request:**
```json
{
  "refresh_token": "uuid-v7"
}
```

**Response:** `200 OK` — same shape as login (new access_token + refresh_token, rotated).

#### `POST /api/v1/auth/logout`
Invalidate the current refresh token.

**Request:**
```json
{
  "refresh_token": "uuid-v7"
}
```

**Response:** `204 No Content`

#### `GET /api/v1/auth/me`
Get the current user's profile.

**Response:** `200 OK`
```json
{
  "id": "uuid-v7",
  "username": "johndoe",
  "display_name": "John Doe",
  "role": "member",
  "created_at": "2024-07-15T14:30:00Z"
}
```

#### `GET /api/v1/auth/config`
Get public auth configuration. Does not require authentication.

**Response:** `200 OK`
```json
{
  "allow_registration": false
}
```

---

### 5.1.1 Upload

#### `POST /api/v1/upload`
Upload one or more files. Multipart form data. Requires authentication.

**Request:**
```
Content-Type: multipart/form-data

files: [binary] (multiple, required)
folder_id: string (optional) — UUID of target folder. If omitted or empty, files go to root (auto-organized by date)
path: string (optional) — target directory path, defaults to auto-organization
relative_path: string (optional per file) — webkitRelativePath from folder picker for recursive directory preservation
skip_name_size_dedup: string (optional) — "true" to skip name+size dedup pre-check (used for folder-scoped uploads). Defaults to false (dedup enabled for root uploads).
```

When a folder is uploaded (via `webkitdirectory`), the browser provides `webkitRelativePath` for each file. If `relative_path` is present, Drive preserves the directory hierarchy relative to the chosen root folder. Example:
- User picks folder `Vacation 2024/` containing `Paris/IMG_1.jpg` and `London/IMG_2.jpg`
- Files arrive with `file=IMG_1.jpg` + `relative_path=Paris/IMG_1.jpg` and `file=IMG_2.jpg` + `relative_path=London/IMG_2.jpg`
- Stored paths: `Vacation 2024/Paris/IMG_1.jpg` and `Vacation 2024/London/IMG_2.jpg`

**Response:** `202 Accepted`
```json
{
  "batch_id": "uuid-v7",
  "jobs": [
    {
      "job_id": "uuid-v7",
      "filename": "IMG_1234.jpg",
      "status": "queued"
    },
    {
      "job_id": "uuid-v7",
      "filename": "IMG_5678.jpg",
      "status": "skipped",
      "reason": "duplicate_name_size",
      "existing_file_id": "uuid-v7"
    }
  ]
}
```

**Deduplication behavior:**
1. **Name+Size check (root uploads only):** When `skip_name_size_dedup` is not set to `"true"` and `folder_id` is null (root uploads), before any processing, check if a file with the same `original_name` AND `size_bytes` already exists **within the same user's files** in the root. If yes → status `skipped` with reason `duplicate_name_size`. The file is silently ignored — no upload, no processing. Folder-scoped uploads set `skip_name_size_dedup=true` and skip this check.
2. **Content hash check (root uploads only):** SHA-256 hash computed during upload. Applied only when `skip_name_size_dedup` is not `"true"` (root uploads). If an identical hash already exists **within the same user's files** → status `skipped` with reason `duplicate_content`. Folder-scoped uploads skip this check.

**Quota enforcement:** Uploads that would exceed the user's `space_quota` (total bytes of non-deleted original files) are rejected with `413 Payload Too Large`:

```json
{
  "error": {
    "code": "QUOTA_EXCEEDED",
    "message": "Upload would exceed space quota (500000 used + 200000 incoming > 600000 limit)"
  }
}
```

#### `GET /api/v1/upload/{batch_id}/status`
Check upload batch progress.

**Response:** `200 OK`
```json
{
  "batch_id": "uuid-v7",
  "total": 10,
  "completed": 7,
  "failed": 1,
  "jobs": [
    {
      "job_id": "uuid-v7",
      "filename": "IMG_1234.jpg",
      "status": "completed",
      "file_id": "uuid-v7"
    },
    {
      "job_id": "uuid-v7",
      "filename": "IMG_5678.jpg",
      "status": "skipped",
      "reason": "duplicate_name_size",
      "existing_file_id": "uuid-v7"
    },
    {
      "job_id": "uuid-v7",
      "filename": "IMG_9999.jpg",
      "status": "failed",
      "error": "unsupported_format"
    }
  ]
}
```

#### `WS /api/v1/upload/ws`
WebSocket endpoint for real-time upload progress. Requires authentication (send JWT as query param: `ws://host/api/v1/upload/ws?token=<access_token>`). Sends JSON messages per file:
```json
{
  "job_id": "uuid-v7",
  "filename": "IMG_1234.jpg",
  "status": "processing",
  "progress": 0.75,
  "stage": "generating_thumbnails",
  "file_id": "uuid-v7",
  "folder_id": "optional-folder-uuid"
}
```
`file_id` is present when the file has been stored in the database. `folder_id` is present when the upload targeted a specific folder; it is `null` for root-scoped uploads.

---

### 5.1.2 Files & Gallery

#### `GET /api/v1/files`
List files with pagination, sorting, and filtering.

**Query Parameters:**
| Param | Type | Default | Description |
|---|---|---|---|
| `folder_id` | string | — | Filter by folder UUID. `folders` layout passes `folder_id=uuid`. Root view omits this param (filters `folder_id IS NULL`). Value `root` also means root. |
| `all_folders` | string | `""` | When `"true"`, includes files from all user folders (no `folder_id` filter). When empty/false, only root-level files (`folder_id IS NULL`). |
| `path` | string | `""` | Directory path to list (empty = root) |
| `cursor` | string | — | Pagination cursor (file ID) |
| `limit` | int | 100 | Items per page (max 500) |
| `sort` | string | `taken_at` | `taken_at`, `created_at`, `filename`, `size` |
| `order` | string | `desc` | `asc` or `desc` |
| `media_type` | string | — | Filter: `photo`, `video`, `file` |
| `date_from` | string | — | ISO 8601 start date |
| `date_to` | string | — | ISO 8601 end date |
| `camera` | string | — | Filter by camera model (substring match) |

**Response:** `200 OK`
```json
{
  "items": [
    {
      "id": "uuid-v7",
      "filename": "2024/07/IMG_1234.jpg",
      "originalName": "IMG_1234.jpg",
      "path": "2024/07",
      "sizeBytes": 4521984,
      "mimeType": "image/jpeg",
      "mediaType": "photo",
      "width": 4032,
      "height": 3024,
      "takenAt": "2024-07-15T14:30:00Z",
      "createdAt": "2024-07-15T14:35:00Z",
      "thumbnails": {
        "sm": { "url": "/api/v1/thumb/abc-123/sm.jpg", "width": 60, "height": 45 },
        "md": { "url": "/api/v1/thumb/abc-123/md.jpg", "width": 600, "height": 450 },
        "preview": { "url": "/api/v1/thumb/abc-123/preview.webp", "width": 960, "height": 720 }
      }
    }
  ],
  "nextCursor": "uuid-v7-of-last-item",
  "total": 15420
}
```

#### `GET /api/v1/files/{id}`
Get single file details with full EXIF data.

**Response:** `200 OK`
```json
{
  "id": "uuid-v7",
  "filename": "2024/07/IMG_1234.jpg",
  "originalName": "IMG_1234.jpg",
  "path": "2024/07",
  "sizeBytes": 4521984,
  "mimeType": "image/jpeg",
  "mediaType": "photo",
  "width": 4032,
  "height": 3024,
  "sha256": "abc123...",
  "takenAt": "2024-07-15T14:30:00Z",
  "createdAt": "2024-07-15T14:35:00Z",
  "updatedAt": "2024-07-15T14:35:00Z",
  "exif": {
    "cameraMake": "Apple",
    "cameraModel": "iPhone 15 Pro",
    "lensMake": "Apple",
    "lensModel": "iPhone 15 Pro back triple camera 6.765mm f/1.78",
    "focalLength": 6.765,
    "aperture": 1.78,
    "shutterSpeed": "1/2500",
    "iso": 80,
    "dateTaken": "2024-07-15T14:30:00Z",
    "gpsLatitude": 52.2297,
    "gpsLongitude": 21.0122,
    "gpsAltitude": 120.5,
    "orientation": 1,
    "colorSpace": "sRGB",
    "flash": 0,
    "software": "Adobe Lightroom 7.2"
  },
  "thumbnails": {
    "sm": { "url": "/api/v1/thumb/abc-123/sm.jpg", "width": 60, "height": 45 },
    "md": { "url": "/api/v1/thumb/abc-123/md.jpg", "width": 600, "height": 450 },
    "preview": { "url": "/api/v1/thumb/abc-123/preview.webp", "width": 960, "height": 720 }
  }
}
```

#### `DELETE /api/v1/files/{id}`
Soft-delete a file (moves to trash).

**Response:** `204 No Content`

#### `PUT /api/v1/files/{id}/rename`
Rename a file or document. For app-managed documents, also updates the filename path.

**Request:**
```json
{
  "name": "new_name.jpg"
}
```

**Response:** `204 No Content`
**Error:** `400 Bad Request` if name is empty. `404 Not Found` if file not owned by user.

#### `DELETE /api/v1/files/{id}/permanent`
Permanently delete a file and all thumbnails from S3 and local cache.

**Response:** `204 No Content`

#### `POST /api/v1/files/batch-delete`
Soft-delete multiple files in a single operation. User-scoped.

**Request:**
```json
{ "ids": ["uuid-1", "uuid-2", "uuid-3"] }
```
**Response:** `204 No Content`

#### `POST /api/v1/files/batch-move`
Move multiple files to a target folder. Set `folder_id` to `null` to move back to root.

**Request:**
```json
{ "ids": ["uuid-1", "uuid-2"], "folder_id": "folder-uuid" }
```
**Response:** `204 No Content`

#### `POST /api/v1/files/batch-copy`
Copy files to a folder. Creates new file records (fresh UUIDs, same storage paths).

**Request:**
```json
{ "ids": ["uuid-1"], "folder_id": "folder-uuid" }
```
**Response:** `200 OK` — `{ "count": 1, "ids": ["new-uuid"] }`

---

### 5.1.3 Thumbnails

#### `GET /api/v1/thumb/{file_id}/{size}.{format}`
Serve a thumbnail image.

**Path Parameters:**
- `file_id`: UUID of the file
- `size`: `sm` (60px), `md` (600px), `preview` (720p), or `video_still` (frame at 5s)
- `format`: `jpg` for sm/md/video_still, `webp` for preview

**Behavior:**
1. Check local cache → serve directly (fast path)
2. Cache miss → regenerate from original or fetch from S3 (if enabled) → serve
3. If neither has it → `404 Not Found`

**Response:** `200 OK` with appropriate `Content-Type` and `Cache-Control: public, max-age=31536000, immutable`

---

### 5.1.3a Video Streaming

#### `GET /api/v1/video/{id}`
Stream a video file with byte range support for rewinding/scrubbing. Requires authentication.

**Query Parameters:**
| Param | Type | Default | Description |
|---|---|---|---|
| `quality` | string | `original` | `original` (full resolution) or `proxy` (720p transcode if available) |

**Behavior:**
1. `quality=proxy` — serves the 720p H.264/AAC MP4 proxy if available. Falls back to original if no proxy exists.
2. `quality=original` (or absent) — serves the full-resolution original file.
3. Supports HTTP Range requests for both proxy and original playback.
4. For local files: uses `http.ServeContent` for native Range support.
5. For S3 files: passes Range header through to S3 GetObject, returning `206 Partial Content` with `Content-Range`.

**Response Headers:**
- `Content-Type: video/mp4`
- `Accept-Ranges: bytes`
- `Content-Length: <file_size_bytes>` (for 200), or
- `Content-Range: bytes <start>-<end>/<total>` (for 206)

**Response:** `200 OK` for full file, `206 Partial Content` for range requests.

---

### 5.1.4 Map & Geo

#### `GET /api/v1/geo/points`
Get all geo-tagged photo points within a bounding box.

**Query Parameters:**
| Param | Type | Required | Description |
|---|---|---|---|
| `lat_min` | float | yes | South latitude |
| `lat_max` | float | yes | North latitude |
| `lon_min` | float | yes | West longitude |
| `lon_max` | float | yes | East longitude |
| `date_from` | string | no | Filter by date |
| `date_to` | string | no | Filter by date |

**Response:** `200 OK`
```json
{
  "points": [
    {
      "fileId": "uuid-v7",
      "latitude": 52.2297,
      "longitude": 21.0122,
      "thumbnailUrl": "/api/v1/thumb/abc-123/sm.jpg",
      "takenAt": "2024-07-15T14:30:00Z"
    }
  ],
  "total": 42
}
```

#### `GET /api/v1/geo/clusters`
Get pre-computed clusters for a given zoom level and bounding box. Uses H3 hexagons at the server side for performance.

**Query Parameters:**
| Param | Type | Required | Description |
|---|---|---|---|
| `zoom` | int | yes | Map zoom level (0-20) |
| `lat_min` | float | yes | South latitude |
| `lat_max` | float | yes | North latitude |
| `lon_min` | float | yes | West longitude |
| `lon_max` | float | yes | East longitude |

**Response:** `200 OK`
```json
{
  "clusters": [
    {
      "latitude": 52.23,
      "longitude": 21.01,
      "count": 156,
      "thumbnailUrl": "/api/v1/thumb/rep-id/sm.jpg"
    }
  ]
}
```

#### `GET /api/v1/geo/heatmap`
Get heatmap data (point density) for a bounding box.

**Response:** `200 OK`
```json
{
  "points": [
    { "latitude": 52.2297, "longitude": 21.0122, "weight": 1.0 }
  ]
}
```

---

### 5.1.5 Timeline

#### `GET /api/v1/timeline`
Get photos grouped by time periods.

**Query Parameters:**
| Param | Type | Default | Description |
|---|---|---|---|
| `granularity` | string | `month` | `year`, `month`, `day` |
| `date_from` | string | — | ISO 8601 |
| `date_to` | string | — | ISO 8601 |

**Response:** `200 OK`
```json
{
  "groups": [
    {
      "period": "2024-07",
      "label": "July 2024",
      "count": 234,
      "thumbnailUrl": "/api/v1/thumb/rep-id/sm.jpg",
      "startDate": "2024-07-01T00:00:00Z",
      "endDate": "2024-07-31T23:59:59Z"
    },
    {
      "period": "2024-06",
      "label": "June 2024",
      "count": 189,
      "thumbnailUrl": "/api/v1/thumb/rep-id-2/sm.jpg",
      "startDate": "2024-06-01T00:00:00Z",
      "endDate": "2024-06-30T23:59:59Z"
    }
  ]
}
```

---

### 5.1.6 Directory Tree

#### `GET /api/v1/dirs`
Get directory tree structure.

**Query Parameters:**
| Param | Type | Default | Description |
|---|---|---|---|
| `all_folders` | string | `""` | When `"true"`, includes directories from files in all user folders. When empty/false, only includes files at root (`folder_id IS NULL`). |

**Response:** `200 OK`
```json
{
  "path": "",
  "name": "root",
  "fileCount": 15420,
  "children": [
    {
      "path": "2024",
      "name": "2024",
      "fileCount": 8500,
      "children": [
        {
          "path": "2024/07",
          "name": "07",
          "fileCount": 234,
          "children": []
        }
      ]
    }
  ]
}
```

---

### 5.1.6a Folders

#### `GET /api/v1/folders`
Get the user's folder tree with file counts and share indicators.

**Response:** `200 OK`
```json
{
  "children": [
    {
      "folder": {
        "id": "uuid-v7",
        "name": "Vacation 2024",
        "parent_id": null,
        "user_id": "uuid-v7",
        "created_at": "2024-07-15T14:30:00Z",
        "updated_at": "2024-07-15T14:30:00Z"
      },
      "fileCount": 156,
      "hasShares": true,
      "children": [
        {
          "folder": {
            "id": "uuid-v7",
            "name": "Paris",
            "parent_id": "uuid-v7",
            "user_id": "uuid-v7",
            "created_at": "2024-07-16T10:00:00Z",
            "updated_at": "2024-07-16T10:00:00Z"
          },
          "fileCount": 42,
          "hasShares": false,
          "children": []
        }
      ]
    }
  ]
}
```

#### `POST /api/v1/folders`
Create a new folder. Can be nested under a parent.

**Request:**
```json
{ "name": "Vacation 2024", "parent_id": null }
```
**Response:** `201 Created` — returns the created `Folder` object.

#### `PUT /api/v1/folders/{id}`
Update a folder — rename, move (change parent), or both. At least one field is required.

**Request:**
```json
{
  "name": "New Name",
  "parent_id": "uuid-or-null"
}
```

- `name` — rename the folder. Must be non-empty.
- `parent_id` — move to a different parent folder. Set to `""` to move to root. Cannot move into itself or its own descendant.

**Response:** `204 No Content`
**Error:** `400 Bad Request` if neither field provided or circular move attempted. `404 Not Found` if folder or target parent not found/owned.

#### `DELETE /api/v1/folders/{id}`
Delete a folder recursively. All files within the folder and its subfolders are soft-deleted (moved to trash). All subfolders are permanently deleted.

**Response:** `200 OK`
```json
{
  "deleted_files": 5,
  "deleted_folders": 3
}
```

**Error:** `404 Not Found` if folder not found or not owned by user.

---

### 5.1.7 Download

#### `GET /api/v1/download/{id}`
Download original file.

**Response:** `200 OK` with `Content-Disposition: attachment; filename="IMG_1234.jpg"` and appropriate `Content-Type`.

#### `POST /api/v1/download/batch`
Download multiple files as a ZIP archive.

**Request:**
```json
{
  "file_ids": ["uuid-1", "uuid-2", "uuid-3"]
}
```

**Response:** `200 OK` with `Content-Type: application/zip` (streamed, not buffered in memory).

---

### 5.1.7b Folder Password Protection

These endpoints require authentication and folder ownership.

#### `POST /api/v1/folders/{id}/password`
Set or update a password on a folder. The password expires after 30 minutes (configurable). All folder contents become protected — the owner must also unlock to access. An optional `password_hint` helps remember the password.

**Request:**
```json
{ "password": "secret123", "password_hint": "My birthday + dog's name" }
```
**Response:** `201 Created`
```json
{ "message": "Password set for folder", "expires_at": "2026-06-02T16:00:00Z" }
```

#### `DELETE /api/v1/folders/{id}/password`
Remove the password protection from a folder.

**Response:** `204 No Content`

#### `POST /api/v1/folders/{id}/unlock`
Unlock a password-protected folder. Returns a short-lived unlock JWT.

**Request:**
```json
{ "password": "secret123" }
```
**Response:** `200 OK`
```json
{
  "unlock_token": "eyJhbGciOi...",
  "expires_at": "2026-06-02T16:00:00Z",
  "folder_id": "uuid"
}
```
**Error:** `401 Unauthorized` for wrong password. `404 Not Found` if no password set.

#### `GET /api/v1/folders/{id}/password`
Check if a folder has an active password.

**Response:** `200 OK`
```json
{ "has_password": true, "expires_at": "2026-06-02T16:00:00Z", "password_hint": "My birthday + dog's name" }
```

---

### 5.1.7c Folder Public Sharing

#### `POST /api/v1/folders/{id}/shares`
Create a public share link for a folder. Owner-only.

**Request:**
```json
{
  "permissions": "read_upload",
  "include_subdirs": true,
  "upload_limit_bytes": 104857600,
  "expires_at": "2026-12-31T23:59:59Z",
  "password": "optional_password"
}
```
**Response:** `201 Created`
```json
{
  "id": "uuid",
  "token": "uuid-token",
  "share_url": "/share/uuid-token",
  "folder_id": "uuid",
  "permissions": "read_upload",
  "include_subdirs": true,
  "upload_limit_bytes": 104857600,
  "expires_at": "2026-12-31T23:59:59Z",
  "has_password": false,
  "created_at": "2026-06-02T15:00:00Z"
}
```

#### `GET /api/v1/folders/{id}/shares`
List all shares for a folder. Owner-only. Returns `uploaded_bytes` per share.

#### `PUT /api/v1/folders/{id}/shares/{shareId}`
Update share permissions, limits, expiry, or password. Owner-only.

#### `DELETE /api/v1/folders/{id}/shares/{shareId}`
Revoke a share link. Owner-only.

---

### 5.1.7d Public Share Access (no authentication required)

All endpoints under `/api/v1/share/{token}` require a valid share session token (obtained via unlock).

#### `GET /api/v1/share/{token}`
Get share info without authentication.

**Response:** `200 OK`
```json
{
  "needs_password": false,
  "permissions": "read_upload",
  "include_subdirs": true,
  "upload_limit_bytes": 104857600,
  "uploaded_bytes": 0,
  "expires_at": "2026-12-31T23:59:59Z",
  "folder_name": "My Folder",
  "file_count": 10
}
```

#### `GET /api/v1/share/{token}/folders`
List subfolders in shared folder. Only available when `include_subdirs` is true. Accepts `?parent_id=` query param. Requires `X-Share-Session-Token` header with read permission.

#### `POST /api/v1/share/{token}/folders`
Create a subfolder within the shared folder tree. Requires `include_subdirs=true` AND `read_write` permission. Body: `{ "name": "New Folder", "parent_id": "optional-parent-id" }`

#### `DELETE /api/v1/share/{token}/folders/{id}`
Delete a subfolder (not the root shared folder). Requires `include_subdirs=true` AND `read_write` permission. Folder must be within the shared folder tree.

#### `POST /api/v1/share/{token}/unlock`
Obtain a share session token (with password if required). No auth needed.

**Request:** `{ "password": "sharepass" }` (if `needs_password` is true, otherwise `{}`)
**Response:** `200 OK`
```json
{
  "share_session_token": "eyJhbGciOi...",
  "expires_at": "2026-06-03T15:00:00Z"
}
```

#### `GET /api/v1/share/{token}/files`
List files in shared folder. Requires `X-Share-Session-Token` header. Accepts `?folder_id=` query param to list files in a subfolder (requires `include_subdirs=true`).
#### `GET /api/v1/share/{token}/files/{id}`
Get file detail. Requires read permission.
#### `GET /api/v1/share/{token}/download/{id}`
Download a file. Requires read permission.
#### `GET /api/v1/share/{token}/thumb/{fileID}/{size}`
Serve a thumbnail from the shared folder. Requires read permission. Pass share session token via query param `?share_session_token=...`.
#### `POST /api/v1/share/{token}/upload`
Upload files to shared folder. Requires upload permission. Enforces `upload_limit_bytes` quota. Multipart form with `files` field and optional `folder_id` field (for subdirectory uploads, requires `include_subdirs=true`).
#### `DELETE /api/v1/share/{token}/files/{id}`
Delete a file from shared folder. Requires write permission.

**Error Codes for Shares:**
| Code | HTTP | Meaning |
|------|------|---------|
| `SHARE_NOT_FOUND` | 404 | Share token does not exist |
| `SHARE_EXPIRED` | 410 | Share link has expired |
| `SHARE_PASSWORD_REQUIRED` | 403 | Share requires password unlock |
| `INVALID_SHARE_PASSWORD` | 401 | Wrong share password |
| `SHARE_TOKEN_REQUIRED` | 403 | Missing share session token |
| `SHARE_QUOTA_EXCEEDED` | 413 | Upload would exceed share upload limit |
| `PERMISSION_DENIED` | 403 | Share does not permit this action |

---

### 5.1.8 Search

#### `GET /api/v1/search`
Full-text search across filenames and EXIF metadata.

**Query Parameters:**
| Param | Type | Description |
|---|---|---|
| `q` | string | Search query |
| `limit` | int | Max results (default 50) |

**Response:** `200 OK` — same shape as `GET /api/v1/files`

---

### 5.1.9 Admin

These endpoints require the `admin` role.

#### `GET /api/v1/admin/users`
List all users.

**Response:** `200 OK`
```json
{
  "users": [
    {
      "id": "uuid-v7",
      "username": "johndoe",
      "role": "member",
      "display_name": "John Doe",
      "created_at": "2024-07-15T14:30:00Z",
      "space_quota": 10737418240,
      "file_count": 15420,
      "total_size_bytes": 8589934592,
      "thumbnail_size_bytes": 2147483648
    }
  ],
  "total": 3
}
```

#### `POST /api/v1/admin/users`
Create a new user. Admin only.

**Request:**
```json
{
  "username": "newuser",
  "password": "password123",
  "role": "member",
  "display_name": "New User"
}
```

**Response:** `201 Created` — returns the created user object (same shape as register response).

#### `DELETE /api/v1/admin/users/{id}`
Delete a user and all their files (soft-delete their files first).

**Response:** `204 No Content`

#### `PUT /api/v1/admin/users/{id}/role`
Change a user's role.

**Request:**
```json
{
  "role": "admin"
}
```

**Response:** `200 OK`

#### `PUT /api/v1/admin/users/{id}/quota`

---

#### `GET /api/v1/admin/files/breakdown?user_id=<optional-uuid>`
Admin file type/ext breakdown. If `user_id` is provided, results are scoped to that user; otherwise global aggregate.

**Response:** `200 OK`
```json
{
  "media_types": [
    { "media_type": "photo", "count": 120, "size_bytes": 800000000 }
  ],
  "extensions": [
    { "extension": "jpeg", "count": 80, "size_bytes": 500000000 }
  ],
  "total_size": 3350000000
}
```

#### `GET /api/v1/admin/thumbnails/stats?user_id=<optional-uuid>`
Admin thumbnail cache breakdown. If `user_id` is provided, results are scoped to that user; otherwise global aggregate.

**Response:** `200 OK`
```json
{
  "breakdown": [
    { "size": "sm", "count": 100, "total_size": 5000000 }
  ],
  "total_count": 400,
  "total_size_bytes": 255000000
}
```
Set a user's space quota (bytes, original files only). NULL = unlimited.

**Request:**
```json
{
  "space_quota": 10737418240
}
```

**Response:** `200 OK` — returns the updated user object with `space_quota`
**Error:** `422 Unprocessable Entity` if quota is below current usage ("QUOTA_BELOW_USAGE")

#### `GET /api/v1/admin/registration`
Get current registration toggle state.

**Response:** `200 OK`
```json
{
  "allow_registration": false
}
```

#### `PUT /api/v1/admin/registration`
Toggle self-registration on or off. Persists to SQLite `settings` table, overriding config file value.

**Request:**
```json
{
  "enabled": true
}
```

**Response:** `200 OK`
```json
{
  "allow_registration": true
}
```

#### `GET /api/v1/admin/jobs`
List upload jobs with pagination and optional status filter. Admin-only.

**Query Parameters:**
| Param | Type | Default | Description |
|---|---|---|---|
| `status` | string | — | Filter by status: `queued`, `processing`, `completed`, `skipped`, `failed` |
| `limit` | int | 50 | Items per page (max 200) |
| `offset` | int | 0 | Pagination offset |

**Response:** `200 OK`
```json
{
  "jobs": [
    {
      "id": "uuid",
      "batch_id": "uuid",
      "user_id": "uuid",
      "filename": "DSC00015.JPG",
      "size_bytes": 4521984,
      "status": "completed",
      "stage": "thumbnails",
      "progress": 1.0,
      "error": null,
      "reason": null,
      "file_id": "uuid",
      "created_at": "2026-05-30T17:25:03Z",
      "updated_at": "2026-05-30T17:25:14Z"
    }
  ],
  "total": 200,
  "summary": {
    "completed": 166,
    "failed": 34,
    "skipped": 0,
    "queued": 0,
    "processing": 0
  }
}
```

#### `POST /api/v1/admin/jobs/{id}/retry`
Retry a failed or skipped job by resetting it to `queued`. Returns `409 Conflict` if the job is not in a retryable state.

**Response:** `200 OK`
```json
{ "status": "ok" }
```

#### `POST /api/v1/admin/jobs/reconcile`
Scan for photos with missing thumbnails and create reconcile jobs to regenerate them. Jobs are picked up by the worker pool automatically.

**Response:** `200 OK`
```json
{
  "created": 237,
  "details": {
    "missing_all_thumbnails": 26,
    "missing_preview_only": 211
  }
}
```

#### `GET /api/v1/admin/events`
Paginated, filterable system events. Admin-only.

**Query Parameters:**
| Param | Type | Default | Description |
|---|---|---|---|
| `event_type` | string | — | Filter by type: `backup_success`, `backup_failure`, `backup_pruned`, `upload_error`, `upload_skip`, `reconciliation`, `cache_eviction`, `s3_connected`, `s3_disconnected`, `server_start`, `server_shutdown` |
| `severity` | string | — | Filter by severity: `info`, `warn`, `err` |
| `date_from` | string | — | ISO 8601 start date |
| `date_to` | string | — | ISO 8601 end date |
| `limit` | int | 50 | Items per page (max 200) |
| `offset` | int | 0 | Pagination offset |

**Response:** `200 OK`
```json
{
  "events": [
    {
      "id": "uuid-v7",
      "event_type": "backup_success",
      "severity": "info",
      "message": "Database backup completed successfully",
      "metadata": { "size_bytes": 5242880, "backup_key": "backups/database/drive-backup-2024-07-15T14:30:00Z.db" },
      "created_at": "2024-07-15T14:30:00Z"
    }
  ],
  "total": 1420
}
```

#### `GET /api/v1/admin/events/counts`
Event counts for UI filter badges. Admin-only.

**Response:** `200 OK`
```json
{
  "by_type": {
    "backup_success": 42,
    "backup_failure": 3,
    "backup_pruned": 30,
    "upload_error": 5,
    "upload_skip": 120,
    "reconciliation": 15,
    "cache_eviction": 864,
    "s3_connected": 2,
    "s3_disconnected": 1,
    "server_start": 5,
    "server_shutdown": 3
  }
}
```

#### `GET /api/v1/admin/backup/status`
Backup configuration and last result. Admin-only.

**Response:** `200 OK`
```json
{
  "enabled": true,
  "interval_h": 24,
  "retention_days": 7,
  "last_result": {
    "status": "success",
    "timestamp": "2024-07-15T14:30:00Z",
    "size_bytes": 5242880
  }
}
```

#### `POST /api/v1/admin/backup`
Trigger immediate database backup. Admin-only.

**Response:** `202 Accepted` — backup job queued.
**Error:** `409 Conflict` — a backup is already in progress.

---

### 5.1.10 Health & Stats

#### `GET /api/v1/health`
Health check. Does NOT require authentication.

**Response:** `200 OK`
```json
{
  "status": "ok",
  "version": "0.1.0",
  "uptime_seconds": 86400,
  "db_connected": true,
  "s3_connected": true
}
```

#### `GET /api/v1/stats`
Storage statistics.

**Response:** `200 OK`
```json
{
  "total_files": 15420,
  "total_photos": 12000,
  "total_videos": 500,
  "total_size_bytes": 128849018880,
  "cache_size_bytes": 52428800000,
  "cache_utilization_pct": 65.5,
  "photos_with_gps": 8500,
  "date_range": {
    "oldest": "2019-03-15T10:00:00Z",
    "newest": "2024-07-20T18:30:00Z"
  }
}
```

---

## 5.2 Error Response Format

All errors follow this structure:

```json
{
  "error": {
    "code": "DUPLICATE_FILE",
    "message": "A file with the same content already exists",
    "details": {
      "existing_file_id": "uuid-v7",
      "existing_filename": "2024/07/IMG_1234.jpg"
    }
  }
}
```

**HTTP Status Codes:**
| Code | Meaning |
|---|---|---|
| 200 | Success |
| 201 | Created |
| 202 | Accepted (async processing) |
| 204 | No Content (successful delete) |
| 400 | Bad Request (validation error) |
| 401 | Unauthorized (missing or invalid JWT) |
| 403 | Forbidden (insufficient role) |
| 404 | Not Found |
| 409 | Conflict (duplicate) |
| 413 | Payload Too Large |
| 415 | Unsupported Media Type |
| 422 | Unprocessable Entity (corrupt file) |
| 429 | Too Many Requests |
| 500 | Internal Server Error |
| 503 | Service Unavailable (storage unreachable) |

---

## 5.3 Authentication Flow

All authenticated endpoints require the `Authorization: Bearer <access_token>` HTTP header.

**Token lifecycle:**
1. User logs in → receives `access_token` (JWT, short-lived, 72h) + `refresh_token` (UUID, single-use)
2. Client sends `access_token` on every request. Auth middleware validates the JWT signature, extracts `user_id` and `role`, and scopes all queries to that user.
3. When `access_token` expires, client uses `POST /api/v1/auth/refresh` with the `refresh_token` to get a new pair.
4. On logout, the refresh token is invalidated server-side. The access_token continues to be valid until expiry (short window).

**Row-level security:** Every database query includes `WHERE user_id = ?` (exceptions: health endpoint, admin user management). Users can only access their own files.

**Admin endpoints** (`/api/v1/admin/*`) require `role: admin` in the JWT claims. Non-admin users receive `403 Forbidden`.

**Rate limiting** (future): Configurable per-endpoint and per-user limits. Returned as `429 Too Many Requests`.

---

### 5.1.11 Shared Albums (v3)

#### `GET /api/v1/albums`
List user's own albums + albums shared with user.

**Response:** `200 OK`
```json
{
  "myAlbums": [{ "id": "uuid", "name": "Vacation", "item_count": 42, "owner_id": "uuid", "is_shared": true, "created_at": "..." }],
  "sharedAlbums": [{ "id": "uuid", "name": "Party", "item_count": 12, "owner_id": "uuid", "owner_name": "bob", "is_shared": true, "created_at": "..." }]
}
```

#### `POST /api/v1/albums`
Create a new album. `{ "name": "Album Name", "description": "optional" }`

#### `GET /api/v1/albums/{id}`
Get album details with owner info, item count, share list.

#### `PUT /api/v1/albums/{id}`
Update album name/description (owner only).

#### `DELETE /api/v1/albums/{id}`
Delete album (owner only).

#### `GET /api/v1/albums/{id}/items`
List files in an album. Same shape as `GET /api/v1/files`.

#### `POST /api/v1/albums/{id}/items`
Add files to album. `{ "file_ids": ["uuid", ...] }` — requires edit permission.

#### `DELETE /api/v1/albums/{id}/items/{itemId}`
Remove file from album — requires edit permission.

#### `POST /api/v1/albums/{id}/shares`
Share album with user. `{ "username": "bob", "permission": "view|comment|edit" }` — owner only.

#### `DELETE /api/v1/albums/{id}/shares/{shareId}`
Remove share access — owner only.

### 5.1.12 Comments & Reactions (v3)

#### `GET /api/v1/files/{id}/comments`
List comments on a file. Returns array with `{ id, user_id, username, content, created_at, reactions[] }`.

#### `POST /api/v1/files/{id}/comments`
Add a comment. `{ "content": "Great photo!" }` — requires file access.

#### `PUT /api/v1/files/{id}/comments/{commentId}`
Edit own comment. `{ "content": "Updated" }`

#### `DELETE /api/v1/files/{id}/comments/{commentId}`
Delete own comment.

#### `POST /api/v1/files/{id}/comments/{commentId}/reactions`
Toggle emoji reaction. `{ "emoji": "👍" }` — returns `{ "emoji": "👍", "added": true|false }`

#### `DELETE /api/v1/files/{id}/comments/{commentId}/reactions/{emoji}`
Remove own reaction.

### 5.1.13 Tags (v3)

#### `GET /api/v1/tags?q=prefix`
List tags matching prefix (for autocomplete). Returns `{ "tags": [{ "id": "uuid", "name": "beach" }] }`

#### `GET /api/v1/files/{id}/tags`
Get tags on a file.

#### `POST /api/v1/files/{id}/tags`
Add tags to file. `{ "tags": ["vacation", "beach"] }` — tag owner only.

#### `DELETE /api/v1/files/{id}/tags/{tagId}`
Remove tag from file — file owner only.

### 5.1.14 Enhanced Search (v3) — Extends `GET /api/v1/search`

New query parameters:

| Param | Type | Description |
|---|---|---|
| `q` | string | Full-text search (filename, FTS5) |
| `size_min` | int | Minimum file size in bytes |
| `size_max` | int | Maximum file size in bytes |
| `created_after` | string | ISO 8601 |
| `created_before` | string | ISO 8601 |
| `taken_after` | string | ISO 8601 |
| `taken_before` | string | ISO 8601 |
| `tags` | string | Comma-separated tag names |
| `limit` | int | Max results (default 50) |

Response includes `folder_path` field for files in folders.

---