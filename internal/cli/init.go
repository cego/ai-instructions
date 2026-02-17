package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/company/ai-instructions/internal/config"
	"github.com/company/ai-instructions/internal/filemanager"
	"github.com/company/ai-instructions/internal/injector"
	"github.com/company/ai-instructions/internal/registry"
	"github.com/company/ai-instructions/internal/resolver"
	"github.com/company/ai-instructions/internal/ui"
	"github.com/spf13/cobra"
)

func (a *App) newInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init [stack...]",
		Short: "Initialize AI instructions for this project",
		Long:  "Set up AI instruction stacks for the current project.\nPass stack names as arguments for non-interactive mode, or run without arguments for the interactive wizard.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.runInit(cmd.Context(), args)
		},
	}
}

func (a *App) runInit(ctx context.Context, stacks []string) error {
	interactive := len(stacks) == 0

	if interactive && ui.IsCI() {
		return &ExitError{Code: 4, Message: "init requires interactive mode — pass stack names as arguments (e.g. ai-instructions init go docker)"}
	}

	// Check if already initialized
	if interactive && config.ConfigExists(a.projectDir) {
		confirmed, err := ui.Confirm("This project is already initialized. Reconfigure?")
		if err != nil {
			return err
		}
		if !confirmed {
			a.output.Info("Aborted.")
			return nil
		}
	} else if interactive && config.OldSettingsExists(a.projectDir) {
		confirmed, err := ui.Confirm("Old settings file detected. Reconfigure and migrate?")
		if err != nil {
			return err
		}
		if !confirmed {
			a.output.Info("Aborted.")
			return nil
		}
	}

	// Step 1: Fetch registry
	client, err := a.newRegistryClient()
	if err != nil {
		return err
	}

	var reg *registry.Registry
	err = ui.WithSpinner("Fetching registry...", func() error {
		var fetchErr error
		reg, fetchErr = client.FetchRegistry(ctx)
		return fetchErr
	})
	if err != nil {
		return err
	}

	var selected []string
	if interactive {
		// Step 2: Stack selection (interactive)
		stackOptions := buildStackOptions(reg)
		selected, err = ui.SelectStacks(stackOptions)
		if err != nil {
			return fmt.Errorf("stack selection: %w", err)
		}
		if len(selected) == 0 {
			a.output.Warning("No stacks selected. Aborted.")
			return nil
		}
	} else {
		// Validate provided stacks exist in registry
		for _, s := range stacks {
			if _, ok := reg.Stacks[s]; !ok {
				return &ExitError{Code: 4, Message: fmt.Sprintf("stack %q not found in registry", s)}
			}
		}
		selected = stacks
	}

	// Step 3: Resolve dependencies
	stackInfoMap := buildStackInfoMap(reg)
	res, err := resolver.NewResolver(stackInfoMap).Resolve(selected)
	if err != nil {
		return fmt.Errorf("dependency resolution: %w", err)
	}

	if interactive {
		// Step 4: Show confirmation
		printResolutionSummary(a.output, res, reg)

		confirmed, err := ui.Confirm("Proceed?")
		if err != nil {
			return err
		}
		if !confirmed {
			a.output.Info("Aborted.")
			return nil
		}
	}

	// Step 5: Build config and download files
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
		Stacks:          selected,
		Resolved:        make(map[string]config.ResolvedStack),
	}

	// Clear managed directory for a fresh start
	os.RemoveAll(filepath.Join(a.projectDir, managedDir))

	fm := filemanager.NewManager(client, a.projectDir, managedDir)

	err = ui.WithSpinner("Downloading instruction files...", func() error {
		for _, stackID := range res.Order {
			manifest, fetchErr := client.FetchStackManifest(ctx, stackID)
			if fetchErr != nil {
				return fetchErr
			}

			files := manifest.Files

			if downloadErr := fm.DownloadStack(ctx, stackID, files); downloadErr != nil {
				return downloadErr
			}

			// Compute hashes of downloaded files
			hash, hashErr := filemanager.HashDir(fm.StackDir(stackID))
			if hashErr != nil {
				return hashErr
			}
			fileHashes, hashErr := filemanager.HashFilesInStack(fm.StackDir(stackID), files)
			if hashErr != nil {
				return hashErr
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
		return nil
	})
	if err != nil {
		return fmt.Errorf("downloading stacks: %w", err)
	}

	// Step 6: Save config
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

	// Step 7: Inject managed blocks
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

func buildStackOptions(reg *registry.Registry) []ui.StackOption {
	var opts []ui.StackOption
	for id, meta := range reg.Stacks {
		opts = append(opts, ui.StackOption{
			ID:          id,
			Name:        meta.Name,
			Description: meta.Description,
			Category:    meta.Category,
		})
	}
	sort.Slice(opts, func(i, j int) bool {
		if opts[i].Category != opts[j].Category {
			return opts[i].Category < opts[j].Category
		}
		return opts[i].ID < opts[j].ID
	})
	return opts
}

func buildStackInfoMap(reg *registry.Registry) map[string]resolver.StackInfo {
	m := make(map[string]resolver.StackInfo)
	for id, meta := range reg.Stacks {
		m[id] = resolver.StackInfo{ID: id, Depends: meta.Depends}
	}
	return m
}

func printResolutionSummary(out *ui.Output, res *resolver.Resolution, reg *registry.Registry) {
	out.Println("\nThe following stacks will be installed:\n")

	out.Println("  Explicit:")
	for _, id := range res.Order {
		if res.Explicit[id] {
			out.Println("    %s", id)
		}
	}

	hasDeps := false
	for _, id := range res.Order {
		if !res.Explicit[id] {
			if !hasDeps {
				out.Println("\n  Auto-included (dependencies):")
				hasDeps = true
			}
			out.Println("    %s (required by %s)", id, res.DependencyOf[id])
		}
	}

	out.Println("\n  Total: %d stacks", len(res.Order))
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
