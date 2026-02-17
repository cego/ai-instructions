package filemanager

import (
	"fmt"
	"os"
	"path/filepath"
)

// CleanupStaleStacks removes stack directories that are no longer in the resolved set.
func CleanupStaleStacks(projectDir, instructionsDir string, resolved map[string]bool) ([]string, error) {
	instrDir := filepath.Join(projectDir, instructionsDir)
	entries, err := os.ReadDir(instrDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading %s dir: %w", instructionsDir, err)
	}

	var removed []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if !resolved[entry.Name()] {
			path := filepath.Join(instrDir, entry.Name())
			if err := os.RemoveAll(path); err != nil {
				return removed, fmt.Errorf("removing stale stack %s: %w", entry.Name(), err)
			}
			removed = append(removed, entry.Name())
		}
	}

	return removed, nil
}

// RemoveStack removes a single stack directory.
func RemoveStack(projectDir, instructionsDir, stackID string) error {
	path := filepath.Join(projectDir, instructionsDir, stackID)
	if err := os.RemoveAll(path); err != nil {
		return fmt.Errorf("removing stack %s: %w", stackID, err)
	}
	return nil
}
