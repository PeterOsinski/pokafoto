CREATE TABLE IF NOT EXISTS upload_chunks (
    upload_id   TEXT NOT NULL,
    chunk_index INTEGER NOT NULL,
    chunk_size  INTEGER NOT NULL,
    offset      INTEGER NOT NULL,
    status      TEXT NOT NULL DEFAULT 'pending'
                    CHECK(status IN ('pending','stored')),
    chunk_sha256 TEXT,
    temp_path   TEXT,
    created_at  TEXT NOT NULL DEFAULT (datetime('now')),
    PRIMARY KEY (upload_id, chunk_index)
);

ALTER TABLE upload_jobs ADD COLUMN upload_mode TEXT NOT NULL DEFAULT 'full';
ALTER TABLE upload_jobs ADD COLUMN chunk_size INTEGER;
ALTER TABLE upload_jobs ADD COLUMN total_chunks INTEGER;
ALTER TABLE upload_jobs ADD COLUMN resume_token TEXT;

CREATE UNIQUE INDEX IF NOT EXISTS idx_upload_jobs_resume_token ON upload_jobs(resume_token);
