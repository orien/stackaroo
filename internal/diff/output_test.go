/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package diff

import (
	"os"
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
	assert.Contains(t, output, "Parameters")
	assert.Contains(t, output, "  + InstanceType: t3.micro")
	assert.Contains(t, output, "  + Environment: prod")
	assert.Contains(t, output, "Tags")
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
	assert.Contains(t, output, "Template")
	assert.Contains(t, output, "Template has modifications")

	// Parameter changes
	assert.Contains(t, output, "Parameters")
	assert.Contains(t, output, "~ InstanceType: t2.micro → t3.micro")
	assert.Contains(t, output, "+ NewParam: newvalue")
	assert.Contains(t, output, "- OldParam: oldvalue")

	// Tag changes
	assert.Contains(t, output, "Tags")
	assert.Contains(t, output, "~ Environment: staging → dev")

	// Changeset info
	assert.Contains(t, output, "CloudFormation Plan")
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
			expected:    []string{"Template"},
			notExpected: []string{"Parameters", "Tags"},
		},
		{
			name:        "parameters only",
			options:     Options{ParametersOnly: true},
			expected:    []string{"Parameters"},
			notExpected: []string{"Template", "Tags"},
		},
		{
			name:        "tags only",
			options:     Options{TagsOnly: true},
			expected:    []string{"Tags"},
			notExpected: []string{"Template", "Parameters"},
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

	// Set NO_COLOR for plain output in tests
	_ = os.Setenv("NO_COLOR", "1")
	defer func() { _ = os.Unsetenv("NO_COLOR") }()

	var output strings.Builder
	styles := NewStyles(false) // Use plain styles for testing
	result.formatNewStackText(&output, styles)
	text := output.String()

	assert.Contains(t, text, "Parameters")
	assert.Contains(t, text, "  + Param1: value1")
	assert.Contains(t, text, "  + Param2: value2")
	assert.Contains(t, text, "Tags")
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
				"Template",
				"✗ No template changes",
			},
		},
		{
			name: "with changes and diff",
			templateChange: &TemplateChange{
				HasChanges: true,
				ResourceCount: struct{ Added, Modified, Removed int }{
					Added: 2, Modified: 1, Removed: 1,
				},
				Diff: "Template diff content here",
			},
			expectedOutput: []string{
				"Template",
				"Template diff content here",
			},
		},
		{
			name: "with changes but no diff",
			templateChange: &TemplateChange{
				HasChanges: true,
				ResourceCount: struct{ Added, Modified, Removed int }{
					Added: 0, Modified: 0, Removed: 0,
				},
				Diff: "",
			},
			expectedOutput: []string{
				"Template",
				"✗ No template changes",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set NO_COLOR for plain output in tests
			_ = os.Setenv("NO_COLOR", "1")
			defer func() { _ = os.Unsetenv("NO_COLOR") }()

			result := &Result{TemplateChange: tt.templateChange}
			var output strings.Builder
			styles := NewStyles(false) // Use plain styles for testing
			result.formatTemplateChangesText(&output, styles)
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

	// Set NO_COLOR for plain output in tests
	_ = os.Setenv("NO_COLOR", "1")
	defer func() { _ = os.Unsetenv("NO_COLOR") }()

	var output strings.Builder
	styles := NewStyles(false) // Use plain styles for testing
	result.formatParameterChangesText(&output, styles)
	text := output.String()

	assert.Contains(t, text, "Parameters")
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

	// Set NO_COLOR for plain output in tests
	_ = os.Setenv("NO_COLOR", "1")
	defer func() { _ = os.Unsetenv("NO_COLOR") }()

	var output strings.Builder
	styles := NewStyles(false) // Use plain styles for testing
	result.formatTagChangesText(&output, styles)
	text := output.String()

	assert.Contains(t, text, "Tags")
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

	// Set NO_COLOR for plain output in tests
	_ = os.Setenv("NO_COLOR", "1")
	defer func() { _ = os.Unsetenv("NO_COLOR") }()

	var output strings.Builder
	styles := NewStyles(false) // Use plain styles for testing
	result.formatChangeSetText(&output, styles)
	text := output.String()

	assert.Contains(t, text, "CloudFormation Plan")
	assert.Contains(t, text, "Resource changes:")

	// Check resource change formatting
	assert.Contains(t, text, "  + NewBucket (AWS::S3::Bucket)")
	assert.Contains(t, text, "  ~ WebServer (AWS::EC2::Instance) [i-1234567890] - ⚠ Replacement: True")
	assert.Contains(t, text, "  - OldQueue (AWS::SQS::Queue) [old-queue-url]")

	// Check details
	assert.Contains(t, text, "    Property: BucketName")
	assert.Contains(t, text, "    Property: InstanceType")
	assert.Contains(t, text, "    Property: SecurityGroups")
}

func TestResult_GetChangeSymbol(t *testing.T) {
	// Set NO_COLOR for plain output in tests
	_ = os.Setenv("NO_COLOR", "1")
	defer func() { _ = os.Unsetenv("NO_COLOR") }()

	styles := NewStyles(false) // Use plain styles for testing

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
			symbol := styles.GetChangeSetSymbol(tt.action)
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

func TestColorizeUnifiedDiff(t *testing.T) {
	t.Run("with colors enabled", func(t *testing.T) {
		diff := `@@ -1,3 +1,4 @@
 context line
-removed line
+added line
 another context`

		styles := NewStyles(true)
		result := ColorizeUnifiedDiff(diff, styles)

		// Verify result is not empty and has correct structure
		assert.NotEmpty(t, result, "Result should not be empty")

		lines := strings.Split(result, "\n")
		assert.Len(t, lines, 5, "Should have 5 lines")

		// Verify content is preserved (even if colors aren't applied in test env)
		assert.Contains(t, lines[0], "@@", "Should contain hunk header")
		assert.Contains(t, lines[1], "context line", "Should contain context text")
		assert.Contains(t, lines[2], "removed line", "Should contain removed text")
		assert.Contains(t, lines[3], "added line", "Should contain added text")
		assert.Contains(t, lines[4], "another context", "Should contain second context text")

		// Verify line prefixes are preserved
		assert.True(t, strings.Contains(lines[0], "@@"), "Hunk header should contain @@")
		assert.True(t, strings.HasPrefix(lines[1], " ") || strings.Contains(lines[1], " context"), "Context line should have space prefix")
		assert.True(t, strings.HasPrefix(lines[2], "-") || strings.Contains(lines[2], "-removed"), "Removed line should have - prefix")
		assert.True(t, strings.HasPrefix(lines[3], "+") || strings.Contains(lines[3], "+added"), "Added line should have + prefix")
	})

	t.Run("with colors disabled", func(t *testing.T) {
		diff := `@@ -1,3 +1,4 @@
 context line
-removed line
+added line`

		styles := NewStyles(false)
		result := ColorizeUnifiedDiff(diff, styles)

		// Should not contain ANSI color codes
		assert.NotContains(t, result, "\x1b[", "Should not contain ANSI escape sequences")

		// Result should match input exactly (no color codes added)
		assert.Equal(t, diff, result, "Output should match input when colors disabled")
	})

	t.Run("empty diff", func(t *testing.T) {
		styles := NewStyles(true)
		result := ColorizeUnifiedDiff("", styles)
		assert.Equal(t, "", result, "Empty diff should return empty string")
	})

	t.Run("each line type colored correctly", func(t *testing.T) {
		styles := NewStyles(true)

		// Test hunk header
		hunkResult := ColorizeUnifiedDiff("@@ -1,2 +1,3 @@", styles)
		expectedHunk := styles.Key.Render("@@ -1,2 +1,3 @@")
		assert.Equal(t, expectedHunk, hunkResult, "Hunk header should use key style")

		// Test added line
		addedResult := ColorizeUnifiedDiff("+added content", styles)
		expectedAdded := styles.Added.Render("+added content")
		assert.Equal(t, expectedAdded, addedResult, "Added line should use added style")

		// Test removed line
		removedResult := ColorizeUnifiedDiff("-removed content", styles)
		expectedRemoved := styles.Removed.Render("-removed content")
		assert.Equal(t, expectedRemoved, removedResult, "Removed line should use removed style")

		// Test context line
		contextResult := ColorizeUnifiedDiff(" context content", styles)
		expectedContext := styles.Value.Render(" context content")
		assert.Equal(t, expectedContext, contextResult, "Context line should use value style")
	})
}
