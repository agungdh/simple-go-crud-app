package render

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"strings"
	"sync"

	"simple-go-crud-app/internal/pages"
)

// Renderer loads and caches one template set per page.
type Renderer struct {
	templates fs.FS
	mu        sync.RWMutex
	cache     map[string]*template.Template
}

// New creates a Renderer that reads templates from the "templates" directory
// of webFS. Parsed template sets are cached for the lifetime of the process.
func New(webFS fs.FS) (*Renderer, error) {
	templates, err := fs.Sub(webFS, "templates")
	if err != nil {
		return nil, fmt.Errorf("templates: %w", err)
	}
	return &Renderer{
		templates: templates,
		cache:     make(map[string]*template.Template),
	}, nil
}

func (r *Renderer) funcs() template.FuncMap {
	return template.FuncMap{
		"hasPrefix": strings.HasPrefix,
		"safeAttr": func(s string) template.HTMLAttr {
			return template.HTMLAttr(s)
		},
		"safeJS": func(s string) template.JS {
			return template.JS(s)
		},
	}
}

func (r *Renderer) partials() []string {
	names := []string{
		"header.html",
		"header-starter.html",
		"sidebar.html",
		"sidebar-starter.html",
		"footer.html",
	}
	out := make([]string, 0, len(names))
	for _, name := range names {
		out = append(out, "partials/"+name)
	}
	return out
}

func (r *Renderer) load(p pages.Page) (*template.Template, error) {
	key := p.Layout + "|" + p.Template
	r.mu.RLock()
	t, ok := r.cache[key]
	r.mu.RUnlock()
	if ok {
		return t, nil
	}

	layoutPath := "layouts/" + p.Layout + ".html"
	pagePath := "pages/" + p.Template

	files := make([]string, 0, len(r.partials())+2)
	files = append(files, layoutPath)
	files = append(files, r.partials()...)
	files = append(files, pagePath)

	t, err := template.New(p.Layout+".html").Funcs(r.funcs()).ParseFS(r.templates, files...)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", pagePath, err)
	}

	r.mu.Lock()
	r.cache[key] = t
	r.mu.Unlock()
	return t, nil
}

// Render executes the page using its layout.
func (r *Renderer) Render(w io.Writer, p pages.Page) error {
	t, err := r.load(p)
	if err != nil {
		return err
	}

	var buf bytes.Buffer
	if err := t.ExecuteTemplate(&buf, p.Layout+".html", p); err != nil {
		return fmt.Errorf("render %s: %w", p.Template, err)
	}
	_, err = w.Write(buf.Bytes())
	return err
}
