package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"
	"uuid"
)

type APIError struct {
	Code    int    `json:"statusCode"`
	Message string `json:"msg"`
	//only for internal logging. Never output to client
	Err error `json:"-"`
}

func NewApiError(code int, msg string, err error) *APIError {
	return &APIError{
		Code:    code,
		Message: msg,
		Err:     err,
	}
}

func (e *APIError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("api error %d: %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("api error %d: %s", e.Code, e.Message)
}

func (e *APIError) Unwrap() error { return e.Err }

type Server struct {
	db              *sql.DB
	imageStorageDir string
}

type handleFunc func(w http.ResponseWriter, r *http.Request) error

func handle(h handleFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		err := h(w, r)
		duration := time.Since(start)

		level := slog.LevelInfo
		msg := "request handled"
		attrs := []slog.Attr{
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Duration("duration", duration),
		}

		if err != nil {
			var apiErr *APIError
			if !errors.As(err, &apiErr) {
				apiErr = &APIError{Code: http.StatusInternalServerError, Message: "internal server error"}
			}
			writeJSON(w, apiErr.Code, apiErr)

			level = slog.LevelError
			msg = "request failed"
			attrs = append(attrs,
				slog.Int("status", apiErr.Code),
				slog.Any("err", err),
			)
		}

		slog.LogAttrs(r.Context(), level, msg, attrs...)
	}
}

func writeJSON(w http.ResponseWriter, code int, v any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	return json.NewEncoder(w).Encode(v)
}

type LatestImagesResponse struct {
	FileNames []string `json:"filenames"`
}

func NewLatestImagesResponse(fileNames []string) LatestImagesResponse {
	return LatestImagesResponse{
		FileNames: fileNames,
	}
}

func (s *Server) handleLatest(w http.ResponseWriter, r *http.Request) error {
	const limit = 20
	n := limit
	if limitParam := r.URL.Query().Get("limit"); limitParam != "" {
		if parsed, err := strconv.Atoi(limitParam); err == nil && parsed > 0 {
			n = min(limit, parsed)
		}
	}

	// we can use uuid since they are v7
	rows, err := s.db.QueryContext(r.Context(), `SELECT filename FROM images ORDER BY id DESC LIMIT ? `, n)
	if err != nil {
		return err
	}
	defer rows.Close()

	filenames := make([]string, 0)
	for rows.Next() {
		var f string
		if err := rows.Scan(&f); err != nil {
			return err
		}
		filenames = append(filenames, f)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	slog.Debug("Fetched files", "filenames", filenames)

	writeJSON(w, http.StatusOK, NewLatestImagesResponse(filenames))

	return nil
}

type UploadResponse struct {
	ID string `json:"id"`
}

func NewUploadResponse(id uuid.UUID) UploadResponse {
	return UploadResponse{ID: id.String()}
}

var allowedExtensions = map[string]string{
	"image/jpeg": ".jpeg",
	"image/png":  ".png",
	"image/webp": ".webp",
}

func parseMimeType(mediatype string) (string, error) {
	mediaType, _, err := mime.ParseMediaType(mediatype)
	if err != nil {
		return "", NewApiError(http.StatusBadRequest, "invalid content type", err)
	}
	ext, ok := allowedExtensions[mediaType]
	if !ok {
		return "", NewApiError(http.StatusUnsupportedMediaType, "unsuported media type", nil)
	}

	return ext, nil
}

func (server *Server) handleUploads(w http.ResponseWriter, r *http.Request) error {
	//TODO: Allow for multipart
	const limit = 1024 * 1024 * 10 //10MB
	maxReader := http.MaxBytesReader(w, r.Body, limit)

	ext, err := parseMimeType(r.Header.Get("content-type"))
	if err != nil {
		return fmt.Errorf("failed to parse mime type %w", err)
	}

	//TODO: duplicate checK? no since the same images can just be uplaoded again by someone in an insta clone.

	id := uuid.NewV7()
	file, err := createFile(id, ext, server)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()
	slog.Debug("Created file", "Path", file.Name())

	n, err := io.Copy(file, maxReader)
	if err != nil {
		return NewApiError(http.StatusBadRequest, "upload failed", fmt.Errorf("failed to copy bytes into file: %w", err))
	}
	slog.Debug("copied bytes into file", "bytes written", n)

	file.Close()

	//TODO: abstract writes behind interface
	err = insertImageInDb(server, r, id, file)
	if err != nil {
		return err
	}
	slog.Debug("Insert image into table", "id", id)

	writeJSON(w, http.StatusCreated, NewUploadResponse(id))
	return nil
}

func insertImageInDb(server *Server, r *http.Request, id uuid.UUID, file *os.File) error {
	_, err := server.db.ExecContext(r.Context(), `INSERT INTO images(id, filename, uploaded_at) VALUES (?,?,?)`,
		id, file.Name(), time.Now())
	if err != nil {
		if err := os.Remove(file.Name()); err != nil {
			slog.Error("Failed to delete file wich was already created", "err", err)
		}
		return fmt.Errorf("failed to insert image info into DB: %w", err)
	}
	return nil
}

func createFile(id uuid.UUID, ext string, s *Server) (*os.File, error) {
	fileName := id.String() + ext
	filePath := filepath.Join(s.imageStorageDir, fileName)
	file, err := os.Create(filePath)
	return file, err
}
