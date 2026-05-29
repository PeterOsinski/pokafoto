CREATE TABLE IF NOT EXISTS upload_jobs (
    id                    TEXT PRIMARY KEY,
    batch_id              TEXT NOT NULL,
    user_id               TEXT NOT NULL REFERENCES users(id),
    filename              TEXT NOT NULL,
    size_bytes            INTEGER NOT NULL,
    temp_path             TEXT NOT NULL,
    folder_id             TEXT,
    skip_name_size_dedup  INTEGER NOT NULL DEFAULT 0,
    status                TEXT NOT NULL DEFAULT 'queued' CHECK(status IN ('queued','processing','completed','skipped','failed')),
    stage                 TEXT,
    progress              REAL NOT NULL DEFAULT 0.0,
    error                 TEXT,
    reason                TEXT,
    file_id               TEXT,
    created_at            TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at            TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX idx_upload_jobs_status ON upload_jobs(status);
CREATE INDEX idx_upload_jobs_batch ON upload_jobs(batch_id);
CREATE INDEX idx_upload_jobs_user ON upload_jobs(user_id);
