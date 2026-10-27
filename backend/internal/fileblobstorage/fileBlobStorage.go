package fileblobstorage

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"uuid"

	"github.com/maxider/insta/internal/server"
)

type FileBlobStorage struct {
	Dir string
}

// nil = file not found
func (storage *FileBlobStorage) GetImageBlob(id uuid.UUID) (io.ReadSeeker, error) {
	file, err := os.Open(filepath.Join(storage.Dir, id.String()))
	if err != nil {
		return nil, err
	}

	return file, nil
}

func (storage *FileBlobStorage) StoreImage(id uuid.UUID, reader io.Reader) (*server.FileMetadata, error) {
	file, err := os.Create(filepath.Join(storage.Dir, id.String()))
	if err != nil {
		return nil, err
	}
	defer file.Close()
	slog.Debug("FileStorage - Created file in filesystem", "path", file.Name())

	n, err := io.Copy(file, reader)
	if err != nil {
		return nil, fmt.Errorf("FileStorage - failed to write bytes to file: %w", err)
	}
	slog.Debug("FileStorage - wrote bytes to file", "bytes writte", n)
	fileInfo, err := file.Stat()
	if err != nil {
		return nil, err
	}
	return &server.FileMetadata{
		Id:       id,
		Filesize: fileInfo.Size(),
	}, nil
}

func (storage *FileBlobStorage) DeleteImage(id uuid.UUID) {
	panic("TODO")
}
