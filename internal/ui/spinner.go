package ui

import (
	"github.com/charmbracelet/huh/spinner"
)

// WithSpinner runs a function with a spinner. In CI mode, runs without spinner.
func WithSpinner(title string, fn func() error) error {
	if IsCI() {
		return fn()
	}
	var actionErr error
	err := spinner.New().
		Title(title).
		Action(func() {
			actionErr = fn()
		}).
		Run()
	if err != nil {
		return err
	}
	return actionErr
}
