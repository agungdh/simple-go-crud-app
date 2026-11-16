-- +goose Up
CREATE TABLE bidang (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    bidang      TEXT NOT NULL,
    search_text TEXT GENERATED ALWAYS AS (bidang) VIRTUAL,
    created_at  INTEGER,
    created_by  BIGINT,
    updated_at  INTEGER,
    updated_by  BIGINT,
    deleted_at  INTEGER,
    deleted_by  BIGINT
);

CREATE UNIQUE INDEX ux_bidang_bidang_active
    ON bidang (bidang)
    WHERE deleted_at IS NULL;

-- External-content FTS5 table: the text lives only in bidang.search_text,
-- while bidang_fts stores the trigram index.
CREATE VIRTUAL TABLE bidang_fts USING fts5(
    search_text,
    content='bidang',
    content_rowid='id',
    tokenize='trigram'
);

-- +goose StatementBegin
CREATE TRIGGER bidang_fts_ai AFTER INSERT ON bidang BEGIN
    INSERT INTO bidang_fts (rowid, search_text)
    VALUES (new.id, new.search_text);
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER bidang_fts_ad AFTER DELETE ON bidang BEGIN
    INSERT INTO bidang_fts (bidang_fts, rowid, search_text)
    VALUES ('delete', old.id, old.search_text);
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER bidang_fts_au AFTER UPDATE ON bidang
WHEN old.search_text IS NOT new.search_text
BEGIN
    INSERT INTO bidang_fts (bidang_fts, rowid, search_text)
    VALUES ('delete', old.id, old.search_text);
    INSERT INTO bidang_fts (rowid, search_text)
    VALUES (new.id, new.search_text);
END;
-- +goose StatementEnd

-- +goose Down
-- Drop the FTS index before its external content table.
DROP TABLE IF EXISTS bidang_fts;
DROP TABLE IF EXISTS bidang;
