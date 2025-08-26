/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package diff

import (
	"context"
	"errors"
	"testing"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/orien/stackaroo/internal/aws"
)

// Mock CloudFormation operations for changeset testing
type MockChangeSetClient struct {
	mock.Mock
}

func (m *MockChangeSetClient) CreateChangeSet(ctx context.Context, params *cloudformation.CreateChangeSetInput, optFns ...func(*cloudformation.Options)) (*cloudformation.CreateChangeSetOutput, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(*cloudformation.CreateChangeSetOutput), args.Error(1)
}

func (m *MockChangeSetClient) DeleteChangeSet(ctx context.Context, params *cloudformation.DeleteChangeSetInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DeleteChangeSetOutput, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(*cloudformation.DeleteChangeSetOutput), args.Error(1)
}

func (m *MockChangeSetClient) DescribeChangeSet(ctx context.Context, params *cloudformation.DescribeChangeSetInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DescribeChangeSetOutput, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(*cloudformation.DescribeChangeSetOutput), args.Error(1)
}

func (m *MockChangeSetClient) ExecuteChangeSet(ctx context.Context, params *cloudformation.ExecuteChangeSetInput, optFns ...func(*cloudformation.Options)) (*cloudformation.ExecuteChangeSetOutput, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(*cloudformation.ExecuteChangeSetOutput), args.Error(1)
}

// Additional methods to satisfy CloudFormationOperations

func (m *MockChangeSetClient) DeployStack(ctx context.Context, input aws.DeployStackInput) error {
	args := m.Called(ctx, input)
	return args.Error(0)
}

func (m *MockChangeSetClient) DeployStackWithCallback(ctx context.Context, input aws.DeployStackInput, eventCallback func(aws.StackEvent)) error {
	args := m.Called(ctx, input, eventCallback)
	return args.Error(0)
}

func (m *MockChangeSetClient) UpdateStack(ctx context.Context, input aws.UpdateStackInput) error {
	args := m.Called(ctx, input)
	return args.Error(0)
}

func (m *MockChangeSetClient) DeleteStack(ctx context.Context, input aws.DeleteStackInput) error {
	args := m.Called(ctx, input)
	return args.Error(0)
}

func (m *MockChangeSetClient) GetStack(ctx context.Context, stackName string) (*aws.Stack, error) {
	args := m.Called(ctx, stackName)
	return args.Get(0).(*aws.Stack), args.Error(1)
}

func (m *MockChangeSetClient) ListStacks(ctx context.Context) ([]*aws.Stack, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*aws.Stack), args.Error(1)
}

func (m *MockChangeSetClient) ValidateTemplate(ctx context.Context, templateBody string) error {
	args := m.Called(ctx, templateBody)
	return args.Error(0)
}

func (m *MockChangeSetClient) StackExists(ctx context.Context, stackName string) (bool, error) {
	args := m.Called(ctx, stackName)
	return args.Bool(0), args.Error(1)
}

func (m *MockChangeSetClient) GetTemplate(ctx context.Context, stackName string) (string, error) {
	args := m.Called(ctx, stackName)
	return args.String(0), args.Error(1)
}

func (m *MockChangeSetClient) DescribeStack(ctx context.Context, stackName string) (*aws.StackInfo, error) {
	args := m.Called(ctx, stackName)
	return args.Get(0).(*aws.StackInfo), args.Error(1)
}

func (m *MockChangeSetClient) DescribeStackEvents(ctx context.Context, stackName string) ([]aws.StackEvent, error) {
	args := m.Called(ctx, stackName)
	return args.Get(0).([]aws.StackEvent), args.Error(1)
}

func (m *MockChangeSetClient) WaitForStackOperation(ctx context.Context, stackName string, eventCallback func(aws.StackEvent)) error {
	args := m.Called(ctx, stackName, eventCallback)
	return args.Error(0)
}

// Helper functions for creating test data

func createTestChangeSetOutput(changeSetId string) *cloudformation.CreateChangeSetOutput {
	return &cloudformation.CreateChangeSetOutput{
		Id: awssdk.String(changeSetId),
	}
}

func createTestDescribeChangeSetOutput(changeSetId string, status types.ChangeSetStatus) *cloudformation.DescribeChangeSetOutput {
	return &cloudformation.DescribeChangeSetOutput{
		ChangeSetId: awssdk.String(changeSetId),
		Status:      status,
		Changes: []types.Change{
			{
				Type: types.ChangeTypeResource,
				ResourceChange: &types.ResourceChange{
					Action:             types.ChangeActionAdd,
					LogicalResourceId:  awssdk.String("MyBucket"),
					PhysicalResourceId: awssdk.String("my-bucket-12345"),
					ResourceType:       awssdk.String("AWS::S3::Bucket"),
					Replacement:        types.ReplacementFalse,
					Details: []types.ResourceChangeDetail{
						{
							Target: &types.ResourceTargetDefinition{
								Attribute: types.ResourceAttributeProperties,
								Name:      awssdk.String("BucketName"),
							},
						},
					},
				},
			},
		},
	}
}

// Tests

func TestNewChangeSetManager(t *testing.T) {
	mockClient := &MockChangeSetClient{}
	manager := NewChangeSetManager(mockClient)

	assert.NotNil(t, manager)

	// Test that the manager can be cast to DefaultChangeSetManager
	defaultManager, ok := manager.(*DefaultChangeSetManager)
	assert.True(t, ok)
	assert.Equal(t, mockClient, defaultManager.cfClient)
}

func TestDefaultChangeSetManager_CreateChangeSet_Success(t *testing.T) {
	ctx := context.Background()
	mockClient := &MockChangeSetClient{}
	manager := NewChangeSetManager(mockClient)

	// Test data
	stackName := "test-stack"
	template := `{"AWSTemplateFormatVersion": "2010-09-09"}`
	parameters := map[string]string{"Param1": "value1"}
	changeSetId := "test-changeset-123"

	// Mock CreateChangeSet
	mockClient.On("CreateChangeSet", ctx, mock.MatchedBy(func(input *cloudformation.CreateChangeSetInput) bool {
		return awssdk.ToString(input.StackName) == stackName &&
			awssdk.ToString(input.TemplateBody) == template &&
			len(input.Parameters) == 1 &&
			awssdk.ToString(input.Parameters[0].ParameterKey) == "Param1" &&
			awssdk.ToString(input.Parameters[0].ParameterValue) == "value1"
	})).Return(createTestChangeSetOutput(changeSetId), nil)

	// Mock DescribeChangeSet for waiting
	mockClient.On("DescribeChangeSet", ctx, mock.MatchedBy(func(input *cloudformation.DescribeChangeSetInput) bool {
		return awssdk.ToString(input.ChangeSetName) == changeSetId
	})).Return(createTestDescribeChangeSetOutput(changeSetId, types.ChangeSetStatusCreateComplete), nil)

	// Mock DeleteChangeSet for cleanup
	mockClient.On("DeleteChangeSet", ctx, mock.MatchedBy(func(input *cloudformation.DeleteChangeSetInput) bool {
		return awssdk.ToString(input.ChangeSetName) == changeSetId
	})).Return(&cloudformation.DeleteChangeSetOutput{}, nil)

	// Execute
	result, err := manager.CreateChangeSet(ctx, stackName, template, parameters)

	// Verify
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, changeSetId, result.ChangeSetID)
	assert.Equal(t, "CREATE_COMPLETE", result.Status)
	assert.Len(t, result.Changes, 1)

	change := result.Changes[0]
	assert.Equal(t, "Add", change.Action)
	assert.Equal(t, "AWS::S3::Bucket", change.ResourceType)
	assert.Equal(t, "MyBucket", change.LogicalID)
	assert.Equal(t, "my-bucket-12345", change.PhysicalID)
	assert.Equal(t, "False", change.Replacement)
	assert.Len(t, change.Details, 1)
	assert.Equal(t, "Property: BucketName (Properties)", change.Details[0])

	mockClient.AssertExpectations(t)
}

func TestDefaultChangeSetManager_CreateChangeSet_CreateError(t *testing.T) {
	ctx := context.Background()
	mockClient := &MockChangeSetClient{}
	manager := NewChangeSetManager(mockClient)

	// Test data
	stackName := "test-stack"
	template := `{"AWSTemplateFormatVersion": "2010-09-09"}`
	parameters := map[string]string{}

	// Mock CreateChangeSet failure
	mockClient.On("CreateChangeSet", ctx, mock.AnythingOfType("*cloudformation.CreateChangeSetInput")).Return((*cloudformation.CreateChangeSetOutput)(nil), errors.New("access denied"))

	// Execute
	result, err := manager.CreateChangeSet(ctx, stackName, template, parameters)

	// Verify
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to create changeset")

	mockClient.AssertExpectations(t)
}

func TestDefaultChangeSetManager_CreateChangeSet_WaitError(t *testing.T) {
	ctx := context.Background()
	mockClient := &MockChangeSetClient{}
	manager := NewChangeSetManager(mockClient)

	// Test data
	stackName := "test-stack"
	template := `{"AWSTemplateFormatVersion": "2010-09-09"}`
	parameters := map[string]string{}
	changeSetId := "test-changeset-123"

	// Mock CreateChangeSet success
	mockClient.On("CreateChangeSet", ctx, mock.AnythingOfType("*cloudformation.CreateChangeSetInput")).Return(createTestChangeSetOutput(changeSetId), nil)

	// Mock DescribeChangeSet for waiting - return failure
	mockClient.On("DescribeChangeSet", ctx, mock.AnythingOfType("*cloudformation.DescribeChangeSetInput")).Return(&cloudformation.DescribeChangeSetOutput{
		Status:       types.ChangeSetStatusFailed,
		StatusReason: awssdk.String("Template validation error"),
	}, nil)

	// Mock DeleteChangeSet for cleanup
	mockClient.On("DeleteChangeSet", ctx, mock.AnythingOfType("*cloudformation.DeleteChangeSetInput")).Return(&cloudformation.DeleteChangeSetOutput{}, nil)

	// Execute
	result, err := manager.CreateChangeSet(ctx, stackName, template, parameters)

	// Verify
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "changeset creation failed")
	assert.Contains(t, err.Error(), "Template validation error")

	mockClient.AssertExpectations(t)
}

func TestDefaultChangeSetManager_CreateChangeSet_DescribeError(t *testing.T) {
	ctx := context.Background()
	mockClient := &MockChangeSetClient{}
	manager := NewChangeSetManager(mockClient)

	// Test data
	stackName := "test-stack"
	template := `{"AWSTemplateFormatVersion": "2010-09-09"}`
	parameters := map[string]string{}
	changeSetId := "test-changeset-123"

	// Mock CreateChangeSet success
	mockClient.On("CreateChangeSet", ctx, mock.AnythingOfType("*cloudformation.CreateChangeSetInput")).Return(createTestChangeSetOutput(changeSetId), nil)

	// Mock DescribeChangeSet for waiting - return success
	mockClient.On("DescribeChangeSet", ctx, mock.AnythingOfType("*cloudformation.DescribeChangeSetInput")).Return(createTestDescribeChangeSetOutput(changeSetId, types.ChangeSetStatusCreateComplete), nil).Once()

	// Mock DescribeChangeSet for final description - return error
	mockClient.On("DescribeChangeSet", ctx, mock.AnythingOfType("*cloudformation.DescribeChangeSetInput")).Return((*cloudformation.DescribeChangeSetOutput)(nil), errors.New("network error")).Once()

	// Mock DeleteChangeSet for cleanup
	mockClient.On("DeleteChangeSet", ctx, mock.AnythingOfType("*cloudformation.DeleteChangeSetInput")).Return(&cloudformation.DeleteChangeSetOutput{}, nil)

	// Execute
	result, err := manager.CreateChangeSet(ctx, stackName, template, parameters)

	// Verify
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to describe changeset")

	mockClient.AssertExpectations(t)
}

func TestDefaultChangeSetManager_DeleteChangeSet_Success(t *testing.T) {
	ctx := context.Background()
	mockClient := &MockChangeSetClient{}
	manager := NewChangeSetManager(mockClient)

	changeSetId := "test-changeset-123"

	// Mock DeleteChangeSet
	mockClient.On("DeleteChangeSet", ctx, mock.MatchedBy(func(input *cloudformation.DeleteChangeSetInput) bool {
		return awssdk.ToString(input.ChangeSetName) == changeSetId
	})).Return(&cloudformation.DeleteChangeSetOutput{}, nil)

	// Execute
	err := manager.DeleteChangeSet(ctx, changeSetId)

	// Verify
	assert.NoError(t, err)
	mockClient.AssertExpectations(t)
}

func TestDefaultChangeSetManager_DeleteChangeSet_Error(t *testing.T) {
	ctx := context.Background()
	mockClient := &MockChangeSetClient{}
	manager := NewChangeSetManager(mockClient)

	changeSetId := "test-changeset-123"

	// Mock DeleteChangeSet failure
	mockClient.On("DeleteChangeSet", ctx, mock.AnythingOfType("*cloudformation.DeleteChangeSetInput")).Return((*cloudformation.DeleteChangeSetOutput)(nil), errors.New("changeset not found"))

	// Execute
	err := manager.DeleteChangeSet(ctx, changeSetId)

	// Verify
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to delete changeset")

	mockClient.AssertExpectations(t)
}

func TestDefaultChangeSetManager_WaitForChangeSet_Pending(t *testing.T) {
	ctx := context.Background()
	mockClient := &MockChangeSetClient{}
	manager := &DefaultChangeSetManager{cfClient: mockClient}

	changeSetId := "test-changeset-123"

	// Mock DescribeChangeSet - first pending, then complete
	mockClient.On("DescribeChangeSet", ctx, mock.AnythingOfType("*cloudformation.DescribeChangeSetInput")).Return(&cloudformation.DescribeChangeSetOutput{
		Status: types.ChangeSetStatusCreatePending,
	}, nil).Once()

	mockClient.On("DescribeChangeSet", ctx, mock.AnythingOfType("*cloudformation.DescribeChangeSetInput")).Return(&cloudformation.DescribeChangeSetOutput{
		Status: types.ChangeSetStatusCreateComplete,
	}, nil).Once()

	// Execute
	err := manager.waitForChangeSet(ctx, changeSetId)

	// Verify
	assert.NoError(t, err)
	mockClient.AssertExpectations(t)
}

func TestDefaultChangeSetManager_WaitForChangeSet_DescribeError(t *testing.T) {
	ctx := context.Background()
	mockClient := &MockChangeSetClient{}
	manager := &DefaultChangeSetManager{cfClient: mockClient}

	changeSetId := "test-changeset-123"

	// Mock DescribeChangeSet failure
	mockClient.On("DescribeChangeSet", ctx, mock.AnythingOfType("*cloudformation.DescribeChangeSetInput")).Return((*cloudformation.DescribeChangeSetOutput)(nil), errors.New("API error"))

	// Execute
	err := manager.waitForChangeSet(ctx, changeSetId)

	// Verify
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to describe changeset while waiting")

	mockClient.AssertExpectations(t)
}

func TestDefaultChangeSetManager_WaitForChangeSet_UnexpectedStatus(t *testing.T) {
	ctx := context.Background()
	mockClient := &MockChangeSetClient{}
	manager := &DefaultChangeSetManager{cfClient: mockClient}

	changeSetId := "test-changeset-123"

	// Mock DescribeChangeSet with unexpected status
	mockClient.On("DescribeChangeSet", ctx, mock.AnythingOfType("*cloudformation.DescribeChangeSetInput")).Return(&cloudformation.DescribeChangeSetOutput{
		Status: "UNKNOWN_STATUS",
	}, nil)

	// Execute
	err := manager.waitForChangeSet(ctx, changeSetId)

	// Verify
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected changeset status")

	mockClient.AssertExpectations(t)
}

func TestDefaultChangeSetManager_WaitForChangeSet_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	mockClient := &MockChangeSetClient{}
	manager := &DefaultChangeSetManager{cfClient: mockClient}

	changeSetId := "test-changeset-123"

	// Cancel context immediately
	cancel()

	// Execute
	err := manager.waitForChangeSet(ctx, changeSetId)

	// Verify
	assert.Error(t, err)
	assert.Equal(t, context.Canceled, err)
}

func TestDefaultChangeSetManager_DescribeChangeSet_Success(t *testing.T) {
	ctx := context.Background()
	mockClient := &MockChangeSetClient{}
	manager := &DefaultChangeSetManager{cfClient: mockClient}

	changeSetId := "test-changeset-123"

	// Mock DescribeChangeSet
	mockClient.On("DescribeChangeSet", ctx, mock.MatchedBy(func(input *cloudformation.DescribeChangeSetInput) bool {
		return awssdk.ToString(input.ChangeSetName) == changeSetId
	})).Return(createTestDescribeChangeSetOutput(changeSetId, types.ChangeSetStatusCreateComplete), nil)

	// Execute
	result, err := manager.describeChangeSet(ctx, changeSetId)

	// Verify
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, changeSetId, result.ChangeSetID)
	assert.Equal(t, "CREATE_COMPLETE", result.Status)
	assert.Len(t, result.Changes, 1)

	mockClient.AssertExpectations(t)
}

func TestDefaultChangeSetManager_DescribeChangeSet_Error(t *testing.T) {
	ctx := context.Background()
	mockClient := &MockChangeSetClient{}
	manager := &DefaultChangeSetManager{cfClient: mockClient}

	changeSetId := "test-changeset-123"

	// Mock DescribeChangeSet failure
	mockClient.On("DescribeChangeSet", ctx, mock.AnythingOfType("*cloudformation.DescribeChangeSetInput")).Return((*cloudformation.DescribeChangeSetOutput)(nil), errors.New("not found"))

	// Execute
	result, err := manager.describeChangeSet(ctx, changeSetId)

	// Verify
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to describe changeset")

	mockClient.AssertExpectations(t)
}

func TestDefaultChangeSetManager_DescribeChangeSet_ComplexChanges(t *testing.T) {
	ctx := context.Background()
	mockClient := &MockChangeSetClient{}
	manager := &DefaultChangeSetManager{cfClient: mockClient}

	changeSetId := "test-changeset-123"

	// Create complex changeset output with multiple changes
	describeOutput := &cloudformation.DescribeChangeSetOutput{
		ChangeSetId: awssdk.String(changeSetId),
		Status:      types.ChangeSetStatusCreateComplete,
		Changes: []types.Change{
			{
				Type: types.ChangeTypeResource,
				ResourceChange: &types.ResourceChange{
					Action:            types.ChangeActionAdd,
					LogicalResourceId: awssdk.String("NewBucket"),
					ResourceType:      awssdk.String("AWS::S3::Bucket"),
					Replacement:       types.ReplacementFalse,
				},
			},
			{
				Type: types.ChangeTypeResource,
				ResourceChange: &types.ResourceChange{
					Action:             types.ChangeActionModify,
					LogicalResourceId:  awssdk.String("ExistingQueue"),
					PhysicalResourceId: awssdk.String("existing-queue-123"),
					ResourceType:       awssdk.String("AWS::SQS::Queue"),
					Replacement:        types.ReplacementTrue,
					Details: []types.ResourceChangeDetail{
						{
							Target: &types.ResourceTargetDefinition{
								Attribute: types.ResourceAttributeProperties,
								Name:      awssdk.String("QueueName"),
							},
						},
						{
							Target: &types.ResourceTargetDefinition{
								Attribute: types.ResourceAttributeMetadata,
								Name:      awssdk.String("Description"),
							},
						},
					},
				},
			},
			{
				Type: types.ChangeTypeResource,
				ResourceChange: &types.ResourceChange{
					Action:             types.ChangeActionRemove,
					LogicalResourceId:  awssdk.String("OldTopic"),
					PhysicalResourceId: awssdk.String("old-topic-456"),
					ResourceType:       awssdk.String("AWS::SNS::Topic"),
					Replacement:        types.ReplacementFalse,
				},
			},
		},
	}

	mockClient.On("DescribeChangeSet", ctx, mock.AnythingOfType("*cloudformation.DescribeChangeSetInput")).Return(describeOutput, nil)

	// Execute
	result, err := manager.describeChangeSet(ctx, changeSetId)

	// Verify
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.Changes, 3)

	// Verify ADD change
	addChange := result.Changes[0]
	assert.Equal(t, "Add", addChange.Action)
	assert.Equal(t, "NewBucket", addChange.LogicalID)
	assert.Equal(t, "AWS::S3::Bucket", addChange.ResourceType)
	assert.Equal(t, "", addChange.PhysicalID)
	assert.Equal(t, "False", addChange.Replacement)

	// Verify MODIFY change
	modifyChange := result.Changes[1]
	assert.Equal(t, "Modify", modifyChange.Action)
	assert.Equal(t, "ExistingQueue", modifyChange.LogicalID)
	assert.Equal(t, "AWS::SQS::Queue", modifyChange.ResourceType)
	assert.Equal(t, "existing-queue-123", modifyChange.PhysicalID)
	assert.Equal(t, "True", modifyChange.Replacement)
	assert.Len(t, modifyChange.Details, 2)
	assert.Equal(t, "Property: QueueName (Properties)", modifyChange.Details[0])
	assert.Equal(t, "Property: Description (Metadata)", modifyChange.Details[1])

	// Verify REMOVE change
	removeChange := result.Changes[2]
	assert.Equal(t, "Remove", removeChange.Action)
	assert.Equal(t, "OldTopic", removeChange.LogicalID)
	assert.Equal(t, "AWS::SNS::Topic", removeChange.ResourceType)
	assert.Equal(t, "old-topic-456", removeChange.PhysicalID)
	assert.Equal(t, "False", removeChange.Replacement)

	mockClient.AssertExpectations(t)
}

func TestDefaultChangeSetManager_CreateChangeSet_WithEmptyParameters(t *testing.T) {
	ctx := context.Background()
	mockClient := &MockChangeSetClient{}
	manager := NewChangeSetManager(mockClient)

	// Test data
	stackName := "test-stack"
	template := `{"AWSTemplateFormatVersion": "2010-09-09"}`
	parameters := map[string]string{} // Empty parameters
	changeSetId := "test-changeset-123"

	// Mock CreateChangeSet
	mockClient.On("CreateChangeSet", ctx, mock.MatchedBy(func(input *cloudformation.CreateChangeSetInput) bool {
		return awssdk.ToString(input.StackName) == stackName &&
			awssdk.ToString(input.TemplateBody) == template &&
			len(input.Parameters) == 0 // Should have no parameters
	})).Return(createTestChangeSetOutput(changeSetId), nil)

	// Mock other calls
	mockClient.On("DescribeChangeSet", ctx, mock.AnythingOfType("*cloudformation.DescribeChangeSetInput")).Return(createTestDescribeChangeSetOutput(changeSetId, types.ChangeSetStatusCreateComplete), nil)
	mockClient.On("DeleteChangeSet", ctx, mock.AnythingOfType("*cloudformation.DeleteChangeSetInput")).Return(&cloudformation.DeleteChangeSetOutput{}, nil)

	// Execute
	result, err := manager.CreateChangeSet(ctx, stackName, template, parameters)

	// Verify
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, changeSetId, result.ChangeSetID)

	mockClient.AssertExpectations(t)
}

func TestDefaultChangeSetManager_CreateChangeSet_ChangeSetNameFormat(t *testing.T) {
	// Test that changeset names have the correct format
	ctx := context.Background()
	mockClient := &MockChangeSetClient{}
	manager := NewChangeSetManager(mockClient)

	stackName := "test-stack"
	template := `{"AWSTemplateFormatVersion": "2010-09-09"}`
	parameters := map[string]string{}

	var changeSetName string

	// Capture changeset name from the call
	mockClient.On("CreateChangeSet", ctx, mock.AnythingOfType("*cloudformation.CreateChangeSetInput")).Return(createTestChangeSetOutput("changeset-123"), nil).Run(func(args mock.Arguments) {
		input := args.Get(1).(*cloudformation.CreateChangeSetInput)
		changeSetName = awssdk.ToString(input.ChangeSetName)
	}).Once()

	// Mock other calls to avoid errors
	mockClient.On("DescribeChangeSet", ctx, mock.AnythingOfType("*cloudformation.DescribeChangeSetInput")).Return(&cloudformation.DescribeChangeSetOutput{
		Status: types.ChangeSetStatusCreateComplete,
	}, nil).Times(2) // wait + final describe calls
	mockClient.On("DeleteChangeSet", ctx, mock.AnythingOfType("*cloudformation.DeleteChangeSetInput")).Return(&cloudformation.DeleteChangeSetOutput{}, nil).Once()

	// Execute
	_, err := manager.CreateChangeSet(ctx, stackName, template, parameters)
	require.NoError(t, err)

	// Verify changeset name format
	assert.Contains(t, changeSetName, "stackaroo-diff-")
	assert.True(t, len(changeSetName) > len("stackaroo-diff-"), "Changeset name should include timestamp")

	mockClient.AssertExpectations(t)
}

func TestDefaultChangeSetManager_CreateChangeSetForDeployment_ExistingStack(t *testing.T) {
	ctx := context.Background()
	mockClient := &MockChangeSetClient{}
	manager := NewChangeSetManager(mockClient)

	// Test data
	stackName := "test-stack"
	template := `{"AWSTemplateFormatVersion": "2010-09-09"}`
	parameters := map[string]string{"Param1": "value1"}
	capabilities := []string{"CAPABILITY_IAM"}
	tags := map[string]string{"Environment": "test"}
	changeSetId := "test-changeset-123"

	// Mock StackExists - stack exists, so should be UPDATE changeset
	mockClient.On("StackExists", ctx, stackName).Return(true, nil)

	// Mock CreateChangeSet
	var changeSetName string
	mockClient.On("CreateChangeSet", ctx, mock.MatchedBy(func(input *cloudformation.CreateChangeSetInput) bool {
		changeSetName = awssdk.ToString(input.ChangeSetName)
		return awssdk.ToString(input.StackName) == stackName &&
			awssdk.ToString(input.TemplateBody) == template &&
			input.ChangeSetType == types.ChangeSetTypeUpdate &&
			len(input.Parameters) == 1 &&
			len(input.Tags) == 1 &&
			len(input.Capabilities) == 1
	})).Return(&cloudformation.CreateChangeSetOutput{
		Id: awssdk.String(changeSetId),
	}, nil)

	// Mock wait for changeset
	mockClient.On("DescribeChangeSet", ctx, mock.AnythingOfType("*cloudformation.DescribeChangeSetInput")).Return(&cloudformation.DescribeChangeSetOutput{
		Status: types.ChangeSetStatusCreateComplete,
	}, nil).Once()

	// Mock final describe
	mockClient.On("DescribeChangeSet", ctx, mock.MatchedBy(func(input *cloudformation.DescribeChangeSetInput) bool {
		return awssdk.ToString(input.ChangeSetName) == changeSetId
	})).Return(createTestDescribeChangeSetOutput(changeSetId, types.ChangeSetStatusCreateComplete), nil)

	// Execute
	result, err := manager.CreateChangeSetForDeployment(ctx, stackName, template, parameters, capabilities, tags)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, changeSetId, result.ChangeSetID)
	assert.Contains(t, changeSetName, "stackaroo-deploy-")

	mockClient.AssertExpectations(t)
}

func TestDefaultChangeSetManager_CreateChangeSetForDeployment_NewStack(t *testing.T) {
	ctx := context.Background()
	mockClient := &MockChangeSetClient{}
	manager := NewChangeSetManager(mockClient)

	// Test data
	stackName := "new-stack"
	template := `{"AWSTemplateFormatVersion": "2010-09-09"}`
	parameters := map[string]string{}
	capabilities := []string{"CAPABILITY_IAM"}
	tags := map[string]string{}
	changeSetId := "new-changeset-123"

	// Mock StackExists - stack doesn't exist, so should be CREATE changeset
	mockClient.On("StackExists", ctx, stackName).Return(false, nil)

	// Mock CreateChangeSet
	mockClient.On("CreateChangeSet", ctx, mock.MatchedBy(func(input *cloudformation.CreateChangeSetInput) bool {
		return awssdk.ToString(input.StackName) == stackName &&
			input.ChangeSetType == types.ChangeSetTypeCreate
	})).Return(&cloudformation.CreateChangeSetOutput{
		Id: awssdk.String(changeSetId),
	}, nil)

	// Mock wait and describe
	mockClient.On("DescribeChangeSet", ctx, mock.AnythingOfType("*cloudformation.DescribeChangeSetInput")).Return(&cloudformation.DescribeChangeSetOutput{
		Status: types.ChangeSetStatusCreateComplete,
	}, nil).Once()

	mockClient.On("DescribeChangeSet", ctx, mock.MatchedBy(func(input *cloudformation.DescribeChangeSetInput) bool {
		return awssdk.ToString(input.ChangeSetName) == changeSetId
	})).Return(createTestDescribeChangeSetOutput(changeSetId, types.ChangeSetStatusCreateComplete), nil)

	// Execute
	result, err := manager.CreateChangeSetForDeployment(ctx, stackName, template, parameters, capabilities, tags)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, changeSetId, result.ChangeSetID)

	mockClient.AssertExpectations(t)
}

func TestDefaultChangeSetManager_CreateChangeSetForDeployment_StackExistsError(t *testing.T) {
	ctx := context.Background()
	mockClient := &MockChangeSetClient{}
	manager := NewChangeSetManager(mockClient)

	// Test data
	stackName := "test-stack"
	template := `{"AWSTemplateFormatVersion": "2010-09-09"}`
	parameters := map[string]string{}
	capabilities := []string{}
	tags := map[string]string{}

	// Mock StackExists failure
	mockClient.On("StackExists", ctx, stackName).Return(false, errors.New("API error"))

	// Execute
	result, err := manager.CreateChangeSetForDeployment(ctx, stackName, template, parameters, capabilities, tags)

	// Assert
	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to check if stack exists")

	mockClient.AssertExpectations(t)
}
