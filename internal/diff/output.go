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
	useColour := shouldUseColour()
	styles := newOutputStyles(useColour)

	// Header
	header := fmt.Sprintf("Stack: %s (Context: %s)", r.StackName, r.Context)
	output.WriteString(styles.header.Render(header))
	output.WriteString("\n")
	output.WriteString(styles.separator.Render(strings.Repeat("═", 60)))
	output.WriteString("\n\n")

	// Handle new stack case
	if !r.StackExists {
		statusLine := styles.statusNew.Render("Status: NEW STACK")
		output.WriteString(statusLine)
		output.WriteString("\n")
		output.WriteString("This stack does not exist in AWS and will be created.\n\n")
		r.formatNewStackText(&output, styles)
		return output.String()
	}

	// Handle existing stack
	if !r.HasChanges() {
		statusLine := styles.statusNoChange.Render("Status: NO CHANGES")
		output.WriteString(statusLine)
		output.WriteString("\n")
		output.WriteString("The deployed stack matches your local configuration.\n")
		return output.String()
	}

	statusLine := styles.statusChanges.Render("Status: CHANGES DETECTED")
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
func (r *Result) formatNewStackText(output *strings.Builder, styles *OutputStyles) {
	if len(r.ParameterDiffs) > 0 {
		output.WriteString(styles.sectionHeader.Render("Parameters to be set:"))
		output.WriteString("\n")
		for _, diff := range r.ParameterDiffs {
			symbol := styles.added.Render("+")
			key := styles.key.Render(diff.Key)
			value := styles.value.Render(diff.ProposedValue)
			fmt.Fprintf(output, "  %s %s: %s\n", symbol, key, value)
		}
		output.WriteString("\n")
	}

	if len(r.TagDiffs) > 0 {
		output.WriteString(styles.sectionHeader.Render("Tags to be set:"))
		output.WriteString("\n")
		for _, diff := range r.TagDiffs {
			symbol := styles.added.Render("+")
			key := styles.key.Render(diff.Key)
			value := styles.value.Render(diff.ProposedValue)
			fmt.Fprintf(output, "  %s %s: %s\n", symbol, key, value)
		}
		output.WriteString("\n")
	}
}

// formatTemplateChangesText formats template change information
func (r *Result) formatTemplateChangesText(output *strings.Builder, styles *OutputStyles) {
	output.WriteString(styles.sectionHeader.Render("Template Changes:"))
	output.WriteString("\n")
	output.WriteString(styles.separator.Render(strings.Repeat("─", 17)))
	output.WriteString("\n")

	if r.TemplateChange.HasChanges {
		checkmark := styles.modified.Render("✓")
		fmt.Fprintf(output, "%s Template has been modified\n", checkmark)

		if r.TemplateChange.ResourceCount.Added > 0 ||
			r.TemplateChange.ResourceCount.Modified > 0 ||
			r.TemplateChange.ResourceCount.Removed > 0 {
			output.WriteString("\nResource changes:\n")
			if r.TemplateChange.ResourceCount.Added > 0 {
				symbol := styles.added.Render("+")
				count := styles.value.Render(fmt.Sprintf("%d", r.TemplateChange.ResourceCount.Added))
				fmt.Fprintf(output, "  %s %s resources to be added\n", symbol, count)
			}
			if r.TemplateChange.ResourceCount.Modified > 0 {
				symbol := styles.modified.Render("~")
				count := styles.value.Render(fmt.Sprintf("%d", r.TemplateChange.ResourceCount.Modified))
				fmt.Fprintf(output, "  %s %s resources to be modified\n", symbol, count)
			}
			if r.TemplateChange.ResourceCount.Removed > 0 {
				symbol := styles.removed.Render("-")
				count := styles.value.Render(fmt.Sprintf("%d", r.TemplateChange.ResourceCount.Removed))
				fmt.Fprintf(output, "  %s %s resources to be removed\n", symbol, count)
			}
		}

		if r.TemplateChange.Diff != "" {
			output.WriteString("\n")
			output.WriteString(styles.subSection.Render("Template diff:"))
			output.WriteString("\n")
			output.WriteString(r.TemplateChange.Diff)
		}
	} else {
		crossmark := styles.statusNoChange.Render("✗")
		fmt.Fprintf(output, "%s No template changes\n", crossmark)
	}
	output.WriteString("\n")
}

// formatParameterChangesText formats parameter change information
func (r *Result) formatParameterChangesText(output *strings.Builder, styles *OutputStyles) {
	output.WriteString(styles.sectionHeader.Render("Parameter Changes:"))
	output.WriteString("\n")
	output.WriteString(styles.separator.Render(strings.Repeat("─", 18)))
	output.WriteString("\n")

	for _, diff := range r.ParameterDiffs {
		symbol := styles.getChangeSymbol(diff.ChangeType)
		key := styles.key.Render(diff.Key)

		switch diff.ChangeType {
		case ChangeTypeAdd:
			value := styles.value.Render(diff.ProposedValue)
			fmt.Fprintf(output, "  %s %s: %s\n", symbol, key, value)
		case ChangeTypeModify:
			currentVal := styles.value.Render(diff.CurrentValue)
			proposedVal := styles.value.Render(diff.ProposedValue)
			arrow := styles.arrow.Render("→")
			fmt.Fprintf(output, "  %s %s: %s %s %s\n", symbol, key, currentVal, arrow, proposedVal)
		case ChangeTypeRemove:
			value := styles.value.Render(diff.CurrentValue)
			fmt.Fprintf(output, "  %s %s: %s\n", symbol, key, value)
		}
	}
	output.WriteString("\n")
}

// formatTagChangesText formats tag change information
func (r *Result) formatTagChangesText(output *strings.Builder, styles *OutputStyles) {
	output.WriteString(styles.sectionHeader.Render("Tag Changes:"))
	output.WriteString("\n")
	output.WriteString(styles.separator.Render(strings.Repeat("─", 12)))
	output.WriteString("\n")

	for _, diff := range r.TagDiffs {
		symbol := styles.getChangeSymbol(diff.ChangeType)
		key := styles.key.Render(diff.Key)

		switch diff.ChangeType {
		case ChangeTypeAdd:
			value := styles.value.Render(diff.ProposedValue)
			fmt.Fprintf(output, "  %s %s: %s\n", symbol, key, value)
		case ChangeTypeModify:
			currentVal := styles.value.Render(diff.CurrentValue)
			proposedVal := styles.value.Render(diff.ProposedValue)
			arrow := styles.arrow.Render("→")
			fmt.Fprintf(output, "  %s %s: %s %s %s\n", symbol, key, currentVal, arrow, proposedVal)
		case ChangeTypeRemove:
			value := styles.value.Render(diff.CurrentValue)
			fmt.Fprintf(output, "  %s %s: %s\n", symbol, key, value)
		}
	}
	output.WriteString("\n")
}

// formatChangeSetText formats AWS changeset information
func (r *Result) formatChangeSetText(output *strings.Builder, styles *OutputStyles) {
	output.WriteString(styles.sectionHeader.Render("AWS CloudFormation Preview:"))
	output.WriteString("\n")
	output.WriteString(styles.separator.Render(strings.Repeat("─", 27)))
	output.WriteString("\n")

	if len(r.ChangeSet.Changes) > 0 {
		output.WriteString("\n")
		output.WriteString(styles.subSection.Render("Resource Changes:"))
		output.WriteString("\n")
		for _, change := range r.ChangeSet.Changes {
			symbol := styles.getChangeSetSymbol(change.Action)
			logicalID := styles.key.Render(change.LogicalID)
			resourceType := styles.value.Render(change.ResourceType)
			fmt.Fprintf(output, "  %s %s (%s)", symbol, logicalID, resourceType)

			if change.PhysicalID != "" {
				physicalID := styles.subSection.Render(fmt.Sprintf("[%s]", change.PhysicalID))
				fmt.Fprintf(output, " %s", physicalID)
			}

			if change.Replacement != "" && change.Replacement != "False" {
				replacement := styles.riskHigh.Render(fmt.Sprintf("⚠ Replacement: %s", change.Replacement))
				fmt.Fprintf(output, " - %s", replacement)
			}

			output.WriteString("\n")

			// Add details if available
			for _, detail := range change.Details {
				detailText := styles.subSection.Render(fmt.Sprintf("    %s", detail))
				output.WriteString(detailText)
				output.WriteString("\n")
			}
		}
	}
	output.WriteString("\n")
}
