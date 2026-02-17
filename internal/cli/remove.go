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

func (a *App) newRemoveCmd() *cobra.Command {
	var autoRemoveOrphans bool

	cmd := &cobra.Command{
		Use:   "remove <stack> [stack...]",
		Short: "Remove stacks from this project",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.runRemove(cmd.Context(), args, autoRemoveOrphans)
		},
	}

	cmd.Flags().BoolVar(&autoRemoveOrphans, "auto-remove-orphans", false, "automatically remove orphaned dependencies (useful in CI)")
	return cmd
}

func (a *App) runRemove(ctx context.Context, stacks []string, autoRemoveOrphans bool) error {
	if err := a.RequireProject(); err != nil {
		return err
	}

	managedDir := a.getManagedDir()

	// Validate stacks are currently installed
	for _, s := range stacks {
		found := false
		for _, existing := range a.config.Stacks {
			if existing == s {
				found = true
				break
			}
		}
		if !found {
			return &ExitError{Code: 4, Message: fmt.Sprintf("stack %q is not installed", s)}
		}
	}

	// Build resolver from registry or current resolved data
	stackInfoMap := a.buildStackInfoFromResolved()

	// Try to fetch registry for better resolution, but fallback to resolved data
	client, err := a.newRegistryClient()
	if err == nil {
		if reg, fetchErr := client.FetchRegistry(ctx); fetchErr == nil {
			stackInfoMap = buildStackInfoMap(reg)
		}
	}

	r := resolver.NewResolver(stackInfoMap)

	// Check for orphans
	orphans := r.ResolveRemoval(a.config.Stacks, stacks)

	removeSet := make(map[string]bool)
	for _, s := range stacks {
		removeSet[s] = true
	}

	if len(orphans) > 0 {
		a.output.Println("\nThe following dependencies are no longer needed:")
		for _, o := range orphans {
			a.output.Println("  %s", o)
		}

		shouldRemoveOrphans := autoRemoveOrphans
		if !shouldRemoveOrphans && !ui.IsCI() {
			var promptErr error
			shouldRemoveOrphans, promptErr = ui.Confirm("Remove these orphaned dependencies?")
			if promptErr != nil {
				return promptErr
			}
		}
		if shouldRemoveOrphans {
			for _, o := range orphans {
				removeSet[o] = true
			}
		}
	}

	// Remove files
	for id := range removeSet {
		if err := filemanager.RemoveStack(a.projectDir, managedDir, id); err != nil {
			a.output.Warning("Could not remove %s: %v", id, err)
		}
		delete(a.config.Resolved, id)
	}

	// Update explicit stacks list
	var remaining []string
	for _, s := range a.config.Stacks {
		if !removeSet[s] {
			remaining = append(remaining, s)
		}
	}
	a.config.Stacks = remaining

	// Re-resolve to get proper order for injection
	if len(remaining) > 0 {
		res, resolveErr := r.Resolve(remaining)
		if resolveErr == nil {
			// Update dependency_of for remaining stacks
			for _, id := range res.Order {
				if rs, ok := a.config.Resolved[id]; ok {
					if res.Explicit[id] {
						rs.Explicit = true
						rs.DependencyOf = ""
					} else {
						rs.Explicit = false
						rs.DependencyOf = res.DependencyOf[id]
					}
					a.config.Resolved[id] = rs
				}
			}

			configs := buildInjectorConfigs(res.Order, a.config.Resolved, managedDir)
			injector.InjectAll(a.projectDir, res.Order, configs, managedDir)
		}
	} else {
		// No stacks left — clear managed blocks
		for _, filename := range []string{"CLAUDE.md", "AGENTS.md", ".cursorrules"} {
			configs := []injector.FileConfig{{Filename: filename, Files: nil}}
			injector.InjectAll(a.projectDir, nil, configs, managedDir)
		}
	}

	// Save config
	if err := config.SaveConfig(a.projectDir, a.config); err != nil {
		return err
	}

	removed := make([]string, 0, len(removeSet))
	for id := range removeSet {
		removed = append(removed, id)
	}
	a.output.Success("Removed %d stack(s): %v", len(removed), removed)
	return nil
}

func (a *App) buildStackInfoFromResolved() map[string]resolver.StackInfo {
	m := make(map[string]resolver.StackInfo)
	for id := range a.config.Resolved {
		m[id] = resolver.StackInfo{ID: id}
	}
	return m
}
