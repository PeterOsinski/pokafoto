# 4. Data Model

## 4.1 Entity Relationship Diagram

```
┌──────────────────────┐       ┌──────────────────────┐
│        users          │       │        files          │
├──────────────────────┤       ├──────────────────────┤
│ id (TEXT PK)         │       │ id (TEXT PK)         │──┐
│ username (TEXT UNQ)  │──┐    │ user_id (TEXT FK)    │  │
│ password_hash (TEXT) │  │    │ filename (TEXT)      │  │
│ role (TEXT)          │  │    │ original_name (TEXT) │  │
│ display_name (TEXT)  │  └───▶│ path (TEXT)          │  │
│ created_at (TEXT)    │       │ size_bytes (INT)     │  │
│ updated_at (TEXT)    │       │ mime_type (TEXT)     │  │
└──────────────────────┘       │ sha256 (TEXT UNIQUE) │  │
           │                   │ media_type (TEXT)    │  │
           │ (1:N)             │ width (INT)          │  │
           ▼                   │ height (INT)         │  │
┌──────────────────────┐       │ duration_sec (REAL)  │  │
│       sessions        │       │ taken_at (TEXT)      │  │
├──────────────────────┤       │ created_at (TEXT)    │  │
│ id (TEXT PK)         │       │ updated_at (TEXT)    │  │
│ user_id (TEXT FK)    │       │ is_deleted (INT)     │  │
│ refresh_token (TEXT) │       └──────────────────────┘  │
│ expires_at (TEXT)    │                 │               │
│ created_at (TEXT)    │                 │ (1:0..1)      │
└──────────────────────┘                 ▼               │
                                ┌──────────────────────┐ │
                                │        exif          │ │
                                ├──────────────────────┤ │
                                │ file_id (TEXT PK/FK) │◀┘
                                │ camera_make (TEXT)   │
                                │ ...                  │
                                └──────────────────────┘
                                          │
                                          │ (1:4)
                                          ▼
                                ┌──────────────────────┐
                                │      thumbnails       │
                                ├──────────────────────┤
                                │ file_id (TEXT PK/FK) │
                                │ size (TEXT PK)       │
                                │   -- 'sm','md','preview','video_still'
                                │ ...                  │
                                └──────────────────────┘

┌──────────────────────┐
│     geo_index         │  (R-tree virtual table)
├──────────────────────┤
│ file_id (TEXT)       │
│ min_lat (REAL)       │
│ max_lat (REAL)       │
│ min_lon (REAL)       │
│ max_lon (REAL)       │
└──────────────────────┘

┌──────────────────────┐       ┌──────────────────────┐
│       folders         │       │        files          │
├──────────────────────┤       │ (folder_id FK)       │
│ id (TEXT PK)         │──┐    └──────────┬───────────┘
│ user_id (TEXT FK)    │  │               │
│ name (TEXT)          │  │    (N:1)       │
│ parent_id (TEXT FK)  │◀─┘               │
│ created_at (TEXT)    │                   │
│ updated_at (TEXT)    │                   │
└──────────────────────┘                   │
      │ (self-ref)                         │
      └───────────────────────────────────┘
                                           (N:1)
    folders.parent_id → folders.id (NULL = root)
    files.folder_id → folders.id (NULL = root, ON DELETE SET NULL)
```

## 4.2 SQL Schema

```sql
-- migrations/001_initial.sql

PRAGMA journal_mode = WAL;
PRAGMA foreign_keys = ON;

-- User accounts
CREATE TABLE IF NOT EXISTS users (
    id              TEXT PRIMARY KEY,          -- UUID v7
    username        TEXT NOT NULL UNIQUE,
    password_hash   TEXT NOT NULL,             -- bcrypt hash
    role            TEXT NOT NULL DEFAULT 'member' CHECK(role IN ('admin', 'member')),
    display_name    TEXT,
    created_at      TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at      TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX idx_users_username ON users(username);

-- Sessions / refresh tokens
CREATE TABLE IF NOT EXISTS sessions (
    id              TEXT PRIMARY KEY,          -- UUID v7
    user_id         TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    refresh_token   TEXT NOT NULL UNIQUE,
    expires_at      TEXT NOT NULL,
    created_at      TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX idx_sessions_user ON sessions(user_id);
CREATE INDEX idx_sessions_token ON sessions(refresh_token);

-- Core file table
CREATE TABLE IF NOT EXISTS files (
    id          TEXT PRIMARY KEY,              -- UUID v7 (time-sortable)
    user_id     TEXT NOT NULL REFERENCES users(id),
    filename    TEXT NOT NULL,                 -- Stored filename (e.g., "2024/07/IMG_1234.jpg")
    original_name TEXT NOT NULL,               -- Original upload filename
    path        TEXT NOT NULL,                 -- Directory path within storage
    size_bytes  INTEGER NOT NULL,
    mime_type   TEXT NOT NULL,
    sha256      TEXT NOT NULL UNIQUE,          -- Content hash for dedup
    media_type  TEXT NOT NULL DEFAULT 'file',  -- 'photo', 'video', 'file'
    width       INTEGER,                       -- Image/video width (null for non-media)
    height      INTEGER,                       -- Image/video height
    duration_sec REAL,                         -- Video duration in seconds
    taken_at    TEXT,                          -- ISO 8601 timestamp from EXIF or file date
    created_at  TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at  TEXT NOT NULL DEFAULT (datetime('now')),
    is_deleted  INTEGER NOT NULL DEFAULT 0    -- Soft delete flag
);

CREATE INDEX idx_files_user ON files(user_id);
CREATE INDEX idx_files_path ON files(path);
CREATE INDEX idx_files_taken_at ON files(taken_at);
CREATE INDEX idx_files_media_type ON files(media_type);
CREATE INDEX idx_files_sha256 ON files(sha256);
CREATE INDEX idx_files_deleted ON files(is_deleted);
-- Composite index for name+size dedup check (photos folder)
CREATE INDEX idx_files_name_size ON files(original_name, size_bytes);

-- EXIF metadata table (1:1 with files for media)
CREATE TABLE IF NOT EXISTS exif (
    file_id         TEXT PRIMARY KEY REFERENCES files(id) ON DELETE CASCADE,
    camera_make     TEXT,
    camera_model    TEXT,
    lens_make       TEXT,
    lens_model      TEXT,
    focal_length    REAL,                   -- mm
    aperture        REAL,                   -- f-number
    shutter_speed   TEXT,                   -- e.g., "1/250"
    iso             INTEGER,
    date_taken      TEXT,                   -- Original EXIF DateTimeOriginal
    gps_latitude    REAL,
    gps_longitude   REAL,
    gps_altitude    REAL,                   -- meters
    orientation     INTEGER,                -- EXIF orientation tag (1-8)
    color_space     TEXT,                   -- sRGB, Adobe RGB, etc.
    flash           INTEGER,                -- 0=no, 1=fired
    software        TEXT,                   -- Processing software
    raw_json        TEXT                    -- Full EXIF dump as JSON for future use
);

CREATE INDEX idx_exif_camera ON exif(camera_make, camera_model);
CREATE INDEX idx_exif_date ON exif(date_taken);
CREATE INDEX idx_exif_gps ON exif(gps_latitude, gps_longitude)
    WHERE gps_latitude IS NOT NULL AND gps_longitude IS NOT NULL;

-- Thumbnail registry
CREATE TABLE IF NOT EXISTS thumbnails (
    file_id     TEXT NOT NULL REFERENCES files(id) ON DELETE CASCADE,
    size        TEXT NOT NULL CHECK(size IN ('sm', 'md', 'preview', 'video_still')),
    width       INTEGER NOT NULL,
    height      INTEGER NOT NULL,
    format      TEXT NOT NULL DEFAULT 'jpeg',  -- 'jpeg' for sm/md/video_still, 'webp' for preview
    local_path  TEXT NOT NULL,                 -- Absolute path on local cache disk
    s3_key      TEXT,                        -- S3 object key (NULL if S3 disabled)
    size_bytes  INTEGER NOT NULL,
    created_at  TEXT NOT NULL DEFAULT (datetime('now')),
    PRIMARY KEY (file_id, size)
);

CREATE INDEX idx_thumbnails_local ON thumbnails(local_path);

-- R-tree spatial index for geo queries
CREATE VIRTUAL TABLE IF NOT EXISTS geo_index USING rtree (
    id,             -- Integer primary key for R-tree
    min_lat, max_lat,
    min_lon, max_lon
);

-- +file_id reference stored separately since R-tree doesn't support foreign keys
CREATE TABLE IF NOT EXISTS geo_index_meta (
    rtree_id    INTEGER PRIMARY KEY,
    file_id     TEXT NOT NULL REFERENCES files(id) ON DELETE CASCADE
);

-- Schema version tracking
CREATE TABLE IF NOT EXISTS schema_migrations (
    version     INTEGER PRIMARY KEY,
    applied_at  TEXT NOT NULL DEFAULT (datetime('now'))
);

-- migrations/002_fts5.sql
-- Full-text search virtual table (separate migration for evolvability)
CREATE VIRTUAL TABLE IF NOT EXISTS files_fts USING fts5(
    filename,
    original_name,
    camera_make,
    camera_model,
    content='files',
    content_rowid='rowid'
);

-- Triggers to keep FTS index in sync
CREATE TRIGGER files_ai AFTER INSERT ON files BEGIN
    INSERT INTO files_fts(rowid, filename, original_name)
    VALUES (new.rowid, new.filename, new.original_name);
END;

CREATE TRIGGER files_ad AFTER DELETE ON files BEGIN
    INSERT INTO files_fts(files_fts, rowid, filename, original_name)
    VALUES ('delete', old.rowid, old.filename, old.original_name);
END;

CREATE TRIGGER files_au AFTER UPDATE ON files BEGIN
    INSERT INTO files_fts(files_fts, rowid, filename, original_name)
    VALUES ('delete', old.rowid, old.filename, old.original_name);
    INSERT INTO files_fts(rowid, filename, original_name)
    VALUES (new.rowid, new.filename, new.original_name);
END;
```

```sql
-- migrations/003_folders.sql
-- User-created folder hierarchy for file organization

CREATE TABLE IF NOT EXISTS folders (
    id          TEXT PRIMARY KEY,
    user_id     TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    parent_id   TEXT REFERENCES folders(id) ON DELETE CASCADE,
    created_at  TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at  TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX idx_folders_user ON folders(user_id);
CREATE INDEX idx_folders_parent ON folders(parent_id);
CREATE UNIQUE INDEX idx_folders_name_parent ON folders(user_id, name, COALESCE(parent_id, ''));

ALTER TABLE files ADD COLUMN folder_id TEXT REFERENCES folders(id) ON DELETE SET NULL;

CREATE INDEX idx_files_folder ON files(folder_id);
```

## 4.3 Go Models

```go
// internal/model/user.go
package model

type UserRole string

const (
    RoleAdmin  UserRole = "admin"
    RoleMember UserRole = "member"
)

type User struct {
    ID           string    `json:"id" db:"id"`
    Username     string    `json:"username" db:"username"`
    PasswordHash string    `json:"-" db:"password_hash"`
    Role         UserRole  `json:"role" db:"role"`
    DisplayName  *string   `json:"display_name,omitempty" db:"display_name"`
    CreatedAt    time.Time `json:"created_at" db:"created_at"`
    UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

// internal/model/session.go
package model

type Session struct {
    ID           string    `json:"id" db:"id"`
    UserID       string    `json:"user_id" db:"user_id"`
    RefreshToken string    `json:"-" db:"refresh_token"`
    ExpiresAt    time.Time `json:"expires_at" db:"expires_at"`
    CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

// internal/model/file.go
package model

import "time"

type MediaType string

const (
    MediaTypePhoto MediaType = "photo"
    MediaTypeVideo MediaType = "video"
    MediaTypeFile  MediaType = "file"
)

type File struct {
    ID           string    `json:"id" db:"id"`
    UserID       string    `json:"user_id" db:"user_id"`
    Filename     string    `json:"filename" db:"filename"`
    OriginalName string    `json:"original_name" db:"original_name"`
    Path         string    `json:"path" db:"path"`
    SizeBytes    int64     `json:"size_bytes" db:"size_bytes"`
    MimeType     string    `json:"mime_type" db:"mime_type"`
    SHA256       string    `json:"sha256" db:"sha256"`
    MediaType    MediaType `json:"media_type" db:"media_type"`
    Width        *int      `json:"width,omitempty" db:"width"`
    Height       *int      `json:"height,omitempty" db:"height"`
    DurationSec  *float64  `json:"duration_sec,omitempty" db:"duration_sec"`
    TakenAt      *string   `json:"taken_at,omitempty" db:"taken_at"`
    FolderID     *string   `json:"folder_id,omitempty" db:"folder_id"`
    CreatedAt    time.Time `json:"created_at" db:"created_at"`
    UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
    IsDeleted    bool      `json:"is_deleted" db:"is_deleted"`
}

// internal/model/folder.go
package model

import "time"

type Folder struct {
    ID        string    `json:"id" db:"id"`
    UserID    string    `json:"user_id" db:"user_id"`
    Name      string    `json:"name" db:"name"`
    ParentID  *string   `json:"parent_id,omitempty" db:"parent_id"`
    CreatedAt time.Time `json:"created_at" db:"created_at"`
    UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

type FolderTreeNode struct {
    Folder    *Folder           `json:"folder"`
    FileCount int64             `json:"fileCount"`
    Children  []*FolderTreeNode `json:"children,omitempty"`
}

// internal/model/exif.go
package model

type ExifData struct {
    FileID       string   `json:"file_id" db:"file_id"`
    CameraMake   *string  `json:"camera_make,omitempty" db:"camera_make"`
    CameraModel  *string  `json:"camera_model,omitempty" db:"camera_model"`
    LensMake     *string  `json:"lens_make,omitempty" db:"lens_make"`
    LensModel    *string  `json:"lens_model,omitempty" db:"lens_model"`
    FocalLength  *float64 `json:"focal_length,omitempty" db:"focal_length"`
    Aperture     *float64 `json:"aperture,omitempty" db:"aperture"`
    ShutterSpeed *string  `json:"shutter_speed,omitempty" db:"shutter_speed"`
    ISO          *int     `json:"iso,omitempty" db:"iso"`
    DateTaken    *string  `json:"date_taken,omitempty" db:"date_taken"`
    GPSLatitude  *float64 `json:"gps_latitude,omitempty" db:"gps_latitude"`
    GPSLongitude *float64 `json:"gps_longitude,omitempty" db:"gps_longitude"`
    GPSAltitude  *float64 `json:"gps_altitude,omitempty" db:"gps_altitude"`
    Orientation  *int     `json:"orientation,omitempty" db:"orientation"`
    ColorSpace   *string  `json:"color_space,omitempty" db:"color_space"`
    Flash        *int     `json:"flash,omitempty" db:"flash"`
    Software     *string  `json:"software,omitempty" db:"software"`
    RawJSON      *string  `json:"raw_json,omitempty" db:"raw_json"`
}

// internal/model/thumbnail.go
package model

type ThumbnailSize string

const (
    ThumbSizeSmall     ThumbnailSize = "sm"
    ThumbSizeMedium    ThumbnailSize = "md"
    ThumbSizePreview   ThumbnailSize = "preview"
    ThumbSizeVideoStill ThumbnailSize = "video_still"
)

type Thumbnail struct {
    FileID    string        `json:"file_id" db:"file_id"`
    Size      ThumbnailSize `json:"size" db:"size"`
    Width     int           `json:"width" db:"width"`
    Height    int           `json:"height" db:"height"`
    Format    string        `json:"format" db:"format"`
    LocalPath string        `json:"local_path" db:"local_path"`
    S3Key     *string       `json:"s3_key,omitempty" db:"s3_key"`
    SizeBytes int64         `json:"size_bytes" db:"size_bytes"`
    CreatedAt time.Time     `json:"created_at" db:"created_at"`
}
```

## 4.4 API Response Shapes

```typescript
// web/src/api/types.ts

interface FileItem {
  id: string;
  userId: string;
  filename: string;
  originalName: string;
  path: string;
  sizeBytes: number;
  mimeType: string;
  sha256: string;
  mediaType: 'photo' | 'video' | 'file';
  width?: number;
  height?: number;
  durationSec?: number;
  takenAt?: string;       // ISO 8601
  folder_id?: string;     // UUID of parent folder, null = root
  createdAt: string;
  updatedAt: string;
  // Joined on read:
  exif?: ExifData;
  thumbnails?: ThumbnailSet;
}

interface FolderNode {
  folder: {
    id: string;
    name: string;
    parent_id: string | null;
    user_id: string;
    created_at: string;
    updated_at: string;
  };
  fileCount: number;
  children: FolderNode[];
}

interface ExifData {
  cameraMake?: string;
  cameraModel?: string;
  lensMake?: string;
  lensModel?: string;
  focalLength?: number;
  aperture?: number;
  shutterSpeed?: string;
  iso?: number;
  dateTaken?: string;
  gpsLatitude?: number;
  gpsLongitude?: number;
  gpsAltitude?: number;
  orientation?: number;
  colorSpace?: string;
  flash?: number;
  software?: string;
}

interface ThumbnailSet {
  sm: ThumbnailInfo;          // 60px JPEG
  md: ThumbnailInfo;          // 600px JPEG
  preview: ThumbnailInfo;    // 720p WebP
  videoStill?: ThumbnailInfo; // frame at 5s JPEG (videos only)
}

interface ThumbnailInfo {
  url: string;           // /api/v1/thumb/{fileId}/{size}.jpg (or .webp for preview)
  width: number;
  height: number;
}

// Paginated response
interface FileListResponse {
  items: FileItem[];
  nextCursor?: string;   // Cursor-based pagination
  total: number;
}

// Geo point for map
interface GeoPoint {
  fileId: string;
  latitude: number;
  longitude: number;
  thumbnailUrl: string;  // sm thumbnail
  takenAt: string;
}

interface GeoClusterResponse {
  clusters: GeoCluster[];
  points: GeoPoint[];    // Unclustered points at current zoom
}

interface GeoCluster {
  latitude: number;      // Cluster center
  longitude: number;
  count: number;
  thumbnailUrl?: string; // Representative thumbnail
}