--id has text type as UUID is not supported in SQLite
CREATE TABLE IF NOT EXISTS images (
    id TEXT PRIMARY KEY NOT NULL,
    image_type TEXT NOT NULL,
    uploaded_at TEXT NOT NULL DEFAULT (datetime('now'))
)
