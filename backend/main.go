package main

import (
	"database/sql"
	"io/fs"
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/maxider/insta/internal/fileblobstorage"

	_ "embed"

	_ "modernc.org/sqlite"
)

//go:embed tables.sql
var schema string

func migrate(db *sql.DB) error {
	_, err := db.Exec(schema)
	return err
}

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
	slog.SetDefault(logger)

	db, err := sql.Open("sqlite", "db.db")
	if err != nil {
		log.Fatalln(err)
	}
	defer db.Close()
	if err := migrate(db); err != nil {
		log.Fatal(err)
	}

	//TODO: pull from args/env/config and abstract behind interface.
	const imageStorageDir = "./file-storage"
	const dirPerm fs.FileMode = 0o755
	if err := os.MkdirAll(imageStorageDir, dirPerm); err != nil {
		log.Fatalln(err)
	}
	storage := fileblobstorage.FileBlobStorage{
		Dir: imageStorageDir,
	}
	server := Server{db: db, blobStorage: &storage}

	if err := run(&server); err != nil {
		log.Fatalln(err)
	}
}

func run(s *Server) error {
	//TODO: Add middleware
	// - rate limiting middleware
	mux := http.NewServeMux()

	//make middleware compatible with http.Handler
	mux.Handle("GET /images/{id}", handle(s.getImage))
	mux.HandleFunc("POST /uploads", handle(s.handleUploads))
	mux.HandleFunc("GET /latest", handle(s.handleLatest))

	server := http.Server{
		Addr:              "localhost:8080",
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	slog.Info("Server running.", "Addr", server.Addr)

	return server.ListenAndServe()
}
