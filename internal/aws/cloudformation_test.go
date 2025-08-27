/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package aws

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestDeployStack_CreateNewStack_Success(t *testing.T) {
	ctx := context.Background()
	mockClient := &MockCloudFormationClient{}
	cfOps := NewCloudFormationOperationsWithClient(mockClient)

	input := DeployStackInput{
		StackName:    "test-stack",
		TemplateBody: `{"AWSTemplateFormatVersion": "2010-09-09"}`,
		Parameters: []Parameter{
			{Key: "Environment", Value: "test"},
		},
		Tags: map[string]string{
			"Project": "stackaroo",
		},
		Capabilities: []string{"CAPABILITY_IAM"},
	}

	// Mock StackExists to return false (new stack) - first call only
	mockClient.On("DescribeStacks", ctx, mock.AnythingOfType("*cloudformation.DescribeStacksInput")).
		Return(nil, errors.New("ValidationError: Stack does not exist")).Once()

	// Mock CreateStack
	mockClient.On("CreateStack", ctx, mock.AnythingOfType("*cloudformation.CreateStackInput")).
		Return(&cloudformation.CreateStackOutput{}, nil)

	// Mock the waiting process - return completed stack for subsequent calls
	completedStack := &cloudformation.DescribeStacksOutput{
		Stacks: []types.Stack{
			{
				StackName:    aws.String("test-stack"),
				StackStatus:  types.StackStatusCreateComplete,
				CreationTime: aws.Time(time.Now()),
			},
		},
	}
	mockClient.On("DescribeStacks", ctx, mock.MatchedBy(func(input *cloudformation.DescribeStacksInput) bool {
		return aws.ToString(input.StackName) == "test-stack"
	})).Return(completedStack, nil)

	// Mock events
	eventsOutput := &cloudformation.DescribeStackEventsOutput{
		StackEvents: []types.StackEvent{
			{
				EventId:           aws.String("event-1"),
				StackName:         aws.String("test-stack"),
				LogicalResourceId: aws.String("test-stack"),
				ResourceType:      aws.String("AWS::CloudFormation::Stack"),
				Timestamp:         aws.Time(time.Now()),
				ResourceStatus:    types.ResourceStatusCreateComplete,
			},
		},
	}
	mockClient.On("DescribeStackEvents", ctx, mock.AnythingOfType("*cloudformation.DescribeStackEventsInput")).
		Return(eventsOutput, nil).Maybe()

	err := cfOps.DeployStack(ctx, input)

	require.NoError(t, err)
	mockClient.AssertExpectations(t)
}

func TestDeployStack_UpdateExistingStack_Success(t *testing.T) {
	ctx := context.Background()
	mockClient := &MockCloudFormationClient{}
	cfOps := NewCloudFormationOperationsWithClient(mockClient)

	input := DeployStackInput{
		StackName:    "existing-stack",
		TemplateBody: `{"AWSTemplateFormatVersion": "2010-09-09"}`,
		Parameters: []Parameter{
			{Key: "Environment", Value: "test"},
		},
		Tags: map[string]string{
			"Project": "stackaroo",
		},
		Capabilities: []string{"CAPABILITY_IAM"},
	}

	// Mock StackExists to return true (existing stack)
	existingStack := &cloudformation.DescribeStacksOutput{
		Stacks: []types.Stack{
			{
				StackName:   aws.String("existing-stack"),
				StackStatus: types.StackStatusCreateComplete,
			},
		},
	}
	mockClient.On("DescribeStacks", ctx, mock.AnythingOfType("*cloudformation.DescribeStacksInput")).
		Return(existingStack, nil)

	// Mock UpdateStack
	mockClient.On("UpdateStack", ctx, mock.AnythingOfType("*cloudformation.UpdateStackInput")).
		Return(&cloudformation.UpdateStackOutput{}, nil)

	// Mock the waiting process - return updated stack
	updatedStack := &cloudformation.DescribeStacksOutput{
		Stacks: []types.Stack{
			{
				StackName:       aws.String("existing-stack"),
				StackStatus:     types.StackStatusUpdateComplete,
				LastUpdatedTime: aws.Time(time.Now()),
			},
		},
	}
	mockClient.On("DescribeStacks", ctx, mock.MatchedBy(func(input *cloudformation.DescribeStacksInput) bool {
		return aws.ToString(input.StackName) == "existing-stack"
	})).Return(updatedStack, nil).Maybe()

	// Mock events
	eventsOutput := &cloudformation.DescribeStackEventsOutput{
		StackEvents: []types.StackEvent{
			{
				EventId:           aws.String("event-1"),
				StackName:         aws.String("existing-stack"),
				LogicalResourceId: aws.String("existing-stack"),
				ResourceType:      aws.String("AWS::CloudFormation::Stack"),
				Timestamp:         aws.Time(time.Now()),
				ResourceStatus:    types.ResourceStatusUpdateComplete,
			},
		},
	}
	mockClient.On("DescribeStackEvents", ctx, mock.AnythingOfType("*cloudformation.DescribeStackEventsInput")).
		Return(eventsOutput, nil).Maybe()

	err := cfOps.DeployStack(ctx, input)

	require.NoError(t, err)
	mockClient.AssertExpectations(t)
}

func TestDeployStack_UpdateNoChanges_Success(t *testing.T) {
	ctx := context.Background()
	mockClient := &MockCloudFormationClient{}
	cfOps := NewCloudFormationOperationsWithClient(mockClient)

	input := DeployStackInput{
		StackName:    "no-change-stack",
		TemplateBody: `{"AWSTemplateFormatVersion": "2010-09-09"}`,
	}

	// Mock StackExists to return true (existing stack)
	existingStack := &cloudformation.DescribeStacksOutput{
		Stacks: []types.Stack{
			{
				StackName:   aws.String("no-change-stack"),
				StackStatus: types.StackStatusCreateComplete,
			},
		},
	}
	mockClient.On("DescribeStacks", ctx, mock.AnythingOfType("*cloudformation.DescribeStacksInput")).
		Return(existingStack, nil)

	// Mock UpdateStack to return "no changes" error
	mockClient.On("UpdateStack", ctx, mock.AnythingOfType("*cloudformation.UpdateStackInput")).
		Return(nil, errors.New("ValidationError: No updates are to be performed"))

	err := cfOps.DeployStack(ctx, input)

	require.Error(t, err)
	var noChangesErr NoChangesError
	require.ErrorAs(t, err, &noChangesErr)
	assert.Equal(t, "no-change-stack", noChangesErr.StackName)
	mockClient.AssertExpectations(t)
}

func TestDeployStack_CreateStack_Failure(t *testing.T) {
	ctx := context.Background()
	mockClient := &MockCloudFormationClient{}
	cfOps := NewCloudFormationOperationsWithClient(mockClient)

	input := DeployStackInput{
		StackName:    "fail-stack",
		TemplateBody: `{"AWSTemplateFormatVersion": "2010-09-09"}`,
	}

	// Mock StackExists to return false (new stack)
	mockClient.On("DescribeStacks", ctx, mock.AnythingOfType("*cloudformation.DescribeStacksInput")).
		Return(nil, errors.New("ValidationError: Stack does not exist"))

	// Mock CreateStack to fail
	mockClient.On("CreateStack", ctx, mock.AnythingOfType("*cloudformation.CreateStackInput")).
		Return(nil, errors.New("ValidationError: Invalid template"))

	err := cfOps.DeployStack(ctx, input)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create stack fail-stack")
	mockClient.AssertExpectations(t)
}

func TestDescribeStackEvents_Success(t *testing.T) {
	ctx := context.Background()
	mockClient := &MockCloudFormationClient{}
	cfOps := NewCloudFormationOperationsWithClient(mockClient)

	expectedEvents := &cloudformation.DescribeStackEventsOutput{
		StackEvents: []types.StackEvent{
			{
				EventId:              aws.String("event-1"),
				StackName:            aws.String("test-stack"),
				LogicalResourceId:    aws.String("MyBucket"),
				PhysicalResourceId:   aws.String("test-bucket-123"),
				ResourceType:         aws.String("AWS::S3::Bucket"),
				Timestamp:            aws.Time(time.Now()),
				ResourceStatus:       types.ResourceStatusCreateComplete,
				ResourceStatusReason: aws.String(""),
			},
		},
	}

	mockClient.On("DescribeStackEvents", ctx, mock.AnythingOfType("*cloudformation.DescribeStackEventsInput")).
		Return(expectedEvents, nil)

	events, err := cfOps.DescribeStackEvents(ctx, "test-stack")

	require.NoError(t, err)
	require.Len(t, events, 1)
	assert.Equal(t, "event-1", events[0].EventId)
	assert.Equal(t, "test-stack", events[0].StackName)
	assert.Equal(t, "MyBucket", events[0].LogicalResourceId)
	assert.Equal(t, "test-bucket-123", events[0].PhysicalResourceId)
	assert.Equal(t, "AWS::S3::Bucket", events[0].ResourceType)
	assert.Equal(t, "CREATE_COMPLETE", events[0].ResourceStatus)
	mockClient.AssertExpectations(t)
}

func TestDescribeStackEvents_Failure(t *testing.T) {
	ctx := context.Background()
	mockClient := &MockCloudFormationClient{}
	cfOps := NewCloudFormationOperationsWithClient(mockClient)

	mockClient.On("DescribeStackEvents", ctx, mock.AnythingOfType("*cloudformation.DescribeStackEventsInput")).
		Return(nil, errors.New("AccessDenied: User not authorized"))

	events, err := cfOps.DescribeStackEvents(ctx, "test-stack")

	require.Error(t, err)
	assert.Nil(t, events)
	assert.Contains(t, err.Error(), "failed to describe stack events")
	mockClient.AssertExpectations(t)
}

func TestIsStackOperationComplete(t *testing.T) {
	tests := []struct {
		name     string
		status   StackStatus
		expected bool
	}{
		{"CreateComplete", StackStatusCreateComplete, true},
		{"CreateFailed", StackStatusCreateFailed, true},
		{"CreateInProgress", StackStatusCreateInProgress, false},
		{"UpdateComplete", StackStatusUpdateComplete, true},
		{"UpdateFailed", StackStatusUpdateFailed, true},
		{"UpdateInProgress", StackStatusUpdateInProgress, false},
		{"UpdateRollbackComplete", StackStatusUpdateRollbackComplete, true},
		{"UpdateRollbackInProgress", StackStatusUpdateRollbackInProgress, false},
		{"DeleteComplete", StackStatusDeleteComplete, true},
		{"DeleteInProgress", StackStatusDeleteInProgress, false},
		{"RollbackComplete", StackStatusRollbackComplete, true},
		{"RollbackInProgress", StackStatusRollbackInProgress, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isStackOperationComplete(tt.status)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsStackOperationSuccessful(t *testing.T) {
	tests := []struct {
		name     string
		status   StackStatus
		expected bool
	}{
		{"CreateComplete", StackStatusCreateComplete, true},
		{"CreateFailed", StackStatusCreateFailed, false},
		{"UpdateComplete", StackStatusUpdateComplete, true},
		{"UpdateFailed", StackStatusUpdateFailed, false},
		{"UpdateRollbackComplete", StackStatusUpdateRollbackComplete, false},
		{"DeleteComplete", StackStatusDeleteComplete, true},
		{"DeleteFailed", StackStatusDeleteFailed, false},
		{"RollbackComplete", StackStatusRollbackComplete, false},
		{"RollbackFailed", StackStatusRollbackFailed, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isStackOperationSuccessful(tt.status)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsNoChangesError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "no changes error",
			err:      errors.New("ValidationError: No updates are to be performed"),
			expected: true,
		},
		{
			name:     "validation error",
			err:      errors.New("ValidationError: Template format error"),
			expected: true,
		},
		{
			name:     "other error",
			err:      errors.New("AccessDenied: User not authorized"),
			expected: false,
		},
		{
			name:     "network error",
			err:      errors.New("RequestTimeout: Connection timeout"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isNoChangesError(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDeployStackWithCallback_Success(t *testing.T) {
	ctx := context.Background()
	mockClient := &MockCloudFormationClient{}
	cfOps := NewCloudFormationOperationsWithClient(mockClient)

	input := DeployStackInput{
		StackName:    "callback-stack",
		TemplateBody: `{"AWSTemplateFormatVersion": "2010-09-09"}`,
		Parameters: []Parameter{
			{Key: "Environment", Value: "test"},
		},
	}

	var capturedEvents []StackEvent
	eventCallback := func(event StackEvent) {
		capturedEvents = append(capturedEvents, event)
	}

	// Mock StackExists to return false (new stack)
	mockClient.On("DescribeStacks", ctx, mock.AnythingOfType("*cloudformation.DescribeStacksInput")).
		Return(nil, errors.New("ValidationError: Stack does not exist")).Once()

	// Mock CreateStack
	mockClient.On("CreateStack", ctx, mock.AnythingOfType("*cloudformation.CreateStackInput")).
		Return(&cloudformation.CreateStackOutput{}, nil)

	// Mock the waiting process - return completed stack
	completedStack := &cloudformation.DescribeStacksOutput{
		Stacks: []types.Stack{
			{
				StackName:    aws.String("callback-stack"),
				StackStatus:  types.StackStatusCreateComplete,
				CreationTime: aws.Time(time.Now()),
			},
		},
	}
	mockClient.On("DescribeStacks", ctx, mock.MatchedBy(func(input *cloudformation.DescribeStacksInput) bool {
		return aws.ToString(input.StackName) == "callback-stack"
	})).Return(completedStack, nil)

	// Mock events
	eventsOutput := &cloudformation.DescribeStackEventsOutput{
		StackEvents: []types.StackEvent{
			{
				EventId:           aws.String("event-1"),
				StackName:         aws.String("callback-stack"),
				LogicalResourceId: aws.String("callback-stack"),
				ResourceType:      aws.String("AWS::CloudFormation::Stack"),
				Timestamp:         aws.Time(time.Now()),
				ResourceStatus:    types.ResourceStatusCreateComplete,
			},
		},
	}
	mockClient.On("DescribeStackEvents", ctx, mock.AnythingOfType("*cloudformation.DescribeStackEventsInput")).
		Return(eventsOutput, nil).Maybe()

	err := cfOps.DeployStackWithCallback(ctx, input, eventCallback)

	require.NoError(t, err)
	assert.Len(t, capturedEvents, 1)
	assert.Equal(t, "event-1", capturedEvents[0].EventId)
	assert.Equal(t, "callback-stack", capturedEvents[0].StackName)
	mockClient.AssertExpectations(t)
}

func TestDefaultCloudFormationOperations_ExecuteChangeSet_Success(t *testing.T) {
	mockClient := &MockCloudFormationClient{}
	cfOps := NewCloudFormationOperationsWithClient(mockClient)
	ctx := context.Background()

	changeSetID := "arn:aws:cloudformation:us-east-1:123456789012:changeSet/test-changeset/test-stack"

	executeInput := &cloudformation.ExecuteChangeSetInput{
		ChangeSetName: aws.String(changeSetID),
	}

	expectedOutput := &cloudformation.ExecuteChangeSetOutput{}

	mockClient.On("ExecuteChangeSet", ctx, executeInput).Return(expectedOutput, nil)

	err := cfOps.ExecuteChangeSet(ctx, changeSetID)

	require.NoError(t, err)
	mockClient.AssertExpectations(t)
}

func TestDefaultCloudFormationOperations_ExecuteChangeSet_Error(t *testing.T) {
	mockClient := &MockCloudFormationClient{}
	cfOps := NewCloudFormationOperationsWithClient(mockClient)
	ctx := context.Background()

	changeSetID := "arn:aws:cloudformation:us-east-1:123456789012:changeSet/test-changeset/test-stack"

	executeInput := &cloudformation.ExecuteChangeSetInput{
		ChangeSetName: aws.String(changeSetID),
	}

	expectedError := errors.New("changeset execution failed")

	mockClient.On("ExecuteChangeSet", ctx, executeInput).Return((*cloudformation.ExecuteChangeSetOutput)(nil), expectedError)

	err := cfOps.ExecuteChangeSet(ctx, changeSetID)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to execute changeset")
	assert.Contains(t, err.Error(), "changeset execution failed")
	mockClient.AssertExpectations(t)
}

// Helper functions for changeset testing

func createTestChangeSetOutput(changeSetId string) *cloudformation.CreateChangeSetOutput {
	return &cloudformation.CreateChangeSetOutput{
		Id: aws.String(changeSetId),
	}
}

func createTestDescribeChangeSetOutput(changeSetId string, status types.ChangeSetStatus) *cloudformation.DescribeChangeSetOutput {
	return &cloudformation.DescribeChangeSetOutput{
		ChangeSetId: aws.String(changeSetId),
		Status:      status,
		Changes: []types.Change{
			{
				Type: types.ChangeTypeResource,
				ResourceChange: &types.ResourceChange{
					Action:             types.ChangeActionAdd,
					LogicalResourceId:  aws.String("MyBucket"),
					PhysicalResourceId: aws.String("my-bucket-12345"),
					ResourceType:       aws.String("AWS::S3::Bucket"),
					Replacement:        types.ReplacementFalse,
					Details: []types.ResourceChangeDetail{
						{
							Target: &types.ResourceTargetDefinition{
								Attribute: types.ResourceAttributeProperties,
								Name:      aws.String("BucketName"),
							},
						},
					},
				},
			},
		},
	}
}

// Changeset Tests

func TestDefaultCloudFormationOperations_CreateChangeSetPreview_Success(t *testing.T) {
	ctx := context.Background()
	mockClient := &MockCloudFormationClient{}
	cf := &DefaultCloudFormationOperations{client: mockClient}

	// Test data
	stackName := "test-stack"
	template := `{"AWSTemplateFormatVersion": "2010-09-09"}`
	parameters := map[string]string{"Param1": "value1"}
	changeSetId := "test-changeset-123"

	// Mock CreateChangeSet
	mockClient.On("CreateChangeSet", ctx, mock.MatchedBy(func(input *cloudformation.CreateChangeSetInput) bool {
		return aws.ToString(input.StackName) == stackName &&
			aws.ToString(input.TemplateBody) == template &&
			len(input.Parameters) == 1 &&
			aws.ToString(input.Parameters[0].ParameterKey) == "Param1" &&
			aws.ToString(input.Parameters[0].ParameterValue) == "value1" &&
			input.ChangeSetType == types.ChangeSetTypeUpdate
	})).Return(createTestChangeSetOutput(changeSetId), nil)

	// Mock DescribeChangeSet for waiting (called once during waitForChangeSet)
	mockClient.On("DescribeChangeSet", ctx, mock.MatchedBy(func(input *cloudformation.DescribeChangeSetInput) bool {
		return aws.ToString(input.ChangeSetName) == changeSetId
	})).Return(createTestDescribeChangeSetOutput(changeSetId, types.ChangeSetStatusCreateComplete), nil).Once()

	// Mock DescribeChangeSet for describing the changeset (called once during describeChangeSetInternal)
	mockClient.On("DescribeChangeSet", ctx, mock.MatchedBy(func(input *cloudformation.DescribeChangeSetInput) bool {
		return aws.ToString(input.ChangeSetName) == changeSetId
	})).Return(createTestDescribeChangeSetOutput(changeSetId, types.ChangeSetStatusCreateComplete), nil).Once()

	// Mock DeleteChangeSet for cleanup
	mockClient.On("DeleteChangeSet", ctx, mock.MatchedBy(func(input *cloudformation.DeleteChangeSetInput) bool {
		return aws.ToString(input.ChangeSetName) == changeSetId
	})).Return(&cloudformation.DeleteChangeSetOutput{}, nil)

	// Execute
	result, err := cf.CreateChangeSetPreview(ctx, stackName, template, parameters)

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

func TestDefaultCloudFormationOperations_CreateChangeSetPreview_CreateError(t *testing.T) {
	ctx := context.Background()
	mockClient := &MockCloudFormationClient{}
	cf := &DefaultCloudFormationOperations{client: mockClient}

	// Test data
	stackName := "test-stack"
	template := `{"AWSTemplateFormatVersion": "2010-09-09"}`
	parameters := map[string]string{}

	// Mock CreateChangeSet failure
	mockClient.On("CreateChangeSet", ctx, mock.AnythingOfType("*cloudformation.CreateChangeSetInput")).Return((*cloudformation.CreateChangeSetOutput)(nil), errors.New("access denied"))

	// Execute
	result, err := cf.CreateChangeSetPreview(ctx, stackName, template, parameters)

	// Verify
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to create changeset")

	mockClient.AssertExpectations(t)
}

func TestDefaultCloudFormationOperations_CreateChangeSetPreview_WaitError(t *testing.T) {
	ctx := context.Background()
	mockClient := &MockCloudFormationClient{}
	cf := &DefaultCloudFormationOperations{client: mockClient}

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
		StatusReason: aws.String("Template validation error"),
	}, nil)

	// Mock DeleteChangeSet for cleanup
	mockClient.On("DeleteChangeSet", ctx, mock.AnythingOfType("*cloudformation.DeleteChangeSetInput")).Return(&cloudformation.DeleteChangeSetOutput{}, nil)

	// Execute
	result, err := cf.CreateChangeSetPreview(ctx, stackName, template, parameters)

	// Verify
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "changeset creation failed")
	assert.Contains(t, err.Error(), "Template validation error")

	mockClient.AssertExpectations(t)
}

func TestDefaultCloudFormationOperations_CreateChangeSetForDeployment_NewStack(t *testing.T) {
	ctx := context.Background()
	mockClient := &MockCloudFormationClient{}
	cf := &DefaultCloudFormationOperations{client: mockClient}

	// Test data
	stackName := "test-stack"
	template := `{"AWSTemplateFormatVersion": "2010-09-09"}`
	parameters := map[string]string{"Param1": "value1"}
	capabilities := []string{"CAPABILITY_IAM"}
	tags := map[string]string{"Environment": "test"}
	changeSetId := "test-changeset-123"

	// Mock DescribeStacks to return error (stack doesn't exist)
	mockClient.On("DescribeStacks", ctx, mock.MatchedBy(func(input *cloudformation.DescribeStacksInput) bool {
		return aws.ToString(input.StackName) == stackName
	})).Return((*cloudformation.DescribeStacksOutput)(nil), errors.New("ValidationError: Stack with id test-stack does not exist"))

	// Mock CreateChangeSet with CREATE type
	mockClient.On("CreateChangeSet", ctx, mock.MatchedBy(func(input *cloudformation.CreateChangeSetInput) bool {
		return aws.ToString(input.StackName) == stackName &&
			aws.ToString(input.TemplateBody) == template &&
			len(input.Parameters) == 1 &&
			aws.ToString(input.Parameters[0].ParameterKey) == "Param1" &&
			aws.ToString(input.Parameters[0].ParameterValue) == "value1" &&
			len(input.Tags) == 1 &&
			aws.ToString(input.Tags[0].Key) == "Environment" &&
			aws.ToString(input.Tags[0].Value) == "test" &&
			len(input.Capabilities) == 1 &&
			string(input.Capabilities[0]) == "CAPABILITY_IAM" &&
			input.ChangeSetType == types.ChangeSetTypeCreate
	})).Return(createTestChangeSetOutput(changeSetId), nil)

	// Mock DescribeChangeSet for waiting
	mockClient.On("DescribeChangeSet", ctx, mock.MatchedBy(func(input *cloudformation.DescribeChangeSetInput) bool {
		return aws.ToString(input.ChangeSetName) == changeSetId
	})).Return(createTestDescribeChangeSetOutput(changeSetId, types.ChangeSetStatusCreateComplete), nil).Once()

	// Mock DescribeChangeSet for describing the changeset
	mockClient.On("DescribeChangeSet", ctx, mock.MatchedBy(func(input *cloudformation.DescribeChangeSetInput) bool {
		return aws.ToString(input.ChangeSetName) == changeSetId
	})).Return(createTestDescribeChangeSetOutput(changeSetId, types.ChangeSetStatusCreateComplete), nil).Once()

	// Execute
	result, err := cf.CreateChangeSetForDeployment(ctx, stackName, template, parameters, capabilities, tags)

	// Verify
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, changeSetId, result.ChangeSetID)
	assert.Equal(t, "CREATE_COMPLETE", result.Status)
	assert.Len(t, result.Changes, 1)

	mockClient.AssertExpectations(t)
}

func TestDefaultCloudFormationOperations_CreateChangeSetForDeployment_ExistingStack(t *testing.T) {
	ctx := context.Background()
	mockClient := &MockCloudFormationClient{}
	cf := &DefaultCloudFormationOperations{client: mockClient}

	// Test data
	stackName := "test-stack"
	template := `{"AWSTemplateFormatVersion": "2010-09-09"}`
	parameters := map[string]string{}
	capabilities := []string{}
	tags := map[string]string{}
	changeSetId := "test-changeset-123"

	// Mock DescribeStacks to return existing stack
	mockClient.On("DescribeStacks", ctx, mock.MatchedBy(func(input *cloudformation.DescribeStacksInput) bool {
		return aws.ToString(input.StackName) == stackName
	})).Return(&cloudformation.DescribeStacksOutput{
		Stacks: []types.Stack{
			{
				StackName:   aws.String(stackName),
				StackStatus: types.StackStatusCreateComplete,
			},
		},
	}, nil)

	// Mock CreateChangeSet with UPDATE type
	mockClient.On("CreateChangeSet", ctx, mock.MatchedBy(func(input *cloudformation.CreateChangeSetInput) bool {
		return aws.ToString(input.StackName) == stackName &&
			input.ChangeSetType == types.ChangeSetTypeUpdate
	})).Return(createTestChangeSetOutput(changeSetId), nil)

	// Mock DescribeChangeSet for waiting and describing
	mockClient.On("DescribeChangeSet", ctx, mock.AnythingOfType("*cloudformation.DescribeChangeSetInput")).Return(
		createTestDescribeChangeSetOutput(changeSetId, types.ChangeSetStatusCreateComplete), nil).Times(2)

	// Execute
	result, err := cf.CreateChangeSetForDeployment(ctx, stackName, template, parameters, capabilities, tags)

	// Verify
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, changeSetId, result.ChangeSetID)

	mockClient.AssertExpectations(t)
}

func TestDefaultCloudFormationOperations_CreateChangeSetForDeployment_StackExistsError(t *testing.T) {
	ctx := context.Background()
	mockClient := &MockCloudFormationClient{}
	cf := &DefaultCloudFormationOperations{client: mockClient}

	// Test data
	stackName := "test-stack"
	template := `{"AWSTemplateFormatVersion": "2010-09-09"}`
	parameters := map[string]string{}
	capabilities := []string{}
	tags := map[string]string{}

	// Mock DescribeStacks failure
	mockClient.On("DescribeStacks", ctx, mock.AnythingOfType("*cloudformation.DescribeStacksInput")).Return(
		(*cloudformation.DescribeStacksOutput)(nil), errors.New("access denied"))

	// Execute
	result, err := cf.CreateChangeSetForDeployment(ctx, stackName, template, parameters, capabilities, tags)

	// Verify
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to check if stack exists")

	mockClient.AssertExpectations(t)
}

func TestDefaultCloudFormationOperations_WaitForChangeSet_Success(t *testing.T) {
	ctx := context.Background()
	mockClient := &MockCloudFormationClient{}
	cf := &DefaultCloudFormationOperations{client: mockClient}

	changeSetId := "test-changeset-123"

	// Mock DescribeChangeSet - return complete immediately
	mockClient.On("DescribeChangeSet", ctx, mock.AnythingOfType("*cloudformation.DescribeChangeSetInput")).Return(&cloudformation.DescribeChangeSetOutput{
		Status: types.ChangeSetStatusCreateComplete,
	}, nil)

	// Execute
	err := cf.waitForChangeSet(ctx, changeSetId)

	// Verify
	assert.NoError(t, err)
	mockClient.AssertExpectations(t)
}

func TestDefaultCloudFormationOperations_WaitForChangeSet_PendingThenComplete(t *testing.T) {
	ctx := context.Background()
	mockClient := &MockCloudFormationClient{}
	cf := &DefaultCloudFormationOperations{client: mockClient}

	changeSetId := "test-changeset-123"

	// Mock DescribeChangeSet - first pending, then complete
	mockClient.On("DescribeChangeSet", ctx, mock.AnythingOfType("*cloudformation.DescribeChangeSetInput")).Return(&cloudformation.DescribeChangeSetOutput{
		Status: types.ChangeSetStatusCreatePending,
	}, nil).Once()

	mockClient.On("DescribeChangeSet", ctx, mock.AnythingOfType("*cloudformation.DescribeChangeSetInput")).Return(&cloudformation.DescribeChangeSetOutput{
		Status: types.ChangeSetStatusCreateComplete,
	}, nil).Once()

	// Execute
	err := cf.waitForChangeSet(ctx, changeSetId)

	// Verify
	assert.NoError(t, err)
	mockClient.AssertExpectations(t)
}

func TestDefaultCloudFormationOperations_WaitForChangeSet_Failed(t *testing.T) {
	ctx := context.Background()
	mockClient := &MockCloudFormationClient{}
	cf := &DefaultCloudFormationOperations{client: mockClient}

	changeSetId := "test-changeset-123"

	// Mock DescribeChangeSet - return failed status
	mockClient.On("DescribeChangeSet", ctx, mock.AnythingOfType("*cloudformation.DescribeChangeSetInput")).Return(&cloudformation.DescribeChangeSetOutput{
		Status:       types.ChangeSetStatusFailed,
		StatusReason: aws.String("No changes to deploy"),
	}, nil)

	// Execute
	err := cf.waitForChangeSet(ctx, changeSetId)

	// Verify
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "changeset creation failed: No changes to deploy")
	mockClient.AssertExpectations(t)
}

func TestDefaultCloudFormationOperations_WaitForChangeSet_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	mockClient := &MockCloudFormationClient{}
	cf := &DefaultCloudFormationOperations{client: mockClient}

	changeSetId := "test-changeset-123"

	// Cancel context immediately
	cancel()

	// Execute
	err := cf.waitForChangeSet(ctx, changeSetId)

	// Verify
	assert.Error(t, err)
	assert.Equal(t, context.Canceled, err)
}

func TestDefaultCloudFormationOperations_DescribeChangeSetInternal_Success(t *testing.T) {
	ctx := context.Background()
	mockClient := &MockCloudFormationClient{}
	cf := &DefaultCloudFormationOperations{client: mockClient}

	changeSetId := "test-changeset-123"

	// Mock DescribeChangeSet
	mockClient.On("DescribeChangeSet", ctx, mock.MatchedBy(func(input *cloudformation.DescribeChangeSetInput) bool {
		return aws.ToString(input.ChangeSetName) == changeSetId
	})).Return(createTestDescribeChangeSetOutput(changeSetId, types.ChangeSetStatusCreateComplete), nil)

	// Execute
	result, err := cf.describeChangeSetInternal(ctx, changeSetId)

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
	assert.Contains(t, change.Details[0], "Property: BucketName")

	mockClient.AssertExpectations(t)
}

func TestDefaultCloudFormationOperations_DescribeChangeSetInternal_Error(t *testing.T) {
	ctx := context.Background()
	mockClient := &MockCloudFormationClient{}
	cf := &DefaultCloudFormationOperations{client: mockClient}

	changeSetId := "test-changeset-123"

	// Mock DescribeChangeSet failure
	mockClient.On("DescribeChangeSet", ctx, mock.AnythingOfType("*cloudformation.DescribeChangeSetInput")).Return(
		(*cloudformation.DescribeChangeSetOutput)(nil), errors.New("not found"))

	// Execute
	result, err := cf.describeChangeSetInternal(ctx, changeSetId)

	// Verify
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to describe changeset")

	mockClient.AssertExpectations(t)
}

func TestDefaultCloudFormationOperations_DeleteChangeSet_Success(t *testing.T) {
	ctx := context.Background()
	mockClient := &MockCloudFormationClient{}
	cf := &DefaultCloudFormationOperations{client: mockClient}

	changeSetId := "test-changeset-123"

	// Mock DeleteChangeSet
	mockClient.On("DeleteChangeSet", ctx, mock.MatchedBy(func(input *cloudformation.DeleteChangeSetInput) bool {
		return aws.ToString(input.ChangeSetName) == changeSetId
	})).Return(&cloudformation.DeleteChangeSetOutput{}, nil)

	// Execute
	err := cf.DeleteChangeSet(ctx, changeSetId)

	// Verify
	assert.NoError(t, err)
	mockClient.AssertExpectations(t)
}

func TestDefaultCloudFormationOperations_DeleteChangeSet_Error(t *testing.T) {
	ctx := context.Background()
	mockClient := &MockCloudFormationClient{}
	cf := &DefaultCloudFormationOperations{client: mockClient}

	changeSetId := "test-changeset-123"

	// Mock DeleteChangeSet failure
	mockClient.On("DeleteChangeSet", ctx, mock.MatchedBy(func(input *cloudformation.DeleteChangeSetInput) bool {
		return aws.ToString(input.ChangeSetName) == changeSetId
	})).Return((*cloudformation.DeleteChangeSetOutput)(nil), errors.New("changeset not found"))

	// Execute
	err := cf.DeleteChangeSet(ctx, changeSetId)

	// Verify
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to delete changeset")

	mockClient.AssertExpectations(t)
}
