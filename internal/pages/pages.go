package pages

// Page describes a single AdminLTE page and how it is rendered.
type Page struct {
	Route       string
	Template    string
	Layout      string
	Title       string
	BodyClass   string
	Onload      string
	Active      string
	Sidebar     string
	FooterClass string
}

// Pegawai is the dynamic CRUD page. It is not part of Pages because it needs
// data from the database and is mounted by the pegawai handler instead.
var Pegawai = Page{
	Route:       "/pegawai",
	Template:    "pegawai.html",
	Layout:      "main",
	Title:       "AdminLTE 2 | Data Pegawai",
	BodyClass:   "hold-transition skin-blue sidebar-mini",
	Active:      "pegawai",
	Sidebar:     "default",
	FooterClass: "main-footer",
}

// PegawaiEdit is the edit form, mounted by the pegawai handler.
var PegawaiEdit = Page{
	Route:       "/pegawai/{id}/edit",
	Template:    "pegawai_form.html",
	Layout:      "main",
	Title:       "AdminLTE 2 | Edit Pegawai",
	BodyClass:   "hold-transition skin-blue sidebar-mini",
	Active:      "pegawai",
	Sidebar:     "default",
	FooterClass: "main-footer",
}

// PegawaiCreate is the add form, mounted by the pegawai handler.
var PegawaiCreate = Page{
	Route:       "/pegawai/create",
	Template:    "pegawai_form.html",
	Layout:      "main",
	Title:       "AdminLTE 2 | Tambah Pegawai",
	BodyClass:   "hold-transition skin-blue sidebar-mini",
	Active:      "pegawai",
	Sidebar:     "default",
	FooterClass: "main-footer",
}

// Bidang is the dynamic CRUD page. It is not part of Pages because it needs
// data from the database and is mounted by the bidang handler instead.
var Bidang = Page{
	Route:       "/bidang",
	Template:    "bidang.html",
	Layout:      "main",
	Title:       "AdminLTE 2 | Data Bidang",
	BodyClass:   "hold-transition skin-blue sidebar-mini",
	Active:      "bidang",
	Sidebar:     "default",
	FooterClass: "main-footer",
}

// BidangEdit is the edit form, mounted by the bidang handler.
var BidangEdit = Page{
	Route:       "/bidang/{id}/edit",
	Template:    "bidang_form.html",
	Layout:      "main",
	Title:       "AdminLTE 2 | Edit Bidang",
	BodyClass:   "hold-transition skin-blue sidebar-mini",
	Active:      "bidang",
	Sidebar:     "default",
	FooterClass: "main-footer",
}

// BidangCreate is the add form, mounted by the bidang handler.
var BidangCreate = Page{
	Route:       "/bidang/create",
	Template:    "bidang_form.html",
	Layout:      "main",
	Title:       "AdminLTE 2 | Tambah Bidang",
	BodyClass:   "hold-transition skin-blue sidebar-mini",
	Active:      "bidang",
	Sidebar:     "default",
	FooterClass: "main-footer",
}

// Pages is the set of static AdminLTE pages kept by this demo.
var Pages = []Page{
	{
		Route:       "/pages/examples/404.html",
		Template:    "examples/404.html",
		Layout:      "main",
		Title:       "AdminLTE 2 | 404 Page not found",
		BodyClass:   "hold-transition skin-blue sidebar-mini",
		Active:      "examples-404",
		Sidebar:     "default",
		FooterClass: "main-footer",
	},
	{
		Route:       "/pages/examples/500.html",
		Template:    "examples/500.html",
		Layout:      "main",
		Title:       "AdminLTE 2 | 500 Error",
		BodyClass:   "hold-transition skin-blue sidebar-mini",
		Active:      "examples-500",
		Sidebar:     "default",
		FooterClass: "main-footer",
	},
}
