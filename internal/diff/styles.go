/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package diff

import (
	"os"

	"github.com/charmbracelet/lipgloss/v2"
)

// Styles contains all the styles for rendering diff output (both plain text and interactive UI)
type Styles struct {
	// Change type styles
	Added    lipgloss.Style
	Removed  lipgloss.Style
	Modified lipgloss.Style

	// Status styles
	StatusNew      lipgloss.Style
	StatusChanges  lipgloss.Style
	StatusNoChange lipgloss.Style

	// Header styles
	Header      lipgloss.Style // Header with border
	HeaderTitle lipgloss.Style
	HeaderValue lipgloss.Style

	// Section styles
	SectionHeader         lipgloss.Style
	SectionHeaderInactive lipgloss.Style
	SectionActive         lipgloss.Style
	SectionInactive       lipgloss.Style
	SubSection            lipgloss.Style

	// Content styles
	Key    lipgloss.Style
	Value  lipgloss.Style
	Arrow  lipgloss.Style
	Subtle lipgloss.Style
	Bold   lipgloss.Style

	// Semantic styles
	Success lipgloss.Style
	Warning lipgloss.Style
	Error   lipgloss.Style

	// Risk level styles
	RiskLow    lipgloss.Style
	RiskMedium lipgloss.Style
	RiskHigh   lipgloss.Style

	// Layout styles
	Footer    lipgloss.Style // Footer with border
	Separator lipgloss.Style

	// Whether colours are enabled
	UseColour bool
}

// colorScheme defines a consistent colour palette for the diff output
type colorScheme struct {
	HeaderText       string
	BaseText         string
	WarningText      string
	SuccessText      string
	KeyText          string
	SubtleText       string
	BorderText       string
	SeparatorText    string
	ActiveText       string
	ErrorText        string
	ActiveBackground string
}

// newColorScheme creates a colour scheme appropriate for the terminal background
func newColorScheme(hasDarkBackground bool) colorScheme {
	if hasDarkBackground {
		// Dark background colours - optimised for readability on dark terminals
		return colorScheme{
			HeaderText:       "12",  // Bright Blue
			BaseText:         "15",  // White
			WarningText:      "11",  // Yellow
			SuccessText:      "10",  // Green
			KeyText:          "14",  // Cyan
			SubtleText:       "8",   // Dark Grey
			BorderText:       "240", // Dimmed Grey
			SeparatorText:    "243", // Medium Grey
			ActiveText:       "13",  // Magenta
			ErrorText:        "9",   // Red
			ActiveBackground: "236", // Dark background
		}
	}

	// Light background colours - optimised for readability on light terminals
	return colorScheme{
		HeaderText:       "4",   // Blue
		BaseText:         "0",   // Black
		WarningText:      "3",   // Yellow/Brown
		SuccessText:      "2",   // Green
		KeyText:          "6",   // Cyan
		SubtleText:       "8",   // Grey
		BorderText:       "245", // Light Grey
		SeparatorText:    "242", // Medium Grey
		ActiveText:       "5",   // Magenta
		ErrorText:        "1",   // Red
		ActiveBackground: "254", // Light background
	}
}

// NewStyles creates a new unified style set for consistent output across both
// plain text and interactive UI.
//
// Colour Mapping:
//   - HeaderText       -> Header titles, section headers
//   - BaseText         -> Normal text, values
//   - WarningText      -> Active sections, changes, warnings
//   - SuccessText      -> Success states, additions, new items
//   - KeyText          -> Keys, parameter names
//   - SubtleText       -> Subtle text, no changes, inactive items
//   - BorderText       -> Borders, inactive sections
//   - SeparatorText    -> Separators, arrows
//   - ActiveText       -> Active sections, highlights
//   - ErrorText        -> Errors, removals
//   - ActiveBackground -> Background for active sections
func NewStyles(useColour bool) *Styles {
	s := &Styles{UseColour: useColour}

	if useColour {
		// Detect dark background and get appropriate colour scheme
		hasDark := lipgloss.HasDarkBackground(os.Stdin, os.Stdout)
		scheme := newColorScheme(hasDark)

		// Border colour - use dimmed colour for borders
		borderColor := lipgloss.Color(scheme.BorderText)

		// Change type colours - use explicit ANSI colours for diff consistency
		// (traditional red/green diff colours are universal and expected)
		s.Added = lipgloss.NewStyle().
			Foreground(lipgloss.Color("10")) // ANSI Green for additions

		s.Removed = lipgloss.NewStyle().
			Foreground(lipgloss.Color("9")) // ANSI Red for removals

		s.Modified = lipgloss.NewStyle().
			Foreground(lipgloss.Color("11")) // ANSI Yellow for modifications

		// Status colours
		s.StatusNew = lipgloss.NewStyle().
			Foreground(lipgloss.Color(scheme.SuccessText)). // Success colour for new/important items
			Bold(true)

		s.StatusChanges = lipgloss.NewStyle().
			Foreground(lipgloss.Color(scheme.WarningText)). // Warning colour for changes
			Bold(true)

		s.StatusNoChange = lipgloss.NewStyle().
			Foreground(lipgloss.Color(scheme.SubtleText)). // Subtle colour for no changes
			Bold(true)

		// Header styles
		s.Header = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderBottom(true).
			BorderForeground(borderColor).
			Padding(0, 1)

		s.HeaderTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(scheme.HeaderText))

		s.HeaderValue = lipgloss.NewStyle().
			Foreground(lipgloss.Color(scheme.BaseText))

		// Section styles
		s.SectionHeader = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(scheme.ActiveText)).
			Background(lipgloss.Color(scheme.ActiveBackground)).
			Padding(0, 1)

		s.SectionHeaderInactive = lipgloss.NewStyle().
			Foreground(lipgloss.Color(scheme.BorderText)).
			Padding(0, 1)

		s.SectionActive = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(scheme.ActiveText)).
			MarginRight(2)

		s.SectionInactive = lipgloss.NewStyle().
			Foreground(lipgloss.Color(scheme.BorderText)).
			MarginRight(2)

		s.SubSection = lipgloss.NewStyle().
			Foreground(lipgloss.Color(scheme.SubtleText)) // Subtle colour for subsections

		// Content styles
		s.Key = lipgloss.NewStyle().
			Foreground(lipgloss.Color(scheme.KeyText)) // Key colour for keys

		s.Value = lipgloss.NewStyle().
			Foreground(lipgloss.Color(scheme.BaseText)) // Base text colour for values

		s.Arrow = lipgloss.NewStyle().
			Foreground(lipgloss.Color(scheme.SeparatorText)) // Separator colour for arrows

		s.Subtle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(scheme.SubtleText)) // Subtle colour for subtle text

		s.Bold = lipgloss.NewStyle().Bold(true)

		// Semantic styles
		s.Success = lipgloss.NewStyle().
			Foreground(lipgloss.Color(scheme.SuccessText)) // Success colour for success

		s.Warning = lipgloss.NewStyle().
			Foreground(lipgloss.Color(scheme.WarningText)). // Warning colour for warnings
			Bold(true)

		s.Error = lipgloss.NewStyle().
			Foreground(lipgloss.Color(scheme.ErrorText)). // Error colour
			Bold(true)

		// Risk level colours
		s.RiskLow = lipgloss.NewStyle().
			Foreground(lipgloss.Color(scheme.SuccessText)).
			Bold(true)

		s.RiskMedium = lipgloss.NewStyle().
			Foreground(lipgloss.Color(scheme.WarningText)).
			Bold(true)

		s.RiskHigh = lipgloss.NewStyle().
			Foreground(lipgloss.Color(scheme.ErrorText)).
			Bold(true)

		// Layout styles
		s.Footer = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderTop(true).
			BorderForeground(borderColor).
			Padding(0, 1)

		s.Separator = lipgloss.NewStyle().
			Foreground(borderColor)

	} else {
		// Plain mode - create empty styles that pass through text unchanged
		// In lipgloss v2, an empty style with no properties set will render text as-is
		plainStyle := lipgloss.NewStyle()

		s.Added = plainStyle
		s.Removed = plainStyle
		s.Modified = plainStyle
		s.StatusNew = plainStyle
		s.StatusChanges = plainStyle
		s.StatusNoChange = plainStyle

		s.Header = plainStyle.
			BorderStyle(lipgloss.NormalBorder()).
			BorderBottom(true).
			Padding(0, 1)
		s.HeaderTitle = plainStyle
		s.HeaderValue = plainStyle

		s.SectionHeader = plainStyle.Padding(0, 1)
		s.SectionHeaderInactive = plainStyle.Padding(0, 1)
		s.SectionActive = plainStyle.MarginRight(2)
		s.SectionInactive = plainStyle.MarginRight(2)
		s.SubSection = plainStyle

		s.Key = plainStyle
		s.Value = plainStyle
		s.Arrow = plainStyle
		s.Subtle = plainStyle
		s.Bold = plainStyle.Bold(true)

		s.Success = plainStyle
		s.Warning = plainStyle
		s.Error = plainStyle

		s.RiskLow = plainStyle
		s.RiskMedium = plainStyle
		s.RiskHigh = plainStyle

		s.Footer = plainStyle.
			BorderStyle(lipgloss.NormalBorder()).
			BorderTop(true).
			Padding(0, 1)
		s.Separator = plainStyle
	}

	return s
}

// GetChangeSymbol returns the appropriate symbol for a change type
func (s *Styles) GetChangeSymbol(changeType ChangeType) string {
	switch changeType {
	case ChangeTypeAdd:
		return s.Added.Render("+")
	case ChangeTypeModify:
		return s.Modified.Render("~")
	case ChangeTypeRemove:
		return s.Removed.Render("-")
	default:
		return "?"
	}
}

// GetChangeSetSymbol returns the appropriate symbol for a changeset action
func (s *Styles) GetChangeSetSymbol(action string) string {
	switch action {
	case "Add":
		return s.Added.Render("+")
	case "Modify":
		return s.Modified.Render("~")
	case "Remove":
		return s.Removed.Render("-")
	default:
		return "?"
	}
}

// ShouldUseColour determines if colour output should be used
func ShouldUseColour() bool {
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
