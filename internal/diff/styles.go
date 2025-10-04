/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package diff

import (
	"os"

	"github.com/charmbracelet/fang"
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

// NewStyles creates a new unified style set using Fang's color scheme for consistency
// across both plain text and interactive UI output.
//
// Color Mapping from Fang ColorScheme:
//   - Title       -> Header titles, section headers
//   - Base        -> Normal text, values
//   - Command     -> Active sections, changes, warnings
//   - Flag        -> Success states, additions, new items
//   - Argument    -> Keys, parameter names
//   - Comment     -> Subtle text, no changes, inactive items
//   - DimmedArgument -> Borders, inactive sections
//   - Dash        -> Separators, arrows
//   - ErrorDetails -> Errors, removals
func NewStyles(useColour bool) *Styles {
	s := &Styles{UseColour: useColour}

	if useColour {
		// Detect dark background and get Fang's color scheme
		hasDark := lipgloss.HasDarkBackground(os.Stdin, os.Stdout)
		lightDark := lipgloss.LightDark(hasDark)
		scheme := fang.DefaultColorScheme(lightDark)

		// Border color - use dimmed color for borders
		borderColor := scheme.DimmedArgument

		// Change type colours - use explicit ANSI colors for diff consistency
		// (traditional red/green diff colors are universal and expected)
		s.Added = lipgloss.NewStyle().
			Foreground(lipgloss.Color("10")) // ANSI Green for additions

		s.Removed = lipgloss.NewStyle().
			Foreground(lipgloss.Color("9")) // ANSI Red for removals

		s.Modified = lipgloss.NewStyle().
			Foreground(lipgloss.Color("11")) // ANSI Yellow for modifications

		// Status colours
		s.StatusNew = lipgloss.NewStyle().
			Foreground(scheme.Flag). // Flag color for new/important items
			Bold(true)

		s.StatusChanges = lipgloss.NewStyle().
			Foreground(scheme.Command). // Command color for changes
			Bold(true)

		s.StatusNoChange = lipgloss.NewStyle().
			Foreground(scheme.Comment). // Comment color for no changes
			Bold(true)

		// Header styles - using Fang's Title and Description colors
		s.Header = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderBottom(true).
			BorderForeground(borderColor).
			Padding(0, 1)

		s.HeaderTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(scheme.Title)

		s.HeaderValue = lipgloss.NewStyle().
			Foreground(scheme.Base)

		// Section styles - using Command color for active sections
		s.SectionHeader = lipgloss.NewStyle().
			Bold(true).
			Foreground(scheme.Command).
			Background(lightDark(
				lipgloss.Color("254"), // light background
				lipgloss.Color("236"), // dark background
			)).
			Padding(0, 1)

		s.SectionHeaderInactive = lipgloss.NewStyle().
			Foreground(scheme.DimmedArgument).
			Padding(0, 1)

		s.SectionActive = lipgloss.NewStyle().
			Bold(true).
			Foreground(scheme.Command).
			MarginRight(2)

		s.SectionInactive = lipgloss.NewStyle().
			Foreground(scheme.DimmedArgument).
			MarginRight(2)

		s.SubSection = lipgloss.NewStyle().
			Foreground(scheme.Comment) // Comment color for subsections

		// Content styles
		s.Key = lipgloss.NewStyle().
			Foreground(scheme.Argument) // Argument color for keys

		s.Value = lipgloss.NewStyle().
			Foreground(scheme.Base) // Base text color for values

		s.Arrow = lipgloss.NewStyle().
			Foreground(scheme.Dash) // Dash color for arrows

		s.Subtle = lipgloss.NewStyle().
			Foreground(scheme.Comment) // Comment color for subtle text

		s.Bold = lipgloss.NewStyle().Bold(true)

		// Semantic styles
		s.Success = lipgloss.NewStyle().
			Foreground(scheme.Flag) // Flag color for success

		s.Warning = lipgloss.NewStyle().
			Foreground(scheme.Command). // Command color for warnings
			Bold(true)

		s.Error = lipgloss.NewStyle().
			Foreground(scheme.ErrorDetails). // Error color from Fang
			Bold(true)

		// Risk level colours
		s.RiskLow = lipgloss.NewStyle().
			Foreground(scheme.Flag).
			Bold(true)

		s.RiskMedium = lipgloss.NewStyle().
			Foreground(scheme.Command).
			Bold(true)

		s.RiskHigh = lipgloss.NewStyle().
			Foreground(scheme.ErrorDetails).
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
