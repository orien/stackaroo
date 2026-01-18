/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package diff

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/x/term"
	"github.com/orien/stackaroo/internal/aws"
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
		statusLine := styles.StatusNew.Render("New Stack")
		output.WriteString(statusLine)
		output.WriteString("\n")
		output.WriteString("This stack does not exist in AWS and will be created.\n\n")
		r.formatNewStackText(&output, styles)
		return output.String()
	}

	// Handle existing stack
	if !r.HasChanges() {
		statusLine := styles.StatusNoChange.Render("No Changes")
		output.WriteString(statusLine)
		output.WriteString("\n")
		output.WriteString("The deployed stack matches your local configuration.\n")
		return output.String()
	}

	statusLine := styles.StatusChanges.Render("Changes Detected")
	output.WriteString(statusLine)
	output.WriteString("\n")
	output.WriteString("Your local configuration differs from the deployed stack.\n\n")

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

	// Changeset information or error
	if r.ChangeSet != nil {
		r.formatChangeSetText(&output, styles)
	} else if r.ChangeSetError != nil {
		// Check if this is a "no infrastructure changes" scenario
		var noChangesErr aws.NoChangesError
		if errors.As(r.ChangeSetError, &noChangesErr) {
			r.formatNoInfrastructureChangesText(&output, styles)
		} else {
			r.formatChangeSetErrorText(&output, styles)
		}
	}

	return output.String()
}

// formatNewStackText formats output for a new stack
func (r *Result) formatNewStackText(output *strings.Builder, styles *Styles) {
	if len(r.ParameterDiffs) > 0 {
		output.WriteString(styles.SectionHeader.Render("PARAMETERS"))
		output.WriteString("\n\n")
		for _, diff := range r.ParameterDiffs {
			symbol := styles.AddedText.Render("+")
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
			symbol := styles.AddedText.Render("+")
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

		var key string
		switch diff.ChangeType {
		case ChangeTypeAdd:
			key = styles.AddedText.Render(diff.Key)
			value := styles.Value.Render(diff.ProposedValue)
			fmt.Fprintf(output, "  %s %s: %s\n", symbol, key, value)
		case ChangeTypeModify:
			key = styles.ModifiedText.Render(diff.Key)
			currentVal := styles.Value.Render(diff.CurrentValue)
			proposedVal := styles.Value.Render(diff.ProposedValue)
			arrow := styles.Arrow.Render("→")
			fmt.Fprintf(output, "  %s %s: %s %s %s\n", symbol, key, currentVal, arrow, proposedVal)
		case ChangeTypeRemove:
			key = styles.RemovedText.Render(diff.Key)
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

		var key string
		switch diff.ChangeType {
		case ChangeTypeAdd:
			key = styles.AddedText.Render(diff.Key)
			fmt.Fprintf(output, "  %s %s: %s\n", symbol, key, diff.ProposedValue)
		case ChangeTypeModify:
			key = styles.ModifiedText.Render(diff.Key)
			fmt.Fprintf(output, "  %s %s: %s → %s\n", symbol, key, diff.CurrentValue, diff.ProposedValue)
		case ChangeTypeRemove:
			key = styles.RemovedText.Render(diff.Key)
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
		for _, change := range r.ChangeSet.Changes {
			symbol := styles.GetChangeSetSymbol(change.Action)

			var logicalID string
			switch change.Action {
			case "Add":
				logicalID = styles.AddedText.Render(change.LogicalID)
			case "Modify":
				logicalID = styles.ModifiedText.Render(change.LogicalID)
			case "Remove":
				logicalID = styles.RemovedText.Render(change.LogicalID)
			default:
				logicalID = styles.Key.Render(change.LogicalID)
			}

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

// formatNoInfrastructureChangesText formats output when template changes don't affect infrastructure
func (r *Result) formatNoInfrastructureChangesText(output *strings.Builder, styles *Styles) {
	output.WriteString(styles.SectionHeader.Render("PLAN"))
	output.WriteString("\n\n")

	// Display as informational, not an error
	infoHeader := styles.StatusNoChange.Render("No Infrastructure Changes")
	fmt.Fprintf(output, "%s\n\n", infoHeader)

	// Explain what this means
	output.WriteString(styles.SubSection.Render("The template changes shown above are metadata-only and do not affect infrastructure."))
	output.WriteString("\n\n")

	output.WriteString(styles.SubSection.Render("Examples of metadata-only changes:"))
	output.WriteString("\n")
	output.WriteString("  • Template Description field\n")
	output.WriteString("  • Metadata section\n")
	output.WriteString("  • Comments or formatting\n\n")

	output.WriteString(styles.SubSection.Render("No deployment is required for these changes."))
	output.WriteString("\n\n")
}

// formatChangeSetErrorText formats changeset generation errors
func (r *Result) formatChangeSetErrorText(output *strings.Builder, styles *Styles) {
	output.WriteString(styles.SectionHeader.Render("PLAN"))
	output.WriteString("\n\n")

	// Display the error prominently
	errorHeader := styles.RiskHigh.Render("Changeset Generation Failed")
	fmt.Fprintf(output, "%s\n\n", errorHeader)

	// Explain what happened
	output.WriteString(styles.SubSection.Render("CloudFormation was unable to generate a detailed change plan:"))
	output.WriteString("\n")

	errorMsg := styles.Value.Render(r.ChangeSetError.Error())
	fmt.Fprintf(output, "  %s\n\n", errorMsg)

	// Reassure the user
	output.WriteString(styles.SubSection.Render("The parameter, tag, and template changes shown above are still accurate."))
	output.WriteString("\n")
	output.WriteString(styles.SubSection.Render("However, resource-level change details are not available."))
	output.WriteString("\n\n")

	// Provide guidance
	output.WriteString(styles.SubSection.Render("Common causes:"))
	output.WriteString("\n")
	output.WriteString("  • Invalid parameter name (parameter not defined in template)\n")
	output.WriteString("  • Invalid parameter value (doesn't meet template constraints)\n")
	output.WriteString("  • Template validation errors\n")
	output.WriteString("  • Missing required parameters\n\n")

	output.WriteString(styles.SubSection.Render("Review the error message and your configuration before proceeding."))
	output.WriteString("\n\n")
}

// ColorizeUnifiedDiff applies color formatting to unified diff output
func ColorizeUnifiedDiff(diff string, styles *Styles) string {
	if !styles.UseColour || diff == "" {
		return diff
	}

	lines := strings.Split(diff, "\n")

	// Drop final empty line from trailing newline artifact
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	// Get terminal width, default to 80 if not available
	termWidth := 80
	if width, _, err := term.GetSize(os.Stdout.Fd()); err == nil && width > 0 {
		termWidth = width
	}

	// Use terminal width for padding (indent will be included in colored output)
	maxLen := termWidth
	for _, line := range lines {
		if len(line) > maxLen {
			maxLen = len(line)
		}
	}

	var colorized strings.Builder

	for _, line := range lines {
		if len(line) == 0 {
			colorized.WriteString("\n")
			continue
		}

		// Pad line to max length for uniform background (accounting for 2-char indent)
		paddedLine := "  " + line
		if len(paddedLine) < maxLen {
			paddedLine = paddedLine + strings.Repeat(" ", maxLen-len(paddedLine))
		}

		switch line[0] {
		case '@':
			// Hunk header
			colorized.WriteString(styles.DiffHunk.Render(paddedLine))
		case '+':
			// Addition
			colorized.WriteString(styles.Added.Render(paddedLine))
		case '-':
			// Deletion
			colorized.WriteString(styles.Removed.Render(paddedLine))
		default:
			// Context line
			colorized.WriteString(styles.DiffContext.Render(paddedLine))
		}

		// Add newline after each line
		colorized.WriteString("\n")
	}

	return colorized.String()
}
