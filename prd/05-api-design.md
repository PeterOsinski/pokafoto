# 5. API Design

## 5.1 REST API Endpoints

### Base URL: `/api/v1`

All endpoints return JSON. Timestamps are ISO 8601. IDs are UUID v7 strings.

**Authentication:** All endpoints require `Authorization: Bearer <token>` header, except for:
- `POST /api/v1/auth/register` (if `auth.allow_registration: true`)
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
skip_name_size_dedup: string (optional) — "true" to skip name+size dedup pre-check (used for inline folder uploads). Defaults to false (dedup enabled).
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
1. **Name+Size check (Upload tab only):** When `skip_name_size_dedup` is not set to `"true"` (i.e., dedicated Upload tab), before any processing, check if a file with the same `original_name` AND `size_bytes` already exists. If yes → status `skipped` with reason `duplicate_name_size`. The file is silently ignored — no upload, no processing. Inline folder uploads (from gallery/folder view) set `skip_name_size_dedup=true` and skip this check.
2. **Content hash check (all files):** SHA-256 hash computed during upload. If an identical hash already exists → status `skipped` with reason `duplicate_content`. This catches renamed duplicates and applies to all uploads.

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
  "stage": "generating_thumbnails"
}
```

---

### 5.1.2 Files & Gallery

#### `GET /api/v1/files`
List files with pagination, sorting, and filtering.

**Query Parameters:**
| Param | Type | Default | Description |
|---|---|---|---|
| `folder_id` | string | — | Filter by folder UUID. `folders` layout passes `folder_id=uuid`. Root view omits this param (filters `folder_id IS NULL`). Value `root` also means root. |
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
| `path` | string | `""` | Root path to start from |
| `depth` | int | `1` | How many levels deep |

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
Get the user's folder tree with file counts.

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
Rename a folder.

**Request:**
```json
{ "name": "Summer Trip 2024" }
```
**Response:** `204 No Content`

#### `DELETE /api/v1/folders/{id}`
Delete a folder. Files inside revert to root via `ON DELETE SET NULL` FK. Nested subfolders cascade-delete via `ON DELETE CASCADE` on `parent_id`.

**Response:** `204 No Content`

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
      "file_count": 15420
    }
  ],
  "total": 3
}
```

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