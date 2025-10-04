/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package diff

import (
	"os"

	"github.com/charmbracelet/lipgloss/v2"
)

// OutputStyles contains all the styles for rendering diff output
type OutputStyles struct {
	// Change type styles
	added    lipgloss.Style
	removed  lipgloss.Style
	modified lipgloss.Style

	// Status styles
	statusNew      lipgloss.Style
	statusChanges  lipgloss.Style
	statusNoChange lipgloss.Style

	// Section styles
	header        lipgloss.Style
	sectionHeader lipgloss.Style
	separator     lipgloss.Style
	subSection    lipgloss.Style

	// Risk level styles
	riskLow    lipgloss.Style
	riskMedium lipgloss.Style
	riskHigh   lipgloss.Style

	// General styles
	key   lipgloss.Style
	value lipgloss.Style
	arrow lipgloss.Style
	bold  lipgloss.Style

	// Whether colours are enabled
	useColour bool
}

// newOutputStyles creates a new set of output styles
func newOutputStyles(useColour bool) *OutputStyles {
	s := &OutputStyles{useColour: useColour}

	if useColour {
		// Change type colours
		s.added = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))    // Green
		s.removed = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))   // Red
		s.modified = lipgloss.NewStyle().Foreground(lipgloss.Color("11")) // Yellow

		// Status colours
		s.statusNew = lipgloss.NewStyle().
			Foreground(lipgloss.Color("10")).
			Bold(true)
		s.statusChanges = lipgloss.NewStyle().
			Foreground(lipgloss.Color("11")).
			Bold(true)
		s.statusNoChange = lipgloss.NewStyle().
			Foreground(lipgloss.Color("8")).
			Bold(true)

		// Section styles
		s.header = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("12")) // Bright Blue
		s.sectionHeader = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("14")) // Bright Cyan
		s.separator = lipgloss.NewStyle().
			Foreground(lipgloss.Color("8")) // Gray
		s.subSection = lipgloss.NewStyle().
			Foreground(lipgloss.Color("7")) // Light Gray

		// Risk level colours
		s.riskLow = lipgloss.NewStyle().
			Foreground(lipgloss.Color("10")).
			Bold(true)
		s.riskMedium = lipgloss.NewStyle().
			Foreground(lipgloss.Color("11")).
			Bold(true)
		s.riskHigh = lipgloss.NewStyle().
			Foreground(lipgloss.Color("9")).
			Bold(true)

		// General styles
		s.key = lipgloss.NewStyle().
			Foreground(lipgloss.Color("14")) // Cyan
		s.value = lipgloss.NewStyle().
			Foreground(lipgloss.Color("15")) // White
		s.arrow = lipgloss.NewStyle().
			Foreground(lipgloss.Color("8")) // Gray
		s.bold = lipgloss.NewStyle().Bold(true)
	} else {
		// Plain mode - create empty styles that pass through text unchanged
		// In lipgloss v2, an empty style with no properties set will render text as-is
		plainStyle := lipgloss.NewStyle()
		s.added = plainStyle
		s.removed = plainStyle
		s.modified = plainStyle
		s.statusNew = plainStyle
		s.statusChanges = plainStyle
		s.statusNoChange = plainStyle
		s.header = plainStyle
		s.sectionHeader = plainStyle
		s.separator = plainStyle
		s.subSection = plainStyle
		s.riskLow = plainStyle
		s.riskMedium = plainStyle
		s.riskHigh = plainStyle
		s.key = plainStyle
		s.value = plainStyle
		s.arrow = plainStyle
		s.bold = plainStyle
	}

	return s
}

// getChangeSymbol returns the appropriate symbol for a change type
func (s *OutputStyles) getChangeSymbol(changeType ChangeType) string {
	switch changeType {
	case ChangeTypeAdd:
		return s.added.Render("+")
	case ChangeTypeModify:
		return s.modified.Render("~")
	case ChangeTypeRemove:
		return s.removed.Render("-")
	default:
		return "?"
	}
}

// getChangeSetSymbol returns the appropriate symbol for a changeset action
func (s *OutputStyles) getChangeSetSymbol(action string) string {
	switch action {
	case "Add":
		return s.added.Render("+")
	case "Modify":
		return s.modified.Render("~")
	case "Remove":
		return s.removed.Render("-")
	default:
		return "?"
	}
}

// shouldUseColour determines if colour output should be used
func shouldUseColour() bool {
	// Check NO_COLOR environment variable (https://no-color.org/)
	if os.Getenv("NO_COLOR") != "" {
		return false
	}

	// Check TERM environment variable
	term := os.Getenv("TERM")
	if term == "dumb" || term == "" {
		return false
	}

	// Check if stdout is a terminal
	fileInfo, err := os.Stdout.Stat()
	if err != nil {
		return false
	}

	// Check if it's a character device (terminal)
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}
