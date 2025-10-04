/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package ui

import (
	"os"

	"github.com/charmbracelet/fang"
	"github.com/charmbracelet/lipgloss/v2"
)

// StyleSet contains all the styles for the interactive diff viewer
type StyleSet struct {
	// Header styles
	Header      lipgloss.Style
	HeaderTitle lipgloss.Style
	HeaderValue lipgloss.Style

	// Status styles
	StatusNew      lipgloss.Style
	StatusChanges  lipgloss.Style
	StatusNoChange lipgloss.Style

	// Section styles
	SectionHeader         lipgloss.Style
	SectionHeaderInactive lipgloss.Style
	SectionActive         lipgloss.Style
	SectionInactive       lipgloss.Style

	// Change type styles
	Added    lipgloss.Style
	Removed  lipgloss.Style
	Modified lipgloss.Style

	// Content styles
	Key    lipgloss.Style
	Value  lipgloss.Style
	Arrow  lipgloss.Style
	Subtle lipgloss.Style

	// Semantic styles
	Success lipgloss.Style
	Warning lipgloss.Style
	Error   lipgloss.Style

	// Layout styles
	Footer    lipgloss.Style
	Separator lipgloss.Style

	useColour bool
}

// NewStyleSet creates a new style set using Fang's color scheme for consistency
// with the rest of the application.
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
func NewStyleSet(useColour bool) *StyleSet {
	s := &StyleSet{useColour: useColour}

	if useColour {
		// Detect dark background and get Fang's color scheme
		hasDark := lipgloss.HasDarkBackground(os.Stdin, os.Stdout)
		lightDark := lipgloss.LightDark(hasDark)
		scheme := fang.DefaultColorScheme(lightDark)

		// Border color - use dimmed color for borders
		borderColor := scheme.DimmedArgument

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

		// Status styles - map to semantic colors
		s.StatusNew = lipgloss.NewStyle().
			Foreground(scheme.Flag). // Use Flag color for new/important items
			Bold(true)

		s.StatusChanges = lipgloss.NewStyle().
			Foreground(scheme.Command). // Command color for changes
			Bold(true)

		s.StatusNoChange = lipgloss.NewStyle().
			Foreground(scheme.Comment). // Comment color for no changes
			Bold(true)

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

		// Change type styles - semantic diff colors
		s.Added = lipgloss.NewStyle().
			Foreground(scheme.Flag). // Flag color (typically green) for additions
			Bold(true)

		s.Removed = lipgloss.NewStyle().
			Foreground(scheme.ErrorDetails). // Error color for removals
			Bold(true)

		s.Modified = lipgloss.NewStyle().
			Foreground(scheme.Command). // Command color for modifications
			Bold(true)

		// Content styles
		s.Key = lipgloss.NewStyle().
			Foreground(scheme.Argument) // Argument color for keys

		s.Value = lipgloss.NewStyle().
			Foreground(scheme.Base) // Base text color for values

		s.Arrow = lipgloss.NewStyle().
			Foreground(scheme.Dash) // Dash color for arrows/separators

		s.Subtle = lipgloss.NewStyle().
			Foreground(scheme.Comment) // Comment color for subtle text

		// Semantic styles
		s.Success = lipgloss.NewStyle().
			Foreground(scheme.Flag) // Flag color for success

		s.Warning = lipgloss.NewStyle().
			Foreground(scheme.Command). // Command color for warnings
			Bold(true)

		s.Error = lipgloss.NewStyle().
			Foreground(scheme.ErrorDetails). // Error color from Fang
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
		// Plain styles without colour
		plainStyle := lipgloss.NewStyle()

		s.Header = plainStyle.
			BorderStyle(lipgloss.NormalBorder()).
			BorderBottom(true).
			Padding(0, 1)

		s.HeaderTitle = plainStyle.Bold(true)
		s.HeaderValue = plainStyle
		s.StatusNew = plainStyle.Bold(true)
		s.StatusChanges = plainStyle.Bold(true)
		s.StatusNoChange = plainStyle.Bold(true)
		s.SectionHeader = plainStyle.Bold(true).Padding(0, 1)
		s.SectionHeaderInactive = plainStyle.Padding(0, 1)
		s.SectionActive = plainStyle.Bold(true).MarginRight(2)
		s.SectionInactive = plainStyle.MarginRight(2)
		s.Added = plainStyle.Bold(true)
		s.Removed = plainStyle.Bold(true)
		s.Modified = plainStyle.Bold(true)
		s.Key = plainStyle
		s.Value = plainStyle
		s.Arrow = plainStyle
		s.Subtle = plainStyle
		s.Success = plainStyle
		s.Warning = plainStyle.Bold(true)
		s.Error = plainStyle.Bold(true)

		s.Footer = plainStyle.
			BorderStyle(lipgloss.NormalBorder()).
			BorderTop(true).
			Padding(0, 1)

		s.Separator = plainStyle
	}

	return s
}
