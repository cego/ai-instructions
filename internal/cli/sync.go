package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/company/ai-instructions/internal/config"
	"github.com/company/ai-instructions/internal/filemanager"
	"github.com/company/ai-instructions/internal/injector"
	"github.com/company/ai-instructions/internal/resolver"
	"github.com/spf13/cobra"
)

func (a *App) newSyncCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "sync",
		Short: "Sync instruction files from registry",
		Long:  "Downloads latest instruction files and updates managed blocks.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.runSync(cmd.Context())
		},
	}
}

func (a *App) runSync(ctx context.Context) error {
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

	// Re-resolve dependencies (in case registry has changed)
	stackInfoMap := buildStackInfoMap(reg)
	res, err := resolver.NewResolver(stackInfoMap).Resolve(a.config.Stacks)
	if err != nil {
		return fmt.Errorf("dependency resolution: %w", err)
	}

	fm := filemanager.NewManager(client, a.projectDir, managedDir)

	var unchanged []string
	type updateInfo struct {
		stack      string
		oldVersion string
		newVersion string
	}
	var updates []updateInfo

	a.output.Info("Syncing instruction files...")
	for _, stackID := range res.Order {
		regMeta, exists := reg.Stacks[stackID]
		if !exists {
			a.output.Warning("Stack %q no longer exists in registry, skipping", stackID)
			continue
		}

		currentResolved, hasExisting := a.config.Resolved[stackID]
		a.debugf("sync %s: registry=%s local=%s", stackID, regMeta.Version, currentResolved.Version)

		// Skip download if version matches and local files are intact
		if hasExisting && currentResolved.Version == regMeta.Version {
			vInfo := filemanager.StackVerifyInfo{
				Hash:       currentResolved.Hash,
				Files:      currentResolved.Files,
				FileHashes: currentResolved.FileHashes,
			}
			result := filemanager.VerifyStack(a.projectDir, managedDir, stackID, vInfo)
			if result.OK {
				a.debugf("sync %s: version match + files intact, skipping", stackID)
				unchanged = append(unchanged, stackID)
				// Still update explicit/dependency_of in case it changed
				rs := currentResolved
				if res.Explicit[stackID] {
					rs.Explicit = true
					rs.DependencyOf = ""
				} else {
					rs.Explicit = false
					rs.DependencyOf = res.DependencyOf[stackID]
				}
				a.config.Resolved[stackID] = rs
				continue
			}
			// Files tampered — re-download below
		}

		manifest, fetchErr := client.FetchStackManifest(ctx, stackID)
		if fetchErr != nil {
			return fmt.Errorf("syncing: %w", fetchErr)
		}

		files := manifest.Files

		if downloadErr := fm.DownloadStack(ctx, stackID, files); downloadErr != nil {
			return fmt.Errorf("syncing: %w", downloadErr)
		}

		hash, hashErr := filemanager.HashDir(fm.StackDir(stackID))
		if hashErr != nil {
			return fmt.Errorf("syncing: %w", hashErr)
		}
		fileHashes, hashErr := filemanager.HashFilesInStack(fm.StackDir(stackID), files)
		if hashErr != nil {
			return fmt.Errorf("syncing: %w", hashErr)
		}

		oldVersion := ""
		if hasExisting {
			oldVersion = currentResolved.Version
		}
		updates = append(updates, updateInfo{
			stack:      stackID,
			oldVersion: oldVersion,
			newVersion: regMeta.Version,
		})

		rs := config.ResolvedStack{
			Version:    regMeta.Version,
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

	// Cleanup stale stacks
	resolvedSet := make(map[string]bool)
	for _, id := range res.Order {
		resolvedSet[id] = true
	}
	staleRemoved, _ := filemanager.CleanupStaleStacks(a.projectDir, managedDir, resolvedSet)
	for _, id := range staleRemoved {
		delete(a.config.Resolved, id)
	}

	// Save config
	if err := config.SaveConfig(a.projectDir, a.config); err != nil {
		return err
	}

	// Cleanup old lockfile if present
	if config.OldLockfileExists(a.projectDir) {
		os.Remove(filepath.Join(a.projectDir, config.LockFile))
	}

	// Re-inject managed blocks
	configs := buildInjectorConfigs(res.Order, a.config.Resolved, managedDir)
	if err := injector.InjectAll(a.projectDir, res.Order, configs, managedDir); err != nil {
		return err
	}

	// Print summary
	if len(updates) > 0 {
		a.output.Success("Synced %d updated stack(s):", len(updates))
		for _, u := range updates {
			if u.oldVersion != "" {
				a.output.Println("  %s   %s → %s", u.stack, u.oldVersion, u.newVersion)
			} else {
				a.output.Println("  %s   (new) %s", u.stack, u.newVersion)
			}
		}
	}
	if len(unchanged) > 0 {
		a.output.Println("\n%d stack(s) unchanged: %v", len(unchanged), unchanged)
	}
	if len(updates) == 0 {
		a.output.Success("Everything is up to date")
	}

	return nil
}
