-- +goose Up
CREATE TABLE pegawai (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    nip         TEXT      NOT NULL UNIQUE,
    nama        TEXT      NOT NULL,
    jabatan     TEXT      NOT NULL DEFAULT '',
    bidang_id   INTEGER   NOT NULL REFERENCES bidang(id),
    email       TEXT      NOT NULL DEFAULT '',
    telepon     TEXT      NOT NULL DEFAULT '',
    alamat      TEXT      NOT NULL DEFAULT '',
    created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_pegawai_bidang_id ON pegawai (bidang_id);

-- +goose Down
DROP TABLE pegawai;
