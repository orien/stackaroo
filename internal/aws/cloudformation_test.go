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

// MockCloudFormationClientForDeployTest is a mock client specifically for deployment tests
type MockCloudFormationClientForDeployTest struct {
	mock.Mock
}

func (m *MockCloudFormationClientForDeployTest) CreateStack(ctx context.Context, params *cloudformation.CreateStackInput, optFns ...func(*cloudformation.Options)) (*cloudformation.CreateStackOutput, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*cloudformation.CreateStackOutput), args.Error(1)
}

func (m *MockCloudFormationClientForDeployTest) UpdateStack(ctx context.Context, params *cloudformation.UpdateStackInput, optFns ...func(*cloudformation.Options)) (*cloudformation.UpdateStackOutput, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*cloudformation.UpdateStackOutput), args.Error(1)
}

func (m *MockCloudFormationClientForDeployTest) DeleteStack(ctx context.Context, params *cloudformation.DeleteStackInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DeleteStackOutput, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(*cloudformation.DeleteStackOutput), args.Error(1)
}

func (m *MockCloudFormationClientForDeployTest) DescribeStacks(ctx context.Context, params *cloudformation.DescribeStacksInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DescribeStacksOutput, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*cloudformation.DescribeStacksOutput), args.Error(1)
}

func (m *MockCloudFormationClientForDeployTest) ListStacks(ctx context.Context, params *cloudformation.ListStacksInput, optFns ...func(*cloudformation.Options)) (*cloudformation.ListStacksOutput, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(*cloudformation.ListStacksOutput), args.Error(1)
}

func (m *MockCloudFormationClientForDeployTest) ValidateTemplate(ctx context.Context, params *cloudformation.ValidateTemplateInput, optFns ...func(*cloudformation.Options)) (*cloudformation.ValidateTemplateOutput, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(*cloudformation.ValidateTemplateOutput), args.Error(1)
}

func (m *MockCloudFormationClientForDeployTest) GetTemplate(ctx context.Context, params *cloudformation.GetTemplateInput, optFns ...func(*cloudformation.Options)) (*cloudformation.GetTemplateOutput, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(*cloudformation.GetTemplateOutput), args.Error(1)
}

func (m *MockCloudFormationClientForDeployTest) CreateChangeSet(ctx context.Context, params *cloudformation.CreateChangeSetInput, optFns ...func(*cloudformation.Options)) (*cloudformation.CreateChangeSetOutput, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(*cloudformation.CreateChangeSetOutput), args.Error(1)
}

func (m *MockCloudFormationClientForDeployTest) ExecuteChangeSet(ctx context.Context, params *cloudformation.ExecuteChangeSetInput, optFns ...func(*cloudformation.Options)) (*cloudformation.ExecuteChangeSetOutput, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(*cloudformation.ExecuteChangeSetOutput), args.Error(1)
}

func (m *MockCloudFormationClientForDeployTest) DeleteChangeSet(ctx context.Context, params *cloudformation.DeleteChangeSetInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DeleteChangeSetOutput, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(*cloudformation.DeleteChangeSetOutput), args.Error(1)
}

func (m *MockCloudFormationClientForDeployTest) DescribeChangeSet(ctx context.Context, params *cloudformation.DescribeChangeSetInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DescribeChangeSetOutput, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(*cloudformation.DescribeChangeSetOutput), args.Error(1)
}

func (m *MockCloudFormationClientForDeployTest) DescribeStackEvents(ctx context.Context, params *cloudformation.DescribeStackEventsInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DescribeStackEventsOutput, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*cloudformation.DescribeStackEventsOutput), args.Error(1)
}

func TestDeployStack_CreateNewStack_Success(t *testing.T) {
	ctx := context.Background()
	mockClient := &MockCloudFormationClientForDeployTest{}
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
	mockClient := &MockCloudFormationClientForDeployTest{}
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
	mockClient := &MockCloudFormationClientForDeployTest{}
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
	mockClient := &MockCloudFormationClientForDeployTest{}
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
	mockClient := &MockCloudFormationClientForDeployTest{}
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
	mockClient := &MockCloudFormationClientForDeployTest{}
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
	mockClient := &MockCloudFormationClientForDeployTest{}
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
	mockClient := &MockCloudFormationClientForDeployTest{}
	cfOps := NewCloudFormationOperationsWithClient(mockClient)
	ctx := context.Background()

	changeSetID := "arn:aws:cloudformation:us-east-1:123456789012:changeSet/test-changeset/test-stack"

	executeInput := &cloudformation.ExecuteChangeSetInput{
		ChangeSetName: aws.String(changeSetID),
	}

	expectedOutput := &cloudformation.ExecuteChangeSetOutput{}

	mockClient.On("ExecuteChangeSet", ctx, executeInput).Return(expectedOutput, nil)

	output, err := cfOps.ExecuteChangeSet(ctx, executeInput)

	require.NoError(t, err)
	assert.Equal(t, expectedOutput, output)
	mockClient.AssertExpectations(t)
}

func TestDefaultCloudFormationOperations_ExecuteChangeSet_Error(t *testing.T) {
	mockClient := &MockCloudFormationClientForDeployTest{}
	cfOps := NewCloudFormationOperationsWithClient(mockClient)
	ctx := context.Background()

	changeSetID := "arn:aws:cloudformation:us-east-1:123456789012:changeSet/test-changeset/test-stack"

	executeInput := &cloudformation.ExecuteChangeSetInput{
		ChangeSetName: aws.String(changeSetID),
	}

	expectedError := errors.New("changeset execution failed")

	mockClient.On("ExecuteChangeSet", ctx, executeInput).Return((*cloudformation.ExecuteChangeSetOutput)(nil), expectedError)

	output, err := cfOps.ExecuteChangeSet(ctx, executeInput)

	assert.Nil(t, output)
	assert.Error(t, err)
	assert.Equal(t, expectedError, err)
	mockClient.AssertExpectations(t)
}

func TestDefaultCloudFormationOperations_ExecuteChangeSetByID_Success(t *testing.T) {
	mockClient := &MockCloudFormationClientForDeployTest{}
	cfOps := NewCloudFormationOperationsWithClient(mockClient)
	ctx := context.Background()

	changeSetID := "arn:aws:cloudformation:us-east-1:123456789012:changeSet/test-changeset/test-stack"

	executeInput := &cloudformation.ExecuteChangeSetInput{
		ChangeSetName: aws.String(changeSetID),
	}

	expectedOutput := &cloudformation.ExecuteChangeSetOutput{}

	mockClient.On("ExecuteChangeSet", ctx, executeInput).Return(expectedOutput, nil)

	err := cfOps.ExecuteChangeSetByID(ctx, changeSetID)

	require.NoError(t, err)
	mockClient.AssertExpectations(t)
}

func TestDefaultCloudFormationOperations_ExecuteChangeSetByID_Error(t *testing.T) {
	mockClient := &MockCloudFormationClientForDeployTest{}
	cfOps := NewCloudFormationOperationsWithClient(mockClient)
	ctx := context.Background()

	changeSetID := "arn:aws:cloudformation:us-east-1:123456789012:changeSet/test-changeset/test-stack"

	executeInput := &cloudformation.ExecuteChangeSetInput{
		ChangeSetName: aws.String(changeSetID),
	}

	expectedError := errors.New("changeset execution failed")

	mockClient.On("ExecuteChangeSet", ctx, executeInput).Return((*cloudformation.ExecuteChangeSetOutput)(nil), expectedError)

	err := cfOps.ExecuteChangeSetByID(ctx, changeSetID)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to execute changeset")
	assert.Contains(t, err.Error(), "changeset execution failed")
	mockClient.AssertExpectations(t)
}
