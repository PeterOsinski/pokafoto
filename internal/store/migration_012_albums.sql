CREATE TABLE IF NOT EXISTS albums (
    id          TEXT PRIMARY KEY,
    user_id     TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    description TEXT,
    created_at  TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at  TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX idx_albums_user ON albums(user_id);

CREATE TABLE IF NOT EXISTS album_items (
    id               TEXT PRIMARY KEY,
    album_id         TEXT NOT NULL REFERENCES albums(id) ON DELETE CASCADE,
    file_id          TEXT NOT NULL REFERENCES files(id) ON DELETE CASCADE,
    added_by_user_id TEXT NOT NULL REFERENCES users(id),
    sort_order       INTEGER NOT NULL DEFAULT 0,
    created_at       TEXT NOT NULL DEFAULT (datetime('now')),
    UNIQUE(album_id, file_id)
);

CREATE INDEX idx_album_items_album ON album_items(album_id);
CREATE INDEX idx_album_items_file ON album_items(file_id);

CREATE TABLE IF NOT EXISTS album_shares (
    id                  TEXT PRIMARY KEY,
    album_id            TEXT NOT NULL REFERENCES albums(id) ON DELETE CASCADE,
    shared_with_user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    permission          TEXT NOT NULL DEFAULT 'view' CHECK(permission IN ('view', 'comment', 'edit')),
    created_at          TEXT NOT NULL DEFAULT (datetime('now')),
    UNIQUE(album_id, shared_with_user_id)
);

CREATE INDEX idx_album_shares_album ON album_shares(album_id);
CREATE INDEX idx_album_shares_user ON album_shares(shared_with_user_id);
