package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"

	_ "modernc.org/sqlite"

	apihttp "github.com/frontpage/quotesvc/internal/http"
	"github.com/frontpage/quotesvc/internal/store"
)

func main() {
	path := os.Getenv("QUOTESVC_DB")
	if path == "" {
		path = "quotesvc.db"
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	st := store.New(db)
	if err := st.Migrate(); err != nil {
		log.Fatalf("migrate: %v", err)
	}
	addr := os.Getenv("QUOTESVC_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	log.Printf("listening on %s (db: %s)", addr, path)
	if err := http.ListenAndServe(addr, apihttp.NewServer(st)); err != nil {
		log.Fatalf("serve: %v", err)
	}
}
