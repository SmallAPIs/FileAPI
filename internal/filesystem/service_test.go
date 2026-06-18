package filesystem

import (
	"os"
	"path/filepath"
	"testing"
)

const testMaxSize = 10 * 1024 * 1024

func newTestService(t *testing.T, root string) *Service {
	t.Helper()
	svc, err := NewService([]string{root}, testMaxSize, testMaxSize)
	if err != nil {
		t.Fatal(err)
	}
	return svc
}

func TestServiceSandbox(t *testing.T) {
	root := t.TempDir()
	nested := filepath.Join(root, "docs")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}

	svc := newTestService(t, root)

	allowed := filepath.Join(nested, "note.txt")
	if _, err := svc.Create(allowed, "hello", false, true); err != nil {
		t.Fatalf("create allowed file: %v", err)
	}

	if _, err := svc.Read(allowed); err != nil {
		t.Fatalf("read allowed file: %v", err)
	}

	outside := filepath.Join(filepath.Dir(root), "outside.txt")
	if _, err := svc.Read(outside); err != ErrPathNotAllowed {
		t.Fatalf("expected ErrPathNotAllowed, got %v", err)
	}

	traversal := filepath.Join(root, "..", filepath.Base(root), "docs", "..", "..", "outside.txt")
	if _, err := svc.Read(traversal); err != ErrPathNotAllowed {
		t.Fatalf("expected traversal block, got %v", err)
	}
}

func TestServiceCRUD(t *testing.T) {
	root := t.TempDir()
	svc := newTestService(t, root)

	path := filepath.Join(root, "a", "b.txt")
	created, err := svc.Create(path, "one", true, true)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if created.Content != "one" {
		t.Fatalf("unexpected content: %q", created.Content)
	}

	updated, err := svc.Edit(path, "two", "overwrite", true)
	if err != nil {
		t.Fatalf("edit overwrite: %v", err)
	}
	if updated.Content != "two" {
		t.Fatalf("unexpected content after overwrite: %q", updated.Content)
	}

	appended, err := svc.Edit(path, "three", "append", true)
	if err != nil {
		t.Fatalf("edit append: %v", err)
	}
	if appended.Content != "twothree" {
		t.Fatalf("unexpected content after append: %q", appended.Content)
	}

	if err := svc.Delete(path); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := svc.Read(path); err != ErrNotFound {
		t.Fatalf("expected not found after delete, got %v", err)
	}
}

func TestServiceCreateWithoutContent(t *testing.T) {
	root := t.TempDir()
	svc := newTestService(t, root)

	path := filepath.Join(root, "meta.txt")
	info, err := svc.Create(path, "payload", false, false)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if info.Content != "" {
		t.Fatalf("expected empty content, got %q", info.Content)
	}
	if info.Size != 7 {
		t.Fatalf("expected size 7, got %d", info.Size)
	}
}

func TestServiceRejectsBinary(t *testing.T) {
	root := t.TempDir()
	svc := newTestService(t, root)

	path := filepath.Join(root, "binary.bin")
	if err := os.WriteFile(path, []byte{0xFF, 0xFE, 0xFD}, 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := svc.Read(path); err != ErrBinaryContent {
		t.Fatalf("expected ErrBinaryContent, got %v", err)
	}
}

func TestServiceRejectsOversizedContent(t *testing.T) {
	root := t.TempDir()
	svc, err := NewService([]string{root}, testMaxSize, 32)
	if err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(root, "big.txt")
	if _, err := svc.Create(path, string(make([]byte, 64)), false, false); err != ErrContentTooLarge {
		t.Fatalf("expected ErrContentTooLarge, got %v", err)
	}
}

func TestConsumeValidUTF8(t *testing.T) {
	valid, remain, err := consumeValidUTF8([]byte("hello"))
	if err != nil || string(valid) != "hello" || len(remain) != 0 {
		t.Fatalf("unexpected result: valid=%q remain=%q err=%v", valid, remain, err)
	}

	_, _, err = consumeValidUTF8([]byte{0xFF})
	if err != ErrBinaryContent {
		t.Fatalf("expected ErrBinaryContent, got %v", err)
	}
}
