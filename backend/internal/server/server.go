package server

import (
	"bytes"
	"context"
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
	Db          *sql.DB
	BlobStorage ImageBlobStore
}

type handleFunc func(w http.ResponseWriter, r *http.Request) error

func Handle(h handleFunc) http.HandlerFunc {
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
	Metadata []ImageMetadataResponse `json:"metadata"`
}

func NewLatestImagesResponse(metadata []ImageMetadata) LatestImagesResponse {
	resp := make([]ImageMetadataResponse, len(metadata))
	for i, m := range metadata {
		resp[i] = ImageMetadataResponse{
			Id:         m.Id.String(),
			Title:      m.Title,
			Tags:       m.Tags,
			Filesize:   m.Filesize,
			MimeType:   m.MimeType,
			Size_x:     m.Size_x,
			Size_y:     m.Size_y,
			Created_at: m.Created_at,
		}
	}
	return LatestImagesResponse{
		Metadata: resp,
	}
}

type ImageMetadata struct {
	Id         uuid.UUID
	Title      string
	Tags       []string
	Filesize   int64 //in bytes
	MimeType   string
	Size_x     uint16
	Size_y     uint16
	Created_at time.Time
}

type ImageMetadataResponse struct {
	Id         string    `json:"id"`
	Title      string    `json:"title"`
	Tags       []string  `json:"tags"`
	Filesize   int64     `json:"filesize"` //in bytes
	MimeType   string    `json:"mimetyp"`
	Size_x     uint16    `json:"size_x"`
	Size_y     uint16    `json:"size_y"`
	Created_at time.Time `json:"created_at"`
}

func (server *Server) HandleLatest(w http.ResponseWriter, r *http.Request) error {
	const limit = 20
	n := limit
	if limitParam := r.URL.Query().Get("limit"); limitParam != "" {
		if parsed, err := strconv.Atoi(limitParam); err == nil && parsed > 0 {
			n = min(limit, parsed)
		}
	}

	// we can use uuid since they are v7
	rows, err := server.Db.QueryContext(r.Context(), `
		SELECT id, title, filesize, mime_type, size_x, size_y, created_at, group_concat(j.tag_name) FROM image_data AS image
			    LEFT JOIN image_tags AS j ON image.id = j.image_id
				GROUP BY image.id
			    ORDER BY image.id DESC LIMIT ?
		`, n)
	if err != nil {
		return fmt.Errorf("error retriving latest: %w", err)
	}
	defer rows.Close()

	metadata := make([]ImageMetadata, 0)
	for rows.Next() {
		var id uuid.UUID
		var title string
		var filesize int
		var mime_type string
		var size_x int
		var size_y int
		var create_at time.Time
		var tags_unparsed string
		if err := rows.Scan(&id, &title, &filesize, &mime_type, &size_x, &size_y, &create_at, &tags_unparsed); err != nil {
			return err
		}

		metadata = append(metadata, ImageMetadata{
			Id:         id,
			Title:      title,
			Tags:       strings.Split(tags_unparsed, ","),
			Filesize:   int64(filesize),
			MimeType:   mime_type,
			Size_x:     uint16(size_x),
			Size_y:     uint16(size_y),
			Created_at: create_at,
		})
	}
	if err := rows.Err(); err != nil {
		return err
	}
	slog.Debug("Fetched image metadata", "metadata", metadata)

	writeJSON(w, http.StatusOK, NewLatestImagesResponse(metadata))

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

func (server *Server) HandleUploads(w http.ResponseWriter, r *http.Request) error {
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
	fileMetadata, err := server.BlobStorage.StoreImage(id, bytes.NewReader(result.ImageBytes))
	if err != nil {
		//TODO: check from error if bad request or internal server error
		return fmt.Errorf("error storing image: %w", err)
	}
	slog.Debug("Created file in blob storage", "metadata", fileMetadata)

	imageMetadata := ImageMetadata{
		Id:         id,
		Title:      result.Title,
		Tags:       result.Tags,
		Filesize:   fileMetadata.Filesize,
		MimeType:   result.ContentType,
		Size_x:     0,          //TODO
		Size_y:     0,          //TODO
		Created_at: time.Now(), //Maybe write this only in the DB
	}
	//TODO: abstract writes behind interface
	err = insertImageInDb(server.Db, r.Context(), imageMetadata)
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

func (server *Server) GetImage(w http.ResponseWriter, r *http.Request) error {
	id := r.PathValue("id")
	uuid, err := uuid.Parse(id)
	if err != nil {
		return NewApiError(http.StatusNotFound, "no image with this id", errors.New("failed to parse param id into uuid"))
	}
	reader, err := server.BlobStorage.GetImageBlob(uuid)
	if err != nil {
		return fmt.Errorf("failed to get image blob: %w", err)
	}

	http.ServeContent(w, r, "TODO", time.Now(), reader)
	return nil
}

func insertImageInDb(db *sql.DB, ctx context.Context, imageMetadata ImageMetadata) error {

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to create transation: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `
		INSERT INTO image_data(id, title, filesize, mime_type, size_x, size_y, created_at) VALUES (?,?,?,?,?,?,?)
		`,
		imageMetadata.Id, imageMetadata.Title, imageMetadata.Filesize, imageMetadata.MimeType, imageMetadata.Size_x, imageMetadata.Size_y, imageMetadata.Created_at)
	if err != nil {
		return fmt.Errorf("error inserting image data in db: %w", err)
	}

	tagStmt, err := tx.PrepareContext(ctx,
		`INSERT INTO tags(name, created_at) VALUES (?, ?) ON CONFLICT(name) DO NOTHING`)
	if err != nil {
		return fmt.Errorf("preparing tag insert: %w", err)
	}
	defer tagStmt.Close()

	linkStmt, err := tx.PrepareContext(ctx,
		`INSERT INTO image_tags(image_id, tag_name) VALUES (?, ?) ON CONFLICT DO NOTHING`)
	if err != nil {
		return fmt.Errorf("preparing image_tags insert: %w", err)
	}
	defer linkStmt.Close()

	now := time.Now()
	for _, t := range imageMetadata.Tags {
		if _, err := tagStmt.ExecContext(ctx, t, now); err != nil {
			return fmt.Errorf("inserting tag %q: %w", t, err)
		}
		if _, err := linkStmt.ExecContext(ctx, imageMetadata.Id, t); err != nil {
			return fmt.Errorf("linking tag %q: %w", t, err)
		}
	}

	tx.Commit()
	return nil
}

type ImageBlobStore interface {
	GetImageBlob(id uuid.UUID) (io.ReadSeeker, error)
	StoreImage(id uuid.UUID, rc io.Reader) (*FileMetadata, error)
	DeleteImage(id uuid.UUID)
}

type FileMetadata struct {
	Id       uuid.UUID
	Filesize int64
}
