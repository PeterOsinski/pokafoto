CREATE TABLE IF NOT EXISTS folder_passwords (
    id              TEXT PRIMARY KEY,
    folder_id       TEXT NOT NULL REFERENCES folders(id) ON DELETE CASCADE,
    password_hash   TEXT NOT NULL,
    expires_at      TEXT NOT NULL,
    created_at      TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at      TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_folder_passwords_folder ON folder_passwords(folder_id);

CREATE TABLE IF NOT EXISTS folder_shares (
    id                  TEXT PRIMARY KEY,
    folder_id           TEXT NOT NULL REFERENCES folders(id) ON DELETE CASCADE,
    token               TEXT NOT NULL UNIQUE,
    permissions         TEXT NOT NULL DEFAULT 'read' CHECK(permissions IN ('read', 'read_upload', 'read_write')),
    upload_limit_bytes  INTEGER,
    expires_at          TEXT,
    has_password        INTEGER NOT NULL DEFAULT 0,
    password_hash       TEXT,
    created_at          TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at          TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_folder_shares_token ON folder_shares(token);
CREATE INDEX IF NOT EXISTS idx_folder_shares_folder ON folder_shares(folder_id);

CREATE TABLE IF NOT EXISTS share_uploads (
    id          TEXT PRIMARY KEY,
    share_id    TEXT NOT NULL REFERENCES folder_shares(id) ON DELETE CASCADE,
    file_id     TEXT NOT NULL REFERENCES files(id) ON DELETE CASCADE,
    size_bytes  INTEGER NOT NULL,
    created_at  TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_share_uploads_share ON share_uploads(share_id);
