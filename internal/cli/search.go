package cli

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

func (a *App) newSearchCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "search <term>",
		Short: "Search available stacks in the registry",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.runSearch(cmd.Context(), args[0])
		},
	}
}

func (a *App) runSearch(ctx context.Context, term string) error {
	client, err := a.newRegistryClient()
	if err != nil {
		return err
	}

	reg, err := client.FetchRegistry(ctx)
	if err != nil {
		return err
	}

	term = strings.ToLower(term)

	type match struct {
		id   string
		meta struct {
			Name        string
			Description string
			Category    string
			Depends     []string
		}
	}

	var matches []match
	for id, meta := range reg.Stacks {
		if strings.Contains(strings.ToLower(id), term) ||
			strings.Contains(strings.ToLower(meta.Name), term) ||
			strings.Contains(strings.ToLower(meta.Description), term) ||
			strings.Contains(strings.ToLower(meta.Category), term) {
			matches = append(matches, match{
				id: id,
				meta: struct {
					Name        string
					Description string
					Category    string
					Depends     []string
				}{meta.Name, meta.Description, meta.Category, meta.Depends},
			})
		}
	}

	sort.Slice(matches, func(i, j int) bool {
		return matches[i].id < matches[j].id
	})

	if len(matches) == 0 {
		a.output.Info("No stacks matching %q", term)
		return nil
	}

	a.output.Println("Available stacks matching %q:\n", term)
	for _, m := range matches {
		deps := ""
		if len(m.meta.Depends) > 0 {
			deps = fmt.Sprintf(", depends: %s", strings.Join(m.meta.Depends, ", "))
		}
		a.output.Println("  %-14s %-50s (%s%s)", m.id, m.meta.Description, m.meta.Category, deps)
	}

	return nil
}
