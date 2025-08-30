/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package describe

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// FormatStackDescription formats stack information for display
func FormatStackDescription(desc *StackDescription) string {
	var output strings.Builder

	// Stack Summary section
	output.WriteString(fmt.Sprintf("Stack: %s\n", desc.Name))
	output.WriteString(fmt.Sprintf("Status: %s\n", desc.Status))
	if !desc.CreatedTime.IsZero() {
		output.WriteString(fmt.Sprintf("Created: %s\n", formatTime(desc.CreatedTime)))
	}

	if desc.UpdatedTime != nil {
		output.WriteString(fmt.Sprintf("Updated: %s\n", formatTime(*desc.UpdatedTime)))
	}

	if desc.StackID != "" && desc.StackID != desc.Name {
		output.WriteString(fmt.Sprintf("Stack ID: %s\n", desc.StackID))
	}

	if desc.Description != "" {
		output.WriteString(fmt.Sprintf("Description: %s\n", desc.Description))
	}

	// Parameters section
	if len(desc.Parameters) > 0 {
		output.WriteString("\nParameters:\n")
		writeKeyValueMap(&output, desc.Parameters)
	}

	// Outputs section
	if len(desc.Outputs) > 0 {
		output.WriteString("\nOutputs:\n")
		writeKeyValueMap(&output, desc.Outputs)
	}

	// Tags section
	if len(desc.Tags) > 0 {
		output.WriteString("\nTags:\n")
		writeKeyValueMap(&output, desc.Tags)
	}

	return output.String()
}

// formatTime formats time in a human-readable format
func formatTime(t time.Time) string {
	// Use ISO 8601 format as specified in AGENTS.md for British standards
	return t.Format("2006-01-02 15:04:05 MST")
}

// writeKeyValueMap writes a sorted map as key-value pairs with indentation
func writeKeyValueMap(output *strings.Builder, m map[string]string) {
	if len(m) == 0 {
		return
	}

	// Sort keys for consistent output
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	// Write each key-value pair with proper indentation
	for _, key := range keys {
		value := m[key]
		fmt.Fprintf(output, "  %s: %s\n", key, value)
	}
}
