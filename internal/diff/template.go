/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package diff

import (
	"context"
	"crypto/sha256"
	"fmt"
	"reflect"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	contextLines = 3 // Number of context lines to show around changes
)

// YAMLTemplateComparator implements TemplateComparator for YAML CloudFormation templates
type YAMLTemplateComparator struct{}

// NewYAMLTemplateComparator creates a new YAML template comparator
func NewYAMLTemplateComparator() TemplateComparator {
	return &YAMLTemplateComparator{}
}

// Compare compares two CloudFormation templates and returns the differences
func (c *YAMLTemplateComparator) Compare(ctx context.Context, currentTemplate, proposedTemplate string) (*TemplateChange, error) {
	// Calculate hashes for quick comparison
	currentHash := c.calculateHash(currentTemplate)
	proposedHash := c.calculateHash(proposedTemplate)

	change := &TemplateChange{
		CurrentHash:  currentHash,
		ProposedHash: proposedHash,
		HasChanges:   currentHash != proposedHash,
	}

	// If hashes are the same, no changes
	if !change.HasChanges {
		return change, nil
	}

	// Parse both templates
	var currentData, proposedData map[string]interface{}

	if err := yaml.Unmarshal([]byte(currentTemplate), &currentData); err != nil {
		return nil, fmt.Errorf("failed to parse current template: %w", err)
	}

	if err := yaml.Unmarshal([]byte(proposedTemplate), &proposedData); err != nil {
		return nil, fmt.Errorf("failed to parse proposed template: %w", err)
	}

	// Compare resources to get counts
	resourceCounts, err := c.compareResources(currentData, proposedData)
	if err != nil {
		return nil, fmt.Errorf("failed to compare resources: %w", err)
	}

	change.ResourceCount = resourceCounts

	// Generate diff text
	diff, err := c.generateDiff(currentTemplate, proposedTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to generate diff: %w", err)
	}

	change.Diff = diff

	return change, nil
}

// calculateHash generates a SHA256 hash of the template content
func (c *YAMLTemplateComparator) calculateHash(template string) string {
	// Normalise whitespace and line endings for consistent hashing
	normalised := strings.TrimSpace(strings.ReplaceAll(template, "\r\n", "\n"))
	hash := sha256.Sum256([]byte(normalised))
	return fmt.Sprintf("%x", hash)[:12] // Use first 12 characters for readability
}

// compareResources compares the Resources sections of two templates
func (c *YAMLTemplateComparator) compareResources(currentData, proposedData map[string]interface{}) (struct{ Added, Modified, Removed int }, error) {
	counts := struct{ Added, Modified, Removed int }{}

	// Extract Resources sections
	currentResources := c.getResourcesSection(currentData)
	proposedResources := c.getResourcesSection(proposedData)

	// Track all resource names
	allResourceNames := make(map[string]bool)
	for name := range currentResources {
		allResourceNames[name] = true
	}
	for name := range proposedResources {
		allResourceNames[name] = true
	}

	// Compare each resource
	for resourceName := range allResourceNames {
		currentResource, currentExists := currentResources[resourceName]
		proposedResource, proposedExists := proposedResources[resourceName]

		if !currentExists && proposedExists {
			counts.Added++
		} else if currentExists && !proposedExists {
			counts.Removed++
		} else if currentExists && proposedExists {
			// Check if resource has been modified
			if !reflect.DeepEqual(currentResource, proposedResource) {
				counts.Modified++
			}
		}
	}

	return counts, nil
}

// getResourcesSection extracts the Resources section from a template
func (c *YAMLTemplateComparator) getResourcesSection(templateData map[string]interface{}) map[string]interface{} {
	if resources, ok := templateData["Resources"]; ok {
		if resourcesMap, ok := resources.(map[string]interface{}); ok {
			return resourcesMap
		}
	}
	return make(map[string]interface{})
}

// generateDiff creates a human-readable diff of the templates with line-by-line comparison
// Uses original template strings to preserve formatting and key order
func (c *YAMLTemplateComparator) generateDiff(currentTemplate, proposedTemplate string) (string, error) {
	var diff strings.Builder

	// Generate unified diff using original templates to preserve formatting
	unifiedDiff := c.generateUnifiedDiff(currentTemplate, proposedTemplate)
	if unifiedDiff != "" {
		diff.WriteString(unifiedDiff)
	}

	return diff.String(), nil
}

// generateUnifiedDiff creates a unified diff between two text strings
func (c *YAMLTemplateComparator) generateUnifiedDiff(current, proposed string) string {
	currentLines := strings.Split(strings.TrimSpace(current), "\n")
	proposedLines := strings.Split(strings.TrimSpace(proposed), "\n")

	// Calculate line differences using simple LCS-based algorithm
	changes := c.calculateLineDiff(currentLines, proposedLines)

	if len(changes) == 0 {
		return ""
	}

	var diff strings.Builder

	// Group changes into hunks
	hunks := c.groupChangesIntoHunks(changes, len(currentLines), len(proposedLines))

	for _, hunk := range hunks {
		// Write hunk header
		diff.WriteString(fmt.Sprintf("@@ -%d,%d +%d,%d @@\n",
			hunk.currentStart+1, hunk.currentCount,
			hunk.proposedStart+1, hunk.proposedCount))

		// Write hunk lines
		for _, line := range hunk.lines {
			diff.WriteString(line)
			diff.WriteString("\n")
		}
	}

	return diff.String()
}

// diffChange represents a single change in the diff
type diffChange struct {
	changeType   string // "context", "delete", "add"
	currentLine  int
	proposedLine int
	content      string
}

// diffHunk represents a group of related changes
type diffHunk struct {
	currentStart  int
	currentCount  int
	proposedStart int
	proposedCount int
	lines         []string
}

// calculateLineDiff computes line-by-line differences
func (c *YAMLTemplateComparator) calculateLineDiff(currentLines, proposedLines []string) []diffChange {
	var changes []diffChange

	// Use a simple diff algorithm (Myers' diff or similar would be better, but this works)
	lcs := c.longestCommonSubsequence(currentLines, proposedLines)

	currentIdx := 0
	proposedIdx := 0
	lcsIdx := 0

	for currentIdx < len(currentLines) || proposedIdx < len(proposedLines) {
		if lcsIdx < len(lcs) &&
			currentIdx < len(currentLines) &&
			proposedIdx < len(proposedLines) &&
			currentLines[currentIdx] == lcs[lcsIdx] &&
			proposedLines[proposedIdx] == lcs[lcsIdx] {
			// Common line
			changes = append(changes, diffChange{
				changeType:   "context",
				currentLine:  currentIdx,
				proposedLine: proposedIdx,
				content:      currentLines[currentIdx],
			})
			currentIdx++
			proposedIdx++
			lcsIdx++
		} else if currentIdx < len(currentLines) &&
			(lcsIdx >= len(lcs) || currentLines[currentIdx] != lcs[lcsIdx]) {
			// Deleted line
			changes = append(changes, diffChange{
				changeType:   "delete",
				currentLine:  currentIdx,
				proposedLine: -1,
				content:      currentLines[currentIdx],
			})
			currentIdx++
		} else if proposedIdx < len(proposedLines) {
			// Added line
			changes = append(changes, diffChange{
				changeType:   "add",
				currentLine:  -1,
				proposedLine: proposedIdx,
				content:      proposedLines[proposedIdx],
			})
			proposedIdx++
		}
	}

	return changes
}

// longestCommonSubsequence finds the LCS of two string slices
func (c *YAMLTemplateComparator) longestCommonSubsequence(a, b []string) []string {
	m := len(a)
	n := len(b)

	// Create DP table
	dp := make([][]int, m+1)
	for i := range dp {
		dp[i] = make([]int, n+1)
	}

	// Fill DP table
	for i := 1; i <= m; i++ {
		for j := 1; j <= n; j++ {
			if a[i-1] == b[j-1] {
				dp[i][j] = dp[i-1][j-1] + 1
			} else {
				dp[i][j] = max(dp[i-1][j], dp[i][j-1])
			}
		}
	}

	// Backtrack to build LCS
	var lcs []string
	i, j := m, n
	for i > 0 && j > 0 {
		if a[i-1] == b[j-1] {
			lcs = append([]string{a[i-1]}, lcs...)
			i--
			j--
		} else if dp[i-1][j] > dp[i][j-1] {
			i--
		} else {
			j--
		}
	}

	return lcs
}

// groupChangesIntoHunks groups related changes into diff hunks
func (c *YAMLTemplateComparator) groupChangesIntoHunks(changes []diffChange, currentTotal, proposedTotal int) []diffHunk {
	if len(changes) == 0 {
		return nil
	}

	// First, find all actual changes (non-context)
	changeIndices := make([]int, 0)
	for i, change := range changes {
		if change.changeType != "context" {
			changeIndices = append(changeIndices, i)
		}
	}

	if len(changeIndices) == 0 {
		return nil
	}

	var hunks []diffHunk

	// Group changes that are within contextLines*2 of each other
	i := 0
	for i < len(changeIndices) {
		// Start a new hunk
		startIdx := changeIndices[i]
		endIdx := startIdx

		// Find all changes that should be in this hunk
		j := i + 1
		for j < len(changeIndices) {
			nextChangeIdx := changeIndices[j]
			// If the next change is more than contextLines*2 away, start a new hunk
			if nextChangeIdx-endIdx > contextLines*2+1 {
				break
			}
			endIdx = nextChangeIdx
			j++
		}

		// Calculate hunk boundaries with context
		hunkStart := max(0, startIdx-contextLines)
		hunkEnd := min(len(changes)-1, endIdx+contextLines)

		// Build the hunk
		hunk := diffHunk{
			currentStart:  changes[hunkStart].currentLine,
			proposedStart: changes[hunkStart].proposedLine,
		}
		if hunk.currentStart < 0 {
			hunk.currentStart = 0
		}
		if hunk.proposedStart < 0 {
			hunk.proposedStart = 0
		}

		// Add all lines in the hunk range
		for k := hunkStart; k <= hunkEnd; k++ {
			change := changes[k]
			var prefix string
			switch change.changeType {
			case "context":
				prefix = " "
				hunk.currentCount++
				hunk.proposedCount++
			case "delete":
				prefix = "-"
				hunk.currentCount++
			case "add":
				prefix = "+"
				hunk.proposedCount++
			}
			hunk.lines = append(hunk.lines, prefix+change.content)
		}

		hunks = append(hunks, hunk)
		i = j
	}

	return hunks
}

// max returns the maximum of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// hasSectionChanged checks if a specific section has changed between templates
func (c *YAMLTemplateComparator) hasSectionChanged(currentData, proposedData map[string]interface{}, sectionName string) bool {
	currentSection, currentExists := currentData[sectionName]
	proposedSection, proposedExists := proposedData[sectionName]

	if currentExists != proposedExists {
		return true
	}

	if currentExists && proposedExists {
		return !reflect.DeepEqual(currentSection, proposedSection)
	}

	return false
}

// generateResourceDiff creates a detailed diff of the Resources section
func (c *YAMLTemplateComparator) generateResourceDiff(currentData, proposedData map[string]interface{}) string {
	var diff strings.Builder

	currentResources := c.getResourcesSection(currentData)
	proposedResources := c.getResourcesSection(proposedData)

	// Track all resource names
	allResourceNames := make(map[string]bool)
	for name := range currentResources {
		allResourceNames[name] = true
	}
	for name := range proposedResources {
		allResourceNames[name] = true
	}

	var resourceChanges []string
	for resourceName := range allResourceNames {
		currentResource, currentExists := currentResources[resourceName]
		proposedResource, proposedExists := proposedResources[resourceName]

		if !currentExists && proposedExists {
			resourceType := c.getResourceType(proposedResource)
			linkedType := HyperlinkResourceType(resourceType)
			resourceChanges = append(resourceChanges, fmt.Sprintf("  + %s (%s)", resourceName, linkedType))
		} else if currentExists && !proposedExists {
			resourceType := c.getResourceType(currentResource)
			linkedType := HyperlinkResourceType(resourceType)
			resourceChanges = append(resourceChanges, fmt.Sprintf("  - %s (%s)", resourceName, linkedType))
		} else if currentExists && proposedExists {
			if !reflect.DeepEqual(currentResource, proposedResource) {
				resourceType := c.getResourceType(proposedResource)
				linkedType := HyperlinkResourceType(resourceType)
				resourceChanges = append(resourceChanges, fmt.Sprintf("  ~ %s (%s)", resourceName, linkedType))
			}
		}
	}

	// Sort changes for consistent output
	sort.Strings(resourceChanges)

	for _, change := range resourceChanges {
		diff.WriteString(change + "\n")
	}

	return diff.String()
}

// getResourceType extracts the Type field from a resource definition
func (c *YAMLTemplateComparator) getResourceType(resource interface{}) string {
	if resourceMap, ok := resource.(map[string]interface{}); ok {
		if resourceType, ok := resourceMap["Type"]; ok {
			if typeStr, ok := resourceType.(string); ok {
				return typeStr
			}
		}
	}
	return "Unknown"
}
