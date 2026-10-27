package fileblobstorage

import (
	"bytes"
	"crypto/rand"
	"testing"
	"uuid"
)

func TestImageStoring(t *testing.T) {
	dir := t.TempDir()
	store := FileBlobStorage{
		Dir: dir,
	}
	buff := make([]byte, 1024)
	rand.Read(buff)

	id := uuid.NewV7()
	err := store.StoreImage(id, bytes.NewReader(buff))
	if err != nil {
		t.Fatal()
	}

	//TODO: check that file contents are identical
}
