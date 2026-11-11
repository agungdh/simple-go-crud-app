-- +goose Up
CREATE TABLE pegawai (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    nama          TEXT NOT NULL,
    nip           TEXT NOT NULL,
    tanggal_lahir INTEGER NOT NULL,
    alamat        TEXT NOT NULL,
    jenis_kelamin TEXT NOT NULL CHECK (jenis_kelamin IN ('L', 'P')),
    created_at    INTEGER,
    created_by    BIGINT,
    updated_at    INTEGER,
    updated_by    BIGINT,
    deleted_at    INTEGER,
    deleted_by    BIGINT
);

CREATE UNIQUE INDEX ux_pegawai_nip_active
    ON pegawai (nip)
    WHERE deleted_at IS NULL;

-- +goose Down
DROP INDEX IF EXISTS ux_pegawai_nip_active;
DROP TABLE IF EXISTS pegawai;
