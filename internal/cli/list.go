package cli

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

func (a *App) newListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all available stacks from the registry",
		Long:  "Shows all registry stacks grouped by category. Installed stacks are marked with a checkmark and show local vs registry version.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.runList(cmd.Context())
		},
	}
}

func (a *App) runList(ctx context.Context) error {
	client, err := a.newRegistryClient()
	if err != nil {
		return err
	}

	reg, err := client.FetchRegistry(ctx)
	if err != nil {
		return err
	}

	// Load project config if available (works without init)
	_ = a.LoadProjectConfig()

	installed := make(map[string]string) // stack ID -> local version
	if a.config != nil && a.config.Resolved != nil {
		for id, rs := range a.config.Resolved {
			installed[id] = rs.Version
		}
	}

	// Group by category
	type stackEntry struct {
		id            string
		description   string
		version       string
		depends       []string
		localVersion  string
		isInstalled   bool
	}

	categories := make(map[string][]stackEntry)
	for id, meta := range reg.Stacks {
		localVersion, isInstalled := installed[id]
		categories[meta.Category] = append(categories[meta.Category], stackEntry{
			id:           id,
			description:  meta.Description,
			version:      meta.Version,
			depends:      meta.Depends,
			localVersion: localVersion,
			isInstalled:  isInstalled,
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
			versionInfo := e.version
			if e.isInstalled {
				status = "* "
				if e.localVersion != e.version {
					versionInfo = fmt.Sprintf("%s (local: %s)", e.version, e.localVersion)
				}
			}

			deps := ""
			if len(e.depends) > 0 {
				deps = fmt.Sprintf(" (depends: %s)", strings.Join(e.depends, ", "))
			}

			a.output.Println("  %s%-14s %s  %s%s", status, e.id, versionInfo, e.description, deps)
		}
		a.output.Println("")
	}

	installedCount := len(installed)
	totalCount := len(reg.Stacks)
	if installedCount > 0 {
		a.output.Println("* = installed (%d/%d)", installedCount, totalCount)
	} else {
		a.output.Println("%d stacks available", totalCount)
	}

	return nil
}
