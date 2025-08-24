/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package diff

import (
	"encoding/json"
	"fmt"
	"strings"
)

// toText returns a human-readable text representation of the diff results
func (r *Result) toText() string {
	var output strings.Builder

	// Header
	output.WriteString(fmt.Sprintf("Stack: %s (Environment: %s)\n", r.StackName, r.Environment))
	output.WriteString(strings.Repeat("=", 50) + "\n\n")

	// Handle new stack case
	if !r.StackExists {
		output.WriteString("Status: NEW STACK\n")
		output.WriteString("This stack does not exist in AWS and will be created.\n\n")
		r.formatNewStackText(&output)
		return output.String()
	}

	// Handle existing stack
	if !r.HasChanges() {
		output.WriteString("Status: NO CHANGES\n")
		output.WriteString("The deployed stack matches your local configuration.\n")
		return output.String()
	}

	output.WriteString("Status: CHANGES DETECTED\n\n")

	// Template changes
	if r.TemplateChange != nil && (!r.Options.ParametersOnly && !r.Options.TagsOnly) {
		r.formatTemplateChangesText(&output)
	}

	// Parameter changes
	if len(r.ParameterDiffs) > 0 && (!r.Options.TemplateOnly && !r.Options.TagsOnly) {
		r.formatParameterChangesText(&output)
	}

	// Tag changes
	if len(r.TagDiffs) > 0 && (!r.Options.TemplateOnly && !r.Options.ParametersOnly) {
		r.formatTagChangesText(&output)
	}

	// Changeset information
	if r.ChangeSet != nil {
		r.formatChangeSetText(&output)
	}

	return output.String()
}

// formatNewStackText formats output for a new stack
func (r *Result) formatNewStackText(output *strings.Builder) {
	if len(r.ParameterDiffs) > 0 {
		output.WriteString("Parameters to be set:\n")
		for _, diff := range r.ParameterDiffs {
			fmt.Fprintf(output, "  + %s: %s\n", diff.Key, diff.ProposedValue)
		}
		output.WriteString("\n")
	}

	if len(r.TagDiffs) > 0 {
		output.WriteString("Tags to be set:\n")
		for _, diff := range r.TagDiffs {
			fmt.Fprintf(output, "  + %s: %s\n", diff.Key, diff.ProposedValue)
		}
		output.WriteString("\n")
	}
}

// formatTemplateChangesText formats template change information
func (r *Result) formatTemplateChangesText(output *strings.Builder) {
	output.WriteString("Template Changes:\n")
	output.WriteString("-----------------\n")

	if r.TemplateChange.HasChanges {
		output.WriteString("✓ Template has been modified\n")

		if r.TemplateChange.ResourceCount.Added > 0 ||
			r.TemplateChange.ResourceCount.Modified > 0 ||
			r.TemplateChange.ResourceCount.Removed > 0 {
			output.WriteString("Resource changes:\n")
			if r.TemplateChange.ResourceCount.Added > 0 {
				fmt.Fprintf(output, "  + %d resources to be added\n", r.TemplateChange.ResourceCount.Added)
			}
			if r.TemplateChange.ResourceCount.Modified > 0 {
				fmt.Fprintf(output, "  ~ %d resources to be modified\n", r.TemplateChange.ResourceCount.Modified)
			}
			if r.TemplateChange.ResourceCount.Removed > 0 {
				fmt.Fprintf(output, "  - %d resources to be removed\n", r.TemplateChange.ResourceCount.Removed)
			}
		}

		if r.TemplateChange.Diff != "" {
			output.WriteString("\nTemplate diff:\n")
			output.WriteString(r.TemplateChange.Diff)
		}
	} else {
		output.WriteString("✗ No template changes\n")
	}
	output.WriteString("\n")
}

// formatParameterChangesText formats parameter change information
func (r *Result) formatParameterChangesText(output *strings.Builder) {
	output.WriteString("Parameter Changes:\n")
	output.WriteString("------------------\n")

	for _, diff := range r.ParameterDiffs {
		switch diff.ChangeType {
		case ChangeTypeAdd:
			fmt.Fprintf(output, "  + %s: %s\n", diff.Key, diff.ProposedValue)
		case ChangeTypeModify:
			fmt.Fprintf(output, "  ~ %s: %s → %s\n", diff.Key, diff.CurrentValue, diff.ProposedValue)
		case ChangeTypeRemove:
			fmt.Fprintf(output, "  - %s: %s\n", diff.Key, diff.CurrentValue)
		}
	}
	output.WriteString("\n")
}

// formatTagChangesText formats tag change information
func (r *Result) formatTagChangesText(output *strings.Builder) {
	output.WriteString("Tag Changes:\n")
	output.WriteString("------------\n")

	for _, diff := range r.TagDiffs {
		switch diff.ChangeType {
		case ChangeTypeAdd:
			fmt.Fprintf(output, "  + %s: %s\n", diff.Key, diff.ProposedValue)
		case ChangeTypeModify:
			fmt.Fprintf(output, "  ~ %s: %s → %s\n", diff.Key, diff.CurrentValue, diff.ProposedValue)
		case ChangeTypeRemove:
			fmt.Fprintf(output, "  - %s: %s\n", diff.Key, diff.CurrentValue)
		}
	}
	output.WriteString("\n")
}

// formatChangeSetText formats AWS changeset information
func (r *Result) formatChangeSetText(output *strings.Builder) {
	output.WriteString("AWS CloudFormation Preview:\n")
	output.WriteString("---------------------------\n")
	fmt.Fprintf(output, "ChangeSet ID: %s\n", r.ChangeSet.ChangeSetID)
	fmt.Fprintf(output, "Status: %s\n", r.ChangeSet.Status)

	if len(r.ChangeSet.Changes) > 0 {
		output.WriteString("\nResource Changes:\n")
		for _, change := range r.ChangeSet.Changes {
			symbol := r.getChangeSymbol(change.Action)
			fmt.Fprintf(output, "  %s %s (%s)", symbol, change.LogicalID, change.ResourceType)

			if change.PhysicalID != "" {
				fmt.Fprintf(output, " [%s]", change.PhysicalID)
			}

			if change.Replacement != "" && change.Replacement != "False" {
				fmt.Fprintf(output, " - Replacement: %s", change.Replacement)
			}

			output.WriteString("\n")

			// Add details if available
			for _, detail := range change.Details {
				fmt.Fprintf(output, "    %s\n", detail)
			}
		}
	}
	output.WriteString("\n")
}

// getChangeSymbol returns the appropriate symbol for a changeset action
func (r *Result) getChangeSymbol(action string) string {
	switch action {
	case "Add":
		return "+"
	case "Modify":
		return "~"
	case "Remove":
		return "-"
	default:
		return "?"
	}
}

// toJSON returns a JSON representation of the diff results
func (r *Result) toJSON() string {
	// Create a simplified structure for JSON output
	jsonResult := map[string]interface{}{
		"stackName":   r.StackName,
		"environment": r.Environment,
		"stackExists": r.StackExists,
		"hasChanges":  r.HasChanges(),
		"options":     r.Options,
	}

	// Add template changes if present
	if r.TemplateChange != nil {
		jsonResult["templateChanges"] = map[string]interface{}{
			"hasChanges":    r.TemplateChange.HasChanges,
			"currentHash":   r.TemplateChange.CurrentHash,
			"proposedHash":  r.TemplateChange.ProposedHash,
			"resourceCount": r.TemplateChange.ResourceCount,
		}
	}

	// Add parameter diffs if present
	if len(r.ParameterDiffs) > 0 {
		paramDiffs := make([]map[string]interface{}, len(r.ParameterDiffs))
		for i, diff := range r.ParameterDiffs {
			paramDiffs[i] = map[string]interface{}{
				"key":           diff.Key,
				"currentValue":  diff.CurrentValue,
				"proposedValue": diff.ProposedValue,
				"changeType":    string(diff.ChangeType),
			}
		}
		jsonResult["parameterDiffs"] = paramDiffs
	}

	// Add tag diffs if present
	if len(r.TagDiffs) > 0 {
		tagDiffs := make([]map[string]interface{}, len(r.TagDiffs))
		for i, diff := range r.TagDiffs {
			tagDiffs[i] = map[string]interface{}{
				"key":           diff.Key,
				"currentValue":  diff.CurrentValue,
				"proposedValue": diff.ProposedValue,
				"changeType":    string(diff.ChangeType),
			}
		}
		jsonResult["tagDiffs"] = tagDiffs
	}

	// Add changeset information if present
	if r.ChangeSet != nil {
		changeSetData := map[string]interface{}{
			"changeSetId": r.ChangeSet.ChangeSetID,
			"status":      r.ChangeSet.Status,
		}

		if len(r.ChangeSet.Changes) > 0 {
			changes := make([]map[string]interface{}, len(r.ChangeSet.Changes))
			for i, change := range r.ChangeSet.Changes {
				changes[i] = map[string]interface{}{
					"action":       change.Action,
					"resourceType": change.ResourceType,
					"logicalId":    change.LogicalID,
					"physicalId":   change.PhysicalID,
					"replacement":  change.Replacement,
					"details":      change.Details,
				}
			}
			changeSetData["changes"] = changes
		}

		jsonResult["changeSet"] = changeSetData
	}

	// Marshal to JSON with proper formatting
	jsonBytes, err := json.MarshalIndent(jsonResult, "", "  ")
	if err != nil {
		return fmt.Sprintf(`{"error": "failed to marshal JSON: %s"}`, err.Error())
	}

	return string(jsonBytes)
}
