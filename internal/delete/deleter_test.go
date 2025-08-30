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
	"github.com/orien/stackaroo/internal/model"
	"github.com/orien/stackaroo/internal/prompt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestNewStackDeleter(t *testing.T) {
	mockCfnOps := &aws.MockCloudFormationOperations{}
	deleter := NewStackDeleter(mockCfnOps)

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
	deleter := NewStackDeleter(mockCfnOps)
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
	deleter := NewStackDeleter(mockCfnOps)
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
	deleter := NewStackDeleter(mockCfnOps)
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
	deleter := NewStackDeleter(mockCfnOps)
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
	deleter := NewStackDeleter(mockCfnOps)
	stack := &model.Stack{
		Name:    "test-stack",
		Context: "dev",
	}

	err := deleter.DeleteStack(ctx, stack)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to describe stack")
	mockCfnOps.AssertExpectations(t)
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
	deleter := NewStackDeleter(mockCfnOps)
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
	deleter := NewStackDeleter(mockCfnOps)
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
	deleter := NewStackDeleter(mockCfnOps)
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
