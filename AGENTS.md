# AGENTS.md

Panduan untuk agent yang bekerja di repo ini.

## TL;DR

| Topik | Intinya |
| --- | --- |
| Stack | Go + chi + Bun + SQLite (`modernc.org/sqlite`) + goose. Jangan ganti. |
| Cek cepat | `make help` (daftar target), `make run`, `make check` (wajib sebelum selesai) |
| DB | `persistence/db/app.db`, WAL, `busy_timeout`, `foreign_keys=ON`, pool kecil |
| Migration | `make migrate-create NAME=nama` → isi Up/Down → `make migrate-up` |
| Layer | `main.go` wiring; `internal/` handler → service → repository → model; tidak campur |
| ID | `INTEGER PRIMARY KEY AUTOINCREMENT`, di Go `int64` |
| Waktu | Audit `*_at` = epoch **ms UTC**; tanggal/waktu bisnis = `TEXT` ISO-8601 (`YYYY-MM-DD`) |
| Audit | Wajib `model.Base`; isi via `MarkCreated/MarkUpdated/MarkDeleted`; jangan expose ke API |
| Soft delete | `model.Active(q)` + `UPDATE deleted_at/deleted_by`; unik = partial index `WHERE deleted_at IS NULL` |
| Query | **Jangan `SELECT *`**; pakai `Column(model.PublicColumns(...)...)` |
| Cari | `search_text` (VIRTUAL) + FTS5 trigram `<tabel>_fts`; `?q=` wajib ≥3 karakter (kurang → `422`) |
| FK | `ON UPDATE RESTRICT ON DELETE RESTRICT`, tipe `BIGINT`, beri index |
| Error | Unique/FK → `409`; format `{"data": ...}` / `{"error": "..."}`; detail internal jangan dibocorkan |
| Test | DB sementara; tiap endpoint: happy path + error |
| Jangan commit | `*.db*`, `bin/`, `.env`, file IDE |

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

Semua perintah harian lewat `make`; jalankan `make` atau `make help` untuk
melihat daftar target.

### Build, test, dan tooling

| Perintah | Ekuivalen manual | Fungsi |
| --- | --- | --- |
| `make run` | `ADDR=:8080 DB_PATH=persistence/db/app.db go run .` | Jalankan aplikasi |
| `make build` | `go build -trimpath -o bin/simple-go-crud-app .` | Build binary ke `bin/` |
| `make test` | `go test ./...` | Jalankan semua test |
| `make vet` | `go vet ./...` | Jalankan `go vet` |
| `make fmt` | `gofmt -w .` | Format semua file Go |
| `make fmt-check` | `gofmt -l .` (harus kosong) | Cek format tanpa mengubah file |
| `make tidy` | `go mod tidy` | Rapikan `go.mod` dan `go.sum` |
| `make check` | `make fmt-check vet test` | Format, vet, dan test sekaligus |
| `make clean` | `rm -rf bin` | Hapus hasil build |

### Migration (goose)

| Perintah | Ekuivalen manual | Fungsi |
| --- | --- | --- |
| `make migrate-up` | `goose -dir migrations sqlite3 persistence/db/app.db up` | Jalankan semua migration |
| `make migrate-down` | `goose -dir migrations sqlite3 persistence/db/app.db down` | Rollback satu migration terakhir |
| `make migrate-status` | `goose -dir migrations sqlite3 persistence/db/app.db status` | Tampilkan status migration |
| `make migrate-create NAME=nama_migration` | `goose -dir migrations sqlite3 persistence/db/app.db create nama_migration sql` | Buat file migration baru |

Target migration memakai `GOOSE_TAGS` (driver SQLite saja) dan menjalankan goose
lewat `go run`, jadi tidak perlu install goose. Override bila perlu, mis.
`make migrate-up GOOSE=goose DB_PATH=/tmp/test.db`.

### Database (opsional)

| Perintah | Fungsi |
| --- | --- |
| `docker compose up -d` | Jalankan sqlite-web di <http://127.0.0.1:8083> |
| `docker compose down` | Hentikan sqlite-web |

### Variabel

| Variabel | Default | Dipakai oleh |
| --- | --- | --- |
| `ADDR` | `:8080` | `make run` |
| `DB_PATH` | `persistence/db/app.db` | `make run`, target migration |
| `NAME` | — (wajib) | `make migrate-create` |
| `GOOSE` | `go run -tags '<GOOSE_TAGS>' github.com/pressly/goose/v3/cmd/goose@v3.30.0` | Target migration |

Selalu jalankan `make check` (fmt, vet, test) sebelum selesai.

## Struktur

- `main.go` — entrypoint, wiring dependency, start server.
- `internal/` — kode aplikasi (handler, service, repository, model, router).
- `migrations/` — file migration goose.
- `persistence/db/` — database runtime (tidak di-commit); hanya `.gitkeep`.
- `Makefile` — target perintah harian; sumber kebenaran daftar perintah.
- `compose.yml` — sqlite-web untuk inspeksi database.
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
- **Hanya kolom audit** `*_at` yang menyimpan **epoch milliseconds UTC**
  (`time.Now().UTC().UnixMilli()`), bukan `TEXT`.
- Kolom tanggal/waktu **bisnis** (mis. `tanggal_lahir`, `jadwal`) disimpan
  sebagai **`TEXT` ISO-8601**: `YYYY-MM-DD` untuk tanggal, `YYYY-MM-DDTHH:MM:SSZ`
  untuk waktu. Jangan memakai epoch milliseconds di luar kolom audit.
- Pelanggaran `UNIQUE` dan `FOREIGN KEY` dipetakan ke error domain (HTTP `409`),
  bukan `500`.

#### Select kolom

- **Jangan `SELECT *`.** Selalu pilih kolom eksplisit dengan `.Column(...)`.
- Default: semua kolom publishable (semua kecuali audit) lewat
  `model.PublicColumns(db, (*Entity)(nil))`.
- Pilih hanya kolom yang benar-benar dipakai response; jangan ikutkan kolom
  audit.
- Untuk JOIN atau query tabel langsung, tulis kolom eksplisit dengan prefix
  alias (`p.id`, `p.nama`, dst.).
- Kolom audit tetap boleh dipakai di klausa `WHERE` (mis. `deleted_at IS NULL`)
  meskipun tidak di-`SELECT`.

```go
q := db.NewSelect().
    Model(&items).
    Column(model.PublicColumns(db, (*Item)(nil))...)
q = model.Active(q)
```

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
- Semua entity **wajib embed `model.Base`** (`internal/model/base.go`); jangan
  mendeklarasikan ulang `id` atau kolom audit.
- Isi audit lewat `MarkCreated(actorID)`, `MarkUpdated(actorID)`, dan
  `MarkDeleted(actorID)`. Jangan mengisi manual.
- Filter baca wajib memakai `model.Active(q)`; untuk query JOIN, filter juga
  tabel yang di-join (`p.deleted_at IS NULL`, `b.deleted_at IS NULL`, dst.).
- Jangan memakai tag `bun:",soft_delete"` bawaan Bun: implementasinya menyimpan
  `UnixNano`, bukan epoch milliseconds.
- Jangan memanggil `db.NewDelete()` untuk soft delete. Lakukan `UPDATE` yang
  mengisi `deleted_at`/`deleted_by` lewat `MarkDeleted`.
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

Index biasa hanya untuk FK dan partial unique di atas. Pencarian memakai FTS5
trigram (lihat [Pencarian](#pencarian-trigram)); jangan membuat index B-tree
biasa untuk kolom pencarian.

Contoh DDL tabel:

```sql
CREATE TABLE items (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    bidang_id   BIGINT NOT NULL
                REFERENCES bidang(id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    kode        TEXT NOT NULL,
    search_text TEXT GENERATED ALWAYS AS (kode) VIRTUAL,
    created_at  INTEGER,
    created_by  BIGINT,
    updated_at  INTEGER,
    updated_by  BIGINT,
    deleted_at  INTEGER,
    deleted_by  BIGINT
);

CREATE INDEX idx_items_bidang_id ON items (bidang_id);

CREATE UNIQUE INDEX ux_items_kode_active
    ON items (kode)
    WHERE deleted_at IS NULL;
```

#### Pencarian (trigram)

Setiap tabel wajib punya kolom `search_text` dan index FTS5 trigram.

- `search_text` adalah **generated column `VIRTUAL`** berisi gabungan semua
  kolom dengan pemisah spasi: **semua kolom kecuali `id`, kolom FK, dan kolom
  audit**. Bungkus kolom nullable dengan `coalesce(kolom, '')`.
- Cast nilai ke bentuk yang **dilihat/diketik user**, bukan bentuk mentah:
  tanggal `YYYY-MM-DD` → `DD-MM-YYYY`
  (`substr(tanggal, 9, 2) || '-' || substr(tanggal, 6, 2) || '-' || substr(tanggal, 1, 4)`),
  boolean/enum → label Indonesia (`pria`/`wanita`), bukan `0`/`1` atau `L`/`P`.
  Kolom lain yang bukan teks di-`CAST(... AS TEXT)`.
- Jangan mendeklarasikan `search_text` di model Bun dan jangan pernah
  mengirimnya ke API. Kolom ini hanya untuk pencarian.
- Index memakai **FTS5 trigram external content**, jadi teks tidak disimpan dua
  kali:

```sql
ALTER TABLE items ADD COLUMN search_text TEXT
    GENERATED ALWAYS AS (kode) VIRTUAL;

CREATE VIRTUAL TABLE items_fts USING fts5(
    search_text,
    content='items',
    content_rowid='id',
    tokenize='trigram'
);
```

- Sinkronkan lewat trigger `AFTER INSERT`, `AFTER DELETE`, dan `AFTER UPDATE`
  dengan `WHEN old.search_text IS NOT new.search_text`. Isi index untuk baris
  lama dengan `INSERT INTO items_fts (items_fts) VALUES ('rebuild')`.
- Query: `JOIN items_fts AS f ON f.rowid = i.id` lalu
  `WHERE f.search_text MATCH ?`. Tetap filter `deleted_at IS NULL` di tabel
  utama; baris soft delete boleh tertinggal di index.
- Trigram hanya cocok untuk **minimal 3 karakter**. Query 1–2 karakter
  **ditolak** (`422`), bukan fallback scan; query kosong berarti tanpa filter.
- Bungkus input user sebagai frasa FTS (`"..."`, escape `"` menjadi `""`) agar
  operator `AND`/`OR`/`NOT`/`*` diperlakukan sebagai teks biasa.

### Migration (goose)

- Semua perubahan skema lewat file migration goose.
- Selama masih **dev** dan belum rilis, migration boleh diedit langsung; hapus
  database lokal (`persistence/db/app.db*`) lalu jalankan ulang. Setelah rilis,
  jangan pernah mengubah migration yang sudah dijalankan — buat migration baru.
- Format file: `<timestamp>_<nama>.sql` (hasil `goose create`).
- Wajib punya blok `-- +goose Up` dan `-- +goose Down`.
- Migration harus idempoten terhadap urutan eksekusi dari awal.
- Trigger atau blok multi-statement wajib dibungkus
  `-- +goose StatementBegin` / `-- +goose StatementEnd`; tanpa itu goose
  memecah statement di dalam `BEGIN ... END`.
- Perhatikan batasan SQLite: `ALTER TABLE` terbatas, dan
  `ADD COLUMN ... NOT NULL REFERENCES` tidak didukung. Untuk perubahan seperti
  itu, buat tabel baru lalu salin data di dalam migration. `ADD COLUMN`
  generated `VIRTUAL` didukung dan tidak menulis ulang tabel.

### API

- Response sukses: `{"data": ...}`.
- Response error: `{"error": "pesan"}` dengan status HTTP yang sesuai
  (`400`, `401`, `403`, `404`, `409`, `422`, `500`).
- Jangan expose kolom audit (`created_*`, `updated_*`, `deleted_*`) di response
  maupun request. Gunakan DTO terpisah atau `json:"-"`.
- Jangan bocorkan detail error internal ke klien; log di server.
- Batasi ukuran body request.
- Pencarian lewat query param `?q=`: trim input, minimal 3 karakter dan
  maksimal 100 karakter (di luar itu `422`), dan jangan expose `search_text` di
  response.
- Tanggal di DTO memakai `DD-MM-YYYY` (format Indonesia). Request menerima
  `DD-MM-YYYY` maupun `YYYY-MM-DD`; penyimpanan tetap `YYYY-MM-DD`.
- Jangan expose kolom internal apa adanya. Boolean/enum dipetakan ke label
  Indonesia di service, mis. `is_male` → `jenis_kelamin`: `Pria`/`Wanita`.

### Testing

- Test memakai database sementara, jangan menyentuh database runtime.
- Setiap endpoint baru punya test happy path + kasus error.
- Test tidak boleh bergantung pada state global atau urutan eksekusi.

## Jangan commit

- File database runtime (`*.db`, `*.db-shm`, `*.db-wal`).
- Binary hasil build (`bin/`).
- File `.env` atau secret.
- Perubahan editor/IDE yang tidak relevan.
