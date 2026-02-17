package filemanager

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/company/ai-instructions/internal/config"
)

func TestVerifyStack(t *testing.T) {
	dir := t.TempDir()
	stackDir := filepath.Join(dir, config.DefaultInstructionsDir, "php")
	os.MkdirAll(stackDir, 0755)

	// Write test files
	os.WriteFile(filepath.Join(stackDir, "coding-standards.md"), []byte("# PHP Standards"), 0644)
	os.WriteFile(filepath.Join(stackDir, "testing.md"), []byte("# PHP Testing"), 0644)

	// Compute correct hash
	hash, err := HashDir(stackDir)
	if err != nil {
		t.Fatal(err)
	}

	// Verify should pass
	result := VerifyStack(dir, config.DefaultInstructionsDir, "php", StackVerifyInfo{
		Hash:  hash,
		Files: []string{"coding-standards.md", "testing.md"},
	})
	if !result.OK {
		t.Errorf("VerifyStack should be OK, missing=%v tampered=%v", result.Missing, result.Tampered)
	}

	// Tamper a file
	os.WriteFile(filepath.Join(stackDir, "coding-standards.md"), []byte("tampered"), 0644)
	result = VerifyStack(dir, config.DefaultInstructionsDir, "php", StackVerifyInfo{
		Hash:  hash,
		Files: []string{"coding-standards.md", "testing.md"},
	})
	if result.OK {
		t.Error("VerifyStack should fail after tampering")
	}
	if len(result.Tampered) == 0 {
		t.Error("should report tampered files")
	}
}

func TestVerifyStackMissingFiles(t *testing.T) {
	dir := t.TempDir()
	stackDir := filepath.Join(dir, config.DefaultInstructionsDir, "php")
	os.MkdirAll(stackDir, 0755)

	// Only write one of two expected files
	os.WriteFile(filepath.Join(stackDir, "coding-standards.md"), []byte("content"), 0644)

	result := VerifyStack(dir, config.DefaultInstructionsDir, "php", StackVerifyInfo{
		Hash:  "sha256:whatever",
		Files: []string{"coding-standards.md", "testing.md"},
	})

	if result.OK {
		t.Error("should fail with missing file")
	}
	if len(result.Missing) != 1 || result.Missing[0] != "testing.md" {
		t.Errorf("Missing = %v, want [testing.md]", result.Missing)
	}
}
