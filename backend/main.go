package main

import (
	"database/sql"
	"io/fs"
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/lmittmann/tint"
	"github.com/maxider/insta/internal/fileblobstorage"
	"github.com/maxider/insta/internal/server"

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
	logger := slog.New(tint.NewTextHandler(os.Stderr, &tint.Options{Level: slog.LevelDebug}))
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
	server := server.Server{Db: db, BlobStorage: &storage}

	if err := run(&server); err != nil {
		log.Fatalln(err)
	}
}

func run(s *server.Server) error {
	//TODO: Add middleware
	// - rate limiting middleware
	mux := http.NewServeMux()

	//make middleware compatible with http.Handler
	mux.Handle("GET /images/{id}", server.Handle(s.GetImage))
	mux.HandleFunc("POST /uploads", server.Handle(s.HandleUploads))
	mux.HandleFunc("GET /latest", server.Handle(s.HandleLatest))

	server := http.Server{
		Addr:              "localhost:8080",
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	slog.Info("Server running.", "Addr", server.Addr)

	return server.ListenAndServe()
}
