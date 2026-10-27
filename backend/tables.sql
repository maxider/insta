--id has text type as UUID is not supported in SQLite
CREATE TABLE IF NOT EXISTS image_data (
    id TEXT PRIMARY KEY NOT NULL,
    title TEXT NOT NULL,
    filesize INT NOT NULL,
    mime_type INT NOT NULL,
    size_x INT NOT NULL,
    size_y INT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS tags (
    name TEXT PRIMARY KEY NOT NULL,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

--join TABLE
-- using UUIDs wastes a lot of space here instead of normal counter. Might reconsider given the context.
CREATE TABLE IF NOT EXISTS image_tags(
    image_id TEXT NOT NULL REFERENCES images(id) ON DELETE CASCADE,
    tag_name TEXT NOT NULL REFERENCES tags(name) ON DELETE CASCADE,
    PRIMARY KEY (image_id, tag_name)
)
