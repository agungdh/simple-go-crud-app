package main

import (
	"flag"
	"log"
	"net/http"

	"simple-go-crud-app/internal/pages"
	"simple-go-crud-app/internal/render"
	"simple-go-crud-app/internal/router"
	"simple-go-crud-app/web"
)

func main() {
	addr := flag.String("addr", ":8080", "HTTP listen address")
	flag.Parse()

	renderer, err := render.New(web.FS)
	if err != nil {
		log.Fatal(err)
	}

	handler, err := router.New(pages.Pages, renderer, web.FS)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("AdminLTE demo listening on http://localhost%s", *addr)
	if err := http.ListenAndServe(*addr, handler); err != nil {
		log.Fatal(err)
	}
}
