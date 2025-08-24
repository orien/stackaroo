/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package diff

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParameterComparator_Compare_NoChanges(t *testing.T) {
	comparator := NewParameterComparator()

	currentParams := map[string]string{
		"Param1": "value1",
		"Param2": "value2",
	}

	proposedParams := map[string]string{
		"Param1": "value1",
		"Param2": "value2",
	}

	diffs, err := comparator.Compare(currentParams, proposedParams)

	require.NoError(t, err)
	assert.Empty(t, diffs)
}

func TestParameterComparator_Compare_EmptyMaps(t *testing.T) {
	comparator := NewParameterComparator()

	currentParams := map[string]string{}
	proposedParams := map[string]string{}

	diffs, err := comparator.Compare(currentParams, proposedParams)

	require.NoError(t, err)
	assert.Empty(t, diffs)
}

func TestParameterComparator_Compare_AddParameters(t *testing.T) {
	comparator := NewParameterComparator()

	currentParams := map[string]string{
		"Param1": "value1",
	}

	proposedParams := map[string]string{
		"Param1": "value1",
		"Param2": "value2",
		"Param3": "value3",
	}

	diffs, err := comparator.Compare(currentParams, proposedParams)

	require.NoError(t, err)
	assert.Len(t, diffs, 2)

	// Check that both new parameters are marked as ADD
	for _, diff := range diffs {
		switch diff.Key {
		case "Param2":
			assert.Equal(t, ChangeTypeAdd, diff.ChangeType)
			assert.Equal(t, "", diff.CurrentValue)
			assert.Equal(t, "value2", diff.ProposedValue)
		case "Param3":
			assert.Equal(t, ChangeTypeAdd, diff.ChangeType)
			assert.Equal(t, "", diff.CurrentValue)
			assert.Equal(t, "value3", diff.ProposedValue)
		default:
			t.Errorf("Unexpected parameter in diff: %s", diff.Key)
		}
	}
}

func TestParameterComparator_Compare_RemoveParameters(t *testing.T) {
	comparator := NewParameterComparator()

	currentParams := map[string]string{
		"Param1": "value1",
		"Param2": "value2",
		"Param3": "value3",
	}

	proposedParams := map[string]string{
		"Param1": "value1",
	}

	diffs, err := comparator.Compare(currentParams, proposedParams)

	require.NoError(t, err)
	assert.Len(t, diffs, 2)

	// Check that both removed parameters are marked as REMOVE
	for _, diff := range diffs {
		switch diff.Key {
		case "Param2":
			assert.Equal(t, ChangeTypeRemove, diff.ChangeType)
			assert.Equal(t, "value2", diff.CurrentValue)
			assert.Equal(t, "", diff.ProposedValue)
		case "Param3":
			assert.Equal(t, ChangeTypeRemove, diff.ChangeType)
			assert.Equal(t, "value3", diff.CurrentValue)
			assert.Equal(t, "", diff.ProposedValue)
		default:
			t.Errorf("Unexpected parameter in diff: %s", diff.Key)
		}
	}
}

func TestParameterComparator_Compare_ModifyParameters(t *testing.T) {
	comparator := NewParameterComparator()

	currentParams := map[string]string{
		"Param1": "oldvalue1",
		"Param2": "value2",
		"Param3": "oldvalue3",
	}

	proposedParams := map[string]string{
		"Param1": "newvalue1",
		"Param2": "value2",
		"Param3": "newvalue3",
	}

	diffs, err := comparator.Compare(currentParams, proposedParams)

	require.NoError(t, err)
	assert.Len(t, diffs, 2)

	// Check that both modified parameters are marked as MODIFY
	for _, diff := range diffs {
		switch diff.Key {
		case "Param1":
			assert.Equal(t, ChangeTypeModify, diff.ChangeType)
			assert.Equal(t, "oldvalue1", diff.CurrentValue)
			assert.Equal(t, "newvalue1", diff.ProposedValue)
		case "Param3":
			assert.Equal(t, ChangeTypeModify, diff.ChangeType)
			assert.Equal(t, "oldvalue3", diff.CurrentValue)
			assert.Equal(t, "newvalue3", diff.ProposedValue)
		default:
			t.Errorf("Unexpected parameter in diff: %s", diff.Key)
		}
	}
}

func TestParameterComparator_Compare_MixedChanges(t *testing.T) {
	comparator := NewParameterComparator()

	currentParams := map[string]string{
		"Param1": "value1",    // unchanged
		"Param2": "oldvalue2", // modified
		"Param3": "value3",    // removed
	}

	proposedParams := map[string]string{
		"Param1": "value1",    // unchanged
		"Param2": "newvalue2", // modified
		"Param4": "value4",    // added
	}

	diffs, err := comparator.Compare(currentParams, proposedParams)

	require.NoError(t, err)
	assert.Len(t, diffs, 3)

	// Sort diffs by type for easier testing
	diffMap := make(map[string]ParameterDiff)
	for _, diff := range diffs {
		diffMap[diff.Key] = diff
	}

	// Check modified parameter
	modifyDiff := diffMap["Param2"]
	assert.Equal(t, ChangeTypeModify, modifyDiff.ChangeType)
	assert.Equal(t, "oldvalue2", modifyDiff.CurrentValue)
	assert.Equal(t, "newvalue2", modifyDiff.ProposedValue)

	// Check removed parameter
	removeDiff := diffMap["Param3"]
	assert.Equal(t, ChangeTypeRemove, removeDiff.ChangeType)
	assert.Equal(t, "value3", removeDiff.CurrentValue)
	assert.Equal(t, "", removeDiff.ProposedValue)

	// Check added parameter
	addDiff := diffMap["Param4"]
	assert.Equal(t, ChangeTypeAdd, addDiff.ChangeType)
	assert.Equal(t, "", addDiff.CurrentValue)
	assert.Equal(t, "value4", addDiff.ProposedValue)
}

func TestParameterComparator_Compare_SortedOutput(t *testing.T) {
	comparator := NewParameterComparator()

	currentParams := map[string]string{}
	proposedParams := map[string]string{
		"ZParam": "valueZ",
		"AParam": "valueA",
		"MParam": "valueM",
	}

	diffs, err := comparator.Compare(currentParams, proposedParams)

	require.NoError(t, err)
	assert.Len(t, diffs, 3)

	// Verify parameters are sorted by key
	assert.Equal(t, "AParam", diffs[0].Key)
	assert.Equal(t, "MParam", diffs[1].Key)
	assert.Equal(t, "ZParam", diffs[2].Key)
}

func TestTagComparator_Compare_NoChanges(t *testing.T) {
	comparator := NewTagComparator()

	currentTags := map[string]string{
		"Environment": "dev",
		"Project":     "test",
	}

	proposedTags := map[string]string{
		"Environment": "dev",
		"Project":     "test",
	}

	diffs, err := comparator.Compare(currentTags, proposedTags)

	require.NoError(t, err)
	assert.Empty(t, diffs)
}

func TestTagComparator_Compare_AddTags(t *testing.T) {
	comparator := NewTagComparator()

	currentTags := map[string]string{
		"Environment": "dev",
	}

	proposedTags := map[string]string{
		"Environment": "dev",
		"Owner":       "team",
		"CostCenter":  "engineering",
	}

	diffs, err := comparator.Compare(currentTags, proposedTags)

	require.NoError(t, err)
	assert.Len(t, diffs, 2)

	// Check that both new tags are marked as ADD
	for _, diff := range diffs {
		switch diff.Key {
		case "Owner":
			assert.Equal(t, ChangeTypeAdd, diff.ChangeType)
			assert.Equal(t, "", diff.CurrentValue)
			assert.Equal(t, "team", diff.ProposedValue)
		case "CostCenter":
			assert.Equal(t, ChangeTypeAdd, diff.ChangeType)
			assert.Equal(t, "", diff.CurrentValue)
			assert.Equal(t, "engineering", diff.ProposedValue)
		default:
			t.Errorf("Unexpected tag in diff: %s", diff.Key)
		}
	}
}

func TestTagComparator_Compare_RemoveTags(t *testing.T) {
	comparator := NewTagComparator()

	currentTags := map[string]string{
		"Environment": "dev",
		"Owner":       "team",
		"CostCenter":  "engineering",
	}

	proposedTags := map[string]string{
		"Environment": "dev",
	}

	diffs, err := comparator.Compare(currentTags, proposedTags)

	require.NoError(t, err)
	assert.Len(t, diffs, 2)

	// Check that both removed tags are marked as REMOVE
	for _, diff := range diffs {
		switch diff.Key {
		case "Owner":
			assert.Equal(t, ChangeTypeRemove, diff.ChangeType)
			assert.Equal(t, "team", diff.CurrentValue)
			assert.Equal(t, "", diff.ProposedValue)
		case "CostCenter":
			assert.Equal(t, ChangeTypeRemove, diff.ChangeType)
			assert.Equal(t, "engineering", diff.CurrentValue)
			assert.Equal(t, "", diff.ProposedValue)
		default:
			t.Errorf("Unexpected tag in diff: %s", diff.Key)
		}
	}
}

func TestTagComparator_Compare_ModifyTags(t *testing.T) {
	comparator := NewTagComparator()

	currentTags := map[string]string{
		"Environment": "dev",
		"Owner":       "oldteam",
		"Project":     "test",
	}

	proposedTags := map[string]string{
		"Environment": "dev",
		"Owner":       "newteam",
		"Project":     "test",
	}

	diffs, err := comparator.Compare(currentTags, proposedTags)

	require.NoError(t, err)
	assert.Len(t, diffs, 1)

	diff := diffs[0]
	assert.Equal(t, "Owner", diff.Key)
	assert.Equal(t, ChangeTypeModify, diff.ChangeType)
	assert.Equal(t, "oldteam", diff.CurrentValue)
	assert.Equal(t, "newteam", diff.ProposedValue)
}

func TestTagComparator_Compare_MixedChanges(t *testing.T) {
	comparator := NewTagComparator()

	currentTags := map[string]string{
		"Environment": "dev",     // unchanged
		"Owner":       "oldteam", // modified
		"OldTag":      "remove",  // removed
	}

	proposedTags := map[string]string{
		"Environment": "dev",     // unchanged
		"Owner":       "newteam", // modified
		"NewTag":      "add",     // added
	}

	diffs, err := comparator.Compare(currentTags, proposedTags)

	require.NoError(t, err)
	assert.Len(t, diffs, 3)

	// Sort diffs by type for easier testing
	diffMap := make(map[string]TagDiff)
	for _, diff := range diffs {
		diffMap[diff.Key] = diff
	}

	// Check added tag
	addDiff := diffMap["NewTag"]
	assert.Equal(t, ChangeTypeAdd, addDiff.ChangeType)
	assert.Equal(t, "", addDiff.CurrentValue)
	assert.Equal(t, "add", addDiff.ProposedValue)

	// Check removed tag
	removeDiff := diffMap["OldTag"]
	assert.Equal(t, ChangeTypeRemove, removeDiff.ChangeType)
	assert.Equal(t, "remove", removeDiff.CurrentValue)
	assert.Equal(t, "", removeDiff.ProposedValue)

	// Check modified tag
	modifyDiff := diffMap["Owner"]
	assert.Equal(t, ChangeTypeModify, modifyDiff.ChangeType)
	assert.Equal(t, "oldteam", modifyDiff.CurrentValue)
	assert.Equal(t, "newteam", modifyDiff.ProposedValue)
}

func TestTagComparator_Compare_SortedOutput(t *testing.T) {
	comparator := NewTagComparator()

	currentTags := map[string]string{}
	proposedTags := map[string]string{
		"ZTag": "valueZ",
		"ATag": "valueA",
		"MTag": "valueM",
	}

	diffs, err := comparator.Compare(currentTags, proposedTags)

	require.NoError(t, err)
	assert.Len(t, diffs, 3)

	// Verify tags are sorted by key
	assert.Equal(t, "ATag", diffs[0].Key)
	assert.Equal(t, "MTag", diffs[1].Key)
	assert.Equal(t, "ZTag", diffs[2].Key)
}
