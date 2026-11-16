-- +goose Up
CREATE TABLE user (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    username    TEXT NOT NULL,
    password    TEXT NOT NULL,
    name        TEXT NOT NULL,
    role        TEXT NOT NULL CHECK (role IN ('admin', 'user')),
    -- Password is intentionally excluded: it must never be searchable or
    -- exposed through the trigram index.
    search_text TEXT GENERATED ALWAYS AS (
                    username || ' ' || name || ' ' || role
                ) VIRTUAL,
    created_at  INTEGER,
    created_by  BIGINT,
    updated_at  INTEGER,
    updated_by  BIGINT,
    deleted_at  INTEGER,
    deleted_by  BIGINT
);

CREATE UNIQUE INDEX ux_user_username_active
    ON user (username)
    WHERE deleted_at IS NULL;

-- External-content FTS5 table: the text lives only in user.search_text,
-- while user_fts stores the trigram index.
CREATE VIRTUAL TABLE user_fts USING fts5(
    search_text,
    content='user',
    content_rowid='id',
    tokenize='trigram'
);

-- +goose StatementBegin
CREATE TRIGGER user_fts_ai AFTER INSERT ON user BEGIN
    INSERT INTO user_fts (rowid, search_text)
    VALUES (new.id, new.search_text);
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER user_fts_ad AFTER DELETE ON user BEGIN
    INSERT INTO user_fts (user_fts, rowid, search_text)
    VALUES ('delete', old.id, old.search_text);
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER user_fts_au AFTER UPDATE ON user
WHEN old.search_text IS NOT new.search_text
BEGIN
    INSERT INTO user_fts (user_fts, rowid, search_text)
    VALUES ('delete', old.id, old.search_text);
    INSERT INTO user_fts (rowid, search_text)
    VALUES (new.id, new.search_text);
END;
-- +goose StatementEnd

-- +goose Down
-- Drop the FTS index before its external content table.
DROP TABLE IF EXISTS user_fts;
DROP TABLE IF EXISTS user;
