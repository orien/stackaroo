/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package ui

import (
	"os"
	"strings"
	"testing"

	"github.com/orien/stackaroo/internal/aws"
	"github.com/orien/stackaroo/internal/diff"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildSections_NewStack(t *testing.T) {
	result := &diff.Result{
		StackName:   "test-stack",
		Context:     "dev",
		StackExists: false,
		ParameterDiffs: []diff.ParameterDiff{
			{Key: "Param1", ProposedValue: "value1", ChangeType: diff.ChangeTypeAdd},
		},
		TagDiffs: []diff.TagDiff{
			{Key: "Environment", ProposedValue: "dev", ChangeType: diff.ChangeTypeAdd},
		},
	}

	sections := buildSections(result)

	require.Len(t, sections, 2, "should have 2 sections for new stack")
	assert.Equal(t, "Parameters", sections[0].Name)
	assert.True(t, sections[0].HasChanges)
	assert.Equal(t, "Tags", sections[1].Name)
	assert.True(t, sections[1].HasChanges)
}

func TestBuildSections_ExistingStackWithAllChanges(t *testing.T) {
	result := &diff.Result{
		StackName:   "test-stack",
		Context:     "dev",
		StackExists: true,
		TemplateChange: &diff.TemplateChange{
			HasChanges: true,
			ResourceCount: struct{ Added, Modified, Removed int }{
				Added:    1,
				Modified: 2,
				Removed:  0,
			},
		},
		ParameterDiffs: []diff.ParameterDiff{
			{Key: "Param1", CurrentValue: "old", ProposedValue: "new", ChangeType: diff.ChangeTypeModify},
		},
		TagDiffs: []diff.TagDiff{
			{Key: "Tag1", ProposedValue: "value1", ChangeType: diff.ChangeTypeAdd},
		},
		ChangeSet: &aws.ChangeSetInfo{
			Changes: []aws.ResourceChange{
				{Action: "Add", LogicalID: "Resource1", ResourceType: "AWS::S3::Bucket"},
			},
		},
	}

	sections := buildSections(result)

	require.Len(t, sections, 4, "should have 4 sections")
	assert.Equal(t, "Template", sections[0].Name)
	assert.Equal(t, "Parameters", sections[1].Name)
	assert.Equal(t, "Tags", sections[2].Name)
	assert.Equal(t, "CloudFormation Plan", sections[3].Name)

	// All should have changes
	for _, section := range sections {
		assert.True(t, section.HasChanges, "section %s should have changes", section.Name)
	}
}

func TestBuildSections_FilteredOptions(t *testing.T) {
	tests := []struct {
		name          string
		options       diff.Options
		expectedCount int
		expectedNames []string
	}{
		{
			name:          "template only",
			options:       diff.Options{TemplateOnly: true},
			expectedCount: 1,
			expectedNames: []string{"Template"},
		},
		{
			name:          "parameters only",
			options:       diff.Options{ParametersOnly: true},
			expectedCount: 1,
			expectedNames: []string{"Parameters"},
		},
		{
			name:          "tags only",
			options:       diff.Options{TagsOnly: true},
			expectedCount: 1,
			expectedNames: []string{"Tags"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &diff.Result{
				StackExists: true,
				TemplateChange: &diff.TemplateChange{
					HasChanges: true,
				},
				ParameterDiffs: []diff.ParameterDiff{
					{Key: "Param1", ChangeType: diff.ChangeTypeAdd},
				},
				TagDiffs: []diff.TagDiff{
					{Key: "Tag1", ChangeType: diff.ChangeTypeAdd},
				},
				Options: tt.options,
			}

			sections := buildSections(result)

			assert.Len(t, sections, tt.expectedCount)
			for i, name := range tt.expectedNames {
				assert.Equal(t, name, sections[i].Name)
			}
		})
	}
}

func TestFormatTemplateSection_WithChanges(t *testing.T) {
	tc := &diff.TemplateChange{
		HasChanges: true,
		ResourceCount: struct{ Added, Modified, Removed int }{
			Added:    2,
			Modified: 1,
			Removed:  1,
		},
		Diff: "Template diff content",
	}

	content := formatTemplateSection(tc)

	assert.Contains(t, content, "Template diff content")
}

func TestFormatTemplateSection_NoChanges(t *testing.T) {
	tc := &diff.TemplateChange{
		HasChanges: false,
	}

	content := formatTemplateSection(tc)

	assert.Contains(t, content, "No template changes")
}

func TestFormatParameterSection_AllChangeTypes(t *testing.T) {
	params := []diff.ParameterDiff{
		{Key: "AddedParam", ProposedValue: "newvalue", ChangeType: diff.ChangeTypeAdd},
		{Key: "ModifiedParam", CurrentValue: "old", ProposedValue: "new", ChangeType: diff.ChangeTypeModify},
		{Key: "RemovedParam", CurrentValue: "oldvalue", ChangeType: diff.ChangeTypeRemove},
	}

	content := formatParameterSection(params, false)

	assert.Contains(t, content, "AddedParam")
	assert.Contains(t, content, "newvalue")
	assert.Contains(t, content, "ModifiedParam")
	assert.Contains(t, content, "old")
	assert.Contains(t, content, "new")
	assert.Contains(t, content, "RemovedParam")
	assert.Contains(t, content, "oldvalue")
}

func TestFormatParameterSection_NewStack(t *testing.T) {
	params := []diff.ParameterDiff{
		{Key: "Param1", ProposedValue: "value1", ChangeType: diff.ChangeTypeAdd},
	}

	content := formatParameterSection(params, true)

	assert.Contains(t, content, "Param1")
}

func TestFormatTagSection_AllChangeTypes(t *testing.T) {
	tags := []diff.TagDiff{
		{Key: "NewTag", ProposedValue: "value1", ChangeType: diff.ChangeTypeAdd},
		{Key: "UpdatedTag", CurrentValue: "old", ProposedValue: "new", ChangeType: diff.ChangeTypeModify},
		{Key: "DeletedTag", CurrentValue: "oldvalue", ChangeType: diff.ChangeTypeRemove},
	}

	content := formatTagSection(tags, false)

	assert.Contains(t, content, "NewTag")
	assert.Contains(t, content, "value1")
	assert.Contains(t, content, "UpdatedTag")
	assert.Contains(t, content, "old")
	assert.Contains(t, content, "new")
	assert.Contains(t, content, "DeletedTag")
	assert.Contains(t, content, "oldvalue")
}

func TestFormatChangeSetSection(t *testing.T) {
	cs := &aws.ChangeSetInfo{
		Changes: []aws.ResourceChange{
			{
				Action:       "Add",
				LogicalID:    "NewBucket",
				ResourceType: "AWS::S3::Bucket",
				PhysicalID:   "",
				Replacement:  "False",
				Details:      []string{"Property: BucketName"},
			},
			{
				Action:       "Modify",
				LogicalID:    "WebServer",
				ResourceType: "AWS::EC2::Instance",
				PhysicalID:   "i-1234567890",
				Replacement:  "True",
				Details:      []string{"Property: InstanceType"},
			},
			{
				Action:       "Remove",
				LogicalID:    "OldQueue",
				ResourceType: "AWS::SQS::Queue",
				PhysicalID:   "queue-url",
			},
		},
	}

	content := formatChangeSetSection(cs)

	assert.Contains(t, content, "NewBucket")
	assert.Contains(t, content, "AWS::S3::Bucket")
	assert.Contains(t, content, "WebServer")
	assert.Contains(t, content, "AWS::EC2::Instance")
	assert.Contains(t, content, "i-1234567890")
	assert.Contains(t, content, "Replacement: True")
	assert.Contains(t, content, "OldQueue")
	assert.Contains(t, content, "AWS::SQS::Queue")
	assert.Contains(t, content, "Property: BucketName")
	assert.Contains(t, content, "Property: InstanceType")
}

func TestGetChangeSymbol(t *testing.T) {
	// Disable colour for consistent testing
	_ = os.Setenv("NO_COLOR", "1")
	defer func() { _ = os.Unsetenv("NO_COLOR") }()

	styles := diff.NewStyles(false)

	tests := []struct {
		changeType diff.ChangeType
		expected   string
	}{
		{diff.ChangeTypeAdd, "+"},
		{diff.ChangeTypeModify, "~"},
		{diff.ChangeTypeRemove, "-"},
	}

	for _, tt := range tests {
		t.Run(string(tt.changeType), func(t *testing.T) {
			symbol := getChangeSymbol(tt.changeType, styles)
			assert.Contains(t, symbol, tt.expected, "symbol should contain expected character")
		})
	}
}

func TestGetChangeSetSymbol(t *testing.T) {
	// Disable colour for consistent testing
	_ = os.Setenv("NO_COLOR", "1")
	defer func() { _ = os.Unsetenv("NO_COLOR") }()

	styles := diff.NewStyles(false)

	tests := []struct {
		action   string
		expected string
	}{
		{"Add", "+"},
		{"Modify", "~"},
		{"Remove", "-"},
		{"Unknown", "?"},
	}

	for _, tt := range tests {
		t.Run(tt.action, func(t *testing.T) {
			symbol := getChangeSetSymbol(tt.action, styles)
			assert.Contains(t, symbol, tt.expected, "symbol should contain expected character")
		})
	}
}

func TestCountLines(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{
			name:     "empty string",
			input:    "",
			expected: 0,
		},
		{
			name:     "single line",
			input:    "single line",
			expected: 0,
		},
		{
			name:     "two lines",
			input:    "line1\nline2",
			expected: 1,
		},
		{
			name:     "multiple lines",
			input:    "line1\nline2\nline3\nline4",
			expected: 3,
		},
		{
			name:     "trailing newline",
			input:    "line1\nline2\n",
			expected: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			count := countLines(tt.input)
			assert.Equal(t, tt.expected, count)
		})
	}
}

func TestShouldUseColour(t *testing.T) {
	tests := []struct {
		name     string
		setup    func()
		teardown func()
		expected bool
	}{
		{
			name: "NO_COLOR set",
			setup: func() {
				_ = os.Setenv("NO_COLOR", "1")
			},
			teardown: func() {
				_ = os.Unsetenv("NO_COLOR")
			},
			expected: false,
		},
		{
			name: "TERM is dumb",
			setup: func() {
				_ = os.Setenv("TERM", "dumb")
			},
			teardown: func() {
				_ = os.Unsetenv("TERM")
			},
			expected: false,
		},
		{
			name: "TERM is empty",
			setup: func() {
				_ = os.Setenv("TERM", "")
			},
			teardown: func() {
				_ = os.Unsetenv("TERM")
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}
			if tt.teardown != nil {
				defer tt.teardown()
			}

			result := diff.ShouldUseColour()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatParameterSection_EmptyList(t *testing.T) {
	params := []diff.ParameterDiff{}
	content := formatParameterSection(params, false)
	assert.Empty(t, content)
}

func TestFormatTagSection_EmptyList(t *testing.T) {
	tags := []diff.TagDiff{}
	content := formatTagSection(tags, false)
	assert.Empty(t, content)
}

func TestBuildSections_StartLineCalculation(t *testing.T) {
	result := &diff.Result{
		StackExists: true,
		TemplateChange: &diff.TemplateChange{
			HasChanges: true,
		},
		ParameterDiffs: []diff.ParameterDiff{
			{Key: "Param1", ChangeType: diff.ChangeTypeAdd},
		},
		TagDiffs: []diff.TagDiff{
			{Key: "Tag1", ChangeType: diff.ChangeTypeAdd},
		},
	}

	sections := buildSections(result)

	// First section should start at line 0
	assert.Equal(t, 0, sections[0].StartLine)

	// Subsequent sections should have increasing start lines
	for i := 1; i < len(sections); i++ {
		assert.Greater(t, sections[i].StartLine, sections[i-1].StartLine,
			"section %d start line should be greater than section %d", i, i-1)
	}
}

func TestFormatTemplateSection_OnlyResourceCounts(t *testing.T) {
	tc := &diff.TemplateChange{
		HasChanges: true,
		ResourceCount: struct{ Added, Modified, Removed int }{
			Added:    1,
			Modified: 0,
			Removed:  0,
		},
		Diff: "", // No diff text
	}

	content := formatTemplateSection(tc)

	// With no diff text, shows no changes message even if HasChanges is true
	assert.Contains(t, content, "No template changes")
}

func TestFormatChangeSetSection_WithoutPhysicalID(t *testing.T) {
	cs := &aws.ChangeSetInfo{
		Changes: []aws.ResourceChange{
			{
				Action:       "Add",
				LogicalID:    "NewResource",
				ResourceType: "AWS::S3::Bucket",
				PhysicalID:   "", // No physical ID
			},
		},
	}

	content := formatChangeSetSection(cs)

	assert.Contains(t, content, "NewResource")
	assert.Contains(t, content, "AWS::S3::Bucket")
	// Should not contain empty brackets
	assert.NotContains(t, content, "[]")
}

func TestFormatChangeSetSection_WithoutReplacement(t *testing.T) {
	cs := &aws.ChangeSetInfo{
		Changes: []aws.ResourceChange{
			{
				Action:       "Modify",
				LogicalID:    "Resource",
				ResourceType: "AWS::EC2::Instance",
				Replacement:  "False",
			},
		},
	}

	content := formatChangeSetSection(cs)

	// Should not show replacement warning for False
	assert.NotContains(t, content, "Replacement:")
}

func TestFormatChangeSetSection_EmptyChangeSet(t *testing.T) {
	cs := &aws.ChangeSetInfo{
		Changes: []aws.ResourceChange{},
	}

	content := formatChangeSetSection(cs)

	// Empty changeset should produce empty content
	assert.Empty(t, strings.TrimSpace(content))
}
