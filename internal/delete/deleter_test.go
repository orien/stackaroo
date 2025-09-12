/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package delete

import (
	"context"
	"errors"
	"testing"

	"github.com/orien/stackaroo/internal/aws"
	"github.com/orien/stackaroo/internal/config"
	"github.com/orien/stackaroo/internal/model"
	"github.com/orien/stackaroo/internal/prompt"
	"github.com/orien/stackaroo/internal/resolve"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestNewStackDeleter(t *testing.T) {
	mockCfnOps := &aws.MockCloudFormationOperations{}
	deleter := NewStackDeleter(mockCfnOps, nil, nil)

	assert.NotNil(t, deleter)
	assert.Equal(t, mockCfnOps, deleter.cfnOps)
}

func TestDeleteStack_StackExists_UserConfirms_Success(t *testing.T) {
	ctx := context.Background()
	mockCfnOps := &aws.MockCloudFormationOperations{}
	mockPrompter := &prompt.MockPrompter{}

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
	deleter := NewStackDeleter(mockCfnOps, nil, nil)
	stack := &model.Stack{
		Name:    "test-stack",
		Context: "dev",
	}

	err := deleter.DeleteStack(ctx, stack)

	assert.NoError(t, err)
	mockCfnOps.AssertExpectations(t)
	mockPrompter.AssertExpectations(t)
}

func TestDeleteStack_StackExists_UserCancels(t *testing.T) {
	ctx := context.Background()
	mockCfnOps := &aws.MockCloudFormationOperations{}
	mockPrompter := &prompt.MockPrompter{}

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
	deleter := NewStackDeleter(mockCfnOps, nil, nil)
	stack := &model.Stack{
		Name:    "test-stack",
		Context: "dev",
	}

	err := deleter.DeleteStack(ctx, stack)

	assert.NoError(t, err) // Should not error when user cancels
	mockCfnOps.AssertExpectations(t)
	mockPrompter.AssertExpectations(t)
}

func TestDeleteStack_StackDoesNotExist(t *testing.T) {
	ctx := context.Background()
	mockCfnOps := &aws.MockCloudFormationOperations{}

	// Set up mock for stack existence check (stack doesn't exist)
	mockCfnOps.On("StackExists", ctx, "test-stack").Return(false, nil)

	// Create deleter and test
	deleter := NewStackDeleter(mockCfnOps, nil, nil)
	stack := &model.Stack{
		Name:    "test-stack",
		Context: "dev",
	}

	err := deleter.DeleteStack(ctx, stack)

	assert.NoError(t, err) // Should not error when stack doesn't exist
	mockCfnOps.AssertExpectations(t)
}

func TestDeleteStack_StackExistsCheckFails(t *testing.T) {
	ctx := context.Background()
	mockCfnOps := &aws.MockCloudFormationOperations{}

	// Set up mock for stack existence check failure
	mockCfnOps.On("StackExists", ctx, "test-stack").Return(false, errors.New("AWS error"))

	// Create deleter and test
	deleter := NewStackDeleter(mockCfnOps, nil, nil)
	stack := &model.Stack{
		Name:    "test-stack",
		Context: "dev",
	}

	err := deleter.DeleteStack(ctx, stack)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to check if stack exists")
	mockCfnOps.AssertExpectations(t)
}

func TestDeleteStack_DescribeStackFails(t *testing.T) {
	ctx := context.Background()
	mockCfnOps := &aws.MockCloudFormationOperations{}

	// Set up mock for stack existence check
	mockCfnOps.On("StackExists", ctx, "test-stack").Return(true, nil)

	// Set up mock for stack description failure
	mockCfnOps.On("DescribeStack", ctx, "test-stack").Return(nil, errors.New("AWS error"))

	// Create deleter and test
	deleter := NewStackDeleter(mockCfnOps, nil, nil)
	stack := &model.Stack{
		Name:    "test-stack",
		Context: "dev",
	}

	err := deleter.DeleteStack(ctx, stack)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to describe stack")
	mockCfnOps.AssertExpectations(t)
}

func TestDeleteSingleStack_Success(t *testing.T) {
	ctx := context.Background()
	mockCfnOps := &aws.MockCloudFormationOperations{}
	mockConfigProvider := &config.MockConfigProvider{}
	mockResolver := &resolve.MockResolver{}

	// Create test stack
	testStack := &model.Stack{
		Name:    "test-stack",
		Context: "dev",
	}

	// Mock resolver to return our test stack
	mockResolver.On("ResolveStack", ctx, "dev", "test-stack").Return(testStack, nil)

	// Mock CloudFormation operations for successful deletion
	mockCfnOps.On("StackExists", ctx, "test-stack").Return(true, nil)
	mockCfnOps.On("DescribeStack", ctx, "test-stack").Return(&aws.StackInfo{
		Status:      "CREATE_COMPLETE",
		Description: "Test stack",
	}, nil)
	mockCfnOps.On("DeleteStack", ctx, aws.DeleteStackInput{StackName: "test-stack"}).Return(nil)
	mockCfnOps.On("WaitForStackOperation", ctx, "test-stack", mock.AnythingOfType("func(aws.StackEvent)")).Return(nil)

	// Mock prompt for confirmation
	mockPrompter := &prompt.MockPrompter{}
	mockPrompter.On("Confirm", mock.AnythingOfType("string")).Return(true, nil)
	originalPrompter := prompt.GetDefaultPrompter()
	prompt.SetPrompter(mockPrompter)
	defer prompt.SetPrompter(originalPrompter)

	// Create deleter and test
	deleter := NewStackDeleter(mockCfnOps, mockConfigProvider, mockResolver)
	err := deleter.DeleteSingleStack(ctx, "test-stack", "dev")

	// Assertions
	assert.NoError(t, err)
	mockResolver.AssertExpectations(t)
	mockCfnOps.AssertExpectations(t)
	mockPrompter.AssertExpectations(t)
}

func TestDeleteSingleStack_ResolverFailure(t *testing.T) {
	ctx := context.Background()
	mockCfnOps := &aws.MockCloudFormationOperations{}
	mockConfigProvider := &config.MockConfigProvider{}
	mockResolver := &resolve.MockResolver{}

	// Mock resolver to return error
	mockResolver.On("ResolveStack", ctx, "dev", "test-stack").Return(nil, errors.New("stack not found"))

	// Create deleter and test
	deleter := NewStackDeleter(mockCfnOps, mockConfigProvider, mockResolver)
	err := deleter.DeleteSingleStack(ctx, "test-stack", "dev")

	// Assertions
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to resolve stack dependencies")
	mockResolver.AssertExpectations(t)
}

func TestDeleteSingleStack_DeleteStackFailure(t *testing.T) {
	ctx := context.Background()
	mockCfnOps := &aws.MockCloudFormationOperations{}
	mockConfigProvider := &config.MockConfigProvider{}
	mockResolver := &resolve.MockResolver{}

	// Create test stack
	testStack := &model.Stack{
		Name:    "test-stack",
		Context: "dev",
	}

	// Mock resolver to return our test stack
	mockResolver.On("ResolveStack", ctx, "dev", "test-stack").Return(testStack, nil)

	// Mock CloudFormation operations for failed deletion
	mockCfnOps.On("StackExists", ctx, "test-stack").Return(true, nil)
	mockCfnOps.On("DescribeStack", ctx, "test-stack").Return(&aws.StackInfo{
		Status:      "CREATE_COMPLETE",
		Description: "Test stack",
	}, nil)
	mockCfnOps.On("DeleteStack", ctx, aws.DeleteStackInput{StackName: "test-stack"}).Return(errors.New("deletion failed"))

	// Mock prompt for confirmation
	mockPrompter := &prompt.MockPrompter{}
	mockPrompter.On("Confirm", mock.AnythingOfType("string")).Return(true, nil)
	originalPrompter := prompt.GetDefaultPrompter()
	prompt.SetPrompter(mockPrompter)
	defer prompt.SetPrompter(originalPrompter)

	// Create deleter and test
	deleter := NewStackDeleter(mockCfnOps, mockConfigProvider, mockResolver)
	err := deleter.DeleteSingleStack(ctx, "test-stack", "dev")

	// Assertions
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error deleting stack test-stack")
	mockResolver.AssertExpectations(t)
	mockCfnOps.AssertExpectations(t)
	mockPrompter.AssertExpectations(t)
}

func TestDeleteAllStacks_Success(t *testing.T) {
	ctx := context.Background()
	mockCfnOps := &aws.MockCloudFormationOperations{}
	mockConfigProvider := &config.MockConfigProvider{}
	mockResolver := &resolve.MockResolver{}

	// Mock config provider to return stack list
	stackNames := []string{"vpc", "app"}
	mockConfigProvider.On("ListStacks", "dev").Return(stackNames, nil)

	// Mock resolver to return dependency order (app before vpc for deletion)
	deploymentOrder := []string{"vpc", "app"}
	mockResolver.On("GetDependencyOrder", "dev", stackNames).Return(deploymentOrder, nil)

	// Create test stacks in reverse order (deletion order: app, vpc)
	appStack := &model.Stack{Name: "app", Context: "dev"}
	vpcStack := &model.Stack{Name: "vpc", Context: "dev"}

	// Mock resolver for individual stack resolution
	mockResolver.On("ResolveStack", ctx, "dev", "app").Return(appStack, nil)
	mockResolver.On("ResolveStack", ctx, "dev", "vpc").Return(vpcStack, nil)

	// Mock CloudFormation operations for both stacks
	for _, stackName := range []string{"app", "vpc"} {
		mockCfnOps.On("StackExists", ctx, stackName).Return(true, nil)
		mockCfnOps.On("DescribeStack", ctx, stackName).Return(&aws.StackInfo{
			Status:      "CREATE_COMPLETE",
			Description: "Test stack",
		}, nil)
		mockCfnOps.On("DeleteStack", ctx, aws.DeleteStackInput{StackName: stackName}).Return(nil)
		mockCfnOps.On("WaitForStackOperation", ctx, stackName, mock.AnythingOfType("func(aws.StackEvent)")).Return(nil)
	}

	// Mock prompt for confirmation
	mockPrompter := &prompt.MockPrompter{}
	mockPrompter.On("Confirm", mock.AnythingOfType("string")).Return(true, nil).Twice()
	originalPrompter := prompt.GetDefaultPrompter()
	prompt.SetPrompter(mockPrompter)
	defer prompt.SetPrompter(originalPrompter)

	// Create deleter and test
	deleter := NewStackDeleter(mockCfnOps, mockConfigProvider, mockResolver)
	err := deleter.DeleteAllStacks(ctx, "dev")

	// Assertions
	assert.NoError(t, err)
	mockConfigProvider.AssertExpectations(t)
	mockResolver.AssertExpectations(t)
	mockCfnOps.AssertExpectations(t)
	mockPrompter.AssertExpectations(t)
}

func TestDeleteAllStacks_NoStacksFound(t *testing.T) {
	ctx := context.Background()
	mockCfnOps := &aws.MockCloudFormationOperations{}
	mockConfigProvider := &config.MockConfigProvider{}
	mockResolver := &resolve.MockResolver{}

	// Mock config provider to return empty stack list
	mockConfigProvider.On("ListStacks", "dev").Return([]string{}, nil)

	// Create deleter and test
	deleter := NewStackDeleter(mockCfnOps, mockConfigProvider, mockResolver)
	err := deleter.DeleteAllStacks(ctx, "dev")

	// Assertions
	assert.NoError(t, err)
	mockConfigProvider.AssertExpectations(t)
	// No other mocks should be called
}

func TestDeleteAllStacks_ListStacksFailure(t *testing.T) {
	ctx := context.Background()
	mockCfnOps := &aws.MockCloudFormationOperations{}
	mockConfigProvider := &config.MockConfigProvider{}
	mockResolver := &resolve.MockResolver{}

	// Mock config provider to return error
	mockConfigProvider.On("ListStacks", "dev").Return(nil, errors.New("context not found"))

	// Create deleter and test
	deleter := NewStackDeleter(mockCfnOps, mockConfigProvider, mockResolver)
	err := deleter.DeleteAllStacks(ctx, "dev")

	// Assertions
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get stacks for context dev")
	mockConfigProvider.AssertExpectations(t)
}

func TestDeleteAllStacks_GetDependencyOrderFailure(t *testing.T) {
	ctx := context.Background()
	mockCfnOps := &aws.MockCloudFormationOperations{}
	mockConfigProvider := &config.MockConfigProvider{}
	mockResolver := &resolve.MockResolver{}

	// Mock config provider to return stack list
	stackNames := []string{"vpc", "app"}
	mockConfigProvider.On("ListStacks", "dev").Return(stackNames, nil)

	// Mock resolver to return dependency order error (circular dependency)
	mockResolver.On("GetDependencyOrder", "dev", stackNames).Return(nil, errors.New("circular dependency detected"))

	// Create deleter and test
	deleter := NewStackDeleter(mockCfnOps, mockConfigProvider, mockResolver)
	err := deleter.DeleteAllStacks(ctx, "dev")

	// Assertions
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to calculate dependency order")
	mockConfigProvider.AssertExpectations(t)
	mockResolver.AssertExpectations(t)
}

func TestDeleteAllStacks_ResolveStackFailure(t *testing.T) {
	ctx := context.Background()
	mockCfnOps := &aws.MockCloudFormationOperations{}
	mockConfigProvider := &config.MockConfigProvider{}
	mockResolver := &resolve.MockResolver{}

	// Mock config provider to return stack list
	stackNames := []string{"vpc"}
	mockConfigProvider.On("ListStacks", "dev").Return(stackNames, nil)

	// Mock resolver to return dependency order
	deploymentOrder := []string{"vpc"}
	mockResolver.On("GetDependencyOrder", "dev", stackNames).Return(deploymentOrder, nil)

	// Mock resolver to fail on individual stack resolution
	mockResolver.On("ResolveStack", ctx, "dev", "vpc").Return(nil, errors.New("template not found"))

	// Create deleter and test
	deleter := NewStackDeleter(mockCfnOps, mockConfigProvider, mockResolver)
	err := deleter.DeleteAllStacks(ctx, "dev")

	// Assertions
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to resolve stack vpc")
	mockConfigProvider.AssertExpectations(t)
	mockResolver.AssertExpectations(t)
}

func TestDeleteAllStacks_DeleteStackFailureStopsExecution(t *testing.T) {
	ctx := context.Background()
	mockCfnOps := &aws.MockCloudFormationOperations{}
	mockConfigProvider := &config.MockConfigProvider{}
	mockResolver := &resolve.MockResolver{}

	// Mock config provider to return stack list
	stackNames := []string{"vpc", "app"}
	mockConfigProvider.On("ListStacks", "dev").Return(stackNames, nil)

	// Mock resolver to return dependency order (app before vpc for deletion)
	deploymentOrder := []string{"vpc", "app"}
	mockResolver.On("GetDependencyOrder", "dev", stackNames).Return(deploymentOrder, nil)

	// Create test stack (only need to resolve the first one that will fail)
	appStack := &model.Stack{Name: "app", Context: "dev"}
	mockResolver.On("ResolveStack", ctx, "dev", "app").Return(appStack, nil)

	// Mock CloudFormation operations for first stack to fail
	mockCfnOps.On("StackExists", ctx, "app").Return(true, nil)
	mockCfnOps.On("DescribeStack", ctx, "app").Return(&aws.StackInfo{
		Status:      "CREATE_COMPLETE",
		Description: "Test stack",
	}, nil)
	mockCfnOps.On("DeleteStack", ctx, aws.DeleteStackInput{StackName: "app"}).Return(errors.New("deletion failed"))

	// Mock prompt for confirmation
	mockPrompter := &prompt.MockPrompter{}
	mockPrompter.On("Confirm", mock.AnythingOfType("string")).Return(true, nil)
	originalPrompter := prompt.GetDefaultPrompter()
	prompt.SetPrompter(mockPrompter)
	defer prompt.SetPrompter(originalPrompter)

	// Create deleter and test
	deleter := NewStackDeleter(mockCfnOps, mockConfigProvider, mockResolver)
	err := deleter.DeleteAllStacks(ctx, "dev")

	// Assertions
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error deleting stack app")
	mockConfigProvider.AssertExpectations(t)
	mockResolver.AssertExpectations(t)
	mockCfnOps.AssertExpectations(t)
	mockPrompter.AssertExpectations(t)

	// Verify that vpc stack resolution was NOT called (execution stopped after app failed)
	mockResolver.AssertNotCalled(t, "ResolveStack", ctx, "dev", "vpc")
}

func TestDeleteStack_ConfirmationPromptFails(t *testing.T) {
	ctx := context.Background()
	mockCfnOps := &aws.MockCloudFormationOperations{}
	mockPrompter := &prompt.MockPrompter{}

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
	deleter := NewStackDeleter(mockCfnOps, nil, nil)
	stack := &model.Stack{
		Name:    "test-stack",
		Context: "dev",
	}

	err := deleter.DeleteStack(ctx, stack)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get user confirmation")
	mockCfnOps.AssertExpectations(t)
	mockPrompter.AssertExpectations(t)
}

func TestDeleteStack_DeleteStackFails(t *testing.T) {
	ctx := context.Background()
	mockCfnOps := &aws.MockCloudFormationOperations{}
	mockPrompter := &prompt.MockPrompter{}

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
	deleter := NewStackDeleter(mockCfnOps, nil, nil)
	stack := &model.Stack{
		Name:    "test-stack",
		Context: "dev",
	}

	err := deleter.DeleteStack(ctx, stack)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to delete stack")
	mockCfnOps.AssertExpectations(t)
	mockPrompter.AssertExpectations(t)
}

func TestDeleteStack_WaitForOperationFails(t *testing.T) {
	ctx := context.Background()
	mockCfnOps := &aws.MockCloudFormationOperations{}
	mockPrompter := &prompt.MockPrompter{}

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
	deleter := NewStackDeleter(mockCfnOps, nil, nil)
	stack := &model.Stack{
		Name:    "test-stack",
		Context: "dev",
	}

	err := deleter.DeleteStack(ctx, stack)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "stack deletion failed or timed out")
	mockCfnOps.AssertExpectations(t)
	mockPrompter.AssertExpectations(t)
}
