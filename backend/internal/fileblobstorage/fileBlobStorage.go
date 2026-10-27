package fileblobstorage

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"uuid"
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

func (storage *FileBlobStorage) StoreImage(id uuid.UUID, reader io.Reader) error {
	file, err := os.Create(filepath.Join(storage.Dir, id.String()))
	if err != nil {
		return err
	}
	defer file.Close()
	slog.Debug("FileStorage - Created file in filesystem", "path", file.Name())

	n, err := io.Copy(file, reader)
	if err != nil {
		return fmt.Errorf("FileStorage - failed to write bytes to file: %w", err)
	}
	slog.Debug("FileStorage - wrote bytes to file", "bytes writte", n)
	return nil
}

func (storage *FileBlobStorage) DeleteImage(id uuid.UUID) {
	panic("TODO")
}
