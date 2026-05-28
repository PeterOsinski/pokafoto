CREATE TABLE IF NOT EXISTS users (
    id              TEXT PRIMARY KEY,
    username        TEXT NOT NULL UNIQUE,
    password_hash   TEXT NOT NULL,
    role            TEXT NOT NULL DEFAULT 'member' CHECK(role IN ('admin', 'member')),
    display_name    TEXT,
    created_at      TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at      TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX idx_users_username ON users(username);

CREATE TABLE IF NOT EXISTS sessions (
    id              TEXT PRIMARY KEY,
    user_id         TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    refresh_token   TEXT NOT NULL UNIQUE,
    expires_at      TEXT NOT NULL,
    created_at      TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX idx_sessions_user ON sessions(user_id);
CREATE INDEX idx_sessions_token ON sessions(refresh_token);

CREATE TABLE IF NOT EXISTS files (
    id          TEXT PRIMARY KEY,
    user_id     TEXT NOT NULL REFERENCES users(id),
    filename    TEXT NOT NULL,
    original_name TEXT NOT NULL,
    path        TEXT NOT NULL,
    size_bytes  INTEGER NOT NULL,
    mime_type   TEXT NOT NULL,
    sha256      TEXT NOT NULL UNIQUE,
    media_type  TEXT NOT NULL DEFAULT 'file',
    width       INTEGER,
    height      INTEGER,
    duration_sec REAL,
    taken_at    TEXT,
    created_at  TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at  TEXT NOT NULL DEFAULT (datetime('now')),
    is_deleted  INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX idx_files_user ON files(user_id);
CREATE INDEX idx_files_path ON files(path);
CREATE INDEX idx_files_taken_at ON files(taken_at);
CREATE INDEX idx_files_media_type ON files(media_type);
CREATE INDEX idx_files_sha256 ON files(sha256);
CREATE INDEX idx_files_deleted ON files(is_deleted);
CREATE INDEX idx_files_name_size ON files(original_name, size_bytes);

CREATE TABLE IF NOT EXISTS exif (
    file_id         TEXT PRIMARY KEY REFERENCES files(id) ON DELETE CASCADE,
    camera_make     TEXT,
    camera_model    TEXT,
    lens_make       TEXT,
    lens_model      TEXT,
    focal_length    REAL,
    aperture        REAL,
    shutter_speed   TEXT,
    iso             INTEGER,
    date_taken      TEXT,
    gps_latitude    REAL,
    gps_longitude   REAL,
    gps_altitude    REAL,
    orientation     INTEGER,
    color_space     TEXT,
    flash           INTEGER,
    software        TEXT,
    raw_json        TEXT
);

CREATE INDEX idx_exif_camera ON exif(camera_make, camera_model);
CREATE INDEX idx_exif_date ON exif(date_taken);
CREATE INDEX idx_exif_gps ON exif(gps_latitude, gps_longitude)
    WHERE gps_latitude IS NOT NULL AND gps_longitude IS NOT NULL;

CREATE TABLE IF NOT EXISTS thumbnails (
    file_id     TEXT NOT NULL REFERENCES files(id) ON DELETE CASCADE,
    size        TEXT NOT NULL CHECK(size IN ('sm', 'md', 'preview', 'video_still')),
    width       INTEGER NOT NULL,
    height      INTEGER NOT NULL,
    format      TEXT NOT NULL DEFAULT 'jpeg',
    local_path  TEXT NOT NULL,
    s3_key      TEXT,
    size_bytes  INTEGER NOT NULL,
    created_at  TEXT NOT NULL DEFAULT (datetime('now')),
    PRIMARY KEY (file_id, size)
);

CREATE INDEX idx_thumbnails_local ON thumbnails(local_path);

CREATE VIRTUAL TABLE IF NOT EXISTS geo_index USING rtree (
    id,
    min_lat, max_lat,
    min_lon, max_lon
);

CREATE TABLE IF NOT EXISTS geo_index_meta (
    rtree_id    INTEGER PRIMARY KEY,
    file_id     TEXT NOT NULL REFERENCES files(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS schema_migrations (
    version     INTEGER PRIMARY KEY,
    applied_at  TEXT NOT NULL DEFAULT (datetime('now'))
);
