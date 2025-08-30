/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package diff

import (
	"strings"
	"testing"

	"github.com/orien/stackaroo/internal/aws"
	"github.com/stretchr/testify/assert"
)

func TestResult_String_TextFormat(t *testing.T) {
	result := &Result{
		StackName:   "test-stack",
		Context:     "dev",
		StackExists: true,
		Options:     Options{},
	}

	output := result.String()

	assert.Contains(t, output, "Stack: test-stack (Context: dev)")
	assert.Contains(t, output, "Status: NO CHANGES")
	assert.Contains(t, output, "The deployed stack matches your local configuration.")
}

func TestResult_ToText_NewStack(t *testing.T) {
	result := &Result{
		StackName:   "new-stack",
		Context:     "prod",
		StackExists: false,
		ParameterDiffs: []ParameterDiff{
			{Key: "InstanceType", ProposedValue: "t3.micro", ChangeType: ChangeTypeAdd},
			{Key: "Environment", ProposedValue: "prod", ChangeType: ChangeTypeAdd},
		},
		TagDiffs: []TagDiff{
			{Key: "Owner", ProposedValue: "team-a", ChangeType: ChangeTypeAdd},
			{Key: "Project", ProposedValue: "webapp", ChangeType: ChangeTypeAdd},
		},
	}

	output := result.toText()

	assert.Contains(t, output, "Stack: new-stack (Context: prod)")
	assert.Contains(t, output, "Status: NEW STACK")
	assert.Contains(t, output, "This stack does not exist in AWS and will be created.")
	assert.Contains(t, output, "Parameters to be set:")
	assert.Contains(t, output, "  + InstanceType: t3.micro")
	assert.Contains(t, output, "  + Environment: prod")
	assert.Contains(t, output, "Tags to be set:")
	assert.Contains(t, output, "  + Owner: team-a")
	assert.Contains(t, output, "  + Project: webapp")
}

func TestResult_ToText_WithChanges(t *testing.T) {
	result := &Result{
		StackName:   "existing-stack",
		Context:     "dev",
		StackExists: true,
		TemplateChange: &TemplateChange{
			HasChanges:   true,
			CurrentHash:  "abc123",
			ProposedHash: "def456",
			Diff:         "Template has modifications",
			ResourceCount: struct{ Added, Modified, Removed int }{
				Added:    2,
				Modified: 1,
				Removed:  1,
			},
		},
		ParameterDiffs: []ParameterDiff{
			{Key: "InstanceType", CurrentValue: "t2.micro", ProposedValue: "t3.micro", ChangeType: ChangeTypeModify},
			{Key: "NewParam", CurrentValue: "", ProposedValue: "newvalue", ChangeType: ChangeTypeAdd},
			{Key: "OldParam", CurrentValue: "oldvalue", ProposedValue: "", ChangeType: ChangeTypeRemove},
		},
		TagDiffs: []TagDiff{
			{Key: "Environment", CurrentValue: "staging", ProposedValue: "dev", ChangeType: ChangeTypeModify},
		},
		ChangeSet: &aws.ChangeSetInfo{
			ChangeSetID: "test-changeset-123",
			Status:      "CREATE_COMPLETE",
			Changes: []aws.ResourceChange{
				{
					Action:       "Modify",
					ResourceType: "AWS::EC2::Instance",
					LogicalID:    "WebServer",
					PhysicalID:   "i-1234567890abcdef0",
					Replacement:  "False",
					Details:      []string{"Property: InstanceType"},
				},
			},
		},
		Options: Options{},
	}

	output := result.toText()

	// Header checks
	assert.Contains(t, output, "Stack: existing-stack (Context: dev)")
	assert.Contains(t, output, "Status: CHANGES DETECTED")

	// Template changes
	assert.Contains(t, output, "Template Changes:")
	assert.Contains(t, output, "✓ Template has been modified")
	assert.Contains(t, output, "+ 2 resources to be added")
	assert.Contains(t, output, "~ 1 resources to be modified")
	assert.Contains(t, output, "- 1 resources to be removed")
	assert.Contains(t, output, "Template has modifications")

	// Parameter changes
	assert.Contains(t, output, "Parameter Changes:")
	assert.Contains(t, output, "~ InstanceType: t2.micro → t3.micro")
	assert.Contains(t, output, "+ NewParam: newvalue")
	assert.Contains(t, output, "- OldParam: oldvalue")

	// Tag changes
	assert.Contains(t, output, "Tag Changes:")
	assert.Contains(t, output, "~ Environment: staging → dev")

	// Changeset info
	assert.Contains(t, output, "AWS CloudFormation Preview:")
	assert.Contains(t, output, "ChangeSet ID: test-changeset-123")
	assert.Contains(t, output, "Status: CREATE_COMPLETE")
	assert.Contains(t, output, "~ WebServer (AWS::EC2::Instance)")
	assert.Contains(t, output, "[i-1234567890abcdef0]")
	assert.Contains(t, output, "Property: InstanceType")
}

func TestResult_ToText_FilteredOptions(t *testing.T) {
	tests := []struct {
		name        string
		options     Options
		expected    []string
		notExpected []string
	}{
		{
			name:        "template only",
			options:     Options{TemplateOnly: true},
			expected:    []string{"Template Changes:"},
			notExpected: []string{"Parameter Changes:", "Tag Changes:"},
		},
		{
			name:        "parameters only",
			options:     Options{ParametersOnly: true},
			expected:    []string{"Parameter Changes:"},
			notExpected: []string{"Template Changes:", "Tag Changes:"},
		},
		{
			name:        "tags only",
			options:     Options{TagsOnly: true},
			expected:    []string{"Tag Changes:"},
			notExpected: []string{"Template Changes:", "Parameter Changes:"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &Result{
				StackName:      "test-stack",
				Context:        "dev",
				StackExists:    true,
				TemplateChange: &TemplateChange{HasChanges: true, Diff: "template changes"},
				ParameterDiffs: []ParameterDiff{{Key: "test", ChangeType: ChangeTypeAdd}},
				TagDiffs:       []TagDiff{{Key: "test", ChangeType: ChangeTypeAdd}},
				Options:        tt.options,
			}

			output := result.toText()

			for _, expected := range tt.expected {
				assert.Contains(t, output, expected)
			}
			for _, notExpected := range tt.notExpected {
				assert.NotContains(t, output, notExpected)
			}
		})
	}
}

func TestResult_FormatNewStackText(t *testing.T) {
	result := &Result{
		ParameterDiffs: []ParameterDiff{
			{Key: "Param1", ProposedValue: "value1", ChangeType: ChangeTypeAdd},
			{Key: "Param2", ProposedValue: "value2", ChangeType: ChangeTypeAdd},
		},
		TagDiffs: []TagDiff{
			{Key: "Environment", ProposedValue: "dev", ChangeType: ChangeTypeAdd},
			{Key: "Project", ProposedValue: "test", ChangeType: ChangeTypeAdd},
		},
	}

	var output strings.Builder
	result.formatNewStackText(&output)
	text := output.String()

	assert.Contains(t, text, "Parameters to be set:")
	assert.Contains(t, text, "  + Param1: value1")
	assert.Contains(t, text, "  + Param2: value2")
	assert.Contains(t, text, "Tags to be set:")
	assert.Contains(t, text, "  + Environment: dev")
	assert.Contains(t, text, "  + Project: test")
}

func TestResult_FormatTemplateChangesText(t *testing.T) {
	tests := []struct {
		name           string
		templateChange *TemplateChange
		expectedOutput []string
	}{
		{
			name: "no changes",
			templateChange: &TemplateChange{
				HasChanges: false,
			},
			expectedOutput: []string{
				"Template Changes:",
				"✗ No template changes",
			},
		},
		{
			name: "with changes and resource counts",
			templateChange: &TemplateChange{
				HasChanges: true,
				ResourceCount: struct{ Added, Modified, Removed int }{
					Added: 2, Modified: 1, Removed: 1,
				},
				Diff: "Template diff content here",
			},
			expectedOutput: []string{
				"Template Changes:",
				"✓ Template has been modified",
				"+ 2 resources to be added",
				"~ 1 resources to be modified",
				"- 1 resources to be removed",
				"Template diff:",
				"Template diff content here",
			},
		},
		{
			name: "with changes but no resource counts",
			templateChange: &TemplateChange{
				HasChanges: true,
				ResourceCount: struct{ Added, Modified, Removed int }{
					Added: 0, Modified: 0, Removed: 0,
				},
			},
			expectedOutput: []string{
				"✓ Template has been modified",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &Result{TemplateChange: tt.templateChange}
			var output strings.Builder
			result.formatTemplateChangesText(&output)
			text := output.String()

			for _, expected := range tt.expectedOutput {
				assert.Contains(t, text, expected)
			}
		})
	}
}

func TestResult_FormatParameterChangesText(t *testing.T) {
	result := &Result{
		ParameterDiffs: []ParameterDiff{
			{Key: "AddedParam", CurrentValue: "", ProposedValue: "newvalue", ChangeType: ChangeTypeAdd},
			{Key: "ModifiedParam", CurrentValue: "oldvalue", ProposedValue: "newvalue", ChangeType: ChangeTypeModify},
			{Key: "RemovedParam", CurrentValue: "oldvalue", ProposedValue: "", ChangeType: ChangeTypeRemove},
		},
	}

	var output strings.Builder
	result.formatParameterChangesText(&output)
	text := output.String()

	assert.Contains(t, text, "Parameter Changes:")
	assert.Contains(t, text, "  + AddedParam: newvalue")
	assert.Contains(t, text, "  ~ ModifiedParam: oldvalue → newvalue")
	assert.Contains(t, text, "  - RemovedParam: oldvalue")
}

func TestResult_FormatTagChangesText(t *testing.T) {
	result := &Result{
		TagDiffs: []TagDiff{
			{Key: "NewTag", CurrentValue: "", ProposedValue: "newvalue", ChangeType: ChangeTypeAdd},
			{Key: "UpdatedTag", CurrentValue: "oldvalue", ProposedValue: "newvalue", ChangeType: ChangeTypeModify},
			{Key: "DeletedTag", CurrentValue: "oldvalue", ProposedValue: "", ChangeType: ChangeTypeRemove},
		},
	}

	var output strings.Builder
	result.formatTagChangesText(&output)
	text := output.String()

	assert.Contains(t, text, "Tag Changes:")
	assert.Contains(t, text, "  + NewTag: newvalue")
	assert.Contains(t, text, "  ~ UpdatedTag: oldvalue → newvalue")
	assert.Contains(t, text, "  - DeletedTag: oldvalue")
}

func TestResult_FormatChangeSetText(t *testing.T) {
	result := &Result{
		ChangeSet: &aws.ChangeSetInfo{
			ChangeSetID: "test-changeset-456",
			Status:      "CREATE_COMPLETE",
			Changes: []aws.ResourceChange{
				{
					Action:       "Add",
					ResourceType: "AWS::S3::Bucket",
					LogicalID:    "NewBucket",
					PhysicalID:   "",
					Replacement:  "False",
					Details:      []string{"Property: BucketName"},
				},
				{
					Action:       "Modify",
					ResourceType: "AWS::EC2::Instance",
					LogicalID:    "WebServer",
					PhysicalID:   "i-1234567890",
					Replacement:  "True",
					Details:      []string{"Property: InstanceType", "Property: SecurityGroups"},
				},
				{
					Action:       "Remove",
					ResourceType: "AWS::SQS::Queue",
					LogicalID:    "OldQueue",
					PhysicalID:   "old-queue-url",
					Replacement:  "False",
					Details:      []string{},
				},
			},
		},
	}

	var output strings.Builder
	result.formatChangeSetText(&output)
	text := output.String()

	assert.Contains(t, text, "AWS CloudFormation Preview:")
	assert.Contains(t, text, "ChangeSet ID: test-changeset-456")
	assert.Contains(t, text, "Status: CREATE_COMPLETE")
	assert.Contains(t, text, "Resource Changes:")

	// Check resource change formatting
	assert.Contains(t, text, "  + NewBucket (AWS::S3::Bucket)")
	assert.Contains(t, text, "  ~ WebServer (AWS::EC2::Instance) [i-1234567890] - Replacement: True")
	assert.Contains(t, text, "  - OldQueue (AWS::SQS::Queue) [old-queue-url]")

	// Check details
	assert.Contains(t, text, "    Property: BucketName")
	assert.Contains(t, text, "    Property: InstanceType")
	assert.Contains(t, text, "    Property: SecurityGroups")
}

func TestResult_GetChangeSymbol(t *testing.T) {
	result := &Result{}

	tests := []struct {
		action   string
		expected string
	}{
		{"Add", "+"},
		{"Modify", "~"},
		{"Remove", "-"},
		{"Unknown", "?"},
		{"", "?"},
	}

	for _, tt := range tests {
		t.Run(tt.action, func(t *testing.T) {
			symbol := result.getChangeSymbol(tt.action)
			assert.Equal(t, tt.expected, symbol)
		})
	}
}

func TestResult_HasChanges(t *testing.T) {
	tests := []struct {
		name     string
		result   *Result
		expected bool
	}{
		{
			name: "new stack",
			result: &Result{
				StackExists: false,
			},
			expected: true,
		},
		{
			name: "template changes",
			result: &Result{
				StackExists:    true,
				TemplateChange: &TemplateChange{HasChanges: true},
			},
			expected: true,
		},
		{
			name: "parameter changes",
			result: &Result{
				StackExists:    true,
				ParameterDiffs: []ParameterDiff{{Key: "test", ChangeType: ChangeTypeAdd}},
			},
			expected: true,
		},
		{
			name: "tag changes",
			result: &Result{
				StackExists: true,
				TagDiffs:    []TagDiff{{Key: "test", ChangeType: ChangeTypeAdd}},
			},
			expected: true,
		},
		{
			name: "no changes",
			result: &Result{
				StackExists:    true,
				TemplateChange: &TemplateChange{HasChanges: false},
				ParameterDiffs: []ParameterDiff{},
				TagDiffs:       []TagDiff{},
			},
			expected: false,
		},
		{
			name: "no changes - nil template change",
			result: &Result{
				StackExists:    true,
				TemplateChange: nil,
				ParameterDiffs: []ParameterDiff{},
				TagDiffs:       []TagDiff{},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasChanges := tt.result.HasChanges()
			assert.Equal(t, tt.expected, hasChanges)
		})
	}
}
