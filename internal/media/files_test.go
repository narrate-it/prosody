package media

import (
	"bytes"
	"context"
	"image"
	"image/png"
	"io"
	"testing"
)

func TestOriginalBytesAndImmutableIdentity(t *testing.T) {
	f, err := NewFiles(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	workspace := "01900000-0000-7000-8000-000000000001"
	asset := "01900000-0000-7000-8000-000000000002"
	original := []byte{0, 1, 2, 255, 13, 10}
	object, err := f.Put(context.Background(), workspace, asset, bytes.NewReader(original), 100)
	if err != nil || object.Size != 6 {
		t.Fatalf("write: %+v %v", object, err)
	}
	if _, err = f.Put(context.Background(), workspace, asset, bytes.NewReader([]byte("replacement")), 100); err == nil {
		t.Fatal("overwrote original")
	}
	file, err := f.Open(workspace, asset)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	saved, err := io.ReadAll(file)
	if err != nil || !bytes.Equal(saved, original) {
		t.Fatalf("changed media: %v", err)
	}
	if _, err = f.Open("../", asset); err == nil {
		t.Fatal("accepted traversal")
	}
}
func TestImageBounds(t *testing.T) {
	var b bytes.Buffer
	if err := png.Encode(&b, image.NewRGBA(image.Rect(0, 0, 10, 20))); err != nil {
		t.Fatal(err)
	}
	if _, err := InspectImage(b.Bytes(), 10000, 200); err != nil {
		t.Fatal(err)
	}
	if _, err := InspectImage(b.Bytes(), 10000, 199); err == nil {
		t.Fatal("accepted oversized dimensions")
	}
	if _, err := InspectImage(b.Bytes()[:30], 10000, 200); err == nil {
		t.Fatal("accepted truncated image")
	}
}
