CREATE VIRTUAL TABLE IF NOT EXISTS files_fts USING fts5(
    filename,
    original_name,
    camera_make,
    camera_model,
    content='files',
    content_rowid='rowid'
);

CREATE TRIGGER IF NOT EXISTS files_ai AFTER INSERT ON files BEGIN
    INSERT INTO files_fts(rowid, filename, original_name)
    VALUES (new.rowid, new.filename, new.original_name);
END;

CREATE TRIGGER IF NOT EXISTS files_ad AFTER DELETE ON files BEGIN
    INSERT INTO files_fts(files_fts, rowid, filename, original_name)
    VALUES ('delete', old.rowid, old.filename, old.original_name);
END;

CREATE TRIGGER IF NOT EXISTS files_au AFTER UPDATE ON files BEGIN
    INSERT INTO files_fts(files_fts, rowid, filename, original_name)
    VALUES ('delete', old.rowid, old.filename, old.original_name);
    INSERT INTO files_fts(rowid, filename, original_name)
    VALUES (new.rowid, new.filename, new.original_name);
END;
