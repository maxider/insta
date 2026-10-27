package fileblobstorage

import (
	"bytes"
	"testing"
	"uuid"
)

func TestImageStoring(t *testing.T) {
	dir := t.TempDir()
	store := FileBlobStorage{
		Dir: dir,
	}
	buff := make([]byte, 1024)

	id := uuid.NewV7()
	err := store.StoreImage(id, bytes.NewReader(buff))
	if err != nil {
		t.Fatal()
	}
}
