/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package diff

import (
	"os"
	"strings"
	"testing"

	"errors"

	"github.com/charmbracelet/x/term"
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

	assert.Contains(t, output, "test-stack - dev")
	assert.Contains(t, output, "No Changes")
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

	assert.Contains(t, output, "new-stack - prod")
	assert.Contains(t, output, "New Stack")
	assert.Contains(t, output, "This stack does not exist in AWS and will be created.")
	assert.Contains(t, output, "PARAMETERS")
	assert.Contains(t, output, "  + InstanceType: t3.micro")
	assert.Contains(t, output, "  + Environment: prod")
	assert.Contains(t, output, "TAGS")
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
	assert.Contains(t, output, "existing-stack - dev")
	assert.Contains(t, output, "Changes Detected")

	// Template changes
	assert.Contains(t, output, "TEMPLATE")
	assert.Contains(t, output, "Template has modifications")

	// Parameter changes
	assert.Contains(t, output, "PARAMETERS")
	assert.Contains(t, output, "~ InstanceType: t2.micro → t3.micro")
	assert.Contains(t, output, "+ NewParam: newvalue")
	assert.Contains(t, output, "- OldParam: oldvalue")

	// Tag changes
	assert.Contains(t, output, "TAGS")
	assert.Contains(t, output, "~ Environment: staging → dev")

	// Changeset info
	assert.Contains(t, output, "PLAN")
	// Check changeset contains resource type (may be hyperlinked)
	assert.Contains(t, output, "WebServer")
	assert.Contains(t, output, "AWS::EC2::Instance")
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
			expected:    []string{"TEMPLATE"},
			notExpected: []string{"PARAMETERS", "TAGS"},
		},
		{
			name:        "parameters only",
			options:     Options{ParametersOnly: true},
			expected:    []string{"PARAMETERS"},
			notExpected: []string{"TEMPLATE", "TAGS"},
		},
		{
			name:        "tags only",
			options:     Options{TagsOnly: true},
			expected:    []string{"TAGS"},
			notExpected: []string{"TEMPLATE", "PARAMETERS"},
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

	assert.Contains(t, text, "PARAMETERS")
	assert.Contains(t, text, "  + Param1: value1")
	assert.Contains(t, text, "  + Param2: value2")
	assert.Contains(t, text, "TAGS")
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
				"TEMPLATE",
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
				"TEMPLATE",
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
				"TEMPLATE",
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

	assert.Contains(t, text, "PARAMETERS")
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

	assert.Contains(t, text, "TAGS")
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

	assert.Contains(t, text, "PLAN")

	// Check resource change formatting (resource types may be hyperlinked)
	assert.Contains(t, text, "NewBucket")
	assert.Contains(t, text, "AWS::S3::Bucket")
	assert.Contains(t, text, "WebServer")
	assert.Contains(t, text, "AWS::EC2::Instance")
	assert.Contains(t, text, "[i-1234567890]")
	assert.Contains(t, text, "REPLACE")
	assert.Contains(t, text, "OldQueue")
	assert.Contains(t, text, "AWS::SQS::Queue")
	assert.Contains(t, text, "[old-queue-url]")

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

func TestResult_FormatChangeSetErrorText(t *testing.T) {
	result := &Result{
		ChangeSetError: assert.AnError,
	}

	// Set NO_COLOR for plain output in tests
	_ = os.Setenv("NO_COLOR", "1")
	defer func() { _ = os.Unsetenv("NO_COLOR") }()

	var output strings.Builder
	styles := NewStyles(false) // Use plain styles for testing
	result.formatChangeSetErrorText(&output, styles)
	text := output.String()

	// Check that error is displayed prominently
	assert.Contains(t, text, "PLAN")
	assert.Contains(t, text, "Changeset Generation Failed")
	assert.Contains(t, text, "CloudFormation was unable to generate a detailed change plan:")
	assert.Contains(t, text, assert.AnError.Error())

	// Check that reassurance is provided
	assert.Contains(t, text, "The parameter, tag, and template changes shown above are still accurate.")
	assert.Contains(t, text, "However, resource-level change details are not available.")

	// Check that guidance is provided
	assert.Contains(t, text, "Common causes:")
	assert.Contains(t, text, "Invalid parameter name")
	assert.Contains(t, text, "Invalid parameter value")
	assert.Contains(t, text, "Template validation errors")
	assert.Contains(t, text, "Missing required parameters")
	assert.Contains(t, text, "Review the error message and your configuration before proceeding.")
}

func TestResult_ToText_WithChangeSetError(t *testing.T) {
	result := &Result{
		StackName:   "test-stack",
		Context:     "dev",
		StackExists: true,
		ParameterDiffs: []ParameterDiff{
			{Key: "InvalidParam", ProposedValue: "value", ChangeType: ChangeTypeAdd},
		},
		ChangeSetError: errors.New("changeset creation failed: parameter InvalidParam does not exist in template"),
		Options:        Options{},
	}

	// Set NO_COLOR for plain output in tests
	_ = os.Setenv("NO_COLOR", "1")
	defer func() { _ = os.Unsetenv("NO_COLOR") }()

	output := result.toText()

	// Should show parameter changes
	assert.Contains(t, output, "PARAMETERS")
	assert.Contains(t, output, "+ InvalidParam: value")

	// Should show changeset error
	assert.Contains(t, output, "PLAN")
	assert.Contains(t, output, "Changeset Generation Failed")
	assert.Contains(t, output, "parameter InvalidParam does not exist in template")
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
		assert.Len(t, lines, 6, "Should have 6 lines (5 content + 1 empty from trailing newline)")

		// Verify content is preserved (even if colors aren't applied in test env)
		assert.Contains(t, lines[0], "@@", "Should contain hunk header")
		assert.Contains(t, lines[1], "context line", "Should contain context text")
		assert.Contains(t, lines[2], "removed line", "Should contain removed text")
		assert.Contains(t, lines[3], "added line", "Should contain added text")
		assert.Contains(t, lines[4], "another context", "Should contain second context text")
		assert.Equal(t, "", lines[5], "Last line should be empty from trailing newline")

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

		// Get terminal width for padding calculation (same as ColorizeUnifiedDiff)
		termWidth := 80
		if width, _, err := term.GetSize(os.Stdout.Fd()); err == nil && width > 0 {
			termWidth = width
		}
		maxLen := termWidth // Use full terminal width

		// Helper to pad a line to maxLen (with 2-char indent)
		padLine := func(line string) string {
			paddedLine := "  " + line
			if len(paddedLine) < maxLen {
				return paddedLine + strings.Repeat(" ", maxLen-len(paddedLine))
			}
			return paddedLine
		}

		// Test hunk header
		hunkResult := ColorizeUnifiedDiff("@@ -1,2 +1,3 @@", styles)
		expectedHunk := styles.DiffHunk.Render(padLine("@@ -1,2 +1,3 @@")) + "\n"
		assert.Equal(t, expectedHunk, hunkResult, "Hunk header should use DiffHunk style")

		// Test added line
		addedResult := ColorizeUnifiedDiff("+added content", styles)
		expectedAdded := styles.Added.Render(padLine("+added content")) + "\n"
		assert.Equal(t, expectedAdded, addedResult, "Added line should use added style")

		// Test removed line
		removedResult := ColorizeUnifiedDiff("-removed content", styles)
		expectedRemoved := styles.Removed.Render(padLine("-removed content")) + "\n"
		assert.Equal(t, expectedRemoved, removedResult, "Removed line should use removed style")

		// Test context line
		contextResult := ColorizeUnifiedDiff(" context content", styles)
		expectedContext := styles.DiffContext.Render(padLine(" context content")) + "\n"
		assert.Equal(t, expectedContext, contextResult, "Context line should use DiffContext style")
	})
}
