package registry

// Registry represents the top-level registry.json.
type Registry struct {
	Version     int                  `json:"version"`
	GeneratedAt string               `json:"generated_at"`
	Stacks      map[string]StackMeta `json:"stacks"`
}

// StackMeta is the summary of a stack in registry.json.
type StackMeta struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Version     string   `json:"version"`
	Hash        string   `json:"hash"`
	Category    string   `json:"category"`
	Depends     []string `json:"depends"`
}

// StackManifest is the full stack.json within a stack folder.
type StackManifest struct {
	Name        string         `json:"name"`
	Version     string         `json:"version"`
	Description string         `json:"description"`
	Depends     []string       `json:"depends"`
	Category    string         `json:"category"`
	Files       []string       `json:"files"`
	Tools       ToolsConfig    `json:"tools"`
}

// ToolsConfig specifies which AI tools a stack targets.
type ToolsConfig struct {
	Claude ClaudeToolConfig `json:"claude"`
	Cursor CursorToolConfig `json:"cursor"`
}

// ClaudeToolConfig controls CLAUDE.md / AGENTS.md inclusion.
type ClaudeToolConfig struct {
	IncludeInClaudeMD bool `json:"include_in_claude_md"`
	IncludeInAgentsMD bool `json:"include_in_agents_md"`
}

// CursorToolConfig controls .cursorrules inclusion.
type CursorToolConfig struct {
	IncludeInCursorRules bool `json:"include_in_cursorrules"`
}

