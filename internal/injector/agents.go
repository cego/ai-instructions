package injector

// AgentsConfig returns the FileConfig for AGENTS.md.
func AgentsConfig(files []string) FileConfig {
	return FileConfig{
		Filename: "AGENTS.md",
		Files:    files,
	}
}
