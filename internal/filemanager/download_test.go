package filemanager

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/company/ai-instructions/internal/config"
	"github.com/company/ai-instructions/internal/registry"
)

func TestDownloadStack(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/company-instructions/php/coding-standards.md":
			w.Write([]byte("# PHP Standards"))
		case "/company-instructions/php/testing.md":
			w.Write([]byte("# PHP Testing"))
		default:
			http.Error(w, "not found", 404)
		}
	}))
	defer server.Close()

	client := registry.NewClient(
		registry.WithBaseURL(server.URL),
		registry.WithHTTPClient(server.Client()),
	)

	dir := t.TempDir()
	fm := NewManager(client, dir, config.DefaultInstructionsDir)

	ctx := context.Background()
	err := fm.DownloadStack(ctx, "php", []string{"coding-standards.md", "testing.md"})
	if err != nil {
		t.Fatalf("DownloadStack() error: %v", err)
	}

	// Verify files exist
	data, err := os.ReadFile(filepath.Join(dir, config.DefaultInstructionsDir, "php", "coding-standards.md"))
	if err != nil {
		t.Fatalf("file should exist: %v", err)
	}
	if string(data) != "# PHP Standards" {
		t.Errorf("content = %q, want %q", string(data), "# PHP Standards")
	}
}

func TestDownloadStacks(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("content of " + r.URL.Path))
	}))
	defer server.Close()

	client := registry.NewClient(
		registry.WithBaseURL(server.URL),
		registry.WithHTTPClient(server.Client()),
	)

	dir := t.TempDir()
	fm := NewManager(client, dir, config.DefaultInstructionsDir)

	ctx := context.Background()
	err := fm.DownloadStacks(ctx, map[string][]string{
		"php":     {"coding-standards.md"},
		"laravel": {"conventions.md"},
	})
	if err != nil {
		t.Fatalf("DownloadStacks() error: %v", err)
	}

	// Verify both stack dirs exist
	if _, err := os.Stat(filepath.Join(dir, config.DefaultInstructionsDir, "php", "coding-standards.md")); err != nil {
		t.Error("php/coding-standards.md should exist")
	}
	if _, err := os.Stat(filepath.Join(dir, config.DefaultInstructionsDir, "laravel", "conventions.md")); err != nil {
		t.Error("laravel/conventions.md should exist")
	}
}

func TestDownloadStack_PathTraversal(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("malicious content"))
	}))
	defer server.Close()

	client := registry.NewClient(
		registry.WithBaseURL(server.URL),
		registry.WithHTTPClient(server.Client()),
	)

	tests := []struct {
		name     string
		stackID  string
		files    []string
		wantErr  string
	}{
		{
			name:    "stack ID with path traversal",
			stackID: "../../etc",
			files:   []string{"passwd"},
			wantErr: "invalid stack ID",
		},
		{
			name:    "stack ID with parent dir",
			stackID: "..",
			files:   []string{"file.md"},
			wantErr: "invalid stack ID",
		},
		{
			name:    "filename with path traversal",
			stackID: "php",
			files:   []string{"../../../.bashrc"},
			wantErr: "invalid filename",
		},
		{
			name:    "filename with parent dir",
			stackID: "php",
			files:   []string{"../secret.md"},
			wantErr: "invalid filename",
		},
		{
			name:    "empty stack ID",
			stackID: "",
			files:   []string{"file.md"},
			wantErr: "empty stack ID",
		},
		{
			name:    "empty filename",
			stackID: "php",
			files:   []string{""},
			wantErr: "empty filename",
		},
		{
			name:    "absolute stack ID",
			stackID: "/etc",
			files:   []string{"passwd"},
			wantErr: "invalid stack ID",
		},
		{
			name:    "absolute filename",
			stackID: "php",
			files:   []string{"/etc/passwd"},
			wantErr: "invalid filename",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			fm := NewManager(client, dir, config.DefaultInstructionsDir)

			err := fm.DownloadStack(context.Background(), tt.stackID, tt.files)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %q, want containing %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestDownloadStacks_PathTraversal(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("malicious content"))
	}))
	defer server.Close()

	client := registry.NewClient(
		registry.WithBaseURL(server.URL),
		registry.WithHTTPClient(server.Client()),
	)

	dir := t.TempDir()
	fm := NewManager(client, dir, config.DefaultInstructionsDir)

	err := fm.DownloadStacks(context.Background(), map[string][]string{
		"../../etc": {"passwd"},
	})
	if err == nil {
		t.Fatal("expected error for path traversal stack ID")
	}
	if !strings.Contains(err.Error(), "invalid stack ID") {
		t.Errorf("error = %q, want containing 'invalid stack ID'", err.Error())
	}
}
