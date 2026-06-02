ALTER TABLE files ADD COLUMN is_app_managed INTEGER NOT NULL DEFAULT 0;
CREATE INDEX idx_files_app_managed ON files(is_app_managed);

CREATE TABLE IF NOT EXISTS documents (
    file_id   TEXT PRIMARY KEY REFERENCES files(id) ON DELETE CASCADE,
    content   TEXT NOT NULL DEFAULT ''
);
