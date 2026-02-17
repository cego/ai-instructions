package cli

import (
	"context"
	"fmt"

	"github.com/company/ai-instructions/internal/exitcodes"
	"github.com/company/ai-instructions/internal/filemanager"
	"github.com/company/ai-instructions/internal/injector"
	"github.com/company/ai-instructions/internal/registry"
	"github.com/spf13/cobra"
)

func (a *App) newVerifyCmd() *cobra.Command {
	var strict bool

	cmd := &cobra.Command{
		Use:   "verify",
		Short: "Verify instruction files are up to date and intact",
		Long:  "CI command: verifies freshness, integrity, and managed blocks. Exit 0 = OK, exit 1 = failed.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.runVerify(cmd.Context(), strict)
		},
	}

	cmd.Flags().BoolVar(&strict, "strict", false, "fail on registry unreachable (default: warn only)")
	return cmd
}

func (a *App) runVerify(ctx context.Context, strict bool) error {
	if err := a.RequireProject(); err != nil {
		return err
	}

	managedDir := a.getManagedDir()

	var issues []string
	var outdatedStacks []string
	var reg *registry.Registry

	// 1. Check freshness against registry
	registryReachable := true
	client, clientErr := a.newRegistryClient()
	if clientErr == nil {
		var fetchErr error
		reg, fetchErr = client.FetchRegistry(ctx)
		if fetchErr != nil {
			registryReachable = false
			if strict {
				return &ExitError{
					Code:    exitcodes.NetworkError,
					Message: fmt.Sprintf("registry unreachable (strict mode): %v", fetchErr),
				}
			}
			a.output.Warning("Registry unreachable, skipping freshness check: %v", fetchErr)
		} else {
			for stackID, resolved := range a.config.Resolved {
				if regMeta, ok := reg.Stacks[stackID]; ok {
					if regMeta.Version != resolved.Version {
						outdatedStacks = append(outdatedStacks, stackID)
						issues = append(issues, fmt.Sprintf(
							"outdated: %s %s → %s",
							stackID, resolved.Version, regMeta.Version,
						))
					}
				}
			}
		}
	} else if strict {
		return &ExitError{
			Code:    exitcodes.ConfigError,
			Message: clientErr.Error(),
		}
	} else {
		registryReachable = false
		a.output.Warning("Registry not configured, skipping freshness check")
	}

	// 2. Verify local file integrity
	verifyInfos := make(map[string]filemanager.StackVerifyInfo)
	for stackID, resolved := range a.config.Resolved {
		verifyInfos[stackID] = filemanager.StackVerifyInfo{
			Hash:       resolved.Hash,
			Files:      resolved.Files,
			FileHashes: resolved.FileHashes,
		}
	}

	results := filemanager.VerifyAll(a.projectDir, managedDir, verifyInfos)
	var tampered []string
	for _, r := range results {
		if !r.OK {
			for _, f := range r.Missing {
				issues = append(issues, fmt.Sprintf("missing: %s/%s", r.Stack, f))
			}
			tampered = append(tampered, r.Tampered...)
		}
	}

	// 3. Verify managed blocks in target files
	var stackOrder []string
	for stackID := range a.config.Resolved {
		stackOrder = append(stackOrder, stackID)
	}
	injectorConfigs := buildInjectorConfigs(stackOrder, a.config.Resolved, managedDir)

	blockResults := injector.VerifyAll(a.projectDir, injectorConfigs)
	var missingBlocks []string
	for _, r := range blockResults {
		if !r.HasBlock {
			missingBlocks = append(missingBlocks, r.Filename)
			issues = append(issues, fmt.Sprintf("missing managed block: %s", r.Filename))
		}
	}

	// Print results
	if len(issues) == 0 {
		totalFiles := countResolvedFiles(a.config.Resolved)
		a.output.Success("All %d stacks verified, %d instruction files up to date", len(a.config.Resolved), totalFiles)
		if !registryReachable {
			a.output.Warning("Freshness could not be verified (registry unreachable)")
		}
		return nil
	}

	a.output.Error("Verification failed")
	fmt.Println()

	if len(outdatedStacks) > 0 {
		a.output.Println("Outdated stacks (registry has newer version):")
		for _, s := range outdatedStacks {
			regVersion := "?"
			if reg != nil {
				if meta, ok := reg.Stacks[s]; ok {
					regVersion = meta.Version
				}
			}
			a.output.Println("  %s   %s → %s", s, a.config.Resolved[s].Version, regVersion)
		}
		fmt.Println()
	}

	if len(tampered) > 0 {
		a.output.Println("Tampered files (local files don't match resolved hashes):")
		for _, f := range tampered {
			a.output.Println("  %s", f)
		}
		fmt.Println()
	}

	if len(missingBlocks) > 0 {
		a.output.Println("Missing managed block:")
		for _, f := range missingBlocks {
			a.output.Println("  %s — AI-INSTRUCTIONS markers not found", f)
		}
		fmt.Println()
	}

	a.output.Println("Run: ai-instructions sync")

	return &ExitError{Code: exitcodes.VerificationFailed, Message: "verification failed"}
}
