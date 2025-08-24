/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package diff

import (
	"sort"
)

// DefaultParameterComparator implements ParameterComparator interface
type DefaultParameterComparator struct{}

// NewParameterComparator creates a new parameter comparator
func NewParameterComparator() ParameterComparator {
	return &DefaultParameterComparator{}
}

// Compare compares current and proposed parameters, returning differences
func (c *DefaultParameterComparator) Compare(currentParams, proposedParams map[string]string) ([]ParameterDiff, error) {
	var diffs []ParameterDiff

	// Track all parameter keys
	allKeys := make(map[string]bool)
	for key := range currentParams {
		allKeys[key] = true
	}
	for key := range proposedParams {
		allKeys[key] = true
	}

	// Compare each parameter
	for key := range allKeys {
		currentValue, currentExists := currentParams[key]
		proposedValue, proposedExists := proposedParams[key]

		var diff ParameterDiff
		diff.Key = key

		if !currentExists && proposedExists {
			// Parameter is being added
			diff.CurrentValue = ""
			diff.ProposedValue = proposedValue
			diff.ChangeType = ChangeTypeAdd
			diffs = append(diffs, diff)
		} else if currentExists && !proposedExists {
			// Parameter is being removed
			diff.CurrentValue = currentValue
			diff.ProposedValue = ""
			diff.ChangeType = ChangeTypeRemove
			diffs = append(diffs, diff)
		} else if currentExists && proposedExists && currentValue != proposedValue {
			// Parameter is being modified
			diff.CurrentValue = currentValue
			diff.ProposedValue = proposedValue
			diff.ChangeType = ChangeTypeModify
			diffs = append(diffs, diff)
		}
		// If currentValue == proposedValue, no diff needed
	}

	// Sort diffs by key for consistent output
	sort.Slice(diffs, func(i, j int) bool {
		return diffs[i].Key < diffs[j].Key
	})

	return diffs, nil
}

// DefaultTagComparator implements TagComparator interface
type DefaultTagComparator struct{}

// NewTagComparator creates a new tag comparator
func NewTagComparator() TagComparator {
	return &DefaultTagComparator{}
}

// Compare compares current and proposed tags, returning differences
func (c *DefaultTagComparator) Compare(currentTags, proposedTags map[string]string) ([]TagDiff, error) {
	var diffs []TagDiff

	// Track all tag keys
	allKeys := make(map[string]bool)
	for key := range currentTags {
		allKeys[key] = true
	}
	for key := range proposedTags {
		allKeys[key] = true
	}

	// Compare each tag
	for key := range allKeys {
		currentValue, currentExists := currentTags[key]
		proposedValue, proposedExists := proposedTags[key]

		var diff TagDiff
		diff.Key = key

		if !currentExists && proposedExists {
			// Tag is being added
			diff.CurrentValue = ""
			diff.ProposedValue = proposedValue
			diff.ChangeType = ChangeTypeAdd
			diffs = append(diffs, diff)
		} else if currentExists && !proposedExists {
			// Tag is being removed
			diff.CurrentValue = currentValue
			diff.ProposedValue = ""
			diff.ChangeType = ChangeTypeRemove
			diffs = append(diffs, diff)
		} else if currentExists && proposedExists && currentValue != proposedValue {
			// Tag is being modified
			diff.CurrentValue = currentValue
			diff.ProposedValue = proposedValue
			diff.ChangeType = ChangeTypeModify
			diffs = append(diffs, diff)
		}
		// If currentValue == proposedValue, no diff needed
	}

	// Sort diffs by key for consistent output
	sort.Slice(diffs, func(i, j int) bool {
		return diffs[i].Key < diffs[j].Key
	})

	return diffs, nil
}
