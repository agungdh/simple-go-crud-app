package render

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"simple-go-crud-app/internal/pages"
)

// Renderer loads and caches one template set per page.
type Renderer struct {
	root  string
	dev   bool
	mu    sync.RWMutex
	cache map[string]*template.Template
}

// New creates a Renderer. When dev is true templates are re-parsed on every
// request so changes show up without restarting the server.
func New(root string, dev bool) *Renderer {
	return &Renderer{
		root:  root,
		dev:   dev,
		cache: make(map[string]*template.Template),
	}
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
		"header-compact.html",
		"header-starter.html",
		"header-topnav.html",
		"sidebar.html",
		"sidebar-starter.html",
		"control-sidebar.html",
		"footer.html",
		"footer-topnav.html",
	}
	out := make([]string, 0, len(names))
	for _, name := range names {
		out = append(out, filepath.Join(r.root, "partials", name))
	}
	return out
}

func (r *Renderer) load(p pages.Page) (*template.Template, error) {
	key := p.Layout + "|" + p.Template
	if !r.dev {
		r.mu.RLock()
		t, ok := r.cache[key]
		r.mu.RUnlock()
		if ok {
			return t, nil
		}
	}

	layoutPath := filepath.Join(r.root, "layouts", p.Layout+".html")
	pagePath := filepath.Join(r.root, "pages", p.Template)

	files := make([]string, 0, len(r.partials())+2)
	files = append(files, layoutPath)
	files = append(files, r.partials()...)
	files = append(files, pagePath)

	for _, f := range files {
		if _, err := os.Stat(f); err != nil {
			return nil, fmt.Errorf("template file %s: %w", f, err)
		}
	}

	name := filepath.Base(layoutPath)
	t, err := template.New(name).Funcs(r.funcs()).ParseFiles(files...)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", pagePath, err)
	}

	if !r.dev {
		r.mu.Lock()
		r.cache[key] = t
		r.mu.Unlock()
	}
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
