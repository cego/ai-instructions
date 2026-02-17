package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/company/ai-instructions/internal/config"
	"github.com/company/ai-instructions/internal/exitcodes"
	"github.com/company/ai-instructions/internal/filemanager"
	"github.com/company/ai-instructions/internal/injector"
	"github.com/company/ai-instructions/internal/registry"
	"github.com/company/ai-instructions/internal/resolver"
	"github.com/spf13/cobra"
)

func (a *App) newInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init <stack> [stack...]",
		Short: "Initialize AI instructions for this project",
		Long:  "Set up AI instruction stacks for the current project.\nPass stack names as arguments (e.g. ai-instructions init php laravel).",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.runInit(cmd.Context(), args)
		},
	}
}

func (a *App) runInit(ctx context.Context, stacks []string) error {
	if a.config != nil && len(a.config.Stacks) > 0 {
		a.output.Warning("Existing config found with stacks: %v", a.config.Stacks)
		a.output.Info("Re-initializing will replace the current configuration.")
	}

	client, err := a.newRegistryClient()
	if err != nil {
		return err
	}

	a.output.Info("Fetching registry...")
	reg, err := client.FetchRegistry(ctx)
	if err != nil {
		return err
	}

	// Validate provided stacks exist in registry
	for _, s := range stacks {
		if _, ok := reg.Stacks[s]; !ok {
			return &ExitError{Code: exitcodes.UsageError, Message: fmt.Sprintf("stack %q not found in registry", s)}
		}
	}

	// Resolve dependencies
	stackInfoMap := buildStackInfoMap(reg)
	res, err := resolver.NewResolver(stackInfoMap).Resolve(stacks)
	if err != nil {
		return fmt.Errorf("dependency resolution: %w", err)
	}

	// Build config and download files
	instrDir := config.DefaultInstructionsDir
	managedDir := instrDir + "/" + config.ManagedDir
	registryURL := a.registryURL
	if registryURL == "" {
		registryURL = config.DefaultRegistryURL
	}
	cfg := &config.Config{
		Version: 1,
		Registry: config.RegistryConfig{
			URL:    registryURL,
			Branch: a.getBranch(),
		},
		InstructionsDir: instrDir,
		Mode:            "platform",
		Stacks:          stacks,
		Resolved:        make(map[string]config.ResolvedStack),
	}

	// Clear managed directory for a fresh start
	os.RemoveAll(filepath.Join(a.projectDir, managedDir))

	fm := filemanager.NewManager(client, a.projectDir, managedDir)

	a.output.Info("Downloading instruction files...")
	for _, stackID := range res.Order {
		manifest, fetchErr := client.FetchStackManifest(ctx, stackID)
		if fetchErr != nil {
			return fmt.Errorf("downloading stacks: %w", fetchErr)
		}

		files := manifest.Files

		if downloadErr := fm.DownloadStack(ctx, stackID, files); downloadErr != nil {
			return fmt.Errorf("downloading stacks: %w", downloadErr)
		}

		// Compute hashes of downloaded files
		hash, hashErr := filemanager.HashDir(fm.StackDir(stackID))
		if hashErr != nil {
			return fmt.Errorf("downloading stacks: %w", hashErr)
		}
		fileHashes, hashErr := filemanager.HashFilesInStack(fm.StackDir(stackID), files)
		if hashErr != nil {
			return fmt.Errorf("downloading stacks: %w", hashErr)
		}

		rs := config.ResolvedStack{
			Version:    reg.Stacks[stackID].Version,
			Hash:       hash,
			Files:      files,
			FileHashes: fileHashes,
			Tools:      toolsConfigFromManifest(manifest.Tools),
		}
		if res.Explicit[stackID] {
			rs.Explicit = true
		} else {
			rs.DependencyOf = res.DependencyOf[stackID]
		}
		cfg.Resolved[stackID] = rs
	}

	// Save config
	if err := config.SaveConfig(a.projectDir, cfg); err != nil {
		return err
	}

	// Cleanup old files
	if config.OldSettingsExists(a.projectDir) {
		os.Remove(filepath.Join(a.projectDir, config.OldSettingsFile))
	}
	if config.OldLockfileExists(a.projectDir) {
		os.Remove(filepath.Join(a.projectDir, config.LockFile))
	}

	// Inject managed blocks
	configs := buildInjectorConfigs(res.Order, cfg.Resolved, managedDir)
	if err := injector.InjectAll(a.projectDir, res.Order, configs, managedDir); err != nil {
		return err
	}

	a.output.Success("Initialized with %d stacks, %d instruction files", len(res.Order), countResolvedFiles(cfg.Resolved))
	a.output.Info("\nRemember to commit the following files:")
	a.output.Info("  - %s", config.ConfigFile)
	a.output.Info("  - %s/", managedDir)
	a.output.Info("  - CLAUDE.md")
	a.output.Info("  - AGENTS.md")
	a.output.Info("  - .cursorrules")

	return nil
}

func buildStackInfoMap(reg *registry.Registry) map[string]resolver.StackInfo {
	m := make(map[string]resolver.StackInfo)
	for id, meta := range reg.Stacks {
		m[id] = resolver.StackInfo{ID: id, Depends: meta.Depends}
	}
	return m
}

func buildInjectorConfigs(order []string, resolved map[string]config.ResolvedStack, instrDir string) []injector.FileConfig {
	var claudeFiles, agentsFiles, cursorFiles []string

	for _, stackID := range order {
		rs := resolved[stackID]
		for _, f := range rs.Files {
			path := fmt.Sprintf("%s/%s/%s", instrDir, stackID, f)
			if rs.Tools.IncludeInClaudeMD {
				claudeFiles = append(claudeFiles, path)
			}
			if rs.Tools.IncludeInAgentsMD {
				agentsFiles = append(agentsFiles, path)
			}
			if rs.Tools.IncludeInCursorRules {
				cursorFiles = append(cursorFiles, path)
			}
		}
	}

	return []injector.FileConfig{
		injector.ClaudeConfig(claudeFiles),
		injector.AgentsConfig(agentsFiles),
		injector.CursorConfig(cursorFiles),
	}
}

// toolsConfigFromManifest converts registry ToolsConfig to config ToolsConfig.
func toolsConfigFromManifest(tools registry.ToolsConfig) config.ToolsConfig {
	return config.ToolsConfig{
		IncludeInClaudeMD:    tools.Claude.IncludeInClaudeMD,
		IncludeInAgentsMD:    tools.Claude.IncludeInAgentsMD,
		IncludeInCursorRules: tools.Cursor.IncludeInCursorRules,
	}
}

func countResolvedFiles(resolved map[string]config.ResolvedStack) int {
	total := 0
	for _, rs := range resolved {
		total += len(rs.Files)
	}
	return total
}
