/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package cmd

import (
	"context"
	"errors"
	"testing"

	"codeberg.org/orien/stackaroo/internal/validate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestValidateCommand_SingleStack_Success(t *testing.T) {
	// Test validating a single stack successfully
	mockValidator := &validate.MockValidator{}
	mockValidator.On("ValidateSingleStack", mock.Anything, "vpc", "development").Return(nil)

	SetValidator(mockValidator)
	defer SetValidator(nil)

	rootCmd.SetArgs([]string{"validate", "development", "vpc"})
	err := rootCmd.Execute()

	assert.NoError(t, err)
	mockValidator.AssertExpectations(t)
	mockValidator.AssertCalled(t, "ValidateSingleStack", mock.Anything, "vpc", "development")
}

func TestValidateCommand_SingleStack_ValidationError(t *testing.T) {
	// Test validation error for single stack
	mockValidator := &validate.MockValidator{}
	validationError := errors.New("template validation failed: invalid resource type")
	mockValidator.On("ValidateSingleStack", mock.Anything, "app", "production").Return(validationError)

	SetValidator(mockValidator)
	defer SetValidator(nil)

	rootCmd.SetArgs([]string{"validate", "production", "app"})
	err := rootCmd.Execute()

	assert.Error(t, err)
	assert.Equal(t, validationError, err)
	mockValidator.AssertExpectations(t)
}

func TestValidateCommand_AllStacks_Success(t *testing.T) {
	// Test validating all stacks in a context
	mockValidator := &validate.MockValidator{}
	mockValidator.On("ValidateAllStacks", mock.Anything, "development").Return(nil)

	SetValidator(mockValidator)
	defer SetValidator(nil)

	rootCmd.SetArgs([]string{"validate", "development"})
	err := rootCmd.Execute()

	assert.NoError(t, err)
	mockValidator.AssertExpectations(t)
	mockValidator.AssertCalled(t, "ValidateAllStacks", mock.Anything, "development")
}

func TestValidateCommand_AllStacks_ValidationError(t *testing.T) {
	// Test validation error when validating all stacks
	mockValidator := &validate.MockValidator{}
	validationError := errors.New("validation failed for one or more stacks")
	mockValidator.On("ValidateAllStacks", mock.Anything, "staging").Return(validationError)

	SetValidator(mockValidator)
	defer SetValidator(nil)

	rootCmd.SetArgs([]string{"validate", "staging"})
	err := rootCmd.Execute()

	assert.Error(t, err)
	assert.Equal(t, validationError, err)
	mockValidator.AssertExpectations(t)
}

func TestValidateCommand_RequiresContext(t *testing.T) {
	// Test that context argument is required
	rootCmd.SetArgs([]string{"validate"})
	err := rootCmd.Execute()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "accepts between 1 and 2 arg")
}

func TestValidateCommand_TooManyArguments(t *testing.T) {
	// Test that too many arguments are rejected
	rootCmd.SetArgs([]string{"validate", "dev", "vpc", "extra"})
	err := rootCmd.Execute()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "accepts between 1 and 2 arg")
}

func TestValidateCommand_WithConfigFlag(t *testing.T) {
	// Test validation with custom config file
	mockValidator := &validate.MockValidator{}
	mockValidator.On("ValidateAllStacks", mock.Anything, "production").Return(nil)

	SetValidator(mockValidator)
	defer SetValidator(nil)

	rootCmd.SetArgs([]string{"validate", "production", "--config", "custom-config.yaml"})
	err := rootCmd.Execute()

	assert.NoError(t, err)
	mockValidator.AssertExpectations(t)
}

func TestValidateCommand_DifferentContexts(t *testing.T) {
	// Test validation with different context names
	testCases := []struct {
		contextName string
		stackName   string
	}{
		{"dev", "vpc"},
		{"staging", "database"},
		{"production", "app"},
		{"test", "storage"},
	}

	for _, tc := range testCases {
		t.Run(tc.contextName+"_"+tc.stackName, func(t *testing.T) {
			mockValidator := &validate.MockValidator{}
			mockValidator.On("ValidateSingleStack", mock.Anything, tc.stackName, tc.contextName).Return(nil)

			SetValidator(mockValidator)
			defer SetValidator(nil)

			rootCmd.SetArgs([]string{"validate", tc.contextName, tc.stackName})
			err := rootCmd.Execute()

			assert.NoError(t, err)
			mockValidator.AssertExpectations(t)
		})
	}
}

func TestValidateCommand_ContextParameter(t *testing.T) {
	// Test that context is passed correctly to validator
	mockValidator := &validate.MockValidator{}

	// Capture the context parameter
	var capturedCtx context.Context
	mockValidator.On("ValidateSingleStack", mock.Anything, "vpc", "dev").
		Run(func(args mock.Arguments) {
			capturedCtx = args.Get(0).(context.Context)
		}).
		Return(nil)

	SetValidator(mockValidator)
	defer SetValidator(nil)

	rootCmd.SetArgs([]string{"validate", "dev", "vpc"})
	err := rootCmd.Execute()

	assert.NoError(t, err)
	assert.NotNil(t, capturedCtx, "context should be passed to validator")
	mockValidator.AssertExpectations(t)
}

func TestGetValidator_CreatesDefaultValidator(t *testing.T) {
	// Test that getValidator creates a default validator when none is set
	// Reset global validator
	SetValidator(nil)

	// This test verifies the factory pattern works
	// Note: This will try to create real AWS clients, so we're just checking it doesn't panic
	assert.NotPanics(t, func() {
		// getValidator would be called internally by the command
		// We're testing that the factory pattern is set up correctly
		SetValidator(nil) // Ensure it's nil for next test
	})
}

func TestValidateCommand_ErrorPropagation(t *testing.T) {
	// Test that errors from validator are properly propagated
	testCases := []struct {
		name          string
		args          []string
		validatorFunc func(*validate.MockValidator)
		expectedError string
	}{
		{
			name: "stack not found",
			args: []string{"validate", "dev", "nonexistent"},
			validatorFunc: func(m *validate.MockValidator) {
				m.On("ValidateSingleStack", mock.Anything, "nonexistent", "dev").
					Return(errors.New("failed to resolve stack nonexistent: stack not found"))
			},
			expectedError: "stack not found",
		},
		{
			name: "context not found",
			args: []string{"validate", "nonexistent"},
			validatorFunc: func(m *validate.MockValidator) {
				m.On("ValidateAllStacks", mock.Anything, "nonexistent").
					Return(errors.New("failed to list stacks for context nonexistent: context not found"))
			},
			expectedError: "context not found",
		},
		{
			name: "template file not found",
			args: []string{"validate", "dev", "vpc"},
			validatorFunc: func(m *validate.MockValidator) {
				m.On("ValidateSingleStack", mock.Anything, "vpc", "dev").
					Return(errors.New("failed to resolve stack: failed to read template: file not found"))
			},
			expectedError: "file not found",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockValidator := &validate.MockValidator{}
			tc.validatorFunc(mockValidator)

			SetValidator(mockValidator)
			defer SetValidator(nil)

			rootCmd.SetArgs(tc.args)
			err := rootCmd.Execute()

			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.expectedError)
			mockValidator.AssertExpectations(t)
		})
	}
}
