package main

import (
	"context"
	"flag"
	"log"
	"net/http"

	"simple-go-crud-app/internal/database"
	"simple-go-crud-app/internal/handler"
	"simple-go-crud-app/internal/pages"
	"simple-go-crud-app/internal/render"
	"simple-go-crud-app/internal/repository"
	"simple-go-crud-app/internal/router"
	"simple-go-crud-app/web"
)

func main() {
	addr := flag.String("addr", ":8080", "HTTP listen address")
	dbPath := flag.String("db", "app.db", "SQLite database file")
	flag.Parse()

	renderer, err := render.New(web.FS)
	if err != nil {
		log.Fatal(err)
	}

	db, err := database.Open(*dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	ctx := context.Background()
	if err := database.Migrate(ctx, db); err != nil {
		log.Fatal(err)
	}

	pegawai := handler.NewPegawai(repository.NewPegawai(db), renderer, pages.Pegawai)
	bidang := handler.NewBidang(repository.NewBidang(db), renderer, pages.Bidang)

	httpHandler, err := router.New(pages.Pages, renderer, web.FS, pegawai, bidang)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("AdminLTE demo listening on http://localhost%s (db: %s)", *addr, *dbPath)
	if err := http.ListenAndServe(*addr, httpHandler); err != nil {
		log.Fatal(err)
	}
}
