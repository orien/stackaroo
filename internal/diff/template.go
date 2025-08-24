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
	diff, err := c.generateDiff(currentData, proposedData)
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

// generateDiff creates a human-readable diff of the templates
func (c *YAMLTemplateComparator) generateDiff(currentData, proposedData map[string]interface{}) (string, error) {
	var diff strings.Builder

	// For now, provide a high-level summary
	// In a more sophisticated implementation, we could do line-by-line YAML diffing

	diff.WriteString("Template sections changed:\n")

	// Compare top-level sections
	allSections := make(map[string]bool)
	for section := range currentData {
		allSections[section] = true
	}
	for section := range proposedData {
		allSections[section] = true
	}

	var sectionChanges []string
	for section := range allSections {
		currentSection, currentExists := currentData[section]
		proposedSection, proposedExists := proposedData[section]

		if !currentExists && proposedExists {
			sectionChanges = append(sectionChanges, fmt.Sprintf("  + %s (added)", section))
		} else if currentExists && !proposedExists {
			sectionChanges = append(sectionChanges, fmt.Sprintf("  - %s (removed)", section))
		} else if currentExists && proposedExists {
			if !reflect.DeepEqual(currentSection, proposedSection) {
				sectionChanges = append(sectionChanges, fmt.Sprintf("  ~ %s (modified)", section))
			}
		}
	}

	// Sort changes for consistent output
	sort.Strings(sectionChanges)

	if len(sectionChanges) == 0 {
		diff.WriteString("  (No section-level changes detected)\n")
	} else {
		for _, change := range sectionChanges {
			diff.WriteString(change + "\n")
		}
	}

	// Add resource-specific details if Resources section changed
	if c.hasSectionChanged(currentData, proposedData, "Resources") {
		resourceDiff := c.generateResourceDiff(currentData, proposedData)
		if resourceDiff != "" {
			diff.WriteString("\nResource changes:\n")
			diff.WriteString(resourceDiff)
		}
	}

	return diff.String(), nil
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
			resourceChanges = append(resourceChanges, fmt.Sprintf("  + %s (%s)", resourceName, resourceType))
		} else if currentExists && !proposedExists {
			resourceType := c.getResourceType(currentResource)
			resourceChanges = append(resourceChanges, fmt.Sprintf("  - %s (%s)", resourceName, resourceType))
		} else if currentExists && proposedExists {
			if !reflect.DeepEqual(currentResource, proposedResource) {
				resourceType := c.getResourceType(proposedResource)
				resourceChanges = append(resourceChanges, fmt.Sprintf("  ~ %s (%s)", resourceName, resourceType))
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
