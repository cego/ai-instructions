package cli

import (
	"fmt"
	"sort"

	"github.com/spf13/cobra"
)

func (a *App) newListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List installed stacks",
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.runList()
		},
	}
}

func (a *App) runList() error {
	if err := a.RequireProject(); err != nil {
		return err
	}

	a.output.Println("Installed stacks:\n")

	// Sort by name
	ids := make([]string, 0, len(a.config.Resolved))
	for id := range a.config.Resolved {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	for _, id := range ids {
		rs := a.config.Resolved[id]
		origin := "(explicit)"
		if rs.DependencyOf != "" {
			origin = fmt.Sprintf("(dep of %s)", rs.DependencyOf)
		}

		fileCount := len(rs.Files)
		fileWord := "files"
		if fileCount == 1 {
			fileWord = "file"
		}

		a.output.Println("  %-25s %-20s %s   %d %s", id, origin, rs.Version, fileCount, fileWord)
	}

	a.output.Println("\nMode: %s", a.config.Mode)
	a.output.Println("Total: %d stacks, %d files", len(a.config.Resolved), countResolvedFiles(a.config.Resolved))

	return nil
}
