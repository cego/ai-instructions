package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const OldSettingsFile = "ai-instructions-settings.yml"

// oldSettings represents the old ai-instructions-settings.yml format.
type oldSettings struct {
	Version         int                      `yaml:"version"`
	Mode            string                   `yaml:"mode"`
	InstructionsDir string                   `yaml:"instructions_dir,omitempty"`
	RegistryURL     string                   `yaml:"registry_url"`
	Branch          string                   `yaml:"branch,omitempty"`
	Stacks          []string                 `yaml:"stacks"`
	Resolved        map[string]ResolvedStack `yaml:"resolved"`
}

// legacyLockfile is used to deserialize the old ai-instructions.lock file during migration.
type legacyLockfile struct {
	Version  int                      `yaml:"version"`
	Resolved map[string]ResolvedStack `yaml:"resolved"`
}

// OldSettingsExists checks whether the old settings file exists in the given directory.
func OldSettingsExists(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, OldSettingsFile))
	return err == nil
}

// OldLockfileExists checks whether the old lockfile exists in the given directory.
func OldLockfileExists(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, LockFile))
	return err == nil
}

// AbsorbLockfile reads the old ai-instructions.lock and merges its Resolved map into the Config.
// It does NOT delete the old file — the caller should do that after verifying the migration.
func AbsorbLockfile(dir string, c *Config) error {
	data, err := os.ReadFile(filepath.Join(dir, LockFile))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("old lockfile not found")
		}
		return fmt.Errorf("reading old lockfile: %w", err)
	}

	var lf legacyLockfile
	if err := yaml.Unmarshal(data, &lf); err != nil {
		return fmt.Errorf("parsing old lockfile: %w", err)
	}

	if lf.Resolved != nil {
		c.Resolved = lf.Resolved
	}

	return nil
}

// MigrateFromOldSettings reads the old settings file and converts it into a Config.
// It does NOT delete the old file — the caller should do that after verifying the migration.
func MigrateFromOldSettings(dir string) (*Config, error) {
	data, err := os.ReadFile(filepath.Join(dir, OldSettingsFile))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("old settings file not found")
		}
		return nil, fmt.Errorf("reading old settings: %w", err)
	}

	var old oldSettings
	if err := yaml.Unmarshal(data, &old); err != nil {
		return nil, fmt.Errorf("parsing old settings: %w", err)
	}

	instrDir := old.InstructionsDir
	if instrDir == "" {
		instrDir = DefaultInstructionsDir
	}

	mode := old.Mode
	if mode == "" {
		mode = "platform"
	}

	branch := old.Branch
	if branch == "" {
		branch = "master"
	}

	cfg := &Config{
		Version: 1,
		Registry: RegistryConfig{
			URL:    old.RegistryURL,
			Branch: branch,
		},
		InstructionsDir: instrDir,
		Mode:            mode,
		Stacks:          old.Stacks,
		Resolved:        old.Resolved,
	}

	if cfg.Resolved == nil {
		cfg.Resolved = make(map[string]ResolvedStack)
	}

	return cfg, nil
}
