# AGENTS.md

Panduan untuk agent yang bekerja di repo ini.

## Proyek

Project **Go** sederhana. Gunakan stack yang sudah ditentukan; jangan ganti
tanpa alasan kuat.

## Stack

- Bahasa: **Go**
- Router: **`github.com/go-chi/chi/v5`**
- Database: **SQLite3** (satu-satunya database), driver **`modernc.org/sqlite`** (pure Go, tanpa CGO)
- ORM: **`github.com/uptrace/bun`** + **`bun/dialect/sqlitedialect`**
- Migration: **`github.com/pressly/goose/v3`**

Buka koneksi dengan `sql.Open("sqlite", dsn)`. Jangan memakai
`bun/driver/sqliteshim` atau `github.com/mattn/go-sqlite3`; keduanya menambah
ketergantungan yang tidak perlu dan `mattn` butuh CGO.

## Perintah

```bash
go run .                  # jalankan aplikasi
go build ./...            # build
go test ./...             # test
go vet ./...              # vet

goose up                  # jalankan migration
goose down                # rollback satu migration
goose status              # cek status migration
goose create <nama> sql   # buat file migration baru
goose -dir migrations sqlite3 app.db up   # jalankan tanpa config
```

Selalu jalankan `gofmt`, `go vet ./...`, dan `go test ./...` sebelum selesai.

## Struktur

- `main.go` — entrypoint, wiring dependency, start server.
- `internal/` — kode aplikasi (handler, service, repository, model).
- `migrations/` — file migration goose.
- Pisahkan layer: handler tidak menulis query, repository tidak menulis response
  HTTP.

## Konvensi

### Umum

- Ikuti pola package yang sudah ada; jangan bikin abstraksi baru kalau belum
  perlu.
- Gunakan `context.Context` untuk operasi DB dan request.
- Bungkus error dengan `fmt.Errorf("...: %w", err)`.
- Jangan menelan error. Kembalikan atau log dengan konteks.
- Komentar kode dalam bahasa Inggris; teks untuk pengguna dalam bahasa
  Indonesia.

### Router (chi)

- Daftarkan route di satu tempat (router), kelompokkan per resource.
- Gunakan middleware chi untuk logging, recovery, dan hal lintas-cutting.
- Path parameter dibaca dengan `chi.URLParam`.

### Database (Bun + SQLite3)

- Database hanya **SQLite3**. Jangan menambah database lain.
- Semua query Bun berada di layer repository.
- Gunakan model dengan tag `bun` dan `json`.
- Pakai `context` pada setiap query.
- Tangani `sql.ErrNoRows` sebagai not-found di layer repository.
- Aktifkan `PRAGMA foreign_keys = ON` di setiap koneksi.
- Gunakan mode WAL dan `busy_timeout` untuk menghindari `database is locked`.
- SQLite hanya punya satu penulis; jaga pool koneksi tetap kecil dan
  serialisasi penulisan. Jangan menaikkan pool tanpa alasan jelas.
- Simpan waktu sebagai **epoch milliseconds UTC** (`time.Now().UTC().UnixMilli()`),
  bukan `TEXT`/`datetime`.
- Pelanggaran `UNIQUE` dan `FOREIGN KEY` dipetakan ke error domain (HTTP `409`),
  bukan `500`.

#### Primary key

- Setiap tabel punya `id` dengan DDL **`INTEGER PRIMARY KEY AUTOINCREMENT`**.
  Di SQLite, `INTEGER` adalah integer 64-bit (setara `BIGINT`) dan satu-satunya
  tipe yang menjadi alias rowid. Jangan menulis `BIGINT PRIMARY KEY` karena
  tidak akan autoincrement.
- Di Go, `id` dipetakan ke `int64`.

#### Kolom audit

Setiap tabel **wajib** punya kolom audit berikut:

| Kolom | Tipe | Nullable | Default |
| --- | --- | --- | --- |
| `created_at` | `INTEGER` (epoch ms) | ya | tidak |
| `created_by` | `BIGINT` | ya | tidak |
| `updated_at` | `INTEGER` (epoch ms) | ya | tidak |
| `updated_by` | `BIGINT` | ya | tidak |
| `deleted_at` | `INTEGER` (epoch ms) | ya | tidak |
| `deleted_by` | `BIGINT` | ya | tidak |

- Semua kolom audit **nullable** dan **tanpa `DEFAULT`**; nilai diisi aplikasi,
  bukan database.
- `*_by` bertipe `BIGINT` (`int64`), mengikuti tipe `id`, dan **bukan FK** —
  jangan menambahkan `REFERENCES` pada kolom audit.
- **Jangan expose kolom audit ke FE/API.** Kolom audit tidak boleh muncul di
  response JSON maupun diterima dari request body. Pakai `json:"-"` pada field
  audit di model, atau map ke DTO response yang hanya berisi field publik.
- Kolom audit tetap dipakai internal (filter soft delete, pengisian otomatis).
- Saat insert: isi `created_at` dan `created_by`; kolom `updated_*`/`deleted_*`
  tetap `NULL`.
- Saat update: isi `updated_at` dan `updated_by`.
- Soft delete: isi `deleted_at` dan `deleted_by`; **jangan** `DELETE` fisik.
- Query baca default wajib menambahkan `WHERE deleted_at IS NULL`.
- Di model Bun, kolom nullable dipetakan ke pointer (`*int64`), bukan
  `time.Time`.

#### Foreign key

- Semua FK wajib eksplisit: **`ON UPDATE RESTRICT ON DELETE RESTRICT`**.
- Kolom FK bertipe `BIGINT` (`int64`).
- Aktifkan `PRAGMA foreign_keys = ON`; tanpa ini FK tidak ditegakkan SQLite.
- Hard delete parent yang masih direferensikan akan gagal; petakan ke error
  domain (HTTP `409`), bukan `500`.
- Beri index pada kolom FK.

#### Unique constraint (soft delete)

Karena memakai soft delete, unique constraint **wajib** berupa **partial unique
index** dengan kondisi `deleted_at IS NULL`. Jangan memakai `UNIQUE` inline di
kolom, karena baris yang sudah di-soft-delete akan tetap bentrok.

```sql
CREATE UNIQUE INDEX ux_items_kode_active
    ON items (kode)
    WHERE deleted_at IS NULL;
```

Index biasa hanya untuk FK dan partial unique di atas. Jangan membuat index
untuk kolom pencarian.

Contoh DDL tabel:

```sql
CREATE TABLE items (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    bidang_id  BIGINT NOT NULL
               REFERENCES bidang(id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    kode       TEXT NOT NULL,
    created_at INTEGER,
    created_by BIGINT,
    updated_at INTEGER,
    updated_by BIGINT,
    deleted_at INTEGER,
    deleted_by BIGINT
);

CREATE INDEX idx_items_bidang_id ON items (bidang_id);

CREATE UNIQUE INDEX ux_items_kode_active
    ON items (kode)
    WHERE deleted_at IS NULL;
```

### Migration (goose)

- Semua perubahan skema lewat file migration baru; jangan mengubah migration
  yang sudah pernah dijalankan.
- Format file: `<timestamp>_<nama>.sql` (hasil `goose create`).
- Wajib punya blok `-- +goose Up` dan `-- +goose Down`.
- Migration harus idempoten terhadap urutan eksekusi dari awal.
- Perhatikan batasan SQLite: `ALTER TABLE` terbatas, dan
  `ADD COLUMN ... NOT NULL REFERENCES` tidak didukung. Untuk perubahan seperti
  itu, buat tabel baru lalu salin data di dalam migration.

### API

- Response sukses: `{"data": ...}`.
- Response error: `{"error": "pesan"}` dengan status HTTP yang sesuai
  (`400`, `401`, `403`, `404`, `409`, `422`, `500`).
- Jangan expose kolom audit (`created_*`, `updated_*`, `deleted_*`) di response
  maupun request. Gunakan DTO terpisah atau `json:"-"`.
- Jangan bocorkan detail error internal ke klien; log di server.
- Batasi ukuran body request.

### Testing

- Test memakai database sementara, jangan menyentuh database runtime.
- Setiap endpoint baru punya test happy path + kasus error.
- Test tidak boleh bergantung pada state global atau urutan eksekusi.

## Jangan commit

- File database runtime (`*.db`, `*.db-shm`, `*.db-wal`).
- Binary hasil build (`bin/`).
- File `.env` atau secret.
- Perubahan editor/IDE yang tidak relevan.
