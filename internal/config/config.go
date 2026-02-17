package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const ConfigFile = "ai-instructions.yml"

// LockFile is the old lockfile name, kept for migration and cleanup.
const LockFile = "ai-instructions.lock"

const resolvedSeparator = "\n# Resolved dependencies — auto-generated, do not edit below this line\n"

// Config represents the ai-instructions.yml file, including resolved state.
type Config struct {
	Version         int            `yaml:"version"`
	Registry        RegistryConfig `yaml:"registry"`
	InstructionsDir string         `yaml:"instructions_dir,omitempty"`
	Mode            string         `yaml:"mode,omitempty"`
	Stacks          []string       `yaml:"stacks"`

	Resolved map[string]ResolvedStack `yaml:"resolved,omitempty"`
}

// configUserFields is the subset of Config that users edit.
// Used for two-pass marshaling so the resolved section stays below a comment.
type configUserFields struct {
	Version         int            `yaml:"version"`
	Registry        RegistryConfig `yaml:"registry"`
	InstructionsDir string         `yaml:"instructions_dir,omitempty"`
	Mode            string         `yaml:"mode,omitempty"`
	Stacks          []string       `yaml:"stacks"`
}

// configResolvedFields is the auto-generated portion of the config file.
type configResolvedFields struct {
	Resolved map[string]ResolvedStack `yaml:"resolved,omitempty"`
}

// RegistryConfig holds registry connection settings.
type RegistryConfig struct {
	URL    string `yaml:"url"`
	Branch string `yaml:"branch,omitempty"`
}

// ConfigExists checks whether the config file exists in the given directory.
func ConfigExists(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, ConfigFile))
	return err == nil
}

// LoadConfig reads and parses the config file from the given directory.
func LoadConfig(dir string) (*Config, error) {
	data, err := os.ReadFile(filepath.Join(dir, ConfigFile))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("config file not found: run 'ai-instructions init' first")
		}
		return nil, fmt.Errorf("reading config: %w", err)
	}

	var c Config
	if err := yaml.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	// Apply defaults
	if c.InstructionsDir == "" {
		c.InstructionsDir = DefaultInstructionsDir
	}
	if c.Mode == "" {
		c.Mode = "platform"
	}
	if c.Registry.Branch == "" {
		c.Registry.Branch = "master"
	}

	if err := ValidateConfig(&c); err != nil {
		return nil, err
	}

	return &c, nil
}

// SaveConfig writes the config file to the given directory.
// It uses two-pass marshaling: user fields first, then a comment separator,
// then the resolved section.
func SaveConfig(dir string, c *Config) error {
	if c.InstructionsDir == "" {
		c.InstructionsDir = DefaultInstructionsDir
	}
	if c.Mode == "" {
		c.Mode = "platform"
	}
	if c.Registry.Branch == "" {
		c.Registry.Branch = "master"
	}

	userPart := configUserFields{
		Version:         c.Version,
		Registry:        c.Registry,
		InstructionsDir: c.InstructionsDir,
		Mode:            c.Mode,
		Stacks:          c.Stacks,
	}

	userBytes, err := yaml.Marshal(userPart)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	var content []byte
	if len(c.Resolved) > 0 {
		resolvedPart := configResolvedFields{Resolved: c.Resolved}
		resolvedBytes, marshalErr := yaml.Marshal(resolvedPart)
		if marshalErr != nil {
			return fmt.Errorf("marshaling resolved: %w", marshalErr)
		}
		content = append(userBytes, []byte(resolvedSeparator)...)
		content = append(content, resolvedBytes...)
	} else {
		content = userBytes
	}

	path := filepath.Join(dir, ConfigFile)
	tmpPath := path + ".tmp"

	if err := os.WriteFile(tmpPath, content, 0644); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("saving config: %w", err)
	}

	return nil
}

// ValidateConfig checks that a Config struct has required fields.
func ValidateConfig(c *Config) error {
	if c.Version < 1 {
		return fmt.Errorf("invalid config version: %d", c.Version)
	}
	if c.Registry.URL == "" {
		return fmt.Errorf("registry url is required")
	}
	if len(c.Stacks) == 0 {
		return fmt.Errorf("at least one stack is required")
	}
	return nil
}
