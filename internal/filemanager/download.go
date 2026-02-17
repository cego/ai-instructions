package filemanager

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cego/ai-instructions/internal/registry"
)

// validatePathComponent rejects path components that could escape the intended directory.
func validatePathComponent(name, label string) error {
	if name == "" {
		return fmt.Errorf("empty %s", label)
	}
	cleaned := filepath.Clean(name)
	if cleaned != name || strings.Contains(cleaned, "..") || filepath.IsAbs(cleaned) {
		return fmt.Errorf("invalid %s: %q", label, name)
	}
	return nil
}

// validateInsideDir checks that resolved is a child of base after symlink-safe cleaning.
func validateInsideDir(base, resolved string) error {
	absBase, err := filepath.Abs(base)
	if err != nil {
		return err
	}
	absResolved, err := filepath.Abs(resolved)
	if err != nil {
		return err
	}
	if !strings.HasPrefix(absResolved, absBase+string(filepath.Separator)) && absResolved != absBase {
		return fmt.Errorf("path %q escapes base directory %q", resolved, base)
	}
	return nil
}

// Manager handles downloading and managing instruction files.
type Manager struct {
	client          *registry.Client
	projectDir      string
	instructionsDir string
}

// NewManager creates a new file manager.
func NewManager(client *registry.Client, projectDir, instructionsDir string) *Manager {
	return &Manager{
		client:          client,
		projectDir:      projectDir,
		instructionsDir: instructionsDir,
	}
}

// InstructionsDir returns the path to the instructions directory.
func (m *Manager) InstructionsDir() string {
	return filepath.Join(m.projectDir, m.instructionsDir)
}

// StackDir returns the path to a specific stack's directory.
func (m *Manager) StackDir(stackID string) string {
	return filepath.Join(m.InstructionsDir(), stackID)
}

// EnsureDir creates the instructions directory if it doesn't exist.
func (m *Manager) EnsureDir() error {
	return os.MkdirAll(m.InstructionsDir(), 0755)
}

// DownloadStack downloads all files for a single stack.
func (m *Manager) DownloadStack(ctx context.Context, stackID string, files []string) error {
	if err := validatePathComponent(stackID, "stack ID"); err != nil {
		return err
	}

	stackDir := m.StackDir(stackID)
	if err := validateInsideDir(m.InstructionsDir(), stackDir); err != nil {
		return fmt.Errorf("invalid stack path: %w", err)
	}

	// Clear existing stack directory to remove stale files from previous versions
	os.RemoveAll(stackDir)
	if err := os.MkdirAll(stackDir, 0755); err != nil {
		return fmt.Errorf("creating stack dir %s: %w", stackID, err)
	}

	for _, filename := range files {
		if err := validatePathComponent(filename, "filename"); err != nil {
			return err
		}

		filePath := filepath.Join(stackDir, filename)
		if err := validateInsideDir(stackDir, filePath); err != nil {
			return fmt.Errorf("invalid file path: %w", err)
		}

		data, err := m.client.DownloadFile(ctx, stackID, filename)
		if err != nil {
			return fmt.Errorf("downloading %s/%s: %w", stackID, filename, err)
		}

		tmpPath := filePath + ".tmp"

		if err := os.WriteFile(tmpPath, data, 0644); err != nil {
			return fmt.Errorf("writing %s/%s: %w", stackID, filename, err)
		}

		if err := os.Rename(tmpPath, filePath); err != nil {
			os.Remove(tmpPath)
			return fmt.Errorf("saving %s/%s: %w", stackID, filename, err)
		}
	}

	return nil
}

// DownloadStacks downloads files for multiple stacks.
func (m *Manager) DownloadStacks(ctx context.Context, stacks map[string][]string) error {
	for stackID := range stacks {
		if err := validatePathComponent(stackID, "stack ID"); err != nil {
			return err
		}
	}

	if err := m.EnsureDir(); err != nil {
		return err
	}

	for stackID, files := range stacks {
		if err := m.DownloadStack(ctx, stackID, files); err != nil {
			return err
		}
	}

	return nil
}
