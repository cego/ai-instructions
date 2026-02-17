package injector

// ClaudeConfig returns the FileConfig for CLAUDE.md.
func ClaudeConfig(files []string) FileConfig {
	return FileConfig{
		Filename: "CLAUDE.md",
		Files:    files,
	}
}
