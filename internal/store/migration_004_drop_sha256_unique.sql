PRAGMA foreign_keys = OFF;

CREATE TABLE files_new (
    id          TEXT PRIMARY KEY,
    user_id     TEXT NOT NULL REFERENCES users(id),
    filename    TEXT NOT NULL,
    original_name TEXT NOT NULL,
    path        TEXT NOT NULL,
    size_bytes  INTEGER NOT NULL,
    mime_type   TEXT NOT NULL,
    sha256      TEXT NOT NULL,
    media_type  TEXT NOT NULL DEFAULT 'file',
    width       INTEGER,
    height      INTEGER,
    duration_sec REAL,
    taken_at    TEXT,
    folder_id   TEXT REFERENCES folders(id) ON DELETE SET NULL,
    created_at  TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at  TEXT NOT NULL DEFAULT (datetime('now')),
    is_deleted  INTEGER NOT NULL DEFAULT 0
);

INSERT INTO files_new (id, user_id, filename, original_name, path, size_bytes, mime_type, sha256, media_type, width, height, duration_sec, taken_at, created_at, updated_at, is_deleted, folder_id)
SELECT id, user_id, filename, original_name, path, size_bytes, mime_type, sha256, media_type, width, height, duration_sec, taken_at, created_at, updated_at, is_deleted, folder_id FROM files;

DROP TABLE files;

ALTER TABLE files_new RENAME TO files;

CREATE INDEX idx_files_user ON files(user_id);
CREATE INDEX idx_files_path ON files(path);
CREATE INDEX idx_files_taken_at ON files(taken_at);
CREATE INDEX idx_files_media_type ON files(media_type);
CREATE INDEX idx_files_sha256 ON files(sha256);
CREATE INDEX idx_files_deleted ON files(is_deleted);
CREATE INDEX idx_files_name_size ON files(original_name, size_bytes);
CREATE INDEX idx_files_folder ON files(folder_id);

DROP TABLE IF EXISTS files_fts;

CREATE VIRTUAL TABLE files_fts USING fts5(
    filename,
    original_name,
    camera_make,
    camera_model,
    content='files',
    content_rowid='rowid'
);

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

INSERT INTO files_fts(rowid, filename, original_name) SELECT rowid, filename, original_name FROM files;

PRAGMA foreign_keys = ON;
