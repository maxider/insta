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
			writeJson(w, apiErr.Code, apiErr)

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

func writeJson(w http.ResponseWriter, code int, v any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	return json.NewEncoder(w).Encode(v)
}

func (s *Server) handleUploads(w http.ResponseWriter, r *http.Request) error {
	//TODO: Allow for multipart
	const limit = 1024 * 1024 * 10 //10MB
	maxReader := http.MaxBytesReader(w, r.Body, limit)

	//TODO: duplicate checK? no since the same images can just be uplaoded again by someone in an insta clone.

	id := uuid.NewV7()
	filePath := filepath.Join(s.imageStorageDir, id.String())
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("Failed to create file: %w", err)
	}
	defer file.Close()
	slog.Debug("Created file", "Path", filePath)

	n, err := io.Copy(file, maxReader)
	if err != nil {
		return NewApiError(http.StatusBadRequest, "upload failed", fmt.Errorf("Failed to copy bytes into file: %w", err))
	}
	slog.Debug("copied bytes into file", "bytes written", n)

	file.Close()

	mediaType, _, err := mime.ParseMediaType(r.Header.Get("content-type"))
	if err != nil {
		return NewApiError(http.StatusBadRequest, "invalid content type", err)
	}

	//abstract writes behind interface
	_, err = s.db.ExecContext(r.Context(), `INSERT INTO images(id, image_type, uploaded_at) VALUES (?,?,?)`,
		id, mediaType, time.Now())
	if err != nil {
		if err := os.Remove(file.Name()); err != nil {
			slog.Error("Failed to delete file wich was already created", "err", err)
		}
		return fmt.Errorf("failed to insert image info into DB: %w", err)
	}
	slog.Debug("Insert image into table", "id", id)

	return nil
}
