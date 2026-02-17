package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/company/ai-instructions/internal/config"
	"github.com/company/ai-instructions/internal/filemanager"
	"github.com/company/ai-instructions/internal/injector"
	"github.com/spf13/cobra"
)

func (a *App) newDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Diagnose common issues",
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.runDoctor(cmd.Context())
		},
	}
}

func (a *App) runDoctor(ctx context.Context) error {
	allOK := true

	// 0. Check for old settings file
	if config.OldSettingsExists(a.projectDir) {
		a.output.Warning("Old %s detected — run 'ai-instructions init' to migrate", config.OldSettingsFile)
	}

	// 0b. Check for old lockfile
	if config.OldLockfileExists(a.projectDir) {
		a.output.Warning("Old %s detected — run 'ai-instructions sync' to migrate to single-file format", config.LockFile)
	}

	// 1. Config file
	if config.ConfigExists(a.projectDir) {
		a.output.Success("%s found", config.ConfigFile)
	} else {
		a.output.Error("%s not found — run: ai-instructions init", config.ConfigFile)
		return nil // Can't check further without config
	}

	// Load config
	if err := a.LoadProjectConfig(); err != nil {
		a.output.Error("Config file invalid: %v", err)
		return nil
	}

	// 2. Resolved stacks
	if a.config.Resolved == nil || len(a.config.Resolved) == 0 {
		a.output.Error("No resolved stacks — run: ai-instructions sync")
		return nil
	}
	a.output.Success("%d stacks resolved", len(a.config.Resolved))

	managedDir := a.getManagedDir()

	// 3. Registry reachable (use a short timeout so doctor doesn't hang)
	registryCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	client, clientErr := a.newRegistryClient()
	if clientErr != nil {
		a.output.Error("Registry not configured: %v", clientErr)
		allOK = false
	} else {
		_, fetchErr := client.FetchRegistry(registryCtx)
		if fetchErr != nil {
			a.output.Error("Registry unreachable at %s: %v", a.getProjectURL(), fetchErr)
			a.output.Info("  Are you connected to Cego Warp?")
			allOK = false
		} else {
			a.output.Success("Registry reachable at %s", a.getProjectURL())
		}
	}

	// 4. Instructions folder
	instrPath := filepath.Join(a.projectDir, managedDir)
	totalFiles := 0
	if info, err := os.Stat(instrPath); err == nil && info.IsDir() {
		for _, rs := range a.config.Resolved {
			totalFiles += len(rs.Files)
		}
		a.output.Success("%s/ folder exists with %d files", managedDir, totalFiles)
	} else {
		a.output.Error("%s/ folder missing — run: ai-instructions sync", managedDir)
		allOK = false
	}

	// 5. Managed blocks
	for _, filename := range []string{"CLAUDE.md", "AGENTS.md", ".cursorrules"} {
		path := filepath.Join(a.projectDir, filename)
		result := injector.VerifyFile(path, filename)
		if result.HasBlock {
			a.output.Success("%s has managed block", filename)
		} else if result.Exists {
			a.output.Error("%s missing managed block — run: ai-instructions sync", filename)
			allOK = false
		} else {
			a.output.Error("%s not found — run: ai-instructions sync", filename)
			allOK = false
		}
	}

	// 6. Hash verification
	allHashesOK := true
	for stackID, rs := range a.config.Resolved {
		stackDir := filepath.Join(a.projectDir, managedDir, stackID)
		if _, err := os.Stat(stackDir); os.IsNotExist(err) {
			allHashesOK = false
			continue
		}
		result := filemanager.VerifyStack(a.projectDir, managedDir, stackID, filemanager.StackVerifyInfo{
			Hash:       rs.Hash,
			Files:      rs.Files,
			FileHashes: rs.FileHashes,
		})
		if !result.OK {
			allHashesOK = false
		}
	}
	if allHashesOK {
		a.output.Success("All file hashes match")
	} else {
		a.output.Error("Some file hashes don't match — run: ai-instructions sync")
		allOK = false
	}

	if allOK {
		fmt.Println()
		a.output.Success("Everything looks good!")
	}

	return nil
}
