/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package diff

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/orien/stackaroo/internal/aws"
)

func TestChangeType_Constants(t *testing.T) {
	// Test that the ChangeType constants have expected values
	assert.Equal(t, ChangeType("ADD"), ChangeTypeAdd)
	assert.Equal(t, ChangeType("MODIFY"), ChangeTypeModify)
	assert.Equal(t, ChangeType("REMOVE"), ChangeTypeRemove)
}

func TestChangeType_StringConversion(t *testing.T) {
	// Test that ChangeType can be converted to string
	assert.Equal(t, "ADD", string(ChangeTypeAdd))
	assert.Equal(t, "MODIFY", string(ChangeTypeModify))
	assert.Equal(t, "REMOVE", string(ChangeTypeRemove))
}

func TestOptions_DefaultValues(t *testing.T) {
	// Test default zero values
	options := Options{}

	assert.False(t, options.TemplateOnly)
	assert.False(t, options.ParametersOnly)
	assert.False(t, options.TagsOnly)
	assert.Equal(t, "", options.Format)
}

func TestOptions_FieldAssignment(t *testing.T) {
	// Test that Options fields can be set and retrieved
	options := Options{
		TemplateOnly:   true,
		ParametersOnly: false,
		TagsOnly:       true,
		Format:         "json",
	}

	assert.True(t, options.TemplateOnly)
	assert.False(t, options.ParametersOnly)
	assert.True(t, options.TagsOnly)
	assert.Equal(t, "json", options.Format)
}

func TestResult_DefaultValues(t *testing.T) {
	// Test default zero values
	result := Result{}

	assert.Equal(t, "", result.StackName)
	assert.Equal(t, "", result.Context)
	assert.False(t, result.StackExists)
	assert.Nil(t, result.TemplateChange)
	assert.Nil(t, result.ParameterDiffs)
	assert.Nil(t, result.TagDiffs)
	assert.Nil(t, result.ChangeSet)
	assert.Equal(t, Options{}, result.Options)
}

func TestResult_FieldAssignment(t *testing.T) {
	// Test that Result fields can be set and retrieved
	templateChange := &TemplateChange{HasChanges: true}
	paramDiffs := []ParameterDiff{{Key: "test"}}
	tagDiffs := []TagDiff{{Key: "test"}}
	changeSet := &aws.ChangeSetInfo{ChangeSetID: "test"}
	options := Options{Format: "text"}

	result := Result{
		StackName:      "test-stack",
		Context:        "prod",
		StackExists:    true,
		TemplateChange: templateChange,
		ParameterDiffs: paramDiffs,
		TagDiffs:       tagDiffs,
		ChangeSet:      changeSet,
		Options:        options,
	}

	assert.Equal(t, "test-stack", result.StackName)
	assert.Equal(t, "prod", result.Context)
	assert.True(t, result.StackExists)
	assert.Equal(t, templateChange, result.TemplateChange)
	assert.Equal(t, paramDiffs, result.ParameterDiffs)
	assert.Equal(t, tagDiffs, result.TagDiffs)
	assert.Equal(t, changeSet, result.ChangeSet)
	assert.Equal(t, options, result.Options)
}

func TestTemplateChange_DefaultValues(t *testing.T) {
	// Test default zero values
	change := TemplateChange{}

	assert.False(t, change.HasChanges)
	assert.Equal(t, "", change.CurrentHash)
	assert.Equal(t, "", change.ProposedHash)
	assert.Equal(t, "", change.Diff)
	assert.Equal(t, 0, change.ResourceCount.Added)
	assert.Equal(t, 0, change.ResourceCount.Modified)
	assert.Equal(t, 0, change.ResourceCount.Removed)
}

func TestTemplateChange_FieldAssignment(t *testing.T) {
	// Test that TemplateChange fields can be set and retrieved
	change := TemplateChange{
		HasChanges:   true,
		CurrentHash:  "abc123",
		ProposedHash: "def456",
		Diff:         "template differences",
		ResourceCount: struct{ Added, Modified, Removed int }{
			Added:    2,
			Modified: 1,
			Removed:  0,
		},
	}

	assert.True(t, change.HasChanges)
	assert.Equal(t, "abc123", change.CurrentHash)
	assert.Equal(t, "def456", change.ProposedHash)
	assert.Equal(t, "template differences", change.Diff)
	assert.Equal(t, 2, change.ResourceCount.Added)
	assert.Equal(t, 1, change.ResourceCount.Modified)
	assert.Equal(t, 0, change.ResourceCount.Removed)
}

func TestParameterDiff_DefaultValues(t *testing.T) {
	// Test default zero values
	diff := ParameterDiff{}

	assert.Equal(t, "", diff.Key)
	assert.Equal(t, "", diff.CurrentValue)
	assert.Equal(t, "", diff.ProposedValue)
	assert.Equal(t, ChangeType(""), diff.ChangeType)
}

func TestParameterDiff_FieldAssignment(t *testing.T) {
	// Test that ParameterDiff fields can be set and retrieved
	diff := ParameterDiff{
		Key:           "InstanceType",
		CurrentValue:  "t2.micro",
		ProposedValue: "t3.micro",
		ChangeType:    ChangeTypeModify,
	}

	assert.Equal(t, "InstanceType", diff.Key)
	assert.Equal(t, "t2.micro", diff.CurrentValue)
	assert.Equal(t, "t3.micro", diff.ProposedValue)
	assert.Equal(t, ChangeTypeModify, diff.ChangeType)
}

func TestParameterDiff_AllChangeTypes(t *testing.T) {
	// Test ParameterDiff with all change types
	tests := []struct {
		name        string
		changeType  ChangeType
		currentVal  string
		proposedVal string
	}{
		{
			name:        "add parameter",
			changeType:  ChangeTypeAdd,
			currentVal:  "",
			proposedVal: "newvalue",
		},
		{
			name:        "modify parameter",
			changeType:  ChangeTypeModify,
			currentVal:  "oldvalue",
			proposedVal: "newvalue",
		},
		{
			name:        "remove parameter",
			changeType:  ChangeTypeRemove,
			currentVal:  "oldvalue",
			proposedVal: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diff := ParameterDiff{
				Key:           "TestParam",
				CurrentValue:  tt.currentVal,
				ProposedValue: tt.proposedVal,
				ChangeType:    tt.changeType,
			}

			assert.Equal(t, "TestParam", diff.Key)
			assert.Equal(t, tt.currentVal, diff.CurrentValue)
			assert.Equal(t, tt.proposedVal, diff.ProposedValue)
			assert.Equal(t, tt.changeType, diff.ChangeType)
		})
	}
}

func TestTagDiff_DefaultValues(t *testing.T) {
	// Test default zero values
	diff := TagDiff{}

	assert.Equal(t, "", diff.Key)
	assert.Equal(t, "", diff.CurrentValue)
	assert.Equal(t, "", diff.ProposedValue)
	assert.Equal(t, ChangeType(""), diff.ChangeType)
}

func TestTagDiff_FieldAssignment(t *testing.T) {
	// Test that TagDiff fields can be set and retrieved
	diff := TagDiff{
		Key:           "Environment",
		CurrentValue:  "dev",
		ProposedValue: "prod",
		ChangeType:    ChangeTypeModify,
	}

	assert.Equal(t, "Environment", diff.Key)
	assert.Equal(t, "dev", diff.CurrentValue)
	assert.Equal(t, "prod", diff.ProposedValue)
	assert.Equal(t, ChangeTypeModify, diff.ChangeType)
}

func TestTagDiff_AllChangeTypes(t *testing.T) {
	// Test TagDiff with all change types
	tests := []struct {
		name        string
		changeType  ChangeType
		currentVal  string
		proposedVal string
	}{
		{
			name:        "add tag",
			changeType:  ChangeTypeAdd,
			currentVal:  "",
			proposedVal: "newvalue",
		},
		{
			name:        "modify tag",
			changeType:  ChangeTypeModify,
			currentVal:  "oldvalue",
			proposedVal: "newvalue",
		},
		{
			name:        "remove tag",
			changeType:  ChangeTypeRemove,
			currentVal:  "oldvalue",
			proposedVal: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diff := TagDiff{
				Key:           "TestTag",
				CurrentValue:  tt.currentVal,
				ProposedValue: tt.proposedVal,
				ChangeType:    tt.changeType,
			}

			assert.Equal(t, "TestTag", diff.Key)
			assert.Equal(t, tt.currentVal, diff.CurrentValue)
			assert.Equal(t, tt.proposedVal, diff.ProposedValue)
			assert.Equal(t, tt.changeType, diff.ChangeType)
		})
	}
}

func TestChangeSetInfo_DefaultValues(t *testing.T) {
	// Test default zero values
	info := aws.ChangeSetInfo{}

	assert.Equal(t, "", info.ChangeSetID)
	assert.Equal(t, "", info.Status)
	assert.Nil(t, info.Changes)
}

func TestChangeSetInfo_FieldAssignment(t *testing.T) {
	// Test that ChangeSetInfo fields can be set and retrieved
	changes := []aws.ResourceChange{
		{Action: "Add", ResourceType: "AWS::S3::Bucket", LogicalID: "MyBucket"},
	}

	info := aws.ChangeSetInfo{
		ChangeSetID: "changeset-123",
		Status:      "CREATE_COMPLETE",
		Changes:     changes,
	}

	assert.Equal(t, "changeset-123", info.ChangeSetID)
	assert.Equal(t, "CREATE_COMPLETE", info.Status)
	assert.Equal(t, changes, info.Changes)
}

func TestResourceChange_DefaultValues(t *testing.T) {
	// Test default zero values
	change := aws.ResourceChange{}

	assert.Equal(t, "", change.Action)
	assert.Equal(t, "", change.ResourceType)
	assert.Equal(t, "", change.LogicalID)
	assert.Equal(t, "", change.PhysicalID)
	assert.Equal(t, "", change.Replacement)
	assert.Nil(t, change.Details)
}

func TestResourceChange_FieldAssignment(t *testing.T) {
	// Test that ResourceChange fields can be set and retrieved
	details := []string{"Property: BucketName", "Property: Tags"}

	change := aws.ResourceChange{
		Action:       "Modify",
		ResourceType: "AWS::S3::Bucket",
		LogicalID:    "MyBucket",
		PhysicalID:   "my-bucket-12345",
		Replacement:  "False",
		Details:      details,
	}

	assert.Equal(t, "Modify", change.Action)
	assert.Equal(t, "AWS::S3::Bucket", change.ResourceType)
	assert.Equal(t, "MyBucket", change.LogicalID)
	assert.Equal(t, "my-bucket-12345", change.PhysicalID)
	assert.Equal(t, "False", change.Replacement)
	assert.Equal(t, details, change.Details)
}

func TestResourceChange_AllActions(t *testing.T) {
	// Test ResourceChange with different action types
	tests := []struct {
		name        string
		action      string
		replacement string
	}{
		{
			name:        "add resource",
			action:      "Add",
			replacement: "N/A",
		},
		{
			name:        "modify resource - no replacement",
			action:      "Modify",
			replacement: "False",
		},
		{
			name:        "modify resource - with replacement",
			action:      "Modify",
			replacement: "True",
		},
		{
			name:        "modify resource - conditional replacement",
			action:      "Modify",
			replacement: "Conditional",
		},
		{
			name:        "remove resource",
			action:      "Remove",
			replacement: "N/A",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			change := aws.ResourceChange{
				Action:       tt.action,
				ResourceType: "AWS::S3::Bucket",
				LogicalID:    "TestBucket",
				Replacement:  tt.replacement,
			}

			assert.Equal(t, tt.action, change.Action)
			assert.Equal(t, "AWS::S3::Bucket", change.ResourceType)
			assert.Equal(t, "TestBucket", change.LogicalID)
			assert.Equal(t, tt.replacement, change.Replacement)
		})
	}
}

func TestResourceChange_WithDetails(t *testing.T) {
	// Test ResourceChange with various details
	tests := []struct {
		name    string
		details []string
	}{
		{
			name:    "no details",
			details: nil,
		},
		{
			name:    "empty details",
			details: []string{},
		},
		{
			name:    "single detail",
			details: []string{"Property: BucketName"},
		},
		{
			name:    "multiple details",
			details: []string{"Property: BucketName", "Property: Tags", "Property: VersioningConfiguration"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			change := aws.ResourceChange{
				Action:       "Modify",
				ResourceType: "AWS::S3::Bucket",
				LogicalID:    "TestBucket",
				Details:      tt.details,
			}

			assert.Equal(t, tt.details, change.Details)
			if tt.details == nil {
				assert.Nil(t, change.Details)
			} else {
				assert.Equal(t, len(tt.details), len(change.Details))
			}
		})
	}
}

func TestResult_HasChanges_EdgeCases(t *testing.T) {
	// Test edge cases for HasChanges method
	tests := []struct {
		name     string
		result   Result
		expected bool
	}{
		{
			name: "new stack always has changes",
			result: Result{
				StackExists: false,
			},
			expected: true,
		},
		{
			name: "existing stack with no template change object",
			result: Result{
				StackExists:    true,
				TemplateChange: nil,
				ParameterDiffs: []ParameterDiff{},
				TagDiffs:       []TagDiff{},
			},
			expected: false,
		},
		{
			name: "existing stack with empty template change",
			result: Result{
				StackExists:    true,
				TemplateChange: &TemplateChange{HasChanges: false},
				ParameterDiffs: []ParameterDiff{},
				TagDiffs:       []TagDiff{},
			},
			expected: false,
		},
		{
			name: "existing stack with template changes but empty arrays",
			result: Result{
				StackExists:    true,
				TemplateChange: &TemplateChange{HasChanges: true},
				ParameterDiffs: []ParameterDiff{},
				TagDiffs:       []TagDiff{},
			},
			expected: true,
		},
		{
			name: "existing stack with parameter changes but no template changes",
			result: Result{
				StackExists:    true,
				TemplateChange: &TemplateChange{HasChanges: false},
				ParameterDiffs: []ParameterDiff{{Key: "test"}},
				TagDiffs:       []TagDiff{},
			},
			expected: true,
		},
		{
			name: "existing stack with tag changes but no other changes",
			result: Result{
				StackExists:    true,
				TemplateChange: &TemplateChange{HasChanges: false},
				ParameterDiffs: []ParameterDiff{},
				TagDiffs:       []TagDiff{{Key: "test"}},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasChanges := tt.result.HasChanges()
			assert.Equal(t, tt.expected, hasChanges)
		})
	}
}

func TestResult_StringMethod_CallsCorrectFormatter(t *testing.T) {
	// Test that String() method calls the correct formatter based on Options.Format
	tests := []struct {
		name           string
		format         string
		expectedOutput string
	}{
		{
			name:           "text format",
			format:         "text",
			expectedOutput: "Stack: test-stack", // Should contain text format elements
		},
		{
			name:           "json format",
			format:         "json",
			expectedOutput: `"stackName": "test-stack"`, // Should contain JSON elements
		},
		{
			name:           "default format (empty)",
			format:         "",
			expectedOutput: "Stack: test-stack", // Should default to text
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Result{
				StackName:   "test-stack",
				Context:     "dev",
				StackExists: true,
				Options:     Options{Format: tt.format},
			}

			output := result.String()
			assert.Contains(t, output, tt.expectedOutput)
		})
	}
}
