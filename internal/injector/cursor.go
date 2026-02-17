package injector

// CursorConfig returns the FileConfig for .cursorrules.
func CursorConfig(files []string) FileConfig {
	return FileConfig{
		Filename: ".cursorrules",
		Files:    files,
	}
}
