# AGENTS.md

Panduan untuk agent yang bekerja di repo ini.

## Proyek

Project **Go** sederhana. Gunakan stack yang sudah ditentukan; jangan ganti
tanpa alasan kuat.

## Stack

- Bahasa: **Go**
- Router: **`github.com/go-chi/chi/v5`**
- Database: **SQLite3** (satu-satunya database)
- ORM: **`github.com/uptrace/bun`** + **`bun/dialect/sqlitedialect`**
- Migration: **`github.com/pressly/goose/v3`**

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
- Simpan waktu dalam UTC dan isi `created_at`/`updated_at` otomatis.
- Index hanya untuk FK dan constraint unik; jangan menambah index untuk kolom
  pencarian.
- Pelanggaran `UNIQUE` dan `FOREIGN KEY` dipetakan ke error domain (HTTP `409`),
  bukan `500`.

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
