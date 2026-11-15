-- +goose Up
-- search_text concatenates every searchable column of pegawai: all columns
-- except id, foreign keys, and audit columns. It is VIRTUAL so the value is
-- never stored twice; ALTER TABLE can add virtual generated columns without
-- rewriting the table.
ALTER TABLE pegawai ADD COLUMN search_text TEXT
    GENERATED ALWAYS AS (
        nama || ' ' ||
        nip || ' ' ||
        CAST(tanggal_lahir AS TEXT) || ' ' ||
        alamat || ' ' ||
        jenis_kelamin
    ) VIRTUAL;

-- External-content FTS5 table: the text lives only in pegawai.search_text,
-- while pegawai_fts stores the trigram index.
CREATE VIRTUAL TABLE pegawai_fts USING fts5(
    search_text,
    content='pegawai',
    content_rowid='id',
    tokenize='trigram'
);

-- Keep the trigram index in sync with the table. Soft delete is an UPDATE
-- that leaves search_text unchanged, so that trigger stays quiet; reads join
-- pegawai and filter deleted_at, making leftover index rows harmless.
-- +goose StatementBegin
CREATE TRIGGER pegawai_fts_ai AFTER INSERT ON pegawai BEGIN
    INSERT INTO pegawai_fts (rowid, search_text)
    VALUES (new.id, new.search_text);
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER pegawai_fts_ad AFTER DELETE ON pegawai BEGIN
    INSERT INTO pegawai_fts (pegawai_fts, rowid, search_text)
    VALUES ('delete', old.id, old.search_text);
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER pegawai_fts_au AFTER UPDATE ON pegawai
WHEN old.search_text IS NOT new.search_text
BEGIN
    INSERT INTO pegawai_fts (pegawai_fts, rowid, search_text)
    VALUES ('delete', old.id, old.search_text);
    INSERT INTO pegawai_fts (rowid, search_text)
    VALUES (new.id, new.search_text);
END;
-- +goose StatementEnd

-- Index rows that existed before the triggers were created.
INSERT INTO pegawai_fts (pegawai_fts) VALUES ('rebuild');

-- +goose Down
DROP TRIGGER IF EXISTS pegawai_fts_au;
DROP TRIGGER IF EXISTS pegawai_fts_ad;
DROP TRIGGER IF EXISTS pegawai_fts_ai;
DROP TABLE IF EXISTS pegawai_fts;
ALTER TABLE pegawai DROP COLUMN search_text;
