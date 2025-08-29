/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package delete

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/orien/stackaroo/internal/aws"
	"github.com/orien/stackaroo/internal/model"
	"github.com/orien/stackaroo/internal/prompt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockClient implements aws.Client for testing
type MockClient struct {
	mock.Mock
}

func (m *MockClient) NewCloudFormationOperations() aws.CloudFormationOperations {
	args := m.Called()
	return args.Get(0).(aws.CloudFormationOperations)
}

// MockCloudFormationOperations implements aws.CloudFormationOperations for testing
type MockCloudFormationOperations struct {
	mock.Mock
}

func (m *MockCloudFormationOperations) DeployStack(ctx context.Context, input aws.DeployStackInput) error {
	args := m.Called(ctx, input)
	return args.Error(0)
}

func (m *MockCloudFormationOperations) DeployStackWithCallback(ctx context.Context, input aws.DeployStackInput, eventCallback func(aws.StackEvent)) error {
	args := m.Called(ctx, input, eventCallback)
	return args.Error(0)
}

func (m *MockCloudFormationOperations) UpdateStack(ctx context.Context, input aws.UpdateStackInput) error {
	args := m.Called(ctx, input)
	return args.Error(0)
}

func (m *MockCloudFormationOperations) DeleteStack(ctx context.Context, input aws.DeleteStackInput) error {
	args := m.Called(ctx, input)
	return args.Error(0)
}

func (m *MockCloudFormationOperations) GetStack(ctx context.Context, stackName string) (*aws.Stack, error) {
	args := m.Called(ctx, stackName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*aws.Stack), args.Error(1)
}

func (m *MockCloudFormationOperations) ListStacks(ctx context.Context) ([]*aws.Stack, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*aws.Stack), args.Error(1)
}

func (m *MockCloudFormationOperations) ValidateTemplate(ctx context.Context, templateBody string) error {
	args := m.Called(ctx, templateBody)
	return args.Error(0)
}

func (m *MockCloudFormationOperations) StackExists(ctx context.Context, stackName string) (bool, error) {
	args := m.Called(ctx, stackName)
	return args.Bool(0), args.Error(1)
}

func (m *MockCloudFormationOperations) GetTemplate(ctx context.Context, stackName string) (string, error) {
	args := m.Called(ctx, stackName)
	return args.String(0), args.Error(1)
}

func (m *MockCloudFormationOperations) DescribeStack(ctx context.Context, stackName string) (*aws.StackInfo, error) {
	args := m.Called(ctx, stackName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*aws.StackInfo), args.Error(1)
}

func (m *MockCloudFormationOperations) ExecuteChangeSet(ctx context.Context, changeSetID string) error {
	args := m.Called(ctx, changeSetID)
	return args.Error(0)
}

func (m *MockCloudFormationOperations) DeleteChangeSet(ctx context.Context, changeSetID string) error {
	args := m.Called(ctx, changeSetID)
	return args.Error(0)
}

func (m *MockCloudFormationOperations) DescribeStackEvents(ctx context.Context, stackName string) ([]aws.StackEvent, error) {
	args := m.Called(ctx, stackName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]aws.StackEvent), args.Error(1)
}

func (m *MockCloudFormationOperations) WaitForStackOperation(ctx context.Context, stackName string, eventCallback func(aws.StackEvent)) error {
	args := m.Called(ctx, stackName, eventCallback)
	// Call the callback with a sample event for testing
	if eventCallback != nil {
		eventCallback(aws.StackEvent{
			EventId:              "event-1",
			StackName:            stackName,
			LogicalResourceId:    stackName,
			ResourceType:         "AWS::CloudFormation::Stack",
			Timestamp:            time.Now(),
			ResourceStatus:       "DELETE_COMPLETE",
			ResourceStatusReason: "",
		})
	}
	return args.Error(0)
}

func (m *MockCloudFormationOperations) CreateChangeSetPreview(ctx context.Context, stackName string, template string, parameters map[string]string) (*aws.ChangeSetInfo, error) {
	args := m.Called(ctx, stackName, template, parameters)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*aws.ChangeSetInfo), args.Error(1)
}

func (m *MockCloudFormationOperations) CreateChangeSetForDeployment(ctx context.Context, stackName string, template string, parameters map[string]string, capabilities []string, tags map[string]string) (*aws.ChangeSetInfo, error) {
	args := m.Called(ctx, stackName, template, parameters, capabilities, tags)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*aws.ChangeSetInfo), args.Error(1)
}

// MockPrompter implements prompt.Prompter for testing
type MockPrompter struct {
	mock.Mock
}

// Confirm mock implementation
func (m *MockPrompter) Confirm(message string) (bool, error) {
	args := m.Called(message)
	return args.Bool(0), args.Error(1)
}

func TestNewStackDeleter(t *testing.T) {
	mockClient := &MockClient{}
	deleter := NewStackDeleter(mockClient)

	assert.NotNil(t, deleter)
	assert.Equal(t, mockClient, deleter.awsClient)
}

func TestDeleteStack_StackExists_UserConfirms_Success(t *testing.T) {
	ctx := context.Background()
	mockClient := &MockClient{}
	mockCfnOps := &MockCloudFormationOperations{}
	mockPrompter := &MockPrompter{}

	// Set up the mock client to return our mock CloudFormation operations
	mockClient.On("NewCloudFormationOperations").Return(mockCfnOps)

	// Set up mock for stack existence check
	mockCfnOps.On("StackExists", ctx, "test-stack").Return(true, nil)

	// Set up mock for stack description
	stackInfo := &aws.StackInfo{
		Name:        "test-stack",
		Status:      aws.StackStatusCreateComplete,
		Description: "Test stack description",
	}
	mockCfnOps.On("DescribeStack", ctx, "test-stack").Return(stackInfo, nil)

	// Set up mock for user confirmation
	// Business logic sends core message, prompter adds formatting
	expectedMessage := "Do you want to delete stack test-stack? This cannot be undone."
	mockPrompter.On("Confirm", expectedMessage).Return(true, nil)

	// Set up mock for stack deletion
	deleteInput := aws.DeleteStackInput{StackName: "test-stack"}
	mockCfnOps.On("DeleteStack", ctx, deleteInput).Return(nil)

	// Set up mock for waiting for deletion
	mockCfnOps.On("WaitForStackOperation", ctx, "test-stack", mock.AnythingOfType("func(aws.StackEvent)")).Return(nil)

	// Set the mock prompter
	originalPrompter := prompt.GetDefaultPrompter()
	prompt.SetPrompter(mockPrompter)
	defer prompt.SetPrompter(originalPrompter)

	// Create deleter and test
	deleter := NewStackDeleter(mockClient)
	stack := &model.Stack{
		Name:    "test-stack",
		Context: "dev",
	}

	err := deleter.DeleteStack(ctx, stack)

	assert.NoError(t, err)
	mockClient.AssertExpectations(t)
	mockCfnOps.AssertExpectations(t)
	mockPrompter.AssertExpectations(t)
}

func TestDeleteStack_StackExists_UserCancels(t *testing.T) {
	ctx := context.Background()
	mockClient := &MockClient{}
	mockCfnOps := &MockCloudFormationOperations{}
	mockPrompter := &MockPrompter{}

	// Set up the mock client to return our mock CloudFormation operations
	mockClient.On("NewCloudFormationOperations").Return(mockCfnOps)

	// Set up mock for stack existence check
	mockCfnOps.On("StackExists", ctx, "test-stack").Return(true, nil)

	// Set up mock for stack description
	stackInfo := &aws.StackInfo{
		Name:        "test-stack",
		Status:      aws.StackStatusCreateComplete,
		Description: "Test stack description",
	}
	mockCfnOps.On("DescribeStack", ctx, "test-stack").Return(stackInfo, nil)

	// Set up mock for user confirmation (user cancels)
	// Business logic sends core message, prompter adds formatting
	expectedMessage := "Do you want to delete stack test-stack? This cannot be undone."
	mockPrompter.On("Confirm", expectedMessage).Return(false, nil)

	// Set the mock prompter
	originalPrompter := prompt.GetDefaultPrompter()
	prompt.SetPrompter(mockPrompter)
	defer prompt.SetPrompter(originalPrompter)

	// Create deleter and test
	deleter := NewStackDeleter(mockClient)
	stack := &model.Stack{
		Name:    "test-stack",
		Context: "dev",
	}

	err := deleter.DeleteStack(ctx, stack)

	assert.NoError(t, err) // Should not error when user cancels
	mockClient.AssertExpectations(t)
	mockCfnOps.AssertExpectations(t)
	mockPrompter.AssertExpectations(t)
}

func TestDeleteStack_StackDoesNotExist(t *testing.T) {
	ctx := context.Background()
	mockClient := &MockClient{}
	mockCfnOps := &MockCloudFormationOperations{}

	// Set up the mock client to return our mock CloudFormation operations
	mockClient.On("NewCloudFormationOperations").Return(mockCfnOps)

	// Set up mock for stack existence check (stack doesn't exist)
	mockCfnOps.On("StackExists", ctx, "test-stack").Return(false, nil)

	// Create deleter and test
	deleter := NewStackDeleter(mockClient)
	stack := &model.Stack{
		Name:    "test-stack",
		Context: "dev",
	}

	err := deleter.DeleteStack(ctx, stack)

	assert.NoError(t, err) // Should not error when stack doesn't exist
	mockClient.AssertExpectations(t)
	mockCfnOps.AssertExpectations(t)
}

func TestDeleteStack_StackExistsCheckFails(t *testing.T) {
	ctx := context.Background()
	mockClient := &MockClient{}
	mockCfnOps := &MockCloudFormationOperations{}

	// Set up the mock client to return our mock CloudFormation operations
	mockClient.On("NewCloudFormationOperations").Return(mockCfnOps)

	// Set up mock for stack existence check failure
	mockCfnOps.On("StackExists", ctx, "test-stack").Return(false, errors.New("AWS error"))

	// Create deleter and test
	deleter := NewStackDeleter(mockClient)
	stack := &model.Stack{
		Name:    "test-stack",
		Context: "dev",
	}

	err := deleter.DeleteStack(ctx, stack)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to check if stack exists")
	mockClient.AssertExpectations(t)
	mockCfnOps.AssertExpectations(t)
}

func TestDeleteStack_DescribeStackFails(t *testing.T) {
	ctx := context.Background()
	mockClient := &MockClient{}
	mockCfnOps := &MockCloudFormationOperations{}

	// Set up the mock client to return our mock CloudFormation operations
	mockClient.On("NewCloudFormationOperations").Return(mockCfnOps)

	// Set up mock for stack existence check
	mockCfnOps.On("StackExists", ctx, "test-stack").Return(true, nil)

	// Set up mock for stack description failure
	mockCfnOps.On("DescribeStack", ctx, "test-stack").Return(nil, errors.New("AWS error"))

	// Create deleter and test
	deleter := NewStackDeleter(mockClient)
	stack := &model.Stack{
		Name:    "test-stack",
		Context: "dev",
	}

	err := deleter.DeleteStack(ctx, stack)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to describe stack")
	mockClient.AssertExpectations(t)
	mockCfnOps.AssertExpectations(t)
}

func TestDeleteStack_ConfirmationPromptFails(t *testing.T) {
	ctx := context.Background()
	mockClient := &MockClient{}
	mockCfnOps := &MockCloudFormationOperations{}
	mockPrompter := &MockPrompter{}

	// Set up the mock client to return our mock CloudFormation operations
	mockClient.On("NewCloudFormationOperations").Return(mockCfnOps)

	// Set up mock for stack existence check
	mockCfnOps.On("StackExists", ctx, "test-stack").Return(true, nil)

	// Set up mock for stack description
	stackInfo := &aws.StackInfo{
		Name:        "test-stack",
		Status:      aws.StackStatusCreateComplete,
		Description: "Test stack description",
	}
	mockCfnOps.On("DescribeStack", ctx, "test-stack").Return(stackInfo, nil)

	// Set up mock for user confirmation failure
	// Business logic sends core message, prompter adds formatting
	expectedMessage := "Do you want to delete stack test-stack? This cannot be undone."
	mockPrompter.On("Confirm", expectedMessage).Return(false, errors.New("prompt error"))

	// Set the mock prompter
	originalPrompter := prompt.GetDefaultPrompter()
	prompt.SetPrompter(mockPrompter)
	defer prompt.SetPrompter(originalPrompter)

	// Create deleter and test
	deleter := NewStackDeleter(mockClient)
	stack := &model.Stack{
		Name:    "test-stack",
		Context: "dev",
	}

	err := deleter.DeleteStack(ctx, stack)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get user confirmation")
	mockClient.AssertExpectations(t)
	mockCfnOps.AssertExpectations(t)
	mockPrompter.AssertExpectations(t)
}

func TestDeleteStack_DeleteStackFails(t *testing.T) {
	ctx := context.Background()
	mockClient := &MockClient{}
	mockCfnOps := &MockCloudFormationOperations{}
	mockPrompter := &MockPrompter{}

	// Set up the mock client to return our mock CloudFormation operations
	mockClient.On("NewCloudFormationOperations").Return(mockCfnOps)

	// Set up mock for stack existence check
	mockCfnOps.On("StackExists", ctx, "test-stack").Return(true, nil)

	// Set up mock for stack description
	stackInfo := &aws.StackInfo{
		Name:        "test-stack",
		Status:      aws.StackStatusCreateComplete,
		Description: "Test stack description",
	}
	mockCfnOps.On("DescribeStack", ctx, "test-stack").Return(stackInfo, nil)

	// Set up mock for user confirmation
	// Business logic sends core message, prompter adds formatting
	expectedMessage := "Do you want to delete stack test-stack? This cannot be undone."
	mockPrompter.On("Confirm", expectedMessage).Return(true, nil)

	// Set up mock for stack deletion failure
	deleteInput := aws.DeleteStackInput{StackName: "test-stack"}
	mockCfnOps.On("DeleteStack", ctx, deleteInput).Return(errors.New("AWS deletion error"))

	// Set the mock prompter
	originalPrompter := prompt.GetDefaultPrompter()
	prompt.SetPrompter(mockPrompter)
	defer prompt.SetPrompter(originalPrompter)

	// Create deleter and test
	deleter := NewStackDeleter(mockClient)
	stack := &model.Stack{
		Name:    "test-stack",
		Context: "dev",
	}

	err := deleter.DeleteStack(ctx, stack)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to delete stack")
	mockClient.AssertExpectations(t)
	mockCfnOps.AssertExpectations(t)
	mockPrompter.AssertExpectations(t)
}

func TestDeleteStack_WaitForOperationFails(t *testing.T) {
	ctx := context.Background()
	mockClient := &MockClient{}
	mockCfnOps := &MockCloudFormationOperations{}
	mockPrompter := &MockPrompter{}

	// Set up the mock client to return our mock CloudFormation operations
	mockClient.On("NewCloudFormationOperations").Return(mockCfnOps)

	// Set up mock for stack existence check
	mockCfnOps.On("StackExists", ctx, "test-stack").Return(true, nil)

	// Set up mock for stack description
	stackInfo := &aws.StackInfo{
		Name:        "test-stack",
		Status:      aws.StackStatusCreateComplete,
		Description: "Test stack description",
	}
	mockCfnOps.On("DescribeStack", ctx, "test-stack").Return(stackInfo, nil)

	// Set up mock for user confirmation
	// Business logic sends core message, prompter adds formatting
	expectedMessage := "Do you want to delete stack test-stack? This cannot be undone."
	mockPrompter.On("Confirm", expectedMessage).Return(true, nil)

	// Set up mock for stack deletion
	deleteInput := aws.DeleteStackInput{StackName: "test-stack"}
	mockCfnOps.On("DeleteStack", ctx, deleteInput).Return(nil)

	// Set up mock for waiting for deletion failure
	mockCfnOps.On("WaitForStackOperation", ctx, "test-stack", mock.AnythingOfType("func(aws.StackEvent)")).Return(errors.New("timeout error"))

	// Set the mock prompter
	originalPrompter := prompt.GetDefaultPrompter()
	prompt.SetPrompter(mockPrompter)
	defer prompt.SetPrompter(originalPrompter)

	// Create deleter and test
	deleter := NewStackDeleter(mockClient)
	stack := &model.Stack{
		Name:    "test-stack",
		Context: "dev",
	}

	err := deleter.DeleteStack(ctx, stack)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "stack deletion failed or timed out")
	mockClient.AssertExpectations(t)
	mockCfnOps.AssertExpectations(t)
	mockPrompter.AssertExpectations(t)
}
