/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package diff

import (
	"os"

	"charm.land/lipgloss/v2"
)

// Styles contains all the styles for rendering diff output
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

// Colours are optimised based on terminal background (dark vs light).
func NewStyles(useColour bool) *Styles {
	s := &Styles{UseColour: useColour}

	if useColour {
		// Detect terminal background and select appropriate colours
		hasDark := lipgloss.HasDarkBackground(os.Stdin, os.Stdout)

		// Define colour palette based on background
		var (
			headerText       string
			baseText         string
			warningText      string
			successText      string
			keyText          string
			subtleText       string
			borderText       string
			separatorText    string
			activeText       string
			errorText        string
			activeBackground string
		)

		if hasDark {
			// Dark background colours - optimised for readability on dark terminals
			headerText = "12"        // Bright Blue
			baseText = "15"          // White
			warningText = "11"       // Yellow
			successText = "10"       // Green
			keyText = "14"           // Cyan
			subtleText = "8"         // Dark Grey
			borderText = "240"       // Dimmed Grey
			separatorText = "243"    // Medium Grey
			activeText = "13"        // Magenta
			errorText = "9"          // Red
			activeBackground = "236" // Dark background
		} else {
			// Light background colours - optimised for readability on light terminals
			headerText = "4"         // Blue
			baseText = "0"           // Black
			warningText = "3"        // Yellow/Brown
			successText = "2"        // Green
			keyText = "6"            // Cyan
			subtleText = "8"         // Grey
			borderText = "245"       // Light Grey
			separatorText = "242"    // Medium Grey
			activeText = "5"         // Magenta
			errorText = "1"          // Red
			activeBackground = "254" // Light background
		}

		// Border colour - use dimmed colour for borders
		borderColor := lipgloss.Color(borderText)

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
			Foreground(lipgloss.Color(successText)).
			Bold(true)

		s.StatusChanges = lipgloss.NewStyle().
			Foreground(lipgloss.Color(warningText)).
			Bold(true)

		s.StatusNoChange = lipgloss.NewStyle().
			Foreground(lipgloss.Color(subtleText)).
			Bold(true)

		// Header styles
		s.Header = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderBottom(true).
			BorderForeground(borderColor).
			Padding(0, 1)

		s.HeaderTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(headerText))

		s.HeaderValue = lipgloss.NewStyle().
			Foreground(lipgloss.Color(baseText))

		// Section styles
		s.SectionHeader = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(activeText)).
			Background(lipgloss.Color(activeBackground)).
			Padding(0, 1)

		s.SectionHeaderInactive = lipgloss.NewStyle().
			Foreground(lipgloss.Color(borderText)).
			Padding(0, 1)

		s.SectionActive = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(activeText)).
			MarginRight(2)

		s.SectionInactive = lipgloss.NewStyle().
			Foreground(lipgloss.Color(borderText)).
			MarginRight(2)

		s.SubSection = lipgloss.NewStyle().
			Foreground(lipgloss.Color(subtleText))

		// Content styles
		s.Key = lipgloss.NewStyle().
			Foreground(lipgloss.Color(keyText))

		s.Value = lipgloss.NewStyle()

		s.Arrow = lipgloss.NewStyle().
			Foreground(lipgloss.Color(separatorText))

		s.Subtle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(subtleText))

		s.Bold = lipgloss.NewStyle().Bold(true)

		// Semantic styles
		s.Success = lipgloss.NewStyle().
			Foreground(lipgloss.Color(successText))

		s.Warning = lipgloss.NewStyle().
			Foreground(lipgloss.Color(warningText)).
			Bold(true)

		s.Error = lipgloss.NewStyle().
			Foreground(lipgloss.Color(errorText)).
			Bold(true)

		// Risk level colours
		s.RiskLow = lipgloss.NewStyle().
			Foreground(lipgloss.Color(successText)).
			Bold(true)

		s.RiskMedium = lipgloss.NewStyle().
			Foreground(lipgloss.Color(warningText)).
			Bold(true)

		s.RiskHigh = lipgloss.NewStyle().
			Foreground(lipgloss.Color(errorText)).
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
