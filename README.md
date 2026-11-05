# AdminLTE 2.4.18 — Go Template Demo

Replika UI lengkap **AdminLTE 2.4.18** yang dirender memakai Go `html/template`
dengan satu layout bersama. Semua aset (CSS, JS, font, gambar, plugin) disajikan
secara lokal oleh server Go — tidak butuh koneksi ke CDN untuk aset utama.

Cakupan saat ini: **UI/demo statis** (37 halaman). Belum ada database, CRUD,
autentikasi, atau logika bisnis.

## Menjalankan

```bash
go run .
```

Buka <http://localhost:8080>. Root `/` langsung menampilkan Dashboard v1.

Opsi yang tersedia:

```bash
go run . -addr :3000     # ganti port
go run . -web web        # lokasi folder template + static
go run . -dev=false      # cache template (default -dev=true, reload tiap request)
```

Build binary:

```bash
go build -o adminlte-demo .
./adminlte-demo -addr :8080
```

## Struktur

```
main.go                     entrypoint + flag
internal/pages/pages.go     registry 37 halaman (route, layout, title, sidebar, dll.)
internal/render/render.go   loader/cache template + template funcs
internal/router/router.go   routing chi, mount aset statis, handler 404
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
- `html/template` menghapus komentar HTML/JS saat render. Ini tidak mengubah
  tampilan maupun perilaku halaman.
- Ukuran repo bertambah sekitar 66 MB karena aset AdminLTE disalin ke
  `web/static/`.
