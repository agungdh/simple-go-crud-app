# AdminLTE 2.4.18 — Go CRUD Pegawai

Aplikasi demo **AdminLTE 2.4.18** dengan Go `html/template`, lengkap dengan CRUD
**pegawai**. Semua aset (CSS, JS, font, gambar, plugin) disajikan secara lokal
oleh server Go — tidak butuh koneksi ke CDN untuk aset utama.

Template dan aset di-embed ke dalam binary memakai `go:embed`, jadi hasil build
berupa **satu file yang bisa langsung dibagikan** tanpa folder `web/`.

## Stack

| Bagian      | Teknologi                                              |
| ----------- | ------------------------------------------------------ |
| Bahasa      | Go 1.27                                                |
| Router      | `github.com/go-chi/chi/v5`                             |
| Database    | SQLite (pure Go, `modernc.org/sqlite`)                 |
| ORM         | `github.com/uptrace/bun` + `sqlitedialect`             |
| Migration   | `github.com/pressly/goose/v3` (embedded, auto-migrate) |
| Template    | `html/template` + `go:embed`                           |
| Tabel       | DataTables 1.13 server-side processing (Bootstrap 3)   |
| Form submit | `fetch()` dari JavaScript, API JSON                    |

Migration dijalankan otomatis saat aplikasi start. File migration tetap
kompatibel dengan goose CLI (`make migrate`).

## Menjalankan

```bash
go run .
```

Atau lewat Makefile (default menampilkan help):

```bash
make              # daftar target
make build        # bin/adminlte-demo
make run          # go run . (bisa: make run ARGS="-addr :3000 -db app.db")
make test         # go test ./...
make vet          # go vet ./...
make migrate      # opsional: goose CLI, butuh `goose` terpasang
make build-prod   # binary statis + stripped untuk distribusi
make clean        # hapus folder bin/
```

Buka <http://localhost:8080>. Root `/` menampilkan halaman Starter, sedangkan
CRUD pegawai ada di <http://localhost:8080/pegawai>.

Opsi yang tersedia:

```bash
go run . -addr :3000          # ganti port
go run . -db data/app.db      # ganti lokasi file SQLite
```

Database default adalah `app.db` di direktori kerja dan dibuat otomatis.
File `*.db`, `*.db-shm`, dan `*.db-wal` sudah di-ignore oleh git.

## CRUD Pegawai

CRUD pegawai memakai **multi-page**: setiap aksi punya halaman sendiri.

- `/pegawai` — tabel daftar pegawai (DataTables server-side: cari, urut, dan
  paginasi diproses di server lewat `/pegawai/datatable`).
- `/pegawai/create` — halaman form tambah.
- `/pegawai/{id}/edit` — halaman form edit dengan data terisi.

Tambah/edit/hapus tetap **disubmit lewat `fetch()` ke endpoint JSON**, bukan
full-page POST. Setelah berhasil, browser diarahkan kembali ke `/pegawai` dengan
query `?status=created|updated|deleted` untuk menampilkan notifikasi. Jika
validasi gagal, pesan error dari API ditampilkan di halaman form tanpa reload.

Field pegawai: `nip` (unik, wajib), `nama` (wajib), `jabatan`, `email`,
`telepon`, `alamat`, serta `created_at`/`updated_at` yang diisi otomatis.

### Route (gaya resource Laravel)

| Method | Path                    | Aksi      | Keterangan                        |
| ------ | ----------------------- | --------- | --------------------------------- |
| GET    | `/pegawai`              | `Index`   | Halaman tabel pegawai             |
| GET    | `/pegawai/datatable`    | `Datatable` | Data server-side untuk DataTables |
| GET    | `/pegawai/create`       | `Create`  | Halaman form tambah               |
| POST   | `/pegawai`              | `Store`   | Tambah pegawai (JSON)             |
| GET    | `/pegawai/{id}/edit`    | `Edit`    | Halaman form edit                 |
| PUT    | `/pegawai/{id}`         | `Update`  | Ubah pegawai (JSON)               |
| DELETE | `/pegawai/{id}`         | `Destroy` | Hapus pegawai (JSON)              |

Modul bidang mengikuti pola yang sama; tambahan `GET /bidang/search?q=` untuk
autocomplete Select2.

Respons sukses memakai bentuk `{"data": ...}` (datatable/detail/mutasi); destroy
mengembalikan `204`. Error memakai `{"error": "pesan"}`. Status yang dipakai:
`200`, `201`, `204`, `400` (JSON/ID tidak valid), `404` (tidak ditemukan), `409`
(NIP/kode duplikat, bidang masih dipakai), `422` (validasi), dan `500`.

Contoh:

```bash
curl -X POST http://localhost:8080/pegawai \
  -H 'Content-Type: application/json' \
  -d '{"nip":"198001012005011001","nama":"Budi Santoso","jabatan":"Staf","bidang_id":1}'

curl 'http://localhost:8080/pegawai/datatable?start=0&length=10'
```

## Build binary mandiri

```bash
CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o adminlte-demo .
./adminlte-demo -addr :8080
```

Binary sudah berisi seluruh template, aset, dan migration; tidak perlu menyalin
folder `web/` ke mesin tujuan. Build untuk platform lain:

```bash
CGO_ENABLED=0 GOOS=linux   GOARCH=amd64 go build -o adminlte-demo-linux .
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o adminlte-demo.exe .
CGO_ENABLED=0 GOOS=darwin  GOARCH=arm64 go build -o adminlte-demo-mac .
```

## Docker & GHCR

`Dockerfile` memakai multi-stage build dan image akhir
`gcr.io/distroless/static-debian13:nonroot`. Karena image berjalan sebagai
non-root, container memakai database di `/tmp/app.db` (ephemeral). Untuk data
persisten, mount volume dan arahkan flag `-db` ke path yang bisa ditulis.

```bash
docker build -t adminlte-demo .
docker run --rm -p 8080:8080 adminlte-demo
```

Build x86-64-v2 secara eksplisit (default di workflow):

```bash
docker build --build-arg GOAMD64=v2 -t adminlte-demo .
```

Workflow `.github/workflows/build.yml` berjalan pada push ke `main`, tag `v*`,
pull request, dan manual (`workflow_dispatch`). Workflow akan:

1. menjalankan `go vet` dan `go test`;
2. membuat binary `linux/amd64` dengan `GOAMD64=v2` sebagai artifact;
3. membangun image `linux/amd64` dan push ke GHCR.

Image tersedia di:

```
ghcr.io/<owner>/<repo>:latest   # push ke main
ghcr.io/<owner>/<repo>:v1.2.3   # tag v1.2.3
ghcr.io/<owner>/<repo>:sha-...  # setiap build
```

Pull dan jalankan:

```bash
docker pull ghcr.io/<owner>/<repo>:latest
docker run --rm -p 8080:8080 ghcr.io/<owner>/<repo>:latest
```

Pull request hanya melakukan build (tanpa push). Package GHCR baru bersifat
private secara default; ubah visibility lewat Settings → Packages jika perlu.

## Struktur

```
main.go                              entrypoint, flag, wiring DB + router
internal/pages/pages.go              registry halaman statis + definisi halaman pegawai
internal/render/render.go            loader/cache template dari embed.FS + template funcs
internal/router/router.go            routing chi, aset embedded, route resource pegawai & bidang, 404
internal/handler/pegawai.go          handler resource pegawai (halaman + JSON API)
internal/repository/pegawai.go       akses data Bun (query, get, create, update, delete)
internal/model/pegawai.go            entity pegawai (tag Bun + JSON)
internal/database/database.go        koneksi SQLite + runner goose (embed)
internal/database/migrations/        file migration goose (*.sql)
web/assets.go                        //go:embed templates + static
web/templates/layouts/               main.html
web/templates/partials/              header, sidebar, footer
web/templates/pages/                 starter, pegawai (list), pegawai_form, examples/{404,500}
web/static/                          dist/, bower_components/, dan plugins/datatables/
```

## Daftar Halaman

- `/starter.html` (root `/` juga menampilkan halaman ini)
- `/pegawai` — daftar pegawai (tabel DataTables: cari, urut, dan paginasi)
- `/pegawai/create` — form tambah pegawai
- `/pegawai/{id}/edit` — form edit pegawai
- `/pages/examples/404.html`
- `/pages/examples/500.html`

Route yang tidak dikenal mengembalikan **HTTP 404** dengan halaman 404 AdminLTE.

## Catatan

- URL aset dan link dinormalisasi ke path absolut (`/dist/...`,
  `/bower_components/...`) sehingga bisa diakses dari route mana pun.
- Template di-parse sekali lalu di-cache; perubahan template/aset/migration
  memerlukan build ulang karena sudah di-embed.
- `html/template` menghapus komentar HTML/JS saat render. Ini tidak mengubah
  tampilan maupun perilaku halaman.
- Koneksi SQLite dibatasi satu koneksi (`SetMaxOpenConns(1)`) untuk menghindari
  error `database is locked` saat ada banyak penulisan bersamaan.
- Lihat `AGENTS.md` untuk panduan kontribusi dan konvensi kode.
