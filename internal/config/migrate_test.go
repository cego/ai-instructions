package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMigrateFromOldSettings(t *testing.T) {
	dir := t.TempDir()

	oldContent := `version: 1
mode: platform
instructions_dir: company-instructions
registry_url: https://ai-ctx.example.com
branch: main
stacks:
    - laravel
    - nuxt
resolved:
    php:
        version: "1.2.0"
        hash: "sha256:abc123"
        files:
            - coding-standards.md
            - testing.md
        dependency_of: laravel
    laravel:
        version: "1.4.0"
        hash: "sha256:def456"
        files:
            - conventions.md
            - eloquent.md
        explicit: true
`

	if err := os.WriteFile(filepath.Join(dir, OldSettingsFile), []byte(oldContent), 0644); err != nil {
		t.Fatalf("writing old settings: %v", err)
	}

	cfg, err := MigrateFromOldSettings(dir)
	if err != nil {
		t.Fatalf("MigrateFromOldSettings() error: %v", err)
	}

	// Verify config
	if cfg.Version != 1 {
		t.Errorf("Config.Version = %d, want 1", cfg.Version)
	}
	if cfg.Registry.URL != "https://ai-ctx.example.com" {
		t.Errorf("Config.Registry.URL = %q, want %q", cfg.Registry.URL, "https://ai-ctx.example.com")
	}
	if cfg.Registry.Branch != "main" {
		t.Errorf("Config.Registry.Branch = %q, want %q", cfg.Registry.Branch, "main")
	}
	if cfg.Mode != "platform" {
		t.Errorf("Config.Mode = %q, want %q", cfg.Mode, "platform")
	}
	if cfg.InstructionsDir != "company-instructions" {
		t.Errorf("Config.InstructionsDir = %q, want %q", cfg.InstructionsDir, "company-instructions")
	}
	if len(cfg.Stacks) != 2 {
		t.Errorf("Config.Stacks len = %d, want 2", len(cfg.Stacks))
	}

	// Verify resolved (now part of config)
	if len(cfg.Resolved) != 2 {
		t.Fatalf("Config.Resolved len = %d, want 2", len(cfg.Resolved))
	}

	php := cfg.Resolved["php"]
	if php.Version != "1.2.0" {
		t.Errorf("php.Version = %q, want %q", php.Version, "1.2.0")
	}
	if php.DependencyOf != "laravel" {
		t.Errorf("php.DependencyOf = %q, want %q", php.DependencyOf, "laravel")
	}

	laravel := cfg.Resolved["laravel"]
	if !laravel.Explicit {
		t.Error("laravel.Explicit should be true")
	}
}

func TestMigrateFromOldSettingsDefaults(t *testing.T) {
	dir := t.TempDir()

	// Minimal old settings with missing optional fields
	oldContent := `version: 1
registry_url: https://ai-ctx.example.com
stacks:
    - php
resolved:
    php:
        version: "1.0.0"
        hash: "sha256:abc"
        files:
            - coding-standards.md
        explicit: true
`

	if err := os.WriteFile(filepath.Join(dir, OldSettingsFile), []byte(oldContent), 0644); err != nil {
		t.Fatalf("writing old settings: %v", err)
	}

	cfg, err := MigrateFromOldSettings(dir)
	if err != nil {
		t.Fatalf("MigrateFromOldSettings() error: %v", err)
	}

	if cfg.InstructionsDir != DefaultInstructionsDir {
		t.Errorf("InstructionsDir = %q, want %q", cfg.InstructionsDir, DefaultInstructionsDir)
	}
	if cfg.Mode != "platform" {
		t.Errorf("Mode = %q, want %q", cfg.Mode, "platform")
	}
	if cfg.Registry.Branch != "master" {
		t.Errorf("Registry.Branch = %q, want %q", cfg.Registry.Branch, "master")
	}
}

func TestOldSettingsExists(t *testing.T) {
	dir := t.TempDir()

	if OldSettingsExists(dir) {
		t.Error("OldSettingsExists() should return false for empty dir")
	}

	os.WriteFile(filepath.Join(dir, OldSettingsFile), []byte("{}"), 0644)

	if !OldSettingsExists(dir) {
		t.Error("OldSettingsExists() should return true when file exists")
	}
}

func TestMigrateNotFound(t *testing.T) {
	dir := t.TempDir()
	_, err := MigrateFromOldSettings(dir)
	if err == nil {
		t.Fatal("MigrateFromOldSettings() should return error for missing file")
	}
}

func TestOldLockfileExists(t *testing.T) {
	dir := t.TempDir()

	if OldLockfileExists(dir) {
		t.Error("OldLockfileExists() should return false for empty dir")
	}

	os.WriteFile(filepath.Join(dir, LockFile), []byte("{}"), 0644)

	if !OldLockfileExists(dir) {
		t.Error("OldLockfileExists() should return true when file exists")
	}
}

func TestAbsorbLockfile(t *testing.T) {
	dir := t.TempDir()

	lockContent := `version: 1
resolved:
    php:
        version: "1.2.0"
        hash: "sha256:abc123"
        files:
            - coding-standards.md
        explicit: true
    laravel:
        version: "1.4.0"
        hash: "sha256:def456"
        files:
            - conventions.md
        dependency_of: php
`

	if err := os.WriteFile(filepath.Join(dir, LockFile), []byte(lockContent), 0644); err != nil {
		t.Fatalf("writing lockfile: %v", err)
	}

	cfg := &Config{
		Version: 1,
		Registry: RegistryConfig{
			URL: "https://ai-ctx.example.com",
		},
		Stacks: []string{"php"},
	}

	if err := AbsorbLockfile(dir, cfg); err != nil {
		t.Fatalf("AbsorbLockfile() error: %v", err)
	}

	if len(cfg.Resolved) != 2 {
		t.Fatalf("Resolved len = %d, want 2", len(cfg.Resolved))
	}

	php := cfg.Resolved["php"]
	if php.Version != "1.2.0" {
		t.Errorf("php.Version = %q, want %q", php.Version, "1.2.0")
	}
	if !php.Explicit {
		t.Error("php.Explicit should be true")
	}

	laravel := cfg.Resolved["laravel"]
	if laravel.DependencyOf != "php" {
		t.Errorf("laravel.DependencyOf = %q, want %q", laravel.DependencyOf, "php")
	}
}

func TestAbsorbLockfileNotFound(t *testing.T) {
	dir := t.TempDir()

	cfg := &Config{
		Version: 1,
		Registry: RegistryConfig{
			URL: "https://ai-ctx.example.com",
		},
		Stacks: []string{"php"},
	}

	err := AbsorbLockfile(dir, cfg)
	if err == nil {
		t.Fatal("AbsorbLockfile() should return error for missing file")
	}
}
