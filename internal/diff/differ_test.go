/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package diff

import (
	"context"
	"errors"
	"testing"

	"github.com/orien/stackaroo/internal/aws"
	"github.com/orien/stackaroo/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Creates a test differ with provided dependencies
func createTestDiffer(cfClient *aws.MockCloudFormationOperations, templateComp *MockTemplateComparator, paramComp *MockParameterComparator, tagComp *MockTagComparator) *StackDiffer {
	return &StackDiffer{
		cfClient:            cfClient,
		templateComparator:  templateComp,
		parameterComparator: paramComp,
		tagComparator:       tagComp,
	}
}

func createTestResolvedStack() *model.Stack {
	return &model.Stack{
		Name:         "test-stack",
		Context:      "dev",
		TemplateBody: `{"AWSTemplateFormatVersion": "2010-09-09"}`,
		Parameters:   map[string]string{"Param1": "value1", "Param2": "value2"},
		Tags:         map[string]string{"Environment": "dev", "Project": "test"},
		Capabilities: []string{"CAPABILITY_IAM"},
	}
}

func createTestStackInfo() *aws.StackInfo {
	return &aws.StackInfo{
		Name:       "test-stack",
		Status:     aws.StackStatusCreateComplete,
		Parameters: map[string]string{"Param1": "oldvalue1", "Param2": "value2"},
		Tags:       map[string]string{"Environment": "dev", "OldTag": "remove"},
		Template:   `{"AWSTemplateFormatVersion": "2010-09-09", "Resources": {}}`,
	}
}

// Tests

func TestStackDiffer_DiffStack_ExistingStack_NoChanges(t *testing.T) {
	// Test diff of existing stack with no changes
	ctx := context.Background()

	// Create mocks
	cfClient := &aws.MockCloudFormationOperations{}
	templateComp := &MockTemplateComparator{}
	paramComp := &MockParameterComparator{}
	tagComp := &MockTagComparator{}
	differ := createTestDiffer(cfClient, templateComp, paramComp, tagComp)

	// Test data
	stack := createTestResolvedStack()
	currentStack := &aws.StackInfo{
		Name:       "test-stack",
		Parameters: map[string]string{"Param1": "value1", "Param2": "value2"},
		Tags:       map[string]string{"Environment": "dev", "Project": "test"},
		Template:   `{"AWSTemplateFormatVersion": "2010-09-09"}`,
	}
	options := Options{}

	// Set up expectations
	cfClient.On("StackExists", ctx, "test-stack").Return(true, nil)
	cfClient.On("DescribeStack", ctx, "test-stack").Return(currentStack, nil)
	cfClient.On("GetTemplate", ctx, "test-stack").Return(currentStack.Template, nil)

	templateComp.On("Compare", ctx, currentStack.Template, stack.TemplateBody).Return(&TemplateChange{
		HasChanges:   false,
		CurrentHash:  "hash1",
		ProposedHash: "hash1",
	}, nil)

	paramComp.On("Compare", currentStack.Parameters, stack.Parameters).Return([]ParameterDiff{}, nil)
	tagComp.On("Compare", currentStack.Tags, stack.Tags).Return([]TagDiff{}, nil)

	// Execute
	result, err := differ.DiffStack(ctx, stack, options)

	// Verify
	require.NoError(t, err)
	// Verify result structure
	assert.NotNil(t, result)
	assert.Equal(t, "test-stack", result.StackName)
	assert.Equal(t, "dev", result.Context)
	assert.True(t, result.StackExists)
	assert.False(t, result.HasChanges())
	assert.Empty(t, result.ParameterDiffs)
	assert.Empty(t, result.TagDiffs)

	cfClient.AssertExpectations(t)
	templateComp.AssertExpectations(t)
	paramComp.AssertExpectations(t)
	tagComp.AssertExpectations(t)
}

func TestStackDiffer_DiffStack_ExistingStack_WithChanges(t *testing.T) {
	// Test diff of existing stack with changes
	ctx := context.Background()

	// Create mocks
	cfClient := &aws.MockCloudFormationOperations{}
	templateComp := &MockTemplateComparator{}
	paramComp := &MockParameterComparator{}
	tagComp := &MockTagComparator{}
	differ := createTestDiffer(cfClient, templateComp, paramComp, tagComp)

	// Test data
	stack := createTestResolvedStack()
	currentStack := createTestStackInfo()
	options := Options{}

	paramDiffs := []ParameterDiff{
		{Key: "Param1", CurrentValue: "oldvalue1", ProposedValue: "value1", ChangeType: ChangeTypeModify},
	}
	tagDiffs := []TagDiff{
		{Key: "OldTag", CurrentValue: "remove", ProposedValue: "", ChangeType: ChangeTypeRemove},
		{Key: "Project", CurrentValue: "", ProposedValue: "test", ChangeType: ChangeTypeAdd},
	}

	// Set up expectations
	cfClient.On("StackExists", ctx, "test-stack").Return(true, nil)
	cfClient.On("DescribeStack", ctx, "test-stack").Return(currentStack, nil)
	cfClient.On("GetTemplate", ctx, "test-stack").Return(currentStack.Template, nil)

	templateComp.On("Compare", ctx, currentStack.Template, stack.TemplateBody).Return(&TemplateChange{
		HasChanges:   true,
		CurrentHash:  "hash1",
		ProposedHash: "hash2",
		Diff:         "Template has changes",
	}, nil)

	paramComp.On("Compare", currentStack.Parameters, stack.Parameters).Return(paramDiffs, nil)
	tagComp.On("Compare", currentStack.Tags, stack.Tags).Return(tagDiffs, nil)

	// Mock changeset creation
	changeSet := &aws.ChangeSetInfo{
		ChangeSetID: "test-changeset-id",
		Status:      "CREATE_COMPLETE",
		Changes: []aws.ResourceChange{
			{Action: "Modify", ResourceType: "AWS::S3::Bucket", LogicalID: "MyBucket"},
		},
	}
	cfClient.On("CreateChangeSetPreview", ctx, "test-stack", stack.TemplateBody, stack.Parameters).Return(changeSet, nil)

	// Execute
	result, err := differ.DiffStack(ctx, stack, options)

	// Verify
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.StackExists)
	assert.True(t, result.HasChanges())
	assert.Len(t, result.ParameterDiffs, 1)
	assert.Len(t, result.TagDiffs, 2)
	assert.NotNil(t, result.TemplateChange)
	assert.True(t, result.TemplateChange.HasChanges)
	assert.NotNil(t, result.ChangeSet)
	assert.Equal(t, "test-changeset-id", result.ChangeSet.ChangeSetID)

	cfClient.AssertExpectations(t)
	templateComp.AssertExpectations(t)
	paramComp.AssertExpectations(t)
	tagComp.AssertExpectations(t)
	cfClient.AssertExpectations(t)
}

func TestStackDiffer_DiffStack_NewStack(t *testing.T) {
	// Test diff of new stack (doesn't exist in AWS)
	ctx := context.Background()

	// Create mocks
	cfClient := &aws.MockCloudFormationOperations{}
	templateComp := &MockTemplateComparator{}
	paramComp := &MockParameterComparator{}
	tagComp := &MockTagComparator{}
	differ := createTestDiffer(cfClient, templateComp, paramComp, tagComp)

	// Test data
	stack := createTestResolvedStack()
	options := Options{}

	// Set up expectations
	cfClient.On("StackExists", ctx, "test-stack").Return(false, nil)

	// Execute
	result, err := differ.DiffStack(ctx, stack, options)

	// Verify
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.StackExists)
	assert.True(t, result.HasChanges())

	// All parameters and tags should be marked as ADD
	assert.Len(t, result.ParameterDiffs, 2)
	for _, diff := range result.ParameterDiffs {
		assert.Equal(t, ChangeTypeAdd, diff.ChangeType)
		assert.Equal(t, "", diff.CurrentValue)
	}

	assert.Len(t, result.TagDiffs, 2)
	for _, diff := range result.TagDiffs {
		assert.Equal(t, ChangeTypeAdd, diff.ChangeType)
		assert.Equal(t, "", diff.CurrentValue)
	}

	assert.NotNil(t, result.TemplateChange)
	assert.True(t, result.TemplateChange.HasChanges)

	cfClient.AssertExpectations(t)
}

func TestStackDiffer_DiffStack_StackExistsError(t *testing.T) {
	// Test error when checking if stack exists
	ctx := context.Background()

	// Create mocks
	cfClient := &aws.MockCloudFormationOperations{}
	templateComp := &MockTemplateComparator{}
	paramComp := &MockParameterComparator{}
	tagComp := &MockTagComparator{}
	differ := createTestDiffer(cfClient, templateComp, paramComp, tagComp)

	// Test data
	stack := createTestResolvedStack()
	options := Options{}

	// Set up expectations
	cfClient.On("StackExists", ctx, "test-stack").Return(false, errors.New("AWS connection error"))

	// Execute
	result, err := differ.DiffStack(ctx, stack, options)

	// Verify
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to check if stack exists")

	cfClient.AssertExpectations(t)
}

func TestStackDiffer_DiffStack_DescribeStackError(t *testing.T) {
	// Test error when describing existing stack
	ctx := context.Background()

	// Create mocks
	cfClient := &aws.MockCloudFormationOperations{}
	templateComp := &MockTemplateComparator{}
	paramComp := &MockParameterComparator{}
	tagComp := &MockTagComparator{}
	differ := createTestDiffer(cfClient, templateComp, paramComp, tagComp)

	// Test data
	stack := createTestResolvedStack()
	options := Options{}

	// Set up expectations
	cfClient.On("StackExists", ctx, "test-stack").Return(true, nil)
	cfClient.On("DescribeStack", ctx, "test-stack").Return((*aws.StackInfo)(nil), errors.New("access denied"))

	// Execute
	result, err := differ.DiffStack(ctx, stack, options)

	// Verify
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to describe stack")

	cfClient.AssertExpectations(t)
}

func TestStackDiffer_DiffStack_FilterOptions(t *testing.T) {
	// Test different filtering options
	tests := []struct {
		name                   string
		options                Options
		expectTemplateCompare  bool
		expectParameterCompare bool
		expectTagCompare       bool
	}{
		{
			name:                   "template only",
			options:                Options{TemplateOnly: true},
			expectTemplateCompare:  true,
			expectParameterCompare: false,
			expectTagCompare:       false,
		},
		{
			name:                   "parameters only",
			options:                Options{ParametersOnly: true},
			expectTemplateCompare:  false,
			expectParameterCompare: true,
			expectTagCompare:       false,
		},
		{
			name:                   "tags only",
			options:                Options{TagsOnly: true},
			expectTemplateCompare:  false,
			expectParameterCompare: false,
			expectTagCompare:       true,
		},
		{
			name:                   "all (default)",
			options:                Options{},
			expectTemplateCompare:  true,
			expectParameterCompare: true,
			expectTagCompare:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			// Create mocks
			cfClient := &aws.MockCloudFormationOperations{}
			templateComp := &MockTemplateComparator{}
			paramComp := &MockParameterComparator{}
			tagComp := &MockTagComparator{}
			differ := createTestDiffer(cfClient, templateComp, paramComp, tagComp)

			// Test data
			stack := createTestResolvedStack()
			currentStack := createTestStackInfo()

			// Set up expectations
			cfClient.On("StackExists", ctx, "test-stack").Return(true, nil)
			cfClient.On("DescribeStack", ctx, "test-stack").Return(currentStack, nil)

			if tt.expectTemplateCompare {
				cfClient.On("GetTemplate", ctx, "test-stack").Return(currentStack.Template, nil)
				templateComp.On("Compare", ctx, currentStack.Template, stack.TemplateBody).Return(&TemplateChange{HasChanges: false}, nil)
			}
			if tt.expectParameterCompare {
				paramComp.On("Compare", currentStack.Parameters, stack.Parameters).Return([]ParameterDiff{}, nil)
			}
			if tt.expectTagCompare {
				tagComp.On("Compare", currentStack.Tags, stack.Tags).Return([]TagDiff{}, nil)
			}

			// Execute
			result, err := differ.DiffStack(ctx, stack, tt.options)

			// Verify
			require.NoError(t, err)
			assert.NotNil(t, result)

			cfClient.AssertExpectations(t)
			templateComp.AssertExpectations(t)
			paramComp.AssertExpectations(t)
			tagComp.AssertExpectations(t)
		})
	}
}

func TestStackDiffer_DiffStack_ChangeSetError(t *testing.T) {
	// Test that changeset errors don't fail the entire diff
	ctx := context.Background()

	// Create mocks
	cfClient := &aws.MockCloudFormationOperations{}
	templateComp := &MockTemplateComparator{}
	paramComp := &MockParameterComparator{}
	tagComp := &MockTagComparator{}
	differ := createTestDiffer(cfClient, templateComp, paramComp, tagComp)

	// Test data
	stack := createTestResolvedStack()
	currentStack := createTestStackInfo()
	options := Options{}

	// Set up expectations
	cfClient.On("StackExists", ctx, "test-stack").Return(true, nil)
	cfClient.On("DescribeStack", ctx, "test-stack").Return(currentStack, nil)
	cfClient.On("GetTemplate", ctx, "test-stack").Return(currentStack.Template, nil)

	templateComp.On("Compare", ctx, currentStack.Template, stack.TemplateBody).Return(&TemplateChange{HasChanges: true}, nil)
	paramComp.On("Compare", currentStack.Parameters, stack.Parameters).Return([]ParameterDiff{{Key: "test", ChangeType: ChangeTypeAdd}}, nil)
	tagComp.On("Compare", currentStack.Tags, stack.Tags).Return([]TagDiff{}, nil)

	// Mock changeset creation failure
	cfClient.On("CreateChangeSetPreview", ctx, "test-stack", stack.TemplateBody, stack.Parameters).Return((*aws.ChangeSetInfo)(nil), errors.New("changeset failed"))

	// Execute
	result, err := differ.DiffStack(ctx, stack, options)

	// Verify - should succeed even though changeset failed
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.HasChanges())
	assert.Nil(t, result.ChangeSet) // Changeset should be nil due to error

	cfClient.AssertExpectations(t)
	templateComp.AssertExpectations(t)
	paramComp.AssertExpectations(t)
	tagComp.AssertExpectations(t)
	cfClient.AssertExpectations(t)
}

func TestStackDiffer_HandleNewStack(t *testing.T) {
	// Test the handleNewStack method directly
	ctx := context.Background()

	// Create differ (mocks not needed for this test)
	differ := &StackDiffer{}

	// Test data
	stack := createTestResolvedStack()
	result := &Result{
		StackName: stack.Name,
		Context:   stack.Context,
		Options:   Options{},
	}

	// Execute
	result, err := differ.handleNewStack(ctx, stack, result)

	// Verify
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.HasChanges())

	// Template should be marked as new
	assert.NotNil(t, result.TemplateChange)
	assert.True(t, result.TemplateChange.HasChanges)

	// All parameters should be ADD
	assert.Len(t, result.ParameterDiffs, 2)
	for _, diff := range result.ParameterDiffs {
		assert.Equal(t, ChangeTypeAdd, diff.ChangeType)
		assert.Equal(t, "", diff.CurrentValue)
		assert.Contains(t, []string{"Param1", "Param2"}, diff.Key)
	}

	// All tags should be ADD
	assert.Len(t, result.TagDiffs, 2)
	for _, diff := range result.TagDiffs {
		assert.Equal(t, ChangeTypeAdd, diff.ChangeType)
		assert.Equal(t, "", diff.CurrentValue)
		assert.Contains(t, []string{"Environment", "Project"}, diff.Key)
	}
}

func TestStackDiffer_CompareTemplates_Error(t *testing.T) {
	// Test template comparison error handling
	ctx := context.Background()

	// Create mocks
	cfClient := &aws.MockCloudFormationOperations{}
	templateComp := &MockTemplateComparator{}
	paramComp := &MockParameterComparator{}
	tagComp := &MockTagComparator{}

	differ := createTestDiffer(cfClient, templateComp, paramComp, tagComp)

	// Test data
	stack := createTestResolvedStack()
	currentStack := createTestStackInfo()

	// Set up expectation for GetTemplate call
	cfClient.On("GetTemplate", ctx, "test-stack").Return(currentStack.Template, nil)

	// Set up expectation for template comparison error
	templateComp.On("Compare", ctx, currentStack.Template, stack.TemplateBody).Return((*TemplateChange)(nil), errors.New("template parse error"))

	// Execute compareTemplates directly (this tests internal method)
	result, err := differ.compareTemplates(ctx, stack, currentStack)

	// Verify
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to compare templates")

	templateComp.AssertExpectations(t)
}
