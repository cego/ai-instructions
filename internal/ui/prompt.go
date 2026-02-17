package ui

import (
	"os"
	"sort"
	"strings"

	"github.com/charmbracelet/huh"
)

// IsCI returns true if running in a CI environment.
// gitlab-ci-local sets GITLAB_CI=false, which should not be treated as CI.
func IsCI() bool {
	return isTruthy(os.Getenv("CI")) ||
		isTruthy(os.Getenv("AI_INSTRUCTIONS_CI")) ||
		isTruthy(os.Getenv("GITHUB_ACTIONS")) ||
		isTruthy(os.Getenv("GITLAB_CI"))
}

func isTruthy(v string) bool {
	return v != "" && v != "false" && v != "0"
}

// CategoryStacks groups stacks by category for display.
type CategoryStacks struct {
	Category string
	Stacks   []StackOption
}

// StackOption represents a selectable stack.
type StackOption struct {
	ID          string
	Name        string
	Description string
	Category    string
}

// SelectStacks prompts the user to select stacks, grouped by category.
func SelectStacks(stacks []StackOption) ([]string, error) {
	// Group by category
	categories := make(map[string][]StackOption)
	for _, s := range stacks {
		categories[s.Category] = append(categories[s.Category], s)
	}

	// Sort category names
	catNames := make([]string, 0, len(categories))
	for c := range categories {
		catNames = append(catNames, c)
	}
	sort.Strings(catNames)

	// Build options with category headers
	var options []huh.Option[string]
	for _, cat := range catNames {
		for _, s := range categories[cat] {
			label := s.ID
			if s.Description != "" {
				label += " — " + s.Description
			}
			catLabel := cat
			if len(cat) > 0 {
				catLabel = strings.ToUpper(cat[:1]) + cat[1:]
			}
			options = append(options, huh.NewOption(catLabel+": "+label, s.ID))
		}
	}

	var selected []string
	err := huh.NewMultiSelect[string]().
		Title("What tech stacks do you want instructions for?").
		Options(options...).
		Value(&selected).
		Run()
	return selected, err
}

// Confirm prompts the user for a yes/no confirmation.
func Confirm(title string) (bool, error) {
	var confirmed bool
	err := huh.NewConfirm().
		Title(title).
		Affirmative("Yes").
		Negative("No").
		Value(&confirmed).
		Run()
	return confirmed, err
}
