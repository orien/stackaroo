/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package diff

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/orien/stackaroo/internal/aws"
	"github.com/orien/stackaroo/internal/model"
)

// Mock implementations for testing

type MockCloudFormationClient struct {
	mock.Mock
}

func (m *MockCloudFormationClient) StackExists(ctx context.Context, stackName string) (bool, error) {
	args := m.Called(ctx, stackName)
	return args.Bool(0), args.Error(1)
}

func (m *MockCloudFormationClient) DescribeStack(ctx context.Context, stackName string) (*aws.StackInfo, error) {
	args := m.Called(ctx, stackName)
	return args.Get(0).(*aws.StackInfo), args.Error(1)
}

func (m *MockCloudFormationClient) GetTemplate(ctx context.Context, stackName string) (string, error) {
	args := m.Called(ctx, stackName)
	return args.String(0), args.Error(1)
}

func (m *MockCloudFormationClient) DeployStack(ctx context.Context, input aws.DeployStackInput) error {
	args := m.Called(ctx, input)
	return args.Error(0)
}

func (m *MockCloudFormationClient) DeployStackWithCallback(ctx context.Context, input aws.DeployStackInput, eventCallback func(aws.StackEvent)) error {
	args := m.Called(ctx, input, eventCallback)
	return args.Error(0)
}

func (m *MockCloudFormationClient) UpdateStack(ctx context.Context, input aws.UpdateStackInput) error {
	args := m.Called(ctx, input)
	return args.Error(0)
}

func (m *MockCloudFormationClient) DeleteStack(ctx context.Context, input aws.DeleteStackInput) error {
	args := m.Called(ctx, input)
	return args.Error(0)
}

func (m *MockCloudFormationClient) GetStack(ctx context.Context, stackName string) (*aws.Stack, error) {
	args := m.Called(ctx, stackName)
	return args.Get(0).(*aws.Stack), args.Error(1)
}

func (m *MockCloudFormationClient) ListStacks(ctx context.Context) ([]*aws.Stack, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*aws.Stack), args.Error(1)
}

func (m *MockCloudFormationClient) ValidateTemplate(ctx context.Context, templateBody string) error {
	args := m.Called(ctx, templateBody)
	return args.Error(0)
}

func (m *MockCloudFormationClient) CreateChangeSet(ctx context.Context, params *cloudformation.CreateChangeSetInput, optFns ...func(*cloudformation.Options)) (*cloudformation.CreateChangeSetOutput, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(*cloudformation.CreateChangeSetOutput), args.Error(1)
}

func (m *MockCloudFormationClient) DeleteChangeSet(ctx context.Context, params *cloudformation.DeleteChangeSetInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DeleteChangeSetOutput, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(*cloudformation.DeleteChangeSetOutput), args.Error(1)
}

func (m *MockCloudFormationClient) DescribeChangeSet(ctx context.Context, params *cloudformation.DescribeChangeSetInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DescribeChangeSetOutput, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(*cloudformation.DescribeChangeSetOutput), args.Error(1)
}

func (m *MockCloudFormationClient) ExecuteChangeSet(ctx context.Context, params *cloudformation.ExecuteChangeSetInput, optFns ...func(*cloudformation.Options)) (*cloudformation.ExecuteChangeSetOutput, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(*cloudformation.ExecuteChangeSetOutput), args.Error(1)
}

func (m *MockCloudFormationClient) DescribeStackEvents(ctx context.Context, stackName string) ([]aws.StackEvent, error) {
	args := m.Called(ctx, stackName)
	return args.Get(0).([]aws.StackEvent), args.Error(1)
}

func (m *MockCloudFormationClient) WaitForStackOperation(ctx context.Context, stackName string, eventCallback func(aws.StackEvent)) error {
	args := m.Called(ctx, stackName, eventCallback)
	return args.Error(0)
}

type MockTemplateComparator struct {
	mock.Mock
}

func (m *MockTemplateComparator) Compare(ctx context.Context, currentTemplate, proposedTemplate string) (*TemplateChange, error) {
	args := m.Called(ctx, currentTemplate, proposedTemplate)
	return args.Get(0).(*TemplateChange), args.Error(1)
}

type MockParameterComparator struct {
	mock.Mock
}

func (m *MockParameterComparator) Compare(currentParams, proposedParams map[string]string) ([]ParameterDiff, error) {
	args := m.Called(currentParams, proposedParams)
	return args.Get(0).([]ParameterDiff), args.Error(1)
}

type MockTagComparator struct {
	mock.Mock
}

func (m *MockTagComparator) Compare(currentTags, proposedTags map[string]string) ([]TagDiff, error) {
	args := m.Called(currentTags, proposedTags)
	return args.Get(0).([]TagDiff), args.Error(1)
}

type MockChangeSetManager struct {
	mock.Mock
}

func (m *MockChangeSetManager) CreateChangeSet(ctx context.Context, stackName string, template string, parameters map[string]string) (*ChangeSetInfo, error) {
	args := m.Called(ctx, stackName, template, parameters)
	return args.Get(0).(*ChangeSetInfo), args.Error(1)
}

func (m *MockChangeSetManager) CreateChangeSetForDeployment(ctx context.Context, stackName string, template string, parameters map[string]string, capabilities []string, tags map[string]string) (*ChangeSetInfo, error) {
	args := m.Called(ctx, stackName, template, parameters, capabilities, tags)
	return args.Get(0).(*ChangeSetInfo), args.Error(1)
}

func (m *MockChangeSetManager) DeleteChangeSet(ctx context.Context, changeSetID string) error {
	args := m.Called(ctx, changeSetID)
	return args.Error(0)
}

// Test helper functions

func createTestDiffer(cfClient *MockCloudFormationClient, templateComp *MockTemplateComparator, paramComp *MockParameterComparator, tagComp *MockTagComparator, changeSetMgr *MockChangeSetManager) *DefaultDiffer {
	return &DefaultDiffer{
		cfClient:            cfClient,
		templateComparator:  templateComp,
		parameterComparator: paramComp,
		tagComparator:       tagComp,
		changeSetManager:    changeSetMgr,
	}
}

func createTestResolvedStack() *model.Stack {
	return &model.Stack{
		Name:         "test-stack",
		Environment:  "dev",
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

func TestNewDefaultDiffer_Success(t *testing.T) {
	// This test would require AWS credentials in a real environment
	// For now, we test that the function handles errors appropriately
	ctx := context.Background()

	differ, err := NewDefaultDiffer(ctx)

	// In environments without AWS credentials, this should fail
	if err != nil {
		assert.Nil(t, differ)
		assert.Contains(t, err.Error(), "failed to create CloudFormation client")
	} else {
		assert.NotNil(t, differ)
	}
}

func TestDefaultDiffer_DiffStack_ExistingStack_NoChanges(t *testing.T) {
	// Test diff of existing stack with no changes
	ctx := context.Background()

	// Create mocks
	cfClient := &MockCloudFormationClient{}
	templateComp := &MockTemplateComparator{}
	paramComp := &MockParameterComparator{}
	tagComp := &MockTagComparator{}
	changeSetMgr := &MockChangeSetManager{}

	differ := createTestDiffer(cfClient, templateComp, paramComp, tagComp, changeSetMgr)

	// Test data
	stack := createTestResolvedStack()
	currentStack := &aws.StackInfo{
		Name:       "test-stack",
		Parameters: map[string]string{"Param1": "value1", "Param2": "value2"},
		Tags:       map[string]string{"Environment": "dev", "Project": "test"},
		Template:   `{"AWSTemplateFormatVersion": "2010-09-09"}`,
	}
	options := Options{Format: "text"}

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
	assert.NotNil(t, result)
	assert.Equal(t, "test-stack", result.StackName)
	assert.Equal(t, "dev", result.Environment)
	assert.True(t, result.StackExists)
	assert.False(t, result.HasChanges())
	assert.Empty(t, result.ParameterDiffs)
	assert.Empty(t, result.TagDiffs)

	cfClient.AssertExpectations(t)
	templateComp.AssertExpectations(t)
	paramComp.AssertExpectations(t)
	tagComp.AssertExpectations(t)
}

func TestDefaultDiffer_DiffStack_ExistingStack_WithChanges(t *testing.T) {
	// Test diff of existing stack with changes
	ctx := context.Background()

	// Create mocks
	cfClient := &MockCloudFormationClient{}
	templateComp := &MockTemplateComparator{}
	paramComp := &MockParameterComparator{}
	tagComp := &MockTagComparator{}
	changeSetMgr := &MockChangeSetManager{}

	differ := createTestDiffer(cfClient, templateComp, paramComp, tagComp, changeSetMgr)

	// Test data
	stack := createTestResolvedStack()
	currentStack := createTestStackInfo()
	options := Options{Format: "text"}

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
	changeSet := &ChangeSetInfo{
		ChangeSetID: "test-changeset-id",
		Status:      "CREATE_COMPLETE",
		Changes: []ResourceChange{
			{Action: "Modify", ResourceType: "AWS::S3::Bucket", LogicalID: "MyBucket"},
		},
	}
	changeSetMgr.On("CreateChangeSet", ctx, "test-stack", stack.TemplateBody, stack.Parameters).Return(changeSet, nil)

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
	changeSetMgr.AssertExpectations(t)
}

func TestDefaultDiffer_DiffStack_NewStack(t *testing.T) {
	// Test diff of new stack (doesn't exist in AWS)
	ctx := context.Background()

	// Create mocks
	cfClient := &MockCloudFormationClient{}
	templateComp := &MockTemplateComparator{}
	paramComp := &MockParameterComparator{}
	tagComp := &MockTagComparator{}
	changeSetMgr := &MockChangeSetManager{}

	differ := createTestDiffer(cfClient, templateComp, paramComp, tagComp, changeSetMgr)

	// Test data
	stack := createTestResolvedStack()
	options := Options{Format: "text"}

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

func TestDefaultDiffer_DiffStack_StackExistsError(t *testing.T) {
	// Test error when checking if stack exists
	ctx := context.Background()

	// Create mocks
	cfClient := &MockCloudFormationClient{}
	templateComp := &MockTemplateComparator{}
	paramComp := &MockParameterComparator{}
	tagComp := &MockTagComparator{}
	changeSetMgr := &MockChangeSetManager{}

	differ := createTestDiffer(cfClient, templateComp, paramComp, tagComp, changeSetMgr)

	// Test data
	stack := createTestResolvedStack()
	options := Options{Format: "text"}

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

func TestDefaultDiffer_DiffStack_DescribeStackError(t *testing.T) {
	// Test error when describing existing stack
	ctx := context.Background()

	// Create mocks
	cfClient := &MockCloudFormationClient{}
	templateComp := &MockTemplateComparator{}
	paramComp := &MockParameterComparator{}
	tagComp := &MockTagComparator{}
	changeSetMgr := &MockChangeSetManager{}

	differ := createTestDiffer(cfClient, templateComp, paramComp, tagComp, changeSetMgr)

	// Test data
	stack := createTestResolvedStack()
	options := Options{Format: "text"}

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

func TestDefaultDiffer_DiffStack_FilterOptions(t *testing.T) {
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
			cfClient := &MockCloudFormationClient{}
			templateComp := &MockTemplateComparator{}
			paramComp := &MockParameterComparator{}
			tagComp := &MockTagComparator{}
			changeSetMgr := &MockChangeSetManager{}

			differ := createTestDiffer(cfClient, templateComp, paramComp, tagComp, changeSetMgr)

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

func TestDefaultDiffer_DiffStack_ChangeSetError(t *testing.T) {
	// Test that changeset errors don't fail the entire diff
	ctx := context.Background()

	// Create mocks
	cfClient := &MockCloudFormationClient{}
	templateComp := &MockTemplateComparator{}
	paramComp := &MockParameterComparator{}
	tagComp := &MockTagComparator{}
	changeSetMgr := &MockChangeSetManager{}

	differ := createTestDiffer(cfClient, templateComp, paramComp, tagComp, changeSetMgr)

	// Test data
	stack := createTestResolvedStack()
	currentStack := createTestStackInfo()
	options := Options{Format: "text"}

	// Set up expectations
	cfClient.On("StackExists", ctx, "test-stack").Return(true, nil)
	cfClient.On("DescribeStack", ctx, "test-stack").Return(currentStack, nil)
	cfClient.On("GetTemplate", ctx, "test-stack").Return(currentStack.Template, nil)

	templateComp.On("Compare", ctx, currentStack.Template, stack.TemplateBody).Return(&TemplateChange{HasChanges: true}, nil)
	paramComp.On("Compare", currentStack.Parameters, stack.Parameters).Return([]ParameterDiff{{Key: "test", ChangeType: ChangeTypeAdd}}, nil)
	tagComp.On("Compare", currentStack.Tags, stack.Tags).Return([]TagDiff{}, nil)

	// Mock changeset creation failure
	changeSetMgr.On("CreateChangeSet", ctx, "test-stack", stack.TemplateBody, stack.Parameters).Return((*ChangeSetInfo)(nil), errors.New("changeset failed"))

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
	changeSetMgr.AssertExpectations(t)
}

func TestDefaultDiffer_HandleNewStack(t *testing.T) {
	// Test the handleNewStack method directly
	ctx := context.Background()

	// Create differ (mocks not needed for this test)
	differ := &DefaultDiffer{}

	// Test data
	stack := createTestResolvedStack()
	result := &Result{
		StackName:   stack.Name,
		Environment: stack.Environment,
		Options:     Options{Format: "text"},
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

func TestDefaultDiffer_CompareTemplates_Error(t *testing.T) {
	// Test template comparison error handling
	ctx := context.Background()

	// Create mocks
	cfClient := &MockCloudFormationClient{}
	templateComp := &MockTemplateComparator{}
	paramComp := &MockParameterComparator{}
	tagComp := &MockTagComparator{}
	changeSetMgr := &MockChangeSetManager{}

	differ := createTestDiffer(cfClient, templateComp, paramComp, tagComp, changeSetMgr)

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
