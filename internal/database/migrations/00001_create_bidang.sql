-- +goose Up
CREATE TABLE bidang (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    kode        TEXT      NOT NULL UNIQUE,
    nama        TEXT      NOT NULL,
    created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- +goose Down
DROP TABLE bidang;
