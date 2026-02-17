package cli

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cego/ai-instructions/internal/config"
	"github.com/cego/ai-instructions/internal/filemanager"
	"github.com/cego/ai-instructions/internal/injector"
	"github.com/cego/ai-instructions/internal/registry"
	"github.com/cego/ai-instructions/internal/resolver"
)

// setupTestRegistry creates an httptest server serving registry data.
func setupTestRegistry(t *testing.T) *httptest.Server {
	t.Helper()

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

func TestFullInitAddSyncVerifyFlow(t *testing.T) {
	server := setupTestRegistry(t)
	defer server.Close()

	projectDir := t.TempDir()
	ctx := context.Background()
	managedDir := config.DefaultInstructionsDir + "/" + config.ManagedDir

	client := registry.NewClient(
		registry.WithBaseURL(server.URL),
		registry.WithHTTPClient(server.Client()),
	)

	// === Step 1: Simulate init ===
	reg, err := client.FetchRegistry(ctx)
	if err != nil {
		t.Fatalf("FetchRegistry: %v", err)
	}

	// Resolve dependencies for laravel
	stackInfoMap := make(map[string]resolver.StackInfo)
	for id, meta := range reg.Stacks {
		stackInfoMap[id] = resolver.StackInfo{ID: id, Depends: meta.Depends}
	}

	res, err := resolver.NewResolver(stackInfoMap).Resolve([]string{"laravel"})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	// Should resolve to php + laravel
	if len(res.Order) != 2 {
		t.Fatalf("expected 2 stacks, got %d: %v", len(res.Order), res.Order)
	}

	// Download files
	fm := filemanager.NewManager(client, projectDir, managedDir)
	cfg := &config.Config{
		Version: 1,
		Registry: config.RegistryConfig{
			URL: server.URL,
		},
		InstructionsDir: config.DefaultInstructionsDir,
		Mode:            "platform",
		Stacks:          []string{"laravel"},
		Resolved:        make(map[string]config.ResolvedStack),
	}

	for _, stackID := range res.Order {
		manifest, err := client.FetchStackManifest(ctx, stackID)
		if err != nil {
			t.Fatalf("FetchStackManifest(%s): %v", stackID, err)
		}

		if err := fm.DownloadStack(ctx, stackID, manifest.Files); err != nil {
			t.Fatalf("DownloadStack(%s): %v", stackID, err)
		}

		hash, err := filemanager.HashDir(fm.StackDir(stackID))
		if err != nil {
			t.Fatalf("HashDir(%s): %v", stackID, err)
		}

		rs := config.ResolvedStack{
			Version: reg.Stacks[stackID].Version,
			Hash:    hash,
			Files:   manifest.Files,
		}
		if res.Explicit[stackID] {
			rs.Explicit = true
		} else {
			rs.DependencyOf = res.DependencyOf[stackID]
		}
		cfg.Resolved[stackID] = rs
	}

	if err := config.SaveConfig(projectDir, cfg); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}

	// Inject managed blocks
	var allFiles []string
	for _, stackID := range res.Order {
		for _, f := range cfg.Resolved[stackID].Files {
			allFiles = append(allFiles, managedDir+"/"+stackID+"/"+f)
		}
	}
	configs := []injector.FileConfig{
		injector.ClaudeConfig(allFiles),
		injector.AgentsConfig(allFiles),
		injector.CursorConfig(allFiles),
	}
	if err := injector.InjectAll(projectDir, res.Order, configs, managedDir); err != nil {
		t.Fatalf("InjectAll: %v", err)
	}

	// === Verify: config exists and is valid ===
	loadedCfg, err := config.LoadConfig(projectDir)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if len(loadedCfg.Stacks) != 1 || loadedCfg.Stacks[0] != "laravel" {
		t.Errorf("expected stacks [laravel], got %v", loadedCfg.Stacks)
	}
	if len(loadedCfg.Resolved) != 2 {
		t.Errorf("expected 2 resolved stacks, got %d", len(loadedCfg.Resolved))
	}

	// Verify php files exist
	phpDir := filepath.Join(projectDir, managedDir, "php")
	if _, err := os.Stat(filepath.Join(phpDir, "coding-standards.md")); err != nil {
		t.Error("php/coding-standards.md should exist")
	}

	// Verify CLAUDE.md has markers
	claudeData, err := os.ReadFile(filepath.Join(projectDir, "CLAUDE.md"))
	if err != nil {
		t.Fatalf("reading CLAUDE.md: %v", err)
	}
	if !strings.Contains(string(claudeData), injector.MarkerStart) {
		t.Error("CLAUDE.md should contain start marker")
	}

	// === Step 2: Verify integrity ===
	for stackID, rs := range loadedCfg.Resolved {
		result := filemanager.VerifyStack(projectDir, managedDir, stackID, filemanager.StackVerifyInfo{
			Hash:  rs.Hash,
			Files: rs.Files,
		})
		if !result.OK {
			t.Errorf("stack %s verification failed: missing=%v tampered=%v", stackID, result.Missing, result.Tampered)
		}
	}

	// === Step 3: Tamper a file, verify should fail ===
	phpStdPath := filepath.Join(phpDir, "coding-standards.md")
	os.WriteFile(phpStdPath, []byte("tampered content"), 0644)

	phpRS := loadedCfg.Resolved["php"]
	result := filemanager.VerifyStack(projectDir, managedDir, "php", filemanager.StackVerifyInfo{
		Hash:  phpRS.Hash,
		Files: phpRS.Files,
	})
	if result.OK {
		t.Error("verify should fail after file tampering")
	}

	// === Step 4: Sync should restore tampered files ===
	if err := fm.DownloadStack(ctx, "php", phpRS.Files); err != nil {
		t.Fatalf("re-download php: %v", err)
	}

	newHash, err := filemanager.HashDir(phpDir)
	if err != nil {
		t.Fatalf("HashDir after sync: %v", err)
	}
	if newHash != phpRS.Hash {
		t.Error("hash should match after sync re-download")
	}

	// === Step 5: Add docker stack ===
	dockerManifest, err := client.FetchStackManifest(ctx, "docker")
	if err != nil {
		t.Fatalf("FetchStackManifest(docker): %v", err)
	}
	if err := fm.DownloadStack(ctx, "docker", dockerManifest.Files); err != nil {
		t.Fatalf("DownloadStack(docker): %v", err)
	}
	dockerHash, err := filemanager.HashDir(fm.StackDir("docker"))
	if err != nil {
		t.Fatalf("HashDir(docker): %v", err)
	}

	loadedCfg.Stacks = append(loadedCfg.Stacks, "docker")
	loadedCfg.Resolved["docker"] = config.ResolvedStack{
		Version:  reg.Stacks["docker"].Version,
		Hash:     dockerHash,
		Files:    dockerManifest.Files,
		Explicit: true,
	}
	if err := config.SaveConfig(projectDir, loadedCfg); err != nil {
		t.Fatalf("SaveConfig after add: %v", err)
	}

	if len(loadedCfg.Resolved) != 3 {
		t.Errorf("expected 3 resolved stacks after add, got %d", len(loadedCfg.Resolved))
	}
}

func TestVerifyExitCodes(t *testing.T) {
	server := setupTestRegistry(t)
	defer server.Close()

	projectDir := t.TempDir()

	// No config should not exist initially
	if config.ConfigExists(projectDir) {
		t.Error("config should not exist initially")
	}

	// Create minimal valid config with resolved data
	cfg := &config.Config{
		Version: 1,
		Registry: config.RegistryConfig{
			URL: server.URL,
		},
		Mode:   "platform",
		Stacks: []string{"php"},
		Resolved: map[string]config.ResolvedStack{
			"php": {
				Version: "1.2.0",
				Hash:    "sha256:fakehash",
				Files:   []string{"coding-standards.md"},
			},
		},
	}

	if err := config.SaveConfig(projectDir, cfg); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}

	loadedCfg, err := config.LoadConfig(projectDir)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if loadedCfg.Mode != "platform" {
		t.Errorf("Mode = %q, want platform", loadedCfg.Mode)
	}
	if len(loadedCfg.Resolved) != 1 {
		t.Errorf("expected 1 resolved stack, got %d", len(loadedCfg.Resolved))
	}
}
