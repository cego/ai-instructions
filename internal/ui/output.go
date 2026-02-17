package ui

import (
	"fmt"
	"os"
	"strings"
)

// Output handles styled terminal output.
type Output struct {
	noColor bool
}

// NewOutput creates a new Output instance.
func NewOutput() *Output {
	return &Output{}
}

// SetNoColor disables colored output.
func (o *Output) SetNoColor(v bool) {
	o.noColor = v
}

// Success prints a success message with a green checkmark.
func (o *Output) Success(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	if o.noColor {
		fmt.Fprintf(os.Stdout, "OK %s\n", msg)
	} else {
		fmt.Fprintf(os.Stdout, "\033[32m✓\033[0m %s\n", msg)
	}
}

// Error prints an error message with a red X.
func (o *Output) Error(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	if o.noColor {
		fmt.Fprintf(os.Stderr, "FAIL %s\n", msg)
	} else {
		fmt.Fprintf(os.Stderr, "\033[31m✗\033[0m %s\n", msg)
	}
}

// Warning prints a warning message with a yellow exclamation.
func (o *Output) Warning(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	if o.noColor {
		fmt.Fprintf(os.Stderr, "WARN %s\n", msg)
	} else {
		fmt.Fprintf(os.Stderr, "\033[33m!\033[0m %s\n", msg)
	}
}

// Info prints an informational message.
func (o *Output) Info(format string, args ...any) {
	fmt.Fprintf(os.Stdout, format+"\n", args...)
}

// Println prints a line to stdout.
func (o *Output) Println(format string, args ...any) {
	fmt.Fprintf(os.Stdout, format+"\n", args...)
}

// Debug prints a debug message to stderr.
func (o *Output) Debug(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	if o.noColor {
		fmt.Fprintf(os.Stderr, "DEBUG %s\n", msg)
	} else {
		fmt.Fprintf(os.Stderr, "\033[36m[debug]\033[0m %s\n", msg)
	}
}

// Table prints a simple aligned table.
func (o *Output) Table(headers []string, rows [][]string) {
	if len(rows) == 0 {
		return
	}

	// Calculate column widths
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = len(h)
	}
	for _, row := range rows {
		for i, cell := range row {
			if i < len(widths) && len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}

	// Print header
	for i, h := range headers {
		fmt.Fprintf(os.Stdout, "%-*s  ", widths[i], h)
	}
	fmt.Fprintln(os.Stdout)

	// Print separator
	for i, w := range widths {
		fmt.Fprintf(os.Stdout, "%s", strings.Repeat("-", w))
		if i < len(widths)-1 {
			fmt.Fprint(os.Stdout, "  ")
		}
	}
	fmt.Fprintln(os.Stdout)

	// Print rows
	for _, row := range rows {
		for i, cell := range row {
			if i < len(widths) {
				fmt.Fprintf(os.Stdout, "%-*s  ", widths[i], cell)
			}
		}
		fmt.Fprintln(os.Stdout)
	}
}

