package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSaveAndLoadConfig(t *testing.T) {
	dir := t.TempDir()

	original := &Config{
		Version: 1,
		Registry: RegistryConfig{
			URL:    "https://ai-ctx.example.com",
			Branch: "main",
		},
		InstructionsDir: DefaultInstructionsDir,
		Mode:            "platform",
		Stacks:          []string{"laravel", "nuxt-ui"},
	}

	if err := SaveConfig(dir, original); err != nil {
		t.Fatalf("SaveConfig() error: %v", err)
	}

	loaded, err := LoadConfig(dir)
	if err != nil {
		t.Fatalf("LoadConfig() error: %v", err)
	}

	if loaded.Version != original.Version {
		t.Errorf("Version = %d, want %d", loaded.Version, original.Version)
	}
	if loaded.Registry.URL != original.Registry.URL {
		t.Errorf("Registry.URL = %q, want %q", loaded.Registry.URL, original.Registry.URL)
	}
	if loaded.Registry.Branch != original.Registry.Branch {
		t.Errorf("Registry.Branch = %q, want %q", loaded.Registry.Branch, original.Registry.Branch)
	}
	if loaded.Mode != original.Mode {
		t.Errorf("Mode = %q, want %q", loaded.Mode, original.Mode)
	}
	if len(loaded.Stacks) != len(original.Stacks) {
		t.Errorf("Stacks len = %d, want %d", len(loaded.Stacks), len(original.Stacks))
	}
	if loaded.InstructionsDir != DefaultInstructionsDir {
		t.Errorf("InstructionsDir = %q, want %q", loaded.InstructionsDir, DefaultInstructionsDir)
	}
}

func TestConfigDefaults(t *testing.T) {
	dir := t.TempDir()

	// Save with empty defaults
	original := &Config{
		Version: 1,
		Registry: RegistryConfig{
			URL: "https://ai-ctx.example.com",
		},
		Stacks: []string{"php"},
	}

	if err := SaveConfig(dir, original); err != nil {
		t.Fatalf("SaveConfig() error: %v", err)
	}

	loaded, err := LoadConfig(dir)
	if err != nil {
		t.Fatalf("LoadConfig() error: %v", err)
	}

	// Defaults should be applied
	if loaded.InstructionsDir != DefaultInstructionsDir {
		t.Errorf("InstructionsDir = %q, want %q", loaded.InstructionsDir, DefaultInstructionsDir)
	}
	if loaded.Mode != "platform" {
		t.Errorf("Mode = %q, want %q", loaded.Mode, "platform")
	}
	if loaded.Registry.Branch != "master" {
		t.Errorf("Registry.Branch = %q, want %q", loaded.Registry.Branch, "master")
	}
}

func TestSaveAndLoadCustomInstructionsDir(t *testing.T) {
	dir := t.TempDir()

	original := &Config{
		Version: 1,
		Registry: RegistryConfig{
			URL: "https://ai-ctx.example.com",
		},
		InstructionsDir: "custom-instructions",
		Mode:            "platform",
		Stacks:          []string{"php"},
	}

	if err := SaveConfig(dir, original); err != nil {
		t.Fatalf("SaveConfig() error: %v", err)
	}

	loaded, err := LoadConfig(dir)
	if err != nil {
		t.Fatalf("LoadConfig() error: %v", err)
	}

	if loaded.InstructionsDir != "custom-instructions" {
		t.Errorf("InstructionsDir = %q, want %q", loaded.InstructionsDir, "custom-instructions")
	}
}

func TestLoadConfigNotFound(t *testing.T) {
	dir := t.TempDir()
	_, err := LoadConfig(dir)
	if err == nil {
		t.Fatal("LoadConfig() should return error for missing file")
	}
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		c       *Config
		wantErr bool
	}{
		{
			name:    "valid",
			c:       &Config{Version: 1, Registry: RegistryConfig{URL: "https://example.com"}, Stacks: []string{"php"}},
			wantErr: false,
		},
		{
			name:    "no version",
			c:       &Config{Version: 0, Registry: RegistryConfig{URL: "https://example.com"}, Stacks: []string{"php"}},
			wantErr: true,
		},
		{
			name:    "no registry url",
			c:       &Config{Version: 1, Registry: RegistryConfig{URL: ""}, Stacks: []string{"php"}},
			wantErr: true,
		},
		{
			name:    "no stacks",
			c:       &Config{Version: 1, Registry: RegistryConfig{URL: "https://example.com"}, Stacks: []string{}},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConfig(tt.c)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfigExists(t *testing.T) {
	dir := t.TempDir()

	if ConfigExists(dir) {
		t.Error("ConfigExists() should return false for empty dir")
	}

	os.WriteFile(filepath.Join(dir, ConfigFile), []byte("{}"), 0644)

	if !ConfigExists(dir) {
		t.Error("ConfigExists() should return true when file exists")
	}
}

func TestSaveAndLoadConfigWithResolved(t *testing.T) {
	dir := t.TempDir()

	original := &Config{
		Version: 1,
		Registry: RegistryConfig{
			URL:    "https://ai-ctx.example.com",
			Branch: "main",
		},
		InstructionsDir: DefaultInstructionsDir,
		Mode:            "platform",
		Stacks:          []string{"laravel"},
		Resolved: map[string]ResolvedStack{
			"php": {
				Version:      "1.2.0",
				Hash:         "sha256:abc123",
				Files:        []string{"coding-standards.md", "testing.md"},
				DependencyOf: "laravel",
			},
			"laravel": {
				Version:  "1.4.0",
				Hash:     "sha256:def456",
				Files:    []string{"conventions.md", "eloquent.md"},
				Explicit: true,
			},
		},
	}

	if err := SaveConfig(dir, original); err != nil {
		t.Fatalf("SaveConfig() error: %v", err)
	}

	loaded, err := LoadConfig(dir)
	if err != nil {
		t.Fatalf("LoadConfig() error: %v", err)
	}

	if len(loaded.Resolved) != 2 {
		t.Fatalf("Resolved len = %d, want 2", len(loaded.Resolved))
	}

	php := loaded.Resolved["php"]
	if php.Version != "1.2.0" {
		t.Errorf("php.Version = %q, want %q", php.Version, "1.2.0")
	}
	if php.DependencyOf != "laravel" {
		t.Errorf("php.DependencyOf = %q, want %q", php.DependencyOf, "laravel")
	}
	if php.Explicit {
		t.Error("php.Explicit should be false")
	}

	laravel := loaded.Resolved["laravel"]
	if !laravel.Explicit {
		t.Error("laravel.Explicit should be true")
	}
}

func TestSaveConfigWithoutResolved(t *testing.T) {
	dir := t.TempDir()

	original := &Config{
		Version: 1,
		Registry: RegistryConfig{
			URL: "https://ai-ctx.example.com",
		},
		Stacks: []string{"php"},
	}

	if err := SaveConfig(dir, original); err != nil {
		t.Fatalf("SaveConfig() error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, ConfigFile))
	if err != nil {
		t.Fatalf("reading config: %v", err)
	}

	// Should not contain resolved section or separator comment
	if strings.Contains(string(data), "resolved:") {
		t.Error("config without resolved should not contain 'resolved:' key")
	}
	if strings.Contains(string(data), "auto-generated") {
		t.Error("config without resolved should not contain separator comment")
	}
}

func TestSaveConfigHasDocumentStart(t *testing.T) {
	dir := t.TempDir()

	original := &Config{
		Version: 1,
		Registry: RegistryConfig{
			URL: "https://ai-ctx.example.com",
		},
		Stacks: []string{"php"},
	}

	if err := SaveConfig(dir, original); err != nil {
		t.Fatalf("SaveConfig() error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, ConfigFile))
	if err != nil {
		t.Fatalf("reading config: %v", err)
	}

	if !strings.HasPrefix(string(data), "---\n") {
		t.Error("config should start with YAML document start marker '---'")
	}
}

func TestSaveConfigHasCommentSeparator(t *testing.T) {
	dir := t.TempDir()

	original := &Config{
		Version: 1,
		Registry: RegistryConfig{
			URL: "https://ai-ctx.example.com",
		},
		Stacks: []string{"php"},
		Resolved: map[string]ResolvedStack{
			"php": {
				Version:  "1.0.0",
				Hash:     "sha256:abc",
				Files:    []string{"coding-standards.md"},
				Explicit: true,
			},
		},
	}

	if err := SaveConfig(dir, original); err != nil {
		t.Fatalf("SaveConfig() error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, ConfigFile))
	if err != nil {
		t.Fatalf("reading config: %v", err)
	}

	if !strings.Contains(string(data), "# Resolved dependencies") {
		t.Error("config with resolved should contain separator comment")
	}
	if !strings.Contains(string(data), "auto-generated, do not edit") {
		t.Error("config with resolved should contain do-not-edit warning")
	}
}
