-- +goose Up
CREATE TABLE IF NOT EXISTS posts (
    id UUID NOT NULL PRIMARY KEY,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    title TEXT,
    url TEXT NOT NULL UNIQUE,
    description TEXT,
    published_at TIMESTAMP NOT NULL,
    feed_id UUID NOT NULL,
    FOREIGN KEY (feed_id) REFERENCES feed(id) ON DELETE CASCADE
);

-- +goose Down
DROP TABLE IF EXISTS posts;