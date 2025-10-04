/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package ui

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/orien/stackaroo/internal/diff"
)

// ShowDiff displays an interactive diff viewer for browsing changes
// Returns an error if the TUI fails to initialise or run
func ShowDiff(result *diff.Result) error {
	// Check if we should use interactive mode
	if !ShouldUseInteractive() {
		// Fall back to plain text output
		fmt.Print(result.String())
		return nil
	}

	// Create and run the interactive viewer
	model := NewModel(result, ViewOnly)
	program := tea.NewProgram(
		model,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	finalModel, err := program.Run()
	if err != nil {
		return fmt.Errorf("failed to run interactive viewer: %w", err)
	}

	// Check if user quit normally
	if m, ok := finalModel.(Model); ok && !m.quitting {
		return fmt.Errorf("viewer exited unexpectedly")
	}

	return nil
}

// ShowDiffWithConfirmation displays an interactive diff viewer and prompts for confirmation
// Returns true if the user confirmed, false if they cancelled or quit
// The message parameter is displayed as context for the confirmation
func ShowDiffWithConfirmation(result *diff.Result, message string) (bool, error) {
	// Check if we should use interactive mode
	if !ShouldUseInteractive() {
		// Fall back to plain text output with simple confirmation
		fmt.Print(result.String())
		fmt.Println()

		// Use the simple prompt package
		fmt.Printf("\n%s [y/N]: ", message)
		var response string
		if _, err := fmt.Scanln(&response); err != nil {
			// Treat read errors as "no" response
			return false, nil
		}

		return response == "y" || response == "Y" || response == "yes" || response == "Yes", nil
	}

	// Create and run the interactive viewer in confirmation mode
	// This shows the diff AND the confirmation prompt simultaneously
	model := NewModel(result, Confirmation)
	program := tea.NewProgram(
		model,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	finalModel, err := program.Run()
	if err != nil {
		return false, fmt.Errorf("failed to run interactive viewer: %w", err)
	}

	// Extract the result
	if m, ok := finalModel.(Model); ok {
		if m.Cancelled() {
			return false, nil
		}
		return m.Confirmed(), nil
	}

	return false, fmt.Errorf("viewer exited unexpectedly")
}

// ShouldUseInteractive determines whether to use the interactive TUI
// Returns false if:
// - NO_COLOR environment variable is set
// - TERM is "dumb" or empty
// - stdout is not a TTY
// - STACKAROO_PLAIN environment variable is set
func ShouldUseInteractive() bool {
	// Check explicit disable via environment
	if os.Getenv("STACKAROO_PLAIN") != "" {
		return false
	}

	// Check NO_COLOR
	if os.Getenv("NO_COLOR") != "" {
		return false
	}

	// Check TERM
	term := os.Getenv("TERM")
	if term == "dumb" || term == "" {
		return false
	}

	// Check if stdout is a TTY
	fileInfo, err := os.Stdout.Stat()
	if err != nil {
		return false
	}

	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}

// ForceInteractive sets an environment variable to force interactive mode
// This is primarily for testing
func ForceInteractive(force bool) {
	if force {
		_ = os.Unsetenv("STACKAROO_PLAIN")
		_ = os.Unsetenv("NO_COLOR")
	} else {
		_ = os.Setenv("STACKAROO_PLAIN", "1")
	}
}
