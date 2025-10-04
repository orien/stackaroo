/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package ui

import (
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

// NewStyleSet creates a new style set
func NewStyleSet(useColour bool) *StyleSet {
	s := &StyleSet{useColour: useColour}

	if useColour {
		// Header styles
		s.Header = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderBottom(true).
			BorderForeground(lipgloss.Color("240")).
			Padding(0, 1)

		s.HeaderTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("12")) // Bright Blue

		s.HeaderValue = lipgloss.NewStyle().
			Foreground(lipgloss.Color("15")) // White

		// Status styles
		s.StatusNew = lipgloss.NewStyle().
			Foreground(lipgloss.Color("10")). // Green
			Bold(true)

		s.StatusChanges = lipgloss.NewStyle().
			Foreground(lipgloss.Color("11")). // Yellow
			Bold(true)

		s.StatusNoChange = lipgloss.NewStyle().
			Foreground(lipgloss.Color("8")). // Gray
			Bold(true)

		// Section styles
		s.SectionHeader = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("14")).  // Cyan
			Background(lipgloss.Color("236")). // Dark gray background
			Padding(0, 1)

		s.SectionHeaderInactive = lipgloss.NewStyle().
			Foreground(lipgloss.Color("8")). // Gray
			Padding(0, 1)

		s.SectionActive = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("14")). // Cyan
			MarginRight(2)

		s.SectionInactive = lipgloss.NewStyle().
			Foreground(lipgloss.Color("8")). // Gray
			MarginRight(2)

		// Change type styles
		s.Added = lipgloss.NewStyle().
			Foreground(lipgloss.Color("10")). // Green
			Bold(true)

		s.Removed = lipgloss.NewStyle().
			Foreground(lipgloss.Color("9")). // Red
			Bold(true)

		s.Modified = lipgloss.NewStyle().
			Foreground(lipgloss.Color("11")). // Yellow
			Bold(true)

		// Content styles
		s.Key = lipgloss.NewStyle().
			Foreground(lipgloss.Color("14")) // Cyan

		s.Value = lipgloss.NewStyle().
			Foreground(lipgloss.Color("15")) // White

		s.Arrow = lipgloss.NewStyle().
			Foreground(lipgloss.Color("8")) // Gray

		s.Subtle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("8")) // Gray

		// Semantic styles
		s.Success = lipgloss.NewStyle().
			Foreground(lipgloss.Color("10")) // Green

		s.Warning = lipgloss.NewStyle().
			Foreground(lipgloss.Color("11")). // Yellow
			Bold(true)

		s.Error = lipgloss.NewStyle().
			Foreground(lipgloss.Color("9")). // Red
			Bold(true)

		// Layout styles
		s.Footer = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderTop(true).
			BorderForeground(lipgloss.Color("240")).
			Padding(0, 1)

		s.Separator = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")) // Dark gray

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
