-- +goose Up
CREATE TABLE pegawai (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    bidang_id     BIGINT NOT NULL
                  REFERENCES bidang(id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    nama          TEXT NOT NULL,
    nip           TEXT NOT NULL,
    tanggal_lahir TEXT NOT NULL,
    alamat        TEXT NOT NULL,
    is_male       INTEGER NOT NULL CHECK (is_male IN (0, 1)),
    search_text   TEXT GENERATED ALWAYS AS (
                      nama || ' ' ||
                      nip || ' ' ||
                      substr(tanggal_lahir, 9, 2) || '-' ||
                      substr(tanggal_lahir, 6, 2) || '-' ||
                      substr(tanggal_lahir, 1, 4) || ' ' ||
                      alamat || ' ' ||
                      CASE WHEN is_male THEN 'pria' ELSE 'wanita' END
                  ) VIRTUAL,
    created_at    INTEGER,
    created_by    BIGINT,
    updated_at    INTEGER,
    updated_by    BIGINT,
    deleted_at    INTEGER,
    deleted_by    BIGINT
);

CREATE INDEX idx_pegawai_bidang_id ON pegawai (bidang_id);

CREATE UNIQUE INDEX ux_pegawai_nip_active
    ON pegawai (nip)
    WHERE deleted_at IS NULL;

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

-- +goose Down
-- Drop the FTS index before its external content table.
DROP TABLE IF EXISTS pegawai_fts;
DROP TABLE IF EXISTS pegawai;
