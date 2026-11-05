package main

import (
	"flag"
	"log"
	"net/http"
	"path/filepath"

	"simple-go-crud-app/internal/pages"
	"simple-go-crud-app/internal/render"
	"simple-go-crud-app/internal/router"
)

func main() {
	addr := flag.String("addr", ":8080", "HTTP listen address")
	webDir := flag.String("web", "web", "directory containing templates and static assets")
	dev := flag.Bool("dev", true, "reload templates on every request")
	flag.Parse()

	renderer := render.New(filepath.Join(*webDir, "templates"), *dev)
	handler := router.New(pages.Pages, renderer, filepath.Join(*webDir, "static"))

	log.Printf("AdminLTE demo listening on http://localhost%s", *addr)
	if err := http.ListenAndServe(*addr, handler); err != nil {
		log.Fatal(err)
	}
}
