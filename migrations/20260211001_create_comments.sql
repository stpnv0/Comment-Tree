-- +goose Up
CREATE EXTENSION IF NOT EXISTS ltree;

CREATE TABLE comments (
    id         BIGSERIAL    PRIMARY KEY,
    parent_id  BIGINT       REFERENCES comments(id) ON DELETE CASCADE,
    author     TEXT         NOT NULL,
    body       TEXT         NOT NULL,
    path       ltree        NOT NULL,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_comments_path_gist   ON comments USING GIST (path);
CREATE INDEX idx_comments_parent_id   ON comments (parent_id);
CREATE INDEX idx_comments_created_at  ON comments (created_at);
CREATE INDEX idx_comments_search      ON comments USING GIN (to_tsvector('simple', body));

-- +goose Down
DROP TABLE IF EXISTS comments;
DROP EXTENSION IF EXISTS ltree;
