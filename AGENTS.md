# AGENTS.md

Panduan untuk agent yang bekerja di repo ini.

## TL;DR

| Topik | Intinya |
| --- | --- |
| Stack | Go + chi + Bun + SQLite (`modernc.org/sqlite`) + goose. Jangan ganti. |
| Status | Kontrak, domain `bidang`/`pegawai`/`auth`, router, dan autentikasi sudah ada; server HTTP aktif di `:8080` |
| Auth | Access token JWT + refresh token buram; role `admin` (baca/tulis) dan `user` (baca saja); semua endpoint data wajib login |
| Cek cepat | `make help` (daftar target), `make check` (wajib sebelum selesai), `make api-validate` (Scalar, butuh Node.js) untuk kontrak |
| DB | `persistence/db/app.db`, WAL, `synchronous=NORMAL`, `busy_timeout`, `foreign_keys=ON`, pool kecil |
| Migration | `make migrate-create NAME=nama` → isi Up/Down → `make migrate-up` |
| Domain | `main.go` wiring + `internal/router`; tiap domain (`auth`, `bidang`, `pegawai`) satu paket berisi model, repository, service, validation, DTO, dan handler; paket shared ada di `internal/shared/` (`entity`, `httpx`, `search`, `apperr`, `database`, `testutil`) |
| Kontrak | `api/openapi.yaml` ditulis **dulu**, baru implementasi; tanpa generator/dependency OpenAPI di Go; DTO tangan di `dto.go` paket domain; `make api-validate` (Scalar) |
| ID | `INTEGER PRIMARY KEY AUTOINCREMENT`, di Go `int64` |
| Waktu | Audit `*_at` = epoch **ms UTC**; tanggal/waktu bisnis = `TEXT` ISO-8601 (`YYYY-MM-DD`) |
| Audit | Wajib `entity.Base`; isi via `MarkCreated/MarkUpdated/MarkDeleted`; jangan expose ke API |
| Soft delete | `entity.Active(q)` + `UPDATE deleted_at/deleted_by`; unik = partial index `WHERE deleted_at IS NULL` |
| Query | **Jangan `SELECT *`**; pakai `Column(entity.PublicColumns(...)...)` |
| Cari | `search_text` (VIRTUAL) + FTS5 trigram `<tabel>_fts`; `?q=` wajib ≥3 karakter (kurang → `422`) |
| FK | `ON UPDATE RESTRICT ON DELETE RESTRICT`, tipe `BIGINT`, beri index |
| Error | Hanya `400`, `401`, `403`, `404`, `422`, `500`; `400`/`401`/`403`/`404`/`500` → `{"error": "..."}`, `422` → `{"error": {"validations": {"field": ["..."]}}}`; validasi termasuk unik/FK → `422`, error tak terduga → `500` |
| Test | DB sementara; tiap endpoint: happy path + error |
| Jangan commit | `*.db*`, `bin/`, `.env`, file IDE |

## Proyek

Project **Go** sederhana. Gunakan stack yang sudah ditentukan; jangan ganti
tanpa alasan kuat.

Urutan kerja wajib **spec-first**: tulis atau ubah `api/openapi.yaml` dulu,
review sampai bentuknya stabil, baru tulis DTO tangan di `dto.go` paket domain,
lalu implementasi di paket domain yang sama (`repository.go`, `service.go`,
`validation.go`, `handler.go`) dan daftarkan route di `internal/router`. Jangan
menulis implementasi untuk endpoint yang belum ada di kontrak, dan jangan
mengubah kontrak supaya cocok dengan kode yang sudah terlanjur dibuat.

Kontrak **ditulis tangan**. Tidak ada generator OpenAPI di repo ini: jangan
menambah `oapi-codegen`, `openapi-generator`, `swag`, `huma`, `go:generate`
untuk kontrak, atau file `*.gen.go`. `main.go` adalah composition root: buka
database, jalankan migration, lalu menyalakan server HTTP lewat `router.New`.

## Stack

- Bahasa: **Go**
- Router: **`github.com/go-chi/chi/v5`**
- Kontrak: **OpenAPI 3.1** di `api/openapi.yaml`, ditulis tangan lebih dulu; tanpa generator/codegen
- Database: **SQLite3** (satu-satunya database), driver **`modernc.org/sqlite`** (pure Go, tanpa CGO)
- ORM: **`github.com/uptrace/bun`** + **`bun/dialect/sqlitedialect`**
- Migration: **`github.com/pressly/goose/v3`**
- Docs UI: **Scalar CLI** lewat `npx` (butuh Node.js) untuk `make docs`,
  `make mock`, dan `make api-validate`; Scalar mendukung OpenAPI 3.1 dan panel
  "Test Request". Scalar hanya untuk memvalidasi, melihat, dan mem-mock
  kontrak, bukan generator spec atau kode.

Buka koneksi dengan `sql.Open("sqlite", dsn)`. Jangan memakai
`bun/driver/sqliteshim` atau `github.com/mattn/go-sqlite3`; keduanya menambah
ketergantungan yang tidak perlu dan `mattn` butuh CGO.

## Perintah

Semua perintah harian lewat `make`; jalankan `make` atau `make help` untuk
melihat daftar target.

### Build, test, dan tooling

| Perintah | Ekuivalen manual | Fungsi |
| --- | --- | --- |
| `make run` | `CORS_ORIGINS=http://localhost:8082 ADDR=:8080 DB_PATH=persistence/db/app.db JWT_SECRET=... ADMIN_PASSWORD=... go run .` | Buka database + migration, lalu jalankan server HTTP |
| `make build` | `go build -trimpath -o bin/simple-go-crud-app .` | Build binary ke `bin/` |
| `make test` | `go test ./...` | Jalankan semua test |
| `make vet` | `go vet ./...` | Jalankan `go vet` |
| `make fmt` | `gofmt -w .` | Format semua file Go |
| `make fmt-check` | `gofmt -l .` (harus kosong) | Cek format tanpa mengubah file |
| `make tidy` | `go mod tidy` | Rapikan `go.mod` dan `go.sum` |
| `make api-validate` | `npx @scalar/cli document validate api/openapi.yaml` | Validasi kontrak OpenAPI (butuh Node.js) |
| `make docs` | `npx @scalar/cli document serve api/openapi.yaml --config api/scalar.config.json -p 8082` | Preview kontrak + "Test Request" di <http://localhost:8082> (butuh Node.js) |
| `make mock` | `npx @scalar/cli document mock api/openapi.yaml -p 8081` | Mock server dari kontrak di <http://127.0.0.1:8081> |
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
| `DB_PATH` | `persistence/db/app.db` | `make run`, target migration |
| `ADDR` | `:8080` | `make run` |
| `CORS_ORIGINS` | `http://localhost:8082` | `make run` |
| `JWT_SECRET` | — (wajib, minimal 32 byte) | `make run`, server |
| `ACCESS_TOKEN_TTL` | `15m` | `make run`, server |
| `REFRESH_TOKEN_TTL` | `168h` | `make run`, server |
| `ADMIN_USERNAME` | `admin` | seed admin pertama saat startup |
| `ADMIN_PASSWORD` | — (wajib saat belum ada user aktif) | seed admin pertama saat startup |
| `ADMIN_NAME` | `Administrator` | seed admin pertama saat startup |
| `NAME` | — (wajib) | `make migrate-create` |
| `GOOSE` | `go run -tags '<GOOSE_TAGS>' github.com/pressly/goose/v3/cmd/goose@v3.30.0` | Target migration |
| `DOCS_PORT` | `8082` | `make docs` |
| `MOCK_PORT` | `8081` | `make mock` |

Selalu jalankan `make check` (fmt, vet, test) sebelum selesai.

## Struktur

- `main.go` — entrypoint; buka database, jalankan migration, dan nyalakan
  server HTTP (`ADDR`, default `:8080`) dengan graceful shutdown.
- `internal/` — kode aplikasi, dipisah antara **paket bisnis per domain** dan
  **paket shared**:
  - `auth/` — domain autentikasi: user, refresh token, JWT, password, middleware,
    service login/refresh/logout, handler, dan DTO.
  - `bidang/` — domain bidang: model, repository, service, validation, DTO, handler.
  - `pegawai/` — domain pegawai: model, repository, service, validation, date,
    DTO, handler.
  - `router/` — komposisi route, CORS, recoverer; satu-satunya tempat yang
    merangkai semua domain.
  - `shared/` — kode yang dipakai lintas domain:
    - `entity/` — `Base` (PK + kolom audit), `Active`, `PublicColumns`, `NowMillis`.
    - `httpx/` — envelope JSON, decode body, mapping error → status, parse `{id}`.
    - `search/` — `Normalize(q)` dan `Phrase(q)` untuk FTS5 trigram.
    - `apperr/` — sentinel error dan `FieldErrors`.
    - `database/` — koneksi SQLite, pragma, dan runner migration.
    - `testutil/` — helper test.
- `api/` — kontrak OpenAPI (`openapi.yaml`); sumber kebenaran bentuk API.
  `api/scalar.config.json` hanya mengatur UI Scalar (skema auth global dan
  persistensi token), bukan bagian dari kontrak.
- DTO ditulis tangan di `dto.go` paket domain setelah kontrak stabil; tidak ada
  kode generated dan tidak ada generator.
- `migrations/` — file migration goose.
- `persistence/db/` — database runtime (tidak di-commit); hanya `.gitkeep`.
- `Makefile` — target perintah harian; sumber kebenaran daftar perintah.
- `compose.yml` — sqlite-web untuk inspeksi database.
- Tanggung jawab tetap dipisah per file walau satu paket: `repository.go` hanya
  menulis query, `service.go` hanya aturan bisnis, `handler.go` hanya HTTP ↔
  DTO ↔ service. Handler tidak menulis query; repository tidak menulis response
  HTTP.

## Konvensi

### Umum

- Ikuti pola package yang sudah ada; jangan bikin abstraksi baru kalau belum
  perlu.
- Jangan membuat paket layer baru (`model`, `repository`, `service`, `handler`,
  `dto`) di root `internal/`. Kode baru masuk ke paket domainnya; helper yang
  dipakai lintas domain masuk ke `internal/shared/` (`entity`, `httpx`, `search`,
  `apperr`).
- Gunakan `context.Context` untuk operasi DB dan request.
- Bungkus error dengan `fmt.Errorf("...: %w", err)`.
- Jangan menelan error. Kembalikan atau log dengan konteks.
- Komentar kode dalam bahasa Inggris; teks untuk pengguna dalam bahasa
  Indonesia.

### HTTP & router (chi)

- Daftarkan route di satu tempat (`internal/router`), kelompokkan per resource.
- Gunakan middleware chi untuk logging, recovery, dan hal lintas-cutting.
  Middleware autentikasi ada di `internal/auth` (`RequireAuth`, `RequireAdmin`).
- Path parameter dibaca lewat `httpx.ParseID` (di baliknya `chi.URLParam`).
- Handler hanya menerjemahkan HTTP ↔ DTO ↔ service domainnya; tidak menulis
  query dan tidak menyimpan aturan bisnis.
- Decode body di handler lewat `httpx.ReadObject`: tepat satu objek JSON,
  maksimal 1 MiB; body yang tidak terbaca dibalas `400`, isi yang tidak valid
  (field asing, tipe salah) `422`.
- Validasi isi (wajib, panjang, tanggal, enum, `q`) ada di `validation.go` paket
  domain dan mengembalikan `apperr.FieldErrors`; aturan pencarian memakai
  `search.Normalize`.
- CORS dibatasi ke origin eksplisit dari `CORS_ORIGINS` (default Scalar docs
  di `:8082`); preflight `OPTIONS` dijawab `204`.

### Database (Bun + SQLite3)

- Database hanya **SQLite3**. Jangan menambah database lain.
- Semua query Bun berada di `repository.go` paket domain.
- Gunakan model dengan tag `bun` dan `json`.
- Pakai `context` pada setiap query.
- Tangani `sql.ErrNoRows` sebagai not-found di repository.
- Aktifkan `PRAGMA foreign_keys = ON` di setiap koneksi.
- Gunakan `PRAGMA journal_mode = WAL` dan `busy_timeout` untuk menghindari
  `database is locked`.
- Gunakan `PRAGMA synchronous = NORMAL`; dengan WAL ini tetap aman terhadap
  korupsi, hanya transaksi terakhir yang bisa hilang saat listrik mati.
- Pragmas dipasang lewat DSN agar berlaku di **setiap** koneksi pool, bukan
  hanya koneksi pertama.
- SQLite hanya punya satu penulis, tapi WAL mengizinkan pembaca berjalan
  bersamaan. Jangan pin pool ke 1 koneksi; pakai pool kecil dan biarkan
  `busy_timeout` yang menangani penulis yang menunggu.
- **Hanya kolom audit** `*_at` yang menyimpan **epoch milliseconds UTC**
  (`time.Now().UTC().UnixMilli()`), bukan `TEXT`.
- Kolom tanggal/waktu **bisnis** (mis. `tanggal_lahir`, `jadwal`) disimpan
  sebagai **`TEXT` ISO-8601**: `YYYY-MM-DD` untuk tanggal, `YYYY-MM-DDTHH:MM:SSZ`
  untuk waktu. Jangan memakai epoch milliseconds di luar kolom audit.
- Pelanggaran `UNIQUE` dan `FOREIGN KEY` yang berasal dari input wajib dicek
  lebih dulu di service dan dibalas `422`, bukan `409`. Constraint tetap ada
  sebagai pertahanan terakhir; error yang tidak bisa dicek dari input (mis.
  hard delete parent yang masih direferensikan) dibalas `500`.

#### Select kolom

- **Jangan `SELECT *`.** Selalu pilih kolom eksplisit dengan `.Column(...)`.
- Default: semua kolom publishable (semua kecuali audit) lewat
  `entity.PublicColumns(db, (*Entity)(nil))`.
- Pilih hanya kolom yang benar-benar dipakai response; jangan ikutkan kolom
  audit.
- Untuk JOIN atau query tabel langsung, tulis kolom eksplisit dengan prefix
  alias (`p.id`, `p.nama`, dst.).
- Kolom audit tetap boleh dipakai di klausa `WHERE` (mis. `deleted_at IS NULL`)
  meskipun tidak di-`SELECT`.

```go
q := db.NewSelect().
    Model(&items).
    Column(entity.PublicColumns(db, (*Item)(nil))...)
q = entity.Active(q)
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
- Semua entity **bisnis wajib embed `entity.Base`** (`internal/shared/entity/base.go`);
  jangan mendeklarasikan ulang `id` atau kolom audit. Tabel teknis tanpa relasi
  bisnis (mis. `refresh_token`) dikecualikan: cukup `id`, kolom teknis, dan
  `created_at`; tidak perlu soft delete/audit lengkap.
- Isi audit lewat `MarkCreated(actorID)`, `MarkUpdated(actorID)`, dan
  `MarkDeleted(actorID)`. Jangan mengisi manual.
- Filter baca wajib memakai `entity.Active(q)`; untuk query JOIN, filter juga
  tabel yang di-join (`p.deleted_at IS NULL`, `b.deleted_at IS NULL`, dst.).
- Jangan memakai tag `bun:",soft_delete"` bawaan Bun: implementasinya menyimpan
  `UnixNano`, bukan epoch milliseconds.
- Jangan memanggil `db.NewDelete()` untuk soft delete. Lakukan `UPDATE` yang
  mengisi `deleted_at`/`deleted_by` lewat `MarkDeleted`.
- Saat insert: isi `created_at` dan `created_by`; kolom `updated_*`/`deleted_*`
  tetap `NULL`.
- Saat update: isi `updated_at` dan `updated_by`.
- Soft delete: isi `deleted_at` dan `deleted_by`; **jangan** `DELETE` fisik.
- Pengecualian tabel teknis: token/sesi seperti `refresh_token` dicabut dengan
  `DELETE` fisik, bukan soft delete, dan tidak memakai kolom audit lengkap.
- Query baca default wajib menambahkan `WHERE deleted_at IS NULL`.
- Di model Bun, kolom nullable dipetakan ke pointer (`*int64`), bukan
  `time.Time`.

#### Foreign key

- Semua FK wajib eksplisit: **`ON UPDATE RESTRICT ON DELETE RESTRICT`**.
- Kolom FK bertipe `BIGINT` (`int64`).
- Aktifkan `PRAGMA foreign_keys = ON`; tanpa ini FK tidak ditegakkan SQLite.
- Hard delete parent yang masih direferensikan akan gagal; itu error
  pasca-validasi, jadi dibalas `500`.
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

Handler HTTP tiap domain sudah ada; kontrak dan DTO menjadi acuannya (lihat
subbagian berikut). Aturan di bawah ini dipegang handler.

- Response `GET` sukses: `{"data": ...}`.
- `POST` sukses `201` tanpa body, `PUT` sukses `200` tanpa body; data yang
  dibuat/diubah tidak dikembalikan sampai ada kebutuhan. `DELETE` membalas
  `204` tanpa body.
- Error non-validasi (`400`, `401`, `403`, `404`, `500`):
  `{"error": "pesan"}`. Status **error** yang boleh dipakai hanya `400`, `401`,
  `403`, `404`, `422`, dan `500`.
- Error validasi (`422`):
  `{"error": {"validations": {"field": ["pesan", ...]}}}`. Key adalah nama
  field JSON; value adalah daftar pesan, minimal satu. Kumpulkan semua error
  yang bisa dideteksi dan kelompokkan per field, jangan berhenti di error
  pertama. Field yang valid tidak muncul.
- `400` untuk body yang tidak terbaca: JSON rusak, bukan satu objek JSON, lebih
  dari satu nilai, body kosong, atau body > 1 MiB.
- `422` untuk body yang terbaca tetapi isinya tidak valid: tipe field salah,
  field tidak dikenal, `id` bukan bilangan bulat positif, field wajib kosong,
  kepanjangan, tanggal tidak valid, `q` di luar 3–100 karakter, nama bidang
  atau NIP yang sudah dipakai, atau `bidang_id` yang tidak menunjuk bidang
  aktif.
- Keunikan dan referensi yang bergantung pada input wajib dicek lebih dulu di
  service (nama bidang unik, NIP unik di antara pegawai aktif, `bidang_id`
  harus bidang aktif) dan dibalas `422`. Setelah validasi lolos, kegagalan tak
  terduga atau error database tetap dibalas `500`.
- `404` hanya untuk `GET`/`PUT`/`DELETE` by id saat data tidak ada atau sudah
  dihapus (soft delete). Endpoint koleksi tidak memakai `404`.
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
  Indonesia saat membentuk DTO, mis. `is_male` → `jenis_kelamin`:
  `Pria`/`Wanita`.

#### Autentikasi

- Endpoint publik hanya `POST /auth/login` dan `POST /auth/refresh`; endpoint
  lain wajib memakai `Authorization: Bearer <access_token>`.
- Access token adalah JWT HS256 berumur pendek (`ACCESS_TOKEN_TTL`, default
  `15m`). Refresh token buram berumur panjang (`REFRESH_TOKEN_TTL`, default
  `168h`), disimpan sebagai hash SHA-256 di `refresh_token`, dan dirotasi pada
  setiap refresh.
- Login gagal, access token tidak valid/kedaluwarsa, dan refresh token
  dicabut/kedaluwarsa dibalas `401`. Token valid tetapi role tidak berhak
  dibalas `403`.
- Role `admin` boleh membaca dan menulis; role `user` hanya boleh `GET`.
- Logout mencabut refresh token yang dikirim. Access token tetap valid sampai
  `exp` karena JWT stateless.
- Password disimpan sebagai hash bcrypt dan tidak pernah masuk response/log.
- Untuk mencoba endpoint lewat Scalar: jalankan `make run` lalu `make docs`,
  panggil `POST /auth/login`, salin `access_token`, dan tempel ke skema
  `bearerAuth`. Refresh token hanya dipakai untuk refresh/logout. Contoh
  kredensial dev ada di `Makefile` (`ADMIN_USERNAME`/`ADMIN_PASSWORD`).
- Auth global di Scalar berasal dari `security` di root `api/openapi.yaml` dan
  `components.securitySchemes.bearerAuth`. `api/scalar.config.json` memilih
  skema itu secara global dan menyimpan token di browser (`persistAuth`).
- Kolom audit `created_by`/`updated_by`/`deleted_by` pada endpoint tulis diisi
  `id` user dari access token; seed admin pertama dicatat sebagai aksi sistem
  (`NULL`).
- Admin pertama di-seed dari `ADMIN_USERNAME`/`ADMIN_PASSWORD`/`ADMIN_NAME`
  hanya saat belum ada user aktif. Tidak ada endpoint manajemen user; akun role
  `user` ditambahkan di luar API.

#### Kontrak API (contract-first)

Alur wajib untuk setiap perubahan API:

1. Tulis atau ubah `api/openapi.yaml`: path, request, response, status, dan
   aturan validasi.
2. Review kontrak sampai bentuknya disetujui; jangan menulis kode dulu.
3. Tulis atau sesuaikan DTO tangan di `dto.go` paket domain.
4. Jalankan `make api-validate` untuk memastikan kontrak valid (Scalar, butuh
   Node.js).
5. Implementasi di paket domain yang sama (`repository.go`, `service.go`,
   `validation.go`, `handler.go`) dan daftarkan route di `internal/router`.

Kalau implementasi ternyata butuh bentuk berbeda, perbaiki kontrak dulu, baru
kode. Jangan mengubah kontrak hanya agar cocok dengan kode yang sudah ada.

- `api/openapi.yaml` adalah **sumber kebenaran** dan ditulis **lebih dulu**.
- Spec hanya untuk konsumen (FE). Jangan tulis detail internal: nama
  tabel/kolom, index, foreign key, soft delete, FTS/trigram, format
  penyimpanan, atau penyebab error database. Cukup bentuk DTO, aturan validasi
  yang bisa dicek client, dan status yang diterima.
- Alasan error cukup di level yang berguna untuk client (`422` = input tidak
  valid, `404` = data tidak ada, `500` = kesalahan server), bukan nama
  constraint atau tabel.
- Tidak ada generator OpenAPI, dependency OpenAPI di Go, dan tidak ada codegen.
  Jangan menambah `oapi-codegen`, `openapi-generator`, `swag`, `huma`,
  `kin-openapi`, `go:generate` untuk kontrak, atau file `*.gen.go`. DTO ditulis
  tangan di `dto.go` paket domain dan wajib cocok dengan schema di kontrak;
  kecocokannya dicek manual saat review, bukan lewat test/library Go.
- Nama field JSON di DTO harus sama persis dengan kontrak. Untuk tipe yang
  berbeda antara request dan response (mis. `jenis_kelamin`), buat dua tipe
  terpisah, jangan dipaksa satu.
- Handler memakai tipe DTO paket domainnya (`bidang.Input`, `pegawai.Response`,
  dst.), bukan struct anonim atau `map`.
- `make api-validate` dijalankan sebelum merge; target ini butuh Node.js dan
  bukan bagian dari `make check`.
- `make docs` untuk melihat kontrak sebagai dokumentasi di
  <http://localhost:8082>. Scalar hanya untuk memvalidasi, melihat, dan
  mem-mock kontrak, bukan generator spec atau kode.
- Perubahan kontrak yang mengubah bentuk response = perubahan breaking untuk
  FE; naikkan `info.version` dan catat di deskripsi operasi terkait.
- Kontrak wajib mendokumentasikan: envelope `data` untuk `GET`, response tanpa
  body untuk `POST`/`PUT`/`DELETE`, envelope `error` (`validations` di dalamnya
  untuk `422`), status yang benar-benar dipakai endpoint (sukses
  `200`/`201`/`204`, error `400`, `401`, `403`, `404`, `422`, `500`), dan
  larangan mengekspos kolom audit/internal.
- Kontrak tidak boleh memuat status `409` atau `413`; body > 1 MiB dipetakan ke
  `400`.
- Tanggal di kontrak memakai `pattern`, bukan `format: date`, karena DTO
  memakai `DD-MM-YYYY`; request menerima dua format.

### Testing

- Test memakai database sementara, jangan menyentuh database runtime.
- Setiap endpoint baru punya test happy path + kasus error.
- Test tidak boleh bergantung pada state global atau urutan eksekusi.
- Test unit diletakkan di paket domain terkait (mis. `pegawai/date_test.go`);
  test perilaku lintas domain lewat router (`internal/router/*_test.go`).
- Tidak ada test Go khusus kontrak dan tidak ada dependency OpenAPI di Go.
  Validasi kontrak lewat `make api-validate` (Scalar CLI, butuh Node.js).
- Kecocokan DTO dengan kontrak tidak dicek otomatis. Review manual saat
  menambah atau mengubah DTO; nama field JSON harus sama persis dengan schema.
- Tulis test perilaku untuk endpoint (happy path + error) lewat router dengan
  `httptest`; test itu tidak memuat atau memvalidasi `api/openapi.yaml`.

## Jangan commit

- File database runtime (`*.db`, `*.db-shm`, `*.db-wal`).
- Binary hasil build (`bin/`).
- File `.env` atau secret.
- Perubahan editor/IDE yang tidak relevan.
