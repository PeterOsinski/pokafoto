CREATE TABLE IF NOT EXISTS tags (
    id   TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE
);

CREATE TABLE IF NOT EXISTS file_tags (
    file_id          TEXT NOT NULL REFERENCES files(id) ON DELETE CASCADE,
    tag_id           TEXT NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    added_by_user_id TEXT NOT NULL REFERENCES users(id),
    created_at       TEXT NOT NULL DEFAULT (datetime('now')),
    PRIMARY KEY (file_id, tag_id)
);

CREATE INDEX idx_file_tags_tag ON file_tags(tag_id);
CREATE INDEX idx_file_tags_file ON file_tags(file_id);
