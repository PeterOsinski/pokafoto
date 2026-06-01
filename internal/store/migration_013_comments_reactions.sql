CREATE TABLE IF NOT EXISTS comments (
    id         TEXT PRIMARY KEY,
    file_id    TEXT NOT NULL REFERENCES files(id) ON DELETE CASCADE,
    user_id    TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    content    TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX idx_comments_file ON comments(file_id);
CREATE INDEX idx_comments_user ON comments(user_id);

CREATE TABLE IF NOT EXISTS reactions (
    id         TEXT PRIMARY KEY,
    comment_id TEXT NOT NULL REFERENCES comments(id) ON DELETE CASCADE,
    user_id    TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    emoji      TEXT NOT NULL CHECK(length(emoji) > 0),
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    UNIQUE(comment_id, user_id, emoji)
);

CREATE INDEX idx_reactions_comment ON reactions(comment_id);
