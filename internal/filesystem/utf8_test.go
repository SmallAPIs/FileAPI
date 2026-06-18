package filesystem

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestStreamValidatedUTF8(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "text.txt")
	if err := os.WriteFile(path, []byte("stream me"), 0o644); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	if err := streamValidatedUTF8(path, testMaxSize, &buf); err != nil {
		t.Fatalf("stream: %v", err)
	}
	if buf.String() != "stream me" {
		t.Fatalf("unexpected content: %q", buf.String())
	}
}

func TestReadValidatedUTF8(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "text.txt")
	if err := os.WriteFile(path, []byte("read me"), 0o644); err != nil {
		t.Fatal(err)
	}

	data, err := readValidatedUTF8(path, testMaxSize)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(data) != "read me" {
		t.Fatalf("unexpected content: %q", data)
	}
}
