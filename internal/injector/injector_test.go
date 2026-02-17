package injector

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/company/ai-instructions/internal/config"
)

func TestBuildBlock(t *testing.T) {
	instrDir := config.DefaultInstructionsDir
	block := BuildBlock(
		[]string{"php", "laravel"},
		[]string{instrDir + "/php/coding-standards.md", instrDir + "/laravel/conventions.md"},
		instrDir,
	)

	if !strings.Contains(block, MarkerStart) {
		t.Error("block should contain start marker")
	}
	if !strings.Contains(block, MarkerEnd) {
		t.Error("block should contain end marker")
	}
	if !strings.Contains(block, "php, laravel") {
		t.Error("block should list stacks")
	}
	if !strings.Contains(block, "- "+instrDir+"/php/coding-standards.md") {
		t.Error("block should list files")
	}
	if !strings.Contains(block, "`"+instrDir+"/`") {
		t.Error("block should reference the instructions dir")
	}
}

func TestInjectNewFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "CLAUDE.md")

	block := BuildBlock([]string{"php"}, []string{config.DefaultInstructionsDir + "/php/coding-standards.md"}, config.DefaultInstructionsDir)
	err := injectIntoFile(path, block)
	if err != nil {
		t.Fatalf("injectIntoFile() error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, MarkerStart) {
		t.Error("file should contain start marker")
	}
	if !strings.Contains(content, MarkerEnd) {
		t.Error("file should contain end marker")
	}
}

func TestInjectPrependExisting(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "CLAUDE.md")

	existing := "# My Project\n\nSome existing content.\n"
	os.WriteFile(path, []byte(existing), 0644)

	block := BuildBlock([]string{"php"}, []string{config.DefaultInstructionsDir + "/php/coding-standards.md"}, config.DefaultInstructionsDir)
	err := injectIntoFile(path, block)
	if err != nil {
		t.Fatalf("injectIntoFile() error: %v", err)
	}

	data, _ := os.ReadFile(path)
	content := string(data)

	// Should have markers before existing content
	startIdx := strings.Index(content, MarkerStart)
	existingIdx := strings.Index(content, "# My Project")
	if startIdx > existingIdx {
		t.Error("managed block should be prepended before existing content")
	}

	// Existing content must be preserved
	if !strings.Contains(content, "Some existing content.") {
		t.Error("existing content should be preserved")
	}
}

func TestInjectUpdateBlock(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "CLAUDE.md")

	// Write initial file with markers
	initial := MarkerStart + "\nold content\n" + MarkerEnd + "\n\n# My Project\n"
	os.WriteFile(path, []byte(initial), 0644)

	block := BuildBlock([]string{"php", "laravel"}, []string{
		config.DefaultInstructionsDir + "/php/coding-standards.md",
		config.DefaultInstructionsDir + "/laravel/conventions.md",
	}, config.DefaultInstructionsDir)
	err := injectIntoFile(path, block)
	if err != nil {
		t.Fatalf("injectIntoFile() error: %v", err)
	}

	data, _ := os.ReadFile(path)
	content := string(data)

	// Old content between markers should be replaced
	if strings.Contains(content, "old content") {
		t.Error("old content between markers should be replaced")
	}

	// New content should be present
	if !strings.Contains(content, "php, laravel") {
		t.Error("new content should be present")
	}

	// Content after markers should be preserved
	if !strings.Contains(content, "# My Project") {
		t.Error("content after markers should be preserved")
	}
}

func TestInjectIdempotent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "CLAUDE.md")

	block := BuildBlock([]string{"php"}, []string{config.DefaultInstructionsDir + "/php/coding-standards.md"}, config.DefaultInstructionsDir)

	// Inject twice
	injectIntoFile(path, block)
	injectIntoFile(path, block)

	data, _ := os.ReadFile(path)
	content := string(data)

	// Should only have one pair of markers
	startCount := strings.Count(content, MarkerStart)
	endCount := strings.Count(content, MarkerEnd)
	if startCount != 1 {
		t.Errorf("start marker count = %d, want 1", startCount)
	}
	if endCount != 1 {
		t.Errorf("end marker count = %d, want 1", endCount)
	}
}

func TestVerifyFile(t *testing.T) {
	dir := t.TempDir()

	// File doesn't exist
	result := VerifyFile(filepath.Join(dir, "CLAUDE.md"), "CLAUDE.md")
	if result.HasBlock || result.Exists {
		t.Error("non-existent file should have HasBlock=false, Exists=false")
	}

	// File exists but no markers
	path := filepath.Join(dir, "CLAUDE.md")
	os.WriteFile(path, []byte("# My Project\n"), 0644)
	result = VerifyFile(path, "CLAUDE.md")
	if result.HasBlock {
		t.Error("file without markers should have HasBlock=false")
	}
	if !result.Exists {
		t.Error("existing file should have Exists=true")
	}

	// File exists with markers
	content := MarkerStart + "\ncontent\n" + MarkerEnd
	os.WriteFile(path, []byte(content), 0644)
	result = VerifyFile(path, "CLAUDE.md")
	if !result.HasBlock {
		t.Error("file with markers should have HasBlock=true")
	}
}

func TestInjectAll(t *testing.T) {
	dir := t.TempDir()

	configs := []FileConfig{
		ClaudeConfig([]string{config.DefaultInstructionsDir + "/php/coding-standards.md"}),
		AgentsConfig([]string{config.DefaultInstructionsDir + "/php/coding-standards.md"}),
		CursorConfig([]string{config.DefaultInstructionsDir + "/php/coding-standards.md"}),
	}

	err := InjectAll(dir, []string{"php"}, configs, config.DefaultInstructionsDir)
	if err != nil {
		t.Fatalf("InjectAll() error: %v", err)
	}

	// Check all files were created
	for _, cfg := range configs {
		path := filepath.Join(dir, cfg.Filename)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("file %s should exist", cfg.Filename)
		}
	}
}
