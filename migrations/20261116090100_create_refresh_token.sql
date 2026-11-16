-- +goose Up
-- Refresh tokens are opaque technical secrets, not business entities: no soft
-- delete and no full audit columns. Only the SHA-256 hash is stored, so a
-- database leak does not expose usable tokens. Logout and rotation hard delete
-- the row; expired rows are ignored on read and can be cleaned up separately.
CREATE TABLE refresh_token (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id    BIGINT NOT NULL
               REFERENCES user(id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    token_hash TEXT NOT NULL,
    expires_at TEXT NOT NULL,
    created_at INTEGER NOT NULL
);

CREATE INDEX idx_refresh_token_user_id ON refresh_token (user_id);

CREATE UNIQUE INDEX ux_refresh_token_hash ON refresh_token (token_hash);

-- +goose Down
DROP TABLE IF EXISTS refresh_token;
