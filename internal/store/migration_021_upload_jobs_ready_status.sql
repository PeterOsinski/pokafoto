PRAGMA foreign_keys = OFF;

CREATE TABLE upload_jobs_new (
    id                    TEXT PRIMARY KEY,
    batch_id              TEXT NOT NULL,
    user_id               TEXT NOT NULL REFERENCES users(id),
    filename              TEXT NOT NULL,
    size_bytes            INTEGER NOT NULL,
    temp_path             TEXT NOT NULL,
    folder_id             TEXT,
    skip_name_size_dedup  INTEGER NOT NULL DEFAULT 0,
    status                TEXT NOT NULL DEFAULT 'queued' CHECK(status IN ('queued','ready','processing','completed','skipped','failed')),
    stage                 TEXT,
    progress              REAL NOT NULL DEFAULT 0.0,
    error                 TEXT,
    reason                TEXT,
    file_id               TEXT,
    upload_mode           TEXT NOT NULL DEFAULT 'full',
    chunk_size            INTEGER,
    total_chunks          INTEGER,
    resume_token          TEXT,
    created_at            TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at            TEXT NOT NULL DEFAULT (datetime('now'))
);

INSERT INTO upload_jobs_new SELECT * FROM upload_jobs;

DROP TABLE upload_jobs;

ALTER TABLE upload_jobs_new RENAME TO upload_jobs;

CREATE INDEX IF NOT EXISTS idx_upload_jobs_status ON upload_jobs(status);
CREATE INDEX IF NOT EXISTS idx_upload_jobs_batch ON upload_jobs(batch_id);
CREATE INDEX IF NOT EXISTS idx_upload_jobs_user ON upload_jobs(user_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_upload_jobs_resume_token ON upload_jobs(resume_token);

PRAGMA foreign_keys = ON;
