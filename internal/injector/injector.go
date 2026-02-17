package injector

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	MarkerStart = "<!-- AI-INSTRUCTIONS:START — managed by ai-instructions, do not edit -->"
	MarkerEnd   = "<!-- AI-INSTRUCTIONS:END -->"
)

// FileConfig describes which files to inject into and what content to include.
type FileConfig struct {
	Filename string
	Files    []string // relative paths like "ai-instructions/php/coding-standards.md"
}

// InjectAll injects managed blocks into all target files.
func InjectAll(projectDir string, stacks []string, configs []FileConfig, instructionsDir string) error {
	for _, cfg := range configs {
		block := BuildBlock(stacks, cfg.Files, instructionsDir)
		if err := injectIntoFile(filepath.Join(projectDir, cfg.Filename), block); err != nil {
			return fmt.Errorf("injecting into %s: %w", cfg.Filename, err)
		}
	}
	return nil
}

// VerifyAll checks that all target files contain the managed block.
func VerifyAll(projectDir string, configs []FileConfig) []VerifyResult {
	var results []VerifyResult
	for _, cfg := range configs {
		path := filepath.Join(projectDir, cfg.Filename)
		result := VerifyFile(path, cfg.Filename)
		results = append(results, result)
	}
	return results
}

// VerifyResult contains the verification result for a single file.
type VerifyResult struct {
	Filename string
	HasBlock bool
	Exists   bool
}

// VerifyFile checks if a file contains the managed block markers.
func VerifyFile(path, filename string) VerifyResult {
	data, err := os.ReadFile(path)
	if err != nil {
		return VerifyResult{Filename: filename, HasBlock: false, Exists: false}
	}
	content := string(data)
	hasStart := strings.Contains(content, MarkerStart)
	hasEnd := strings.Contains(content, MarkerEnd)
	return VerifyResult{Filename: filename, HasBlock: hasStart && hasEnd, Exists: true}
}

// BuildBlock generates the managed content block.
func BuildBlock(stacks []string, files []string, instructionsDir string) string {
	var b strings.Builder

	b.WriteString(MarkerStart)
	b.WriteString("\n")
	b.WriteString("# Company AI Instructions\n\n")
	b.WriteString("If any instruction file is missing or inaccessible, stop and ask for it before proceeding.\n\n")
	b.WriteString(fmt.Sprintf("This project uses the following instruction stacks: %s\n\n", strings.Join(stacks, ", ")))
	b.WriteString(fmt.Sprintf("Read and follow ALL instruction files in the `%s/` folder:\n", instructionsDir))

	for _, f := range files {
		b.WriteString(fmt.Sprintf("- %s\n", f))
	}

	b.WriteString("\nThese are mandatory company standards. Follow them strictly.\n")
	b.WriteString(MarkerEnd)

	return b.String()
}

// injectIntoFile creates or updates the managed block in a file.
func injectIntoFile(path, block string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist — create with just the block
			return atomicWrite(path, block+"\n")
		}
		return err
	}

	content := string(data)

	startIdx := strings.Index(content, MarkerStart)
	endIdx := strings.Index(content, MarkerEnd)

	var newContent string
	if startIdx >= 0 && endIdx >= 0 && endIdx > startIdx {
		// Both markers found in correct order — replace between them (inclusive)
		endIdx += len(MarkerEnd)
		newContent = content[:startIdx] + block + content[endIdx:]
	} else if startIdx >= 0 || endIdx >= 0 {
		// Malformed: one marker without the other — strip the broken marker and prepend
		cleaned := content
		cleaned = strings.Replace(cleaned, MarkerStart, "", 1)
		cleaned = strings.Replace(cleaned, MarkerEnd, "", 1)
		cleaned = strings.TrimLeft(cleaned, "\n")
		newContent = block + "\n\n" + cleaned
	} else {
		// No markers at all — prepend block above existing content
		newContent = block + "\n\n" + content
	}

	return atomicWrite(path, newContent)
}

// atomicWrite writes content to a file using a temp file and rename.
func atomicWrite(path, content string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, []byte(content), 0644); err != nil {
		return err
	}

	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return err
	}

	return nil
}
