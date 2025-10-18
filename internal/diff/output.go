/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package diff

import (
	"fmt"
	"strings"
)

// toText returns a human-readable text representation of the diff results
func (r *Result) toText() string {
	var output strings.Builder

	// Detect if we should use colour
	useColour := ShouldUseColour()
	styles := NewStyles(useColour)

	// Header
	header := fmt.Sprintf("%s - %s", r.StackName, r.Context)
	output.WriteString("\n")
	output.WriteString(styles.HeaderTitle.Render(header))
	output.WriteString("\n\n")

	// Handle new stack case
	if !r.StackExists {
		statusLine := styles.StatusNew.Render("NEW STACK")
		output.WriteString(statusLine)
		output.WriteString("\n")
		output.WriteString("This stack does not exist in AWS and will be created.\n\n")
		r.formatNewStackText(&output, styles)
		return output.String()
	}

	// Handle existing stack
	if !r.HasChanges() {
		statusLine := styles.StatusNoChange.Render("NO CHANGES")
		output.WriteString(statusLine)
		output.WriteString("\n")
		output.WriteString("The deployed stack matches your local configuration.\n")
		return output.String()
	}

	statusLine := styles.StatusChanges.Render("CHANGES DETECTED")
	output.WriteString(statusLine)
	output.WriteString("\n\n")

	// Template changes
	if r.TemplateChange != nil && (!r.Options.ParametersOnly && !r.Options.TagsOnly) {
		r.formatTemplateChangesText(&output, styles)
	}

	// Parameter changes
	if len(r.ParameterDiffs) > 0 && (!r.Options.TemplateOnly && !r.Options.TagsOnly) {
		r.formatParameterChangesText(&output, styles)
	}

	// Tag changes
	if len(r.TagDiffs) > 0 && (!r.Options.TemplateOnly && !r.Options.ParametersOnly) {
		r.formatTagChangesText(&output, styles)
	}

	// Changeset information
	if r.ChangeSet != nil {
		r.formatChangeSetText(&output, styles)
	}

	return output.String()
}

// formatNewStackText formats output for a new stack
func (r *Result) formatNewStackText(output *strings.Builder, styles *Styles) {
	if len(r.ParameterDiffs) > 0 {
		output.WriteString(styles.SectionHeader.Render("PARAMETERS"))
		output.WriteString("\n\n")
		for _, diff := range r.ParameterDiffs {
			symbol := styles.Added.Render("+")
			key := styles.Key.Render(diff.Key)
			value := styles.Value.Render(diff.ProposedValue)
			fmt.Fprintf(output, "  %s %s: %s\n", symbol, key, value)
		}
		output.WriteString("\n")
	}

	if len(r.TagDiffs) > 0 {
		output.WriteString(styles.SectionHeader.Render("TAGS"))
		output.WriteString("\n\n")
		for _, diff := range r.TagDiffs {
			symbol := styles.Added.Render("+")
			key := styles.Key.Render(diff.Key)
			value := styles.Value.Render(diff.ProposedValue)
			fmt.Fprintf(output, "  %s %s: %s\n", symbol, key, value)
		}
		output.WriteString("\n")
	}
}

// formatTemplateChangesText formats template change information
func (r *Result) formatTemplateChangesText(output *strings.Builder, styles *Styles) {
	output.WriteString(styles.SectionHeader.Render("TEMPLATE"))
	output.WriteString("\n\n")

	if r.TemplateChange.HasChanges && r.TemplateChange.Diff != "" {
		output.WriteString(ColorizeUnifiedDiff(r.TemplateChange.Diff, styles))
	} else {
		crossmark := styles.StatusNoChange.Render("✗")
		fmt.Fprintf(output, "%s No template changes\n", crossmark)
	}
	output.WriteString("\n")
}

// formatParameterChangesText formats parameter change information
func (r *Result) formatParameterChangesText(output *strings.Builder, styles *Styles) {
	output.WriteString(styles.SectionHeader.Render("PARAMETERS"))
	output.WriteString("\n\n")

	for _, diff := range r.ParameterDiffs {
		symbol := styles.GetChangeSymbol(diff.ChangeType)
		key := styles.Key.Render(diff.Key)

		switch diff.ChangeType {
		case ChangeTypeAdd:
			value := styles.Value.Render(diff.ProposedValue)
			fmt.Fprintf(output, "  %s %s: %s\n", symbol, key, value)
		case ChangeTypeModify:
			currentVal := styles.Value.Render(diff.CurrentValue)
			proposedVal := styles.Value.Render(diff.ProposedValue)
			arrow := styles.Arrow.Render("→")
			fmt.Fprintf(output, "  %s %s: %s %s %s\n", symbol, key, currentVal, arrow, proposedVal)
		case ChangeTypeRemove:
			value := styles.Value.Render(diff.CurrentValue)
			fmt.Fprintf(output, "  %s %s: %s\n", symbol, key, value)
		}
	}
	output.WriteString("\n")
}

// formatTagChangesText formats tag change information
func (r *Result) formatTagChangesText(output *strings.Builder, styles *Styles) {
	output.WriteString(styles.SectionHeader.Render("TAGS"))
	output.WriteString("\n\n")

	for _, diff := range r.TagDiffs {
		symbol := styles.GetChangeSymbol(diff.ChangeType)
		key := styles.Key.Render(diff.Key)

		switch diff.ChangeType {
		case ChangeTypeAdd:
			fmt.Fprintf(output, "  %s %s: %s\n", symbol, key, diff.ProposedValue)
		case ChangeTypeModify:
			fmt.Fprintf(output, "  %s %s: %s → %s\n", symbol, key, diff.CurrentValue, diff.ProposedValue)
		case ChangeTypeRemove:
			fmt.Fprintf(output, "  %s %s: %s\n", symbol, key, diff.CurrentValue)
		}
	}
	output.WriteString("\n")
}

// formatChangeSetText formats AWS changeset information
func (r *Result) formatChangeSetText(output *strings.Builder, styles *Styles) {
	output.WriteString(styles.SectionHeader.Render("PLAN"))
	output.WriteString("\n\n")

	if len(r.ChangeSet.Changes) > 0 {
		output.WriteString(styles.SubSection.Render("Resource changes:"))
		output.WriteString("\n")
		for _, change := range r.ChangeSet.Changes {
			symbol := styles.GetChangeSetSymbol(change.Action)
			logicalID := styles.Key.Render(change.LogicalID)
			resourceType := styles.Value.Render(change.ResourceType)
			fmt.Fprintf(output, "  %s %s (%s)", symbol, logicalID, resourceType)

			if change.PhysicalID != "" {
				physicalID := styles.SubSection.Render(fmt.Sprintf("[%s]", change.PhysicalID))
				fmt.Fprintf(output, " %s", physicalID)
			}

			switch change.Replacement {
			case "True":
				output.WriteString(styles.RiskHigh.Render(" REPLACE"))
			case "Conditional":
				output.WriteString(styles.RiskHigh.Render(" REPLACE (conditional)"))
			}
			output.WriteString("\n")

			// Add details if available
			for _, detail := range change.Details {
				detailText := styles.SubSection.Render(fmt.Sprintf("    %s", detail))
				output.WriteString(detailText)
				output.WriteString("\n")
			}
		}
	}
	output.WriteString("\n")
}

// ColorizeUnifiedDiff applies color formatting to unified diff output
func ColorizeUnifiedDiff(diff string, styles *Styles) string {
	if !styles.UseColour || diff == "" {
		return diff
	}

	lines := strings.Split(diff, "\n")
	var colorized strings.Builder

	for i, line := range lines {
		if len(line) == 0 {
			colorized.WriteString("\n")
			continue
		}

		switch line[0] {
		case '@':
			// Hunk header - use cyan/key style
			colorized.WriteString(styles.Key.Render(line))
		case '+':
			// Addition - use green
			colorized.WriteString(styles.Added.Render(line))
		case '-':
			// Deletion - use red
			colorized.WriteString(styles.Removed.Render(line))
		default:
			// Unknown - leave as is
			colorized.WriteString(line)
		}

		// Add newline except for the last line if it was empty
		if i < len(lines)-1 {
			colorized.WriteString("\n")
		}
	}

	return colorized.String()
}
