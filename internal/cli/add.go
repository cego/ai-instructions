package cli

import (
	"context"
	"fmt"

	"github.com/company/ai-instructions/internal/config"
	"github.com/company/ai-instructions/internal/filemanager"
	"github.com/company/ai-instructions/internal/injector"
	"github.com/company/ai-instructions/internal/resolver"
	"github.com/company/ai-instructions/internal/ui"
	"github.com/spf13/cobra"
)

func (a *App) newAddCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "add <stack> [stack...]",
		Short: "Add stacks to this project",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.runAdd(cmd.Context(), args)
		},
	}
}

func (a *App) runAdd(ctx context.Context, stacks []string) error {
	if err := a.RequireProject(); err != nil {
		return err
	}

	managedDir := a.getManagedDir()

	client, err := a.newRegistryClient()
	if err != nil {
		return err
	}

	reg, err := client.FetchRegistry(ctx)
	if err != nil {
		return err
	}

	// Check for already installed stacks
	existingSet := make(map[string]bool)
	for _, s := range a.config.Stacks {
		existingSet[s] = true
	}

	var newStacks []string
	for _, s := range stacks {
		if _, ok := reg.Stacks[s]; !ok {
			return &ExitError{Code: 4, Message: fmt.Sprintf("stack %q not found in registry", s)}
		}
		if existingSet[s] {
			a.output.Warning("Stack %q is already installed, skipping", s)
			continue
		}
		newStacks = append(newStacks, s)
	}

	if len(newStacks) == 0 {
		a.output.Info("Nothing to add.")
		return nil
	}

	// Merge with existing stacks and resolve
	allExplicit := make([]string, 0, len(a.config.Stacks)+len(newStacks))
	allExplicit = append(allExplicit, a.config.Stacks...)
	allExplicit = append(allExplicit, newStacks...)
	stackInfoMap := buildStackInfoMap(reg)
	res, err := resolver.NewResolver(stackInfoMap).Resolve(allExplicit)
	if err != nil {
		return fmt.Errorf("dependency resolution: %w", err)
	}

	// Download only new stacks
	fm := filemanager.NewManager(client, a.projectDir, managedDir)

	err = ui.WithSpinner("Downloading instruction files...", func() error {
		for _, stackID := range res.Order {
			if _, exists := a.config.Resolved[stackID]; exists {
				continue // already downloaded
			}

			manifest, fetchErr := client.FetchStackManifest(ctx, stackID)
			if fetchErr != nil {
				return fetchErr
			}

			files := manifest.Files

			if downloadErr := fm.DownloadStack(ctx, stackID, files); downloadErr != nil {
				return downloadErr
			}

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
			a.config.Resolved[stackID] = rs
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("downloading stacks: %w", err)
	}

	// Update config (stacks list + resolved entries)
	a.config.Stacks = allExplicit
	if err := config.SaveConfig(a.projectDir, a.config); err != nil {
		return err
	}

	// Re-inject managed blocks
	configs := buildInjectorConfigs(res.Order, a.config.Resolved, managedDir)
	if err := injector.InjectAll(a.projectDir, res.Order, configs, managedDir); err != nil {
		return err
	}

	a.output.Success("Added %d stack(s): %v", len(newStacks), newStacks)
	return nil
}
