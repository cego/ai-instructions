package filemanager

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHashBytes(t *testing.T) {
	hash := HashBytes([]byte("hello world"))
	if !strings.HasPrefix(hash, "sha256:") {
		t.Errorf("hash should start with sha256: prefix, got %q", hash)
	}
	if len(hash) != 71 { // "sha256:" (7) + 64 hex chars
		t.Errorf("hash length = %d, want 71", len(hash))
	}

	// Same input should produce same hash
	hash2 := HashBytes([]byte("hello world"))
	if hash != hash2 {
		t.Error("same input should produce same hash")
	}

	// Different input should produce different hash
	hash3 := HashBytes([]byte("hello world!"))
	if hash == hash3 {
		t.Error("different input should produce different hash")
	}
}

func TestHashFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	os.WriteFile(path, []byte("hello world"), 0644)

	hash, err := HashFile(path)
	if err != nil {
		t.Fatalf("HashFile() error: %v", err)
	}

	expected := HashBytes([]byte("hello world"))
	if hash != expected {
		t.Errorf("HashFile() = %q, want %q", hash, expected)
	}
}

func TestHashFileNotFound(t *testing.T) {
	_, err := HashFile("/nonexistent/file")
	if err == nil {
		t.Error("HashFile() should error for missing file")
	}
}

func TestHashDir(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.md"), []byte("file a"), 0644)
	os.WriteFile(filepath.Join(dir, "b.md"), []byte("file b"), 0644)

	hash1, err := HashDir(dir)
	if err != nil {
		t.Fatalf("HashDir() error: %v", err)
	}

	if !strings.HasPrefix(hash1, "sha256:") {
		t.Errorf("hash should start with sha256: prefix")
	}

	// Same content should produce same hash
	dir2 := t.TempDir()
	os.WriteFile(filepath.Join(dir2, "a.md"), []byte("file a"), 0644)
	os.WriteFile(filepath.Join(dir2, "b.md"), []byte("file b"), 0644)

	hash2, err := HashDir(dir2)
	if err != nil {
		t.Fatalf("HashDir() error: %v", err)
	}
	if hash1 != hash2 {
		t.Error("identical directories should produce same hash")
	}

	// Modified content should produce different hash
	os.WriteFile(filepath.Join(dir2, "a.md"), []byte("modified"), 0644)
	hash3, err := HashDir(dir2)
	if err != nil {
		t.Fatalf("HashDir() error: %v", err)
	}
	if hash1 == hash3 {
		t.Error("modified directory should produce different hash")
	}
}

func TestHashDirDeterministic(t *testing.T) {
	// Order of file creation shouldn't matter
	dir1 := t.TempDir()
	os.WriteFile(filepath.Join(dir1, "z.md"), []byte("last"), 0644)
	os.WriteFile(filepath.Join(dir1, "a.md"), []byte("first"), 0644)

	dir2 := t.TempDir()
	os.WriteFile(filepath.Join(dir2, "a.md"), []byte("first"), 0644)
	os.WriteFile(filepath.Join(dir2, "z.md"), []byte("last"), 0644)

	hash1, _ := HashDir(dir1)
	hash2, _ := HashDir(dir2)

	if hash1 != hash2 {
		t.Error("directory hash should be deterministic regardless of file creation order")
	}
}
