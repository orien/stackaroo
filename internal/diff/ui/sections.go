/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package ui

import (
	"fmt"
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
			Name:       "CloudFormation Plan",
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
	useColour := diff.ShouldUseColour()
	styles := diff.NewStyles(useColour)

	if tc.HasChanges && tc.Diff != "" {
		s.WriteString(diff.ColorizeUnifiedDiff(tc.Diff, styles))
	} else {
		s.WriteString(styles.Subtle.Render("✗ No template changes"))
	}

	return s.String()
}

// formatParameterSection formats parameter changes
func formatParameterSection(params []diff.ParameterDiff, isNewStack bool) string {
	var s strings.Builder
	useColour := diff.ShouldUseColour()
	styles := diff.NewStyles(useColour)

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
	useColour := diff.ShouldUseColour()
	styles := diff.NewStyles(useColour)

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
	useColour := diff.ShouldUseColour()
	styles := diff.NewStyles(useColour)

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
func getChangeSymbol(changeType diff.ChangeType, styles *diff.Styles) string {
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
func getChangeSetSymbol(action string, styles *diff.Styles) string {
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
