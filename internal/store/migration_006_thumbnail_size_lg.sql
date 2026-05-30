CREATE TABLE IF NOT EXISTS thumbnails_new (
    file_id     TEXT NOT NULL REFERENCES files(id) ON DELETE CASCADE,
    size        TEXT NOT NULL CHECK(size IN ('sm', 'md', 'lg', 'preview', 'video_still')),
    width       INTEGER NOT NULL,
    height      INTEGER NOT NULL,
    format      TEXT NOT NULL DEFAULT 'jpeg',
    local_path  TEXT NOT NULL,
    s3_key      TEXT,
    size_bytes  INTEGER NOT NULL,
    created_at  TEXT NOT NULL DEFAULT (datetime('now')),
    PRIMARY KEY (file_id, size)
);

INSERT OR IGNORE INTO thumbnails_new SELECT * FROM thumbnails;

DROP TABLE thumbnails;

ALTER TABLE thumbnails_new RENAME TO thumbnails;

CREATE INDEX IF NOT EXISTS idx_thumbnails_local ON thumbnails(local_path);
