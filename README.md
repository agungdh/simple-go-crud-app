# AdminLTE 2.4.18 — Go Template Demo

Replika UI lengkap **AdminLTE 2.4.18** yang dirender memakai Go `html/template`
dengan satu layout bersama. Semua aset (CSS, JS, font, gambar, plugin) disajikan
secara lokal oleh server Go — tidak butuh koneksi ke CDN untuk aset utama.

Template dan aset di-embed ke dalam binary memakai `go:embed`, jadi hasil build
berupa **satu file yang bisa langsung dibagikan** tanpa folder `web/`.

Cakupan saat ini: **UI/demo statis** (37 halaman). Belum ada database, CRUD,
autentikasi, atau logika bisnis.

## Menjalankan

```bash
go run .
```

Atau lewat Makefile (default menampilkan help):

```bash
make              # daftar target
make build        # bin/adminlte-demo
make run          # go run . (bisa: make run ARGS="-addr :3000")
make build-prod   # binary statis + stripped untuk distribusi
make clean        # hapus folder bin/
```

Buka <http://localhost:8080>. Root `/` langsung menampilkan Dashboard v1.

Opsi yang tersedia:

```bash
go run . -addr :3000     # ganti port
```

Build binary mandiri:

```bash
CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o adminlte-demo .
./adminlte-demo -addr :8080
```

Binary sudah berisi seluruh template dan aset; tidak perlu menyalin folder
`web/` ke mesin tujuan. Build untuk platform lain:

```bash
CGO_ENABLED=0 GOOS=linux   GOARCH=amd64 go build -o adminlte-demo-linux .
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o adminlte-demo.exe .
CGO_ENABLED=0 GOOS=darwin  GOARCH=arm64 go build -o adminlte-demo-mac .
```

## Docker & GHCR

`Dockerfile` memakai multi-stage build dan image akhir
`gcr.io/distroless/static-debian13:nonroot`.

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
main.go                     entrypoint + flag
internal/pages/pages.go     registry 37 halaman (route, layout, title, sidebar, dll.)
internal/render/render.go   loader/cache template dari embed.FS + template funcs
internal/router/router.go   routing chi, mount aset embedded, handler 404
web/assets.go               //go:embed templates + static
web/templates/layouts/      main.html, topnav.html, standalone.html
web/templates/partials/     header (default/compact/starter), sidebar, control-sidebar, footer
web/templates/pages/        konten tiap halaman
web/static/                 dist/, plugins/, bower_components/ (salinan AdminLTE)
```

Tiga keluarga layout:

- `main` — sidebar + header + control sidebar (mayoritas halaman)
- `topnav` — navigasi atas tanpa sidebar (`pages/layout/top-nav.html`)
- `standalone` — login, register, lockscreen, invoice-print

## Daftar Halaman

Dashboard & starter:

- `/index.html`, `/index2.html`, `/starter.html`

Pages:

- `/pages/calendar.html`, `/pages/widgets.html`
- `/pages/UI/{general,icons,buttons,sliders,timeline,modals}.html`
- `/pages/forms/{general,advanced,editors}.html`
- `/pages/tables/{simple,data}.html`
- `/pages/charts/{chartjs,morris,flot,inline}.html`
- `/pages/mailbox/{mailbox,compose,read-mail}.html`
- `/pages/examples/{invoice,invoice-print,profile,login,register,lockscreen,404,500,blank,pace}.html`
- `/pages/layout/{top-nav,boxed,fixed,collapsed-sidebar}.html`

Route yang tidak dikenal mengembalikan **HTTP 404** dengan halaman 404 AdminLTE.

## Catatan

- URL aset dan link dinormalisasi ke path absolut (`/dist/...`, `/plugins/...`,
  `/bower_components/...`) sehingga bisa diakses dari route mana pun.
- Template di-parse sekali lalu di-cache; perubahan template/aset memerlukan
  build ulang karena sudah di-embed.
- `html/template` menghapus komentar HTML/JS saat render. Ini tidak mengubah
  tampilan maupun perilaku halaman.
- Ukuran repo bertambah sekitar 66 MB karena aset AdminLTE disalin ke
  `web/static/`; binary hasil build juga membesar sesuai ukuran aset.
