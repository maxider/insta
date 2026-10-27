package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
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
	db          *sql.DB
	blobStorage ImageBlobStore
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
	Ids []string `json:"ids"`
}

func NewLatestImagesResponse(uuids []uuid.UUID) LatestImagesResponse {
	ids := make([]string, len(uuids))
	for i, id := range uuids {
		ids[i] = id.String()
	}
	return LatestImagesResponse{
		Ids: ids,
	}
}

func (server *Server) handleLatest(w http.ResponseWriter, r *http.Request) error {
	const limit = 20
	n := limit
	if limitParam := r.URL.Query().Get("limit"); limitParam != "" {
		if parsed, err := strconv.Atoi(limitParam); err == nil && parsed > 0 {
			n = min(limit, parsed)
		}
	}

	// we can use uuid since they are v7
	rows, err := server.db.QueryContext(r.Context(), `SELECT id FROM images ORDER BY id DESC LIMIT ? `, n)
	if err != nil {
		return fmt.Errorf("error retriving latest: %w", err)
	}
	defer rows.Close()

	ids := make([]uuid.UUID, 0)
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	slog.Debug("Fetched image ids", "ids", ids)

	writeJSON(w, http.StatusOK, NewLatestImagesResponse(ids))

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
	const limit = 1024 * 1024 * 10 //10MB
	r.Body = http.MaxBytesReader(w, r.Body, limit)

	reader, err := r.MultipartReader()
	if err != nil {
		return err
	}
	result, err := extractParts(reader)
	if err != nil {
		return err
	}

	//TODO: use extension
	_, err = parseMimeType(result.ContentType)
	if err != nil {
		return fmt.Errorf("failed to parse mime type %w", err)
	}

	//TODO: duplicate checK? no since the same images can just be uplaoded again by someone in an insta clone.

	id := uuid.NewV7()
	if err = server.blobStorage.StoreImage(id, bytes.NewReader(result.ImageBytes)); err != nil {
		//TODO: check from error if bad request or internal server error
		return fmt.Errorf("error storing image: %w", err)
	}
	slog.Debug("Created file in blob storage", "id", id)

	//TODO: abstract writes behind interface
	err = insertImageInDb(server.db, r, id, result.Title)
	if err != nil {
		return err
	}
	slog.Debug("Insert image into table", "id", id)

	writeJSON(w, http.StatusCreated, NewUploadResponse(id))
	return nil
}

type PartsResult struct {
	Title       string
	Tags        []string
	ImageBytes  []byte
	ContentType string
}

func extractParts(reader *multipart.Reader) (*PartsResult, error) {
	var title string
	var tags []string
	var imageBytes []byte
	var contentType string
	for {
		part, err := reader.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read part: %w", err)
		}
		switch part.FormName() {
		case "title":
			{
				title, err = readPart(part)
				if err != nil {
					return nil, err
				}
			}
		case "tags":
			{
				tags_unparsed, err := readPart(part)
				if err != nil {
					return nil, err
				}
				tags = strings.Split(tags_unparsed, ",")
			}
		case "file":
			{
				slog.Debug("file")
				imageBytes, err = io.ReadAll(part)
				if err != nil {
					return nil, err
				}
				contentType = part.Header.Get("content-type")
			}
		}
	}
	return &PartsResult{
		Title:       title,
		Tags:        tags,
		ImageBytes:  imageBytes,
		ContentType: contentType,
	}, nil
}

func readPart(p *multipart.Part) (string, error) {
	content, err := io.ReadAll(p)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func (server *Server) getImage(w http.ResponseWriter, r *http.Request) error {
	id := r.PathValue("id")
	uuid, err := uuid.Parse(id)
	if err != nil {
		return NewApiError(http.StatusNotFound, "no image with this id", errors.New("failed to parse param id into uuid"))
	}
	reader, err := server.blobStorage.GetImageBlob(uuid)
	if err != nil {
		return fmt.Errorf("failed to get image blob: %w", err)
	}

	http.ServeContent(w, r, "TODO", time.Now(), reader)
	return nil
}

func insertImageInDb(db *sql.DB, r *http.Request, id uuid.UUID, title string) error {
	_, err := db.ExecContext(r.Context(), `INSERT INTO images(id, title, uploaded_at) VALUES (?,?,?)`,
		id, title, time.Now())
	if err != nil {
		return fmt.Errorf("error inserting image data in db: %w", err)
	}

	return nil
}

type ImageData struct {
	id        uuid.UUID
	title     string
	extension string
	createdAt time.Time
}

type ImageBlobStore interface {
	GetImageBlob(id uuid.UUID) (io.ReadSeeker, error)
	StoreImage(id uuid.UUID, rc io.Reader) error
	DeleteImage(id uuid.UUID)
}
