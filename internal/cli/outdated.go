package cli

import (
	"context"
	"fmt"
	"sort"

	"github.com/spf13/cobra"
)

func (a *App) newOutdatedCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "outdated",
		Short: "Show outdated stacks",
		Long:  "Compare locked versions against the registry to show available updates.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.runOutdated(cmd.Context())
		},
	}
}

func (a *App) runOutdated(ctx context.Context) error {
	if err := a.RequireProject(); err != nil {
		return err
	}

	client, err := a.newRegistryClient()
	if err != nil {
		return err
	}

	reg, err := client.FetchRegistry(ctx)
	if err != nil {
		return err
	}

	ids := make([]string, 0, len(a.config.Resolved))
	for id := range a.config.Resolved {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	headers := []string{"Stack", "Locked", "Latest", "Status"}
	var rows [][]string

	hasOutdated := false
	for _, id := range ids {
		rs := a.config.Resolved[id]
		latest := "removed"
		status := "removed from registry"

		if meta, ok := reg.Stacks[id]; ok {
			latest = meta.Version
			if meta.Version == rs.Version {
				status = "up to date"
			} else {
				status = "update available"
				hasOutdated = true
			}
		}

		rows = append(rows, []string{id, rs.Version, latest, status})
	}

	a.output.Table(headers, rows)

	if !hasOutdated {
		fmt.Println()
		a.output.Success("All stacks are up to date")
	}

	return nil
}
