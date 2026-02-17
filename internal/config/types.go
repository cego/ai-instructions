package config

const DefaultInstructionsDir = "ai-instructions"
const ManagedDir = "company-instructions"
const DefaultRegistryURL = "https://gitlab.cego.dk/cego/ai-marketplace"
const DefaultBranch = "master"

// ResolvedStack represents a single resolved stack in the lockfile.
type ResolvedStack struct {
	Version      string            `yaml:"version"`
	Hash         string            `yaml:"hash"`
	Files        []string          `yaml:"files"`
	FileHashes   map[string]string `yaml:"file_hashes,omitempty"`
	Tools        ToolsConfig       `yaml:"tools"`
	Explicit     bool              `yaml:"explicit,omitempty"`
	DependencyOf string            `yaml:"dependency_of,omitempty"`
}

// ToolsConfig specifies which AI tool files a stack targets.
type ToolsConfig struct {
	IncludeInClaudeMD    bool `yaml:"include_in_claude_md"`
	IncludeInAgentsMD    bool `yaml:"include_in_agents_md"`
	IncludeInCursorRules bool `yaml:"include_in_cursorrules"`
}
