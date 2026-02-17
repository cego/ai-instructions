package registry

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupTestServer(t *testing.T) *httptest.Server {
	t.Helper()

	// Find testdata directory
	testdataDir := filepath.Join("..", "..", "testdata", "registry")

	mux := http.NewServeMux()
	mux.HandleFunc("/company-instructions/", func(w http.ResponseWriter, r *http.Request) {
		relPath := r.URL.Path[len("/company-instructions/"):]
		data, err := os.ReadFile(filepath.Join(testdataDir, "company-instructions", relPath))
		if err != nil {
			http.Error(w, "not found", 404)
			return
		}
		if filepath.Ext(relPath) == ".json" {
			w.Header().Set("Content-Type", "application/json")
		}
		w.Write(data)
	})

	return httptest.NewServer(mux)
}

func TestFetchRegistry(t *testing.T) {
	server := setupTestServer(t)
	defer server.Close()

	client := NewClient(
		WithBaseURL(server.URL),
		WithHTTPClient(server.Client()),
	)

	ctx := context.Background()
	reg, err := client.FetchRegistry(ctx)
	if err != nil {
		t.Fatalf("FetchRegistry() error: %v", err)
	}

	if reg.Version != 1 {
		t.Errorf("Version = %d, want 1", reg.Version)
	}

	if _, ok := reg.Stacks["php"]; !ok {
		t.Error("registry should contain php stack")
	}

	if _, ok := reg.Stacks["laravel"]; !ok {
		t.Error("registry should contain laravel stack")
	}

	// Verify caching - second call should use cache
	reg2, err := client.FetchRegistry(ctx)
	if err != nil {
		t.Fatalf("cached FetchRegistry() error: %v", err)
	}
	if reg2 != reg {
		t.Error("second call should return cached registry")
	}
}

func TestFetchStackManifest(t *testing.T) {
	server := setupTestServer(t)
	defer server.Close()

	client := NewClient(
		WithBaseURL(server.URL),
		WithHTTPClient(server.Client()),
	)

	ctx := context.Background()
	manifest, err := client.FetchStackManifest(ctx, "laravel")
	if err != nil {
		t.Fatalf("FetchStackManifest() error: %v", err)
	}

	if manifest.Name != "Laravel" {
		t.Errorf("Name = %q, want %q", manifest.Name, "Laravel")
	}

	if len(manifest.Files) != 4 {
		t.Errorf("Files len = %d, want 4", len(manifest.Files))
	}

	if len(manifest.Depends) != 1 || manifest.Depends[0] != "php" {
		t.Errorf("Depends = %v, want [php]", manifest.Depends)
	}
}

func TestDownloadFile(t *testing.T) {
	server := setupTestServer(t)
	defer server.Close()

	client := NewClient(
		WithBaseURL(server.URL),
		WithHTTPClient(server.Client()),
	)

	ctx := context.Background()
	data, err := client.DownloadFile(ctx, "php", "coding-standards.md")
	if err != nil {
		t.Fatalf("DownloadFile() error: %v", err)
	}

	if len(data) == 0 {
		t.Error("downloaded file should not be empty")
	}
}

func TestAuthToken(t *testing.T) {
	var receivedToken string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedToken = r.Header.Get("PRIVATE-TOKEN")
		reg := Registry{Version: 1, Stacks: map[string]StackMeta{}}
		json.NewEncoder(w).Encode(reg)
	}))
	defer server.Close()

	client := NewClient(
		WithBaseURL(server.URL),
		WithToken("test-token-123"),
		WithHTTPClient(server.Client()),
	)

	client.FetchRegistry(context.Background())

	if receivedToken != "test-token-123" {
		t.Errorf("token = %q, want %q", receivedToken, "test-token-123")
	}
}

func TestHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", 404)
	}))
	defer server.Close()

	client := NewClient(
		WithBaseURL(server.URL),
		WithHTTPClient(server.Client()),
	)

	_, err := client.FetchRegistry(context.Background())
	if err == nil {
		t.Error("should return error for 404")
	}
}

func TestResponseSizeLimit(t *testing.T) {
	// Serve a response larger than maxResponseSize
	oversized := strings.Repeat("x", maxResponseSize+1024)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(oversized))
	}))
	defer server.Close()

	client := NewClient(
		WithBaseURL(server.URL),
		WithHTTPClient(server.Client()),
	)

	data, err := client.DownloadFile(context.Background(), "php", "huge.md")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(data) > maxResponseSize {
		t.Errorf("response size = %d, want <= %d", len(data), maxResponseSize)
	}
}
