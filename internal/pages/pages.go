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

// Pages is the set of static AdminLTE pages kept by this demo.
var Pages = []Page{
	{
		Route:       "/starter.html",
		Template:    "starter.html",
		Layout:      "main",
		Title:       "AdminLTE 2 | Starter",
		BodyClass:   "hold-transition skin-blue sidebar-mini",
		Active:      "starter",
		Sidebar:     "starter",
		FooterClass: "main-footer",
	},
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
