ALTER TABLE files ADD COLUMN deleted_at TEXT;

UPDATE files SET deleted_at = updated_at WHERE is_deleted = 1;
