CREATE TABLE IF NOT EXISTS system_events (
    id              TEXT PRIMARY KEY,
    event_type      TEXT NOT NULL,
    severity        TEXT NOT NULL CHECK(severity IN ('info','warning','error')),
    message         TEXT NOT NULL,
    metadata        TEXT,
    created_at      TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX idx_system_events_type ON system_events(event_type);
CREATE INDEX idx_system_events_severity ON system_events(severity);
CREATE INDEX idx_system_events_created ON system_events(created_at DESC);
CREATE INDEX idx_system_events_type_severity ON system_events(event_type, severity);
