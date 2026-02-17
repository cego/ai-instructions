package cli

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

func (a *App) newStacksCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stacks",
		Short: "List all available stacks from the registry",
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.runStacks(cmd.Context())
		},
	}
}

func (a *App) runStacks(ctx context.Context) error {
	client, err := a.newRegistryClient()
	if err != nil {
		return err
	}

	reg, err := client.FetchRegistry(ctx)
	if err != nil {
		return err
	}

	// Load project config if available (ignore errors — project may not be initialized)
	_ = a.LoadProjectConfig()

	installed := make(map[string]bool)
	if a.config != nil {
		for id := range a.config.Resolved {
			installed[id] = true
		}
	}

	// Group by category
	type stackEntry struct {
		id          string
		description string
		version     string
		depends     []string
		installed   bool
	}

	categories := make(map[string][]stackEntry)
	for id, meta := range reg.Stacks {
		categories[meta.Category] = append(categories[meta.Category], stackEntry{
			id:          id,
			description: meta.Description,
			version:     meta.Version,
			depends:     meta.Depends,
			installed:   installed[id],
		})
	}

	catNames := make([]string, 0, len(categories))
	for c := range categories {
		catNames = append(catNames, c)
	}
	sort.Strings(catNames)

	for _, cat := range catNames {
		entries := categories[cat]
		sort.Slice(entries, func(i, j int) bool {
			return entries[i].id < entries[j].id
		})

		label := cat
		if len(cat) > 0 {
			label = strings.ToUpper(cat[:1]) + cat[1:]
		}
		a.output.Println("%s:", label)

		for _, e := range entries {
			status := "  "
			if e.installed {
				status = "✓ "
			}

			deps := ""
			if len(e.depends) > 0 {
				deps = fmt.Sprintf(" (depends: %s)", strings.Join(e.depends, ", "))
			}

			a.output.Println("  %s%-14s %s  %s%s", status, e.id, e.version, e.description, deps)
		}
		a.output.Println("")
	}

	installedCount := len(installed)
	totalCount := len(reg.Stacks)
	if installedCount > 0 {
		a.output.Println("✓ = installed (%d/%d)", installedCount, totalCount)
	} else {
		a.output.Println("%d stacks available", totalCount)
	}

	return nil
}
