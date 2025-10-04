/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package ui

import (
	"fmt"
	"os"
	"strings"

	"github.com/orien/stackaroo/internal/aws"
	"github.com/orien/stackaroo/internal/diff"
)

// buildSections creates navigable sections from a diff result
func buildSections(result *diff.Result) []Section {
	var sections []Section
	currentLine := 0

	// New stack case - simplified sections
	if !result.StackExists {
		if len(result.ParameterDiffs) > 0 {
			content := formatParameterSection(result.ParameterDiffs, true)
			sections = append(sections, Section{
				Name:       "Parameters",
				Content:    content,
				HasChanges: true,
				StartLine:  currentLine,
			})
			currentLine += countLines(content) + 4 // +4 for section header/separator
		}

		if len(result.TagDiffs) > 0 {
			content := formatTagSection(result.TagDiffs, true)
			sections = append(sections, Section{
				Name:       "Tags",
				Content:    content,
				HasChanges: true,
				StartLine:  currentLine,
			})
		}

		return sections
	}

	// Existing stack - full diff sections
	if result.TemplateChange != nil && (!result.Options.ParametersOnly && !result.Options.TagsOnly) {
		content := formatTemplateSection(result.TemplateChange)
		sections = append(sections, Section{
			Name:       "Template",
			Content:    content,
			HasChanges: result.TemplateChange.HasChanges,
			StartLine:  currentLine,
		})
		currentLine += countLines(content) + 4
	}

	if len(result.ParameterDiffs) > 0 && (!result.Options.TemplateOnly && !result.Options.TagsOnly) {
		content := formatParameterSection(result.ParameterDiffs, false)
		sections = append(sections, Section{
			Name:       "Parameters",
			Content:    content,
			HasChanges: true,
			StartLine:  currentLine,
		})
		currentLine += countLines(content) + 4
	}

	if len(result.TagDiffs) > 0 && (!result.Options.TemplateOnly && !result.Options.ParametersOnly) {
		content := formatTagSection(result.TagDiffs, false)
		sections = append(sections, Section{
			Name:       "Tags",
			Content:    content,
			HasChanges: true,
			StartLine:  currentLine,
		})
		currentLine += countLines(content) + 4
	}

	if result.ChangeSet != nil && len(result.ChangeSet.Changes) > 0 {
		content := formatChangeSetSection(result.ChangeSet)
		sections = append(sections, Section{
			Name:       "AWS Resources",
			Content:    content,
			HasChanges: true,
			StartLine:  currentLine,
		})
	}

	return sections
}

// formatTemplateSection formats template changes
func formatTemplateSection(tc *diff.TemplateChange) string {
	var s strings.Builder
	useColour := shouldUseColour()
	styles := NewStyleSet(useColour)

	if tc.HasChanges {
		s.WriteString(styles.Success.Render("✓ Template has been modified"))
		s.WriteString("\n\n")

		if tc.ResourceCount.Added > 0 || tc.ResourceCount.Modified > 0 || tc.ResourceCount.Removed > 0 {
			s.WriteString("Resource changes:\n")
			if tc.ResourceCount.Added > 0 {
				s.WriteString(fmt.Sprintf("  %s %d resources to be added\n",
					styles.Added.Render("+"),
					tc.ResourceCount.Added))
			}
			if tc.ResourceCount.Modified > 0 {
				s.WriteString(fmt.Sprintf("  %s %d resources to be modified\n",
					styles.Modified.Render("~"),
					tc.ResourceCount.Modified))
			}
			if tc.ResourceCount.Removed > 0 {
				s.WriteString(fmt.Sprintf("  %s %d resources to be removed\n",
					styles.Removed.Render("-"),
					tc.ResourceCount.Removed))
			}
		}

		if tc.Diff != "" {
			s.WriteString("\nTemplate diff:\n")
			s.WriteString(tc.Diff)
		}
	} else {
		s.WriteString(styles.Subtle.Render("✗ No template changes"))
	}

	return s.String()
}

// formatParameterSection formats parameter changes
func formatParameterSection(params []diff.ParameterDiff, isNewStack bool) string {
	var s strings.Builder
	useColour := shouldUseColour()
	styles := NewStyleSet(useColour)

	if isNewStack {
		s.WriteString("Parameters to be set:\n\n")
	}

	for _, p := range params {
		symbol := getChangeSymbol(p.ChangeType, styles)
		key := styles.Key.Render(p.Key)

		switch p.ChangeType {
		case diff.ChangeTypeAdd:
			value := styles.Value.Render(p.ProposedValue)
			s.WriteString(fmt.Sprintf("  %s %s: %s\n", symbol, key, value))
		case diff.ChangeTypeModify:
			currentVal := styles.Value.Render(p.CurrentValue)
			proposedVal := styles.Value.Render(p.ProposedValue)
			arrow := styles.Arrow.Render("→")
			s.WriteString(fmt.Sprintf("  %s %s: %s %s %s\n", symbol, key, currentVal, arrow, proposedVal))
		case diff.ChangeTypeRemove:
			value := styles.Value.Render(p.CurrentValue)
			s.WriteString(fmt.Sprintf("  %s %s: %s\n", symbol, key, value))
		}
	}

	return s.String()
}

// formatTagSection formats tag changes
func formatTagSection(tags []diff.TagDiff, isNewStack bool) string {
	var s strings.Builder
	useColour := shouldUseColour()
	styles := NewStyleSet(useColour)

	if isNewStack {
		s.WriteString("Tags to be set:\n\n")
	}

	for _, t := range tags {
		symbol := getChangeSymbol(t.ChangeType, styles)
		key := styles.Key.Render(t.Key)

		switch t.ChangeType {
		case diff.ChangeTypeAdd:
			value := styles.Value.Render(t.ProposedValue)
			s.WriteString(fmt.Sprintf("  %s %s: %s\n", symbol, key, value))
		case diff.ChangeTypeModify:
			currentVal := styles.Value.Render(t.CurrentValue)
			proposedVal := styles.Value.Render(t.ProposedValue)
			arrow := styles.Arrow.Render("→")
			s.WriteString(fmt.Sprintf("  %s %s: %s %s %s\n", symbol, key, currentVal, arrow, proposedVal))
		case diff.ChangeTypeRemove:
			value := styles.Value.Render(t.CurrentValue)
			s.WriteString(fmt.Sprintf("  %s %s: %s\n", symbol, key, value))
		}
	}

	return s.String()
}

// formatChangeSetSection formats AWS changeset information
func formatChangeSetSection(cs *aws.ChangeSetInfo) string {
	var s strings.Builder
	useColour := shouldUseColour()
	styles := NewStyleSet(useColour)

	s.WriteString("Resource Changes:\n\n")

	for _, change := range cs.Changes {
		symbol := getChangeSetSymbol(change.Action, styles)
		logicalID := styles.Key.Render(change.LogicalID)
		resourceType := styles.Value.Render(change.ResourceType)

		s.WriteString(fmt.Sprintf("  %s %s (%s)", symbol, logicalID, resourceType))

		if change.PhysicalID != "" {
			physicalID := styles.Subtle.Render(fmt.Sprintf("[%s]", change.PhysicalID))
			s.WriteString(fmt.Sprintf(" %s", physicalID))
		}

		if change.Replacement != "" && change.Replacement != "False" {
			replacement := styles.Warning.Render(fmt.Sprintf("⚠ Replacement: %s", change.Replacement))
			s.WriteString(fmt.Sprintf(" - %s", replacement))
		}

		s.WriteString("\n")

		// Add details if available
		for _, detail := range change.Details {
			detailText := styles.Subtle.Render(fmt.Sprintf("    %s", detail))
			s.WriteString(detailText)
			s.WriteString("\n")
		}
	}

	return s.String()
}

// getChangeSymbol returns the styled symbol for a change type
func getChangeSymbol(changeType diff.ChangeType, styles *StyleSet) string {
	switch changeType {
	case diff.ChangeTypeAdd:
		return styles.Added.Render("+")
	case diff.ChangeTypeModify:
		return styles.Modified.Render("~")
	case diff.ChangeTypeRemove:
		return styles.Removed.Render("-")
	default:
		return "?"
	}
}

// getChangeSetSymbol returns the styled symbol for a changeset action
func getChangeSetSymbol(action string, styles *StyleSet) string {
	switch action {
	case "Add":
		return styles.Added.Render("+")
	case "Modify":
		return styles.Modified.Render("~")
	case "Remove":
		return styles.Removed.Render("-")
	default:
		return "?"
	}
}

// countLines counts the number of lines in a string
func countLines(s string) int {
	if s == "" {
		return 0
	}
	return strings.Count(s, "\n")
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

	// In TUI mode, we're always in a terminal, so default to true
	// The lipgloss renderer will handle the actual colour profile detection
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}
