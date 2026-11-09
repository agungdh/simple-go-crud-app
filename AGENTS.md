# AGENTS.md

Panduan untuk agent/kontributor yang bekerja di repo ini.

## Ringkasan proyek

Aplikasi Go + AdminLTE 2 untuk CRUD **pegawai** dan **bidang** dengan
arsitektur **multi-page** (list, tambah, edit punya route/halaman
masing-masing). Setiap pegawai **wajib** punya bidang (FK ke tabel `bidang`).
Halaman dirender server-side dengan `html/template` sebagai **shell HTML tanpa
data**; seluruh data diambil browser lewat `fetch()` ke endpoint JSON
(`datatable` untuk tabel dan `show` untuk form edit). Submit form juga memakai
`fetch()`. Database SQLite diakses lewat ORM Bun, dan migration memakai goose.

## Stack & dependensi

- Go 1.27, router `go-chi/chi/v5`.
- SQLite pure Go: `modernc.org/sqlite` via `uptrace/bun/driver/sqliteshim`.
- ORM: `github.com/uptrace/bun` + `bun/dialect/sqlitedialect`.
- Migration: `github.com/pressly/goose/v3`, file SQL di
  `internal/database/migrations/`, di-embed dan dijalankan otomatis saat start.
- Tabel daftar memakai DataTables 1.13 server-side processing + integrasi
  Bootstrap 3, aset lokal di `web/static/plugins/datatables/` (tanpa CDN).
- Select2 4.0.13 untuk autocomplete bidang, aset lokal di
  `web/static/plugins/select2/` (tanpa CDN).
- Alpine.js 3.17.5 untuk state form dan submit `fetch()`, aset lokal di
  `web/static/plugins/alpinejs/` (tanpa CDN).
- Template & aset di-embed dengan `go:embed` (`web/assets.go`).

Jangan menambah driver SQLite berbasis CGO; build harus tetap jalan dengan
`CGO_ENABLED=0`.

## Perintah penting

```bash
make run                 # go run . (auto-migrate, default :8080, db app.db)
make run ARGS="-addr :3000 -db app.db"
make test                # go test ./...
make vet                 # go vet ./...
make build               # bin/adminlte-demo
make build-prod          # CGO_ENABLED=0, stripped
make migrate             # opsional: goose CLI
```

Selalu jalankan `gofmt`, `go vet ./...`, dan `go test ./...` sebelum selesai.

## Struktur & tanggung jawab

| Path | Isi |
| --- | --- |
| `main.go` | flag, buka DB, migrate, wiring handler, start server |
| `internal/pages/pages.go` | definisi `Page`; `Pages` untuk halaman statis, `Pegawai`/`Bidang` untuk halaman dinamis |
| `internal/render/render.go` | parse/cache template per layout+page, `Render`/`RenderData` |
| `internal/router/router.go` | mount aset, halaman, route resource pegawai & bidang, dan 404 |
| `internal/handler/pegawai.go` | handler resource pegawai (Index/Create/Store/Show/Edit/Update/Destroy + Datatable), validasi request, mapping error |
| `internal/handler/bidang.go` | handler resource bidang (termasuk Show dan endpoint autocomplete select2) |
| `internal/repository/pegawai.go` | query Bun pegawai (JOIN bidang), error domain (`ErrNotFound`, `ErrDuplicateNIP`) |
| `internal/repository/bidang.go` | query Bun bidang, error domain (`ErrNotFound`, `ErrDuplicateKode`, `ErrInUse`) |
| `internal/model/pegawai.go` | entity pegawai + tag Bun/JSON (termasuk `bidang_id`, `bidang_nama`) |
| `internal/model/bidang.go` | entity bidang + tag Bun/JSON |
| `internal/database/` | koneksi SQLite, runner goose, file migration |
| `web/templates/` | layout, partial, halaman |
| `web/static/` | aset AdminLTE + DataTables + Select2 + Alpine.js; disajikan dari `/dist`, `/bower_components`, `/plugins` |

## Konvensi wajib

### Routing (gaya resource Laravel)

- Nama aksi handler mengikuti resource controller Laravel: `Index`, `Create`,
  `Store`, `Edit`, `Update`, `Destroy`.
- Route mengikuti pola resource, **tanpa prefix `/api`**:
  - `GET /pegawai` → `Index` (halaman tabel; DataTables memuat baris via
    `GET /pegawai/datatable`).
  - `GET /pegawai/create` → `Create` (halaman form tambah).
  - `POST /pegawai` → `Store` (API JSON, kembalikan `201` + data).
  - `GET /pegawai/{id}` → `Show` (API JSON, `{"data": ...}`; dipakai form
    edit untuk mengisi field).
  - `GET /pegawai/{id}/edit` → `Edit` (halaman form edit, tanpa data).
  - `PUT /pegawai/{id}` → `Update` (API JSON).
  - `DELETE /pegawai/{id}` → `Destroy` (API JSON, `204`).
  - Modul bidang mengikuti pola yang sama; tambahan endpoint pendukung:
    `GET /bidang/datatable` dan `GET /bidang/search?q=`.
- **Halaman HTML tidak boleh membaca database.** `Index`, `Create`, dan `Edit`
  hanya merender shell HTML (judul, form kosong, tabel kosong). Data selalu
  diambil browser lewat `fetch()`:
  - tabel daftar memakai `GET /{resource}/datatable`;
  - form edit memakai `GET /{resource}/{id}` (`Show`), lalu mengisi field dan
    Select2 di sisi klien.
- Endpoint `Show` mengembalikan `404` bila data tidak ada; halaman edit tetap
  merender `200` karena halaman itu statis dan error baru muncul saat fetch.
- Route didaftarkan di `internal/router/router.go` memakai konstanta
  `pages.Pegawai.Route` / `pages.Bidang.Route` (bukan string hardcoded), supaya
  path halaman dan path API selalu konsisten.
- Jangan menambah prefix `/api` pada route baru.

### Form submission

- Arsitektur **multi-page**: daftar di `/pegawai`, form tambah di
  `/pegawai/create`, form edit di `/pegawai/{id}/edit`. Modul bidang mengikuti
  pola yang sama (`/bidang`, `/bidang/create`, `/bidang/{id}/edit`). Jangan
  memakai modal untuk CRUD.
- Form edit **tidak boleh diisi data oleh server**. Saat halaman dimuat, JS
  memanggil `fetch()` ke `GET /pegawai/{id}` (endpoint `Show`) lalu mengisi
  input dan Select2 dari `body.data`. Template form hanya berisi field kosong.
- Halaman form memakai Alpine.js (`pegawaiForm` / `bidangForm`) untuk state dan
  submit. Tetap gunakan `data-action`/`data-method` pada `<form>` sebagai
  sumber URL/metode, dan `x-model` untuk field biasa.
- **Semua submit form harus lewat `fetch()`** ke route resource (`POST /pegawai`
  atau `PUT /pegawai/{id}`) yang mengembalikan JSON. Jangan membuat handler POST
  yang merender HTML.
- Set `Content-Type: application/json` dan kirim body JSON.
- Setelah mutasi berhasil, redirect ke `/pegawai?status=created|updated|deleted`
  (atau `/bidang?status=...`); halaman daftar menampilkan notifikasi dari query
  `status`. Notifikasi ini satu-satunya nilai dinamis yang boleh dirender
  server-side pada halaman daftar; data tabel tetap dari `datatable`.
- Jika API mengembalikan error, tampilkan pesannya di halaman form tanpa reload.
- Link edit memakai `<a href="/pegawai/{id}/edit">`, bukan handler JS.
- Hapus data memakai `fetch()` DELETE ke `/pegawai/{id}` lalu redirect ke daftar;
  tombol aksi memakai event delegation pada `<tbody>`.
- Saat membangun elemen dari data user, gunakan `textContent` (bukan
  `innerHTML`) untuk mencegah XSS.
- Field bidang pada form pegawai memakai Select2 dengan autocomplete
  server-side ke `GET /bidang/search?q=`. Saat memuat data edit, tambahkan
  `<option>` terpilih dari `bidang_id`/`bidang_nama` lalu trigger `change`.
  `<select name="bidang_id">` ikut terkirim lewat `FormData`, tetapi nilainya
  **wajib dikonversi ke Number** sebelum `JSON.stringify` agar bisa di-decode ke
  `int64`.

### Tabel daftar (DataTables)

- Tabel pegawai dan bidang memakai DataTables 1.13 dengan **server-side
  processing**; aset dimuat lokal dari `/plugins/datatables/`.
- Endpoint `GET /pegawai/datatable` dan `GET /bidang/datatable`
  menerima parameter DataTables (`draw`, `start`, `length`, `search[value]`,
  `order[0][column]`, `order[0][dir]`) dan mengembalikan
  `{draw, recordsTotal, recordsFiltered, data}`.
- Pencarian, pengurutan, dan paginasi dilakukan di `repository.Query`, bukan di
  browser. Jangan mengubah tabel kembali ke mode client-side.
- Query pegawai memakai `LEFT JOIN bidang AS b ON b.id = p.bidang_id` dan
  `ColumnExpr("b.nama AS bidang_nama")`. Pencarian pegawai ikut mencocokkan
  `b.kode` dan `b.nama`.
- Kolom order dipetakan lewat whitelist `datatableColumns` /
  `bidangDatatableColumns`; entri berupa ekspresi ber-alias penuh (mis.
  `p.nama`, `b.nama`), bukan nama kolom mentah. Arah hanya `asc`/`desc`.
  Jangan pernah memasukkan input user langsung ke `OrderExpr`.
- Batasi `length` (maksimal 100) dan panjang kata kunci pencarian.
- Pencarian harus **case-insensitive**: pakai `LOWER(kolom) LIKE ?` dan pola
  yang juga di-`strings.ToLower`. Jangan andalkan perilaku default SQLite.
- Jangan membuat index untuk kolom yang dipakai pencarian; index hanya untuk
  FK dan unique (lihat bagian Database & migration).
- Escape wildcard `%`, `_`, dan `\` saat membangun pola `LIKE`.
- Jangan menambahkan baris placeholder "belum ada data" ke `<tbody>`; gunakan
  opsi `language.emptyTable`.
- Kolom aksi diatur `orderable: false, searchable: false`; render tombolnya
  dengan HTML yang di-escape.

### Template & render

- Halaman statis: tambahkan `pages.Page` ke `pages.Pages`; router otomatis
  mendaftarkannya.
- Halaman dinamis: definisikan `pages.Page` terpisah (contoh `pages.Pegawai`,
  `pages.PegawaiCreate`, `pages.PegawaiEdit`, `pages.Bidang`, dst.), mount
  handler-nya secara eksplisit di router, dan render dengan
  `renderer.RenderData`.
- Handler halaman (`Index`/`Create`/`Edit`) **tidak boleh** memanggil
  repository. Data user tidak boleh masuk ke template; jangan menambahkan field
  seperti `Item` pada data render. Form edit diisi lewat `fetch()` ke `Show`.
- Form tambah dan edit berbagi satu template per modul
  (`pegawai_form.html` / `bidang_form.html`); bedakan lewat `Method`
  (`POST`/`PUT`) dan `FormAction` pada page data. `FormAction` untuk edit adalah
  `/{resource}/{id}`, sekaligus URL yang di-`fetch()` untuk memuat data.
- Data untuk `RenderData` harus struct yang **embed `pages.Page`**, karena
  layout/partial membaca `.Title`, `.Active`, `.Sidebar`, dan `.BodyClass`.
- Partial baru harus didaftarkan di `Renderer.partials()` agar ikut di-parse.
- CSS/JS khusus halaman diletakkan di blok `page-css`/`page-js`
  (jangan tambah script global di layout). Select2 dan Alpine.js dimuat di
  halaman form.
- Select2 men-trigger event jQuery `change`, bukan native event; sinkronkan
  state Alpine lewat handler jQuery pada elemen `<select>`.
- Template dan aset di-embed: perubahan memerlukan restart/build ulang.

### Database & migration

- Semua perubahan skema lewat file goose baru di
  `internal/database/migrations/` dengan format `00002_nama_perubahan.sql`,
  berisi `-- +goose Up` dan `-- +goose Down`.
- Jangan mengubah migration yang sudah pernah dijalankan; buat file baru.
- Baseline skema: `00001_create_bidang.sql` membuat tabel `bidang`, lalu
  `00002_create_pegawai.sql` membuat `pegawai` dengan `bidang_id` NOT NULL dan
  FK ke `bidang(id)`. Urutan ini wajib (bidang harus ada lebih dulu).
  `pegawai.bidang_id` tidak bisa ditambahkan lewat `ALTER TABLE` karena SQLite
  melarang `ADD COLUMN ... NOT NULL REFERENCES`; ubah lewat migration rebuild
  baru bila perlu.
- Index hanya untuk FK (`idx_pegawai_bidang_id`) dan constraint unik
  (`pegawai.nip`, `bidang.kode`). Jangan membuat index untuk kolom pencarian
  (`nama`, dll); pencarian tetap memakai `LOWER(...) LIKE`.
- Akses DB hanya lewat repository; handler tidak boleh menulis query Bun.
- Timestamp disimpan UTC (`time.Now().UTC()`), field `created_at`/`updated_at`
  diisi otomatis.
- SQLite hanya mendukung satu penulis. Connection pool dibatasi kecil
  (`maxOpenConns`/`maxIdleConns` = 4, lihat `internal/database/database.go`)
  karena target deploy RAM 512 MB / 1 vCPU; WAL memungkinkan pembaca paralel,
  penulis diserialisasi SQLite dan di-backoff lewat `busy_timeout`. Jangan
  menaikkan pool tanpa mengukur memori dan kontensi tulis.
- NIP unik: tangani duplikat sebagai `ErrDuplicateNIP` (HTTP 409), bukan 500.
- Kode bidang unik: tangani duplikat sebagai `ErrDuplicateKode` (HTTP 409).
- Hapus bidang yang masih dipakai pegawai akan gagal FK; petakan ke `ErrInUse`
  (HTTP 409), bukan 500.

### API & error

- Sukses: `{"data": ...}` untuk `Show`, `Store`, `Update`, `datatable`, dan
  `search`; `Destroy` mengembalikan `204` tanpa body.
- Error: `{"error": "pesan"}` dengan status yang sesuai: `400`, `404`, `409`,
  `422`, `500`.
- `GET /{resource}/{id}` (`Show`) mengembalikan `400` untuk ID tidak valid dan
  `404` bila data tidak ada.
- Trim input dan validasi di handler. Field wajib pegawai: `nip`, `nama`,
  `bidang_id`; email harus valid jika diisi. Field wajib bidang: `kode`, `nama`.
- Endpoint pendukung: `GET /bidang/search?q=` mengembalikan
  `{"data": [{id, kode, nama}]}` untuk Select2 (limit 20).
- Gunakan `http.MaxBytesReader` untuk membatasi body request.
- Jangan bocorkan error internal ke klien; log detailnya di server.

### Gaya kode

- Ikuti pola package yang sudah ada; komentar kode dalam bahasa Inggris,
  teks UI/pesan error untuk pengguna dalam bahasa Indonesia.
- Gunakan `fmt.Errorf("...: %w", err)` untuk membungkus error.
- Pertahankan konsistensi penamaan: `Pegawai`, `pegawai` (bukan employee).

## Testing

- Test router ada di `internal/router/router_test.go` dan memakai SQLite
  sementara (`t.TempDir()`), jadi tidak menyentuh `app.db`.
- Setiap fitur/endpoint baru harus punya test happy path + validasi/error.
- Test form edit memverifikasi halaman tetap `200` tanpa data ter-render, lalu
  memanggil endpoint `Show` untuk data, `404` saat tidak ada, dan `400` saat ID
  tidak valid.
- Test pegawai harus membuat bidang lebih dulu (helper `seedBidang`) dan
  menyertakan `bidang_id` pada payload.
- Jalankan `go test ./...`; test tidak boleh bergantung pada state global atau
  urutan eksekusi.

## Jangan commit

- File database runtime (`*.db`, `*.db-shm`, `*.db-wal`).
- Binary hasil build (`bin/`).
- Perubahan `.idea/` yang tidak relevan.
