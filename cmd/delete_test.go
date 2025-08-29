/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package cmd

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/orien/stackaroo/internal/delete"
	"github.com/orien/stackaroo/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockDeleter implements delete.Deleter for testing
type MockDeleter struct {
	mock.Mock
}

// Ensure MockDeleter implements delete.Deleter interface
var _ delete.Deleter = (*MockDeleter)(nil)

func (m *MockDeleter) DeleteStack(ctx context.Context, stack *model.Stack) error {
	args := m.Called(ctx, stack)
	return args.Error(0)
}

func TestDeleteCommand_Exists(t *testing.T) {
	// Test that delete command is registered with root command
	deleteCmd := findCommand(rootCmd, "delete")

	assert.NotNil(t, deleteCmd, "delete command should be registered")
	assert.Equal(t, "delete <context> [stack-name]", deleteCmd.Use)
}

func TestDeleteCommand_AcceptsStackName(t *testing.T) {
	// Test that delete command accepts a stack name argument
	deleteCmd := findCommand(rootCmd, "delete")
	assert.NotNil(t, deleteCmd)

	// Test that Args validation is set
	assert.NotNil(t, deleteCmd.Args, "delete command should have Args validation set")
}

func TestDeleteCommand_AcceptsTwoArgs(t *testing.T) {
	// Test that delete command accepts one or two arguments (context and optional stack name)
	deleteCmd := findCommand(rootCmd, "delete")
	assert.NotNil(t, deleteCmd)

	// Test that Args validation accepts 1-2 arguments
	err := deleteCmd.Args(deleteCmd, []string{"dev", "vpc"})
	assert.NoError(t, err, "Two arguments should be valid")

	err = deleteCmd.Args(deleteCmd, []string{"dev"})
	assert.NoError(t, err, "One argument should be valid")

	err = deleteCmd.Args(deleteCmd, []string{})
	assert.Error(t, err, "No arguments should be invalid")
}

func TestDeleteCommand_RequiresAtLeastOneArg(t *testing.T) {
	// Test that delete command requires at least a context argument

	// Mock deleter that shouldn't be called
	mockDeleter := &MockDeleter{}

	oldDeleter := deleter
	SetDeleter(mockDeleter)
	defer SetDeleter(oldDeleter)

	// Execute with no arguments - should fail
	rootCmd.SetArgs([]string{"delete"})

	err := rootCmd.Execute()
	assert.Error(t, err, "delete command should require at least a context argument")
	assert.Contains(t, err.Error(), "accepts between 1 and 2 arg(s), received 0")

	// Verify no deleter calls were made
	mockDeleter.AssertExpectations(t)
}

func TestDeleteCommand_DeleteSingleStack(t *testing.T) {
	// Test deleting a single stack
	mockDeleter := &MockDeleter{}

	oldDeleter := deleter
	SetDeleter(mockDeleter)
	defer SetDeleter(oldDeleter)

	// Create temporary config and template files
	configContent := `
project: test-project
region: us-east-1

contexts:
  dev:
    account: "123456789012"
    region: us-west-2

stacks:
  - name: vpc
    template: templates/vpc.yaml
    parameters:
      VpcCidr: 10.0.0.0/16
`

	tmpDir := createTempConfigWithTemplates(t, configContent, []string{"vpc.yaml"})

	// Change to temp directory
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	err = os.Chdir(tmpDir)
	require.NoError(t, err)
	defer func() {
		err := os.Chdir(oldWd)
		require.NoError(t, err)
	}()

	// Set up expected stack
	expectedStack := &model.Stack{
		Name:    "vpc",
		Context: "dev",
	}

	// Set up mock expectations
	mockDeleter.On("DeleteStack", mock.Anything, mock.MatchedBy(func(stack *model.Stack) bool {
		return stack.Name == expectedStack.Name && stack.Context == expectedStack.Context
	})).Return(nil)

	// Execute command
	rootCmd.SetArgs([]string{"delete", "dev", "vpc"})
	err = rootCmd.Execute()

	// Assertions
	require.NoError(t, err)
	mockDeleter.AssertExpectations(t)
}

func TestDeleteCommand_DeleteAllStacksInContext(t *testing.T) {
	// Test deleting all stacks in a context with dependency ordering
	mockDeleter := &MockDeleter{}

	oldDeleter := deleter
	SetDeleter(mockDeleter)
	defer SetDeleter(oldDeleter)

	// Create temporary config and template files with dependencies
	configContent := `
project: test-project
region: us-east-1

contexts:
  dev:
    account: "123456789012"
    region: us-west-2

stacks:
  - name: vpc
    template: templates/vpc.yaml
    parameters:
      VpcCidr: 10.0.0.0/16
  - name: app
    template: templates/app.yaml
    depends_on:
      - vpc
    parameters:
      AppName: test-app
`

	tmpDir := createTempConfigWithTemplates(t, configContent, []string{"vpc.yaml", "app.yaml"})

	// Change to temp directory
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	err = os.Chdir(tmpDir)
	require.NoError(t, err)
	defer func() {
		err := os.Chdir(oldWd)
		require.NoError(t, err)
	}()

	// Set up mock expectations - app should be deleted before vpc (reverse dependency order)
	mockDeleter.On("DeleteStack", mock.Anything, mock.MatchedBy(func(stack *model.Stack) bool {
		return stack.Name == "app" && stack.Context == "dev"
	})).Return(nil).Once()

	mockDeleter.On("DeleteStack", mock.Anything, mock.MatchedBy(func(stack *model.Stack) bool {
		return stack.Name == "vpc" && stack.Context == "dev"
	})).Return(nil).Once()

	// Execute command
	rootCmd.SetArgs([]string{"delete", "dev"})
	err = rootCmd.Execute()

	// Assertions
	require.NoError(t, err)
	mockDeleter.AssertExpectations(t)
}

func TestDeleteCommand_DeletionFails(t *testing.T) {
	// Test handling of deletion failure
	mockDeleter := &MockDeleter{}

	oldDeleter := deleter
	SetDeleter(mockDeleter)
	defer SetDeleter(oldDeleter)

	// Create temporary config and template files
	configContent := `
project: test-project
region: us-east-1

contexts:
  dev:
    account: "123456789012"
    region: us-west-2

stacks:
  - name: vpc
    template: templates/vpc.yaml
    parameters:
      VpcCidr: 10.0.0.0/16
`

	tmpDir := createTempConfigWithTemplates(t, configContent, []string{"vpc.yaml"})

	// Change to temp directory
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	err = os.Chdir(tmpDir)
	require.NoError(t, err)
	defer func() {
		err := os.Chdir(oldWd)
		require.NoError(t, err)
	}()

	// Set up mock expectations with error
	mockDeleter.On("DeleteStack", mock.Anything, mock.MatchedBy(func(stack *model.Stack) bool {
		return stack.Name == "vpc" && stack.Context == "dev"
	})).Return(errors.New("deletion failed"))

	// Execute command
	rootCmd.SetArgs([]string{"delete", "dev", "vpc"})
	err = rootCmd.Execute()

	// Assertions
	require.Error(t, err)
	assert.Contains(t, err.Error(), "error deleting stack vpc")
	mockDeleter.AssertExpectations(t)
}

func TestDeleteCommand_NoStacksInContext(t *testing.T) {
	// Test handling when no stacks exist in context
	mockDeleter := &MockDeleter{}

	oldDeleter := deleter
	SetDeleter(mockDeleter)
	defer SetDeleter(oldDeleter)

	// Create temporary config with empty stacks
	configContent := `
project: test-project
region: us-east-1

contexts:
  dev:
    account: "123456789012"
    region: us-west-2

stacks: []
`

	tmpDir := createTempConfigWithTemplates(t, configContent, []string{})

	// Change to temp directory
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	err = os.Chdir(tmpDir)
	require.NoError(t, err)
	defer func() {
		err := os.Chdir(oldWd)
		require.NoError(t, err)
	}()

	// Execute command
	rootCmd.SetArgs([]string{"delete", "dev"})
	err = rootCmd.Execute()

	// Should succeed with no stacks to delete
	require.NoError(t, err)
	// Should not call DeleteStack
	mockDeleter.AssertNotCalled(t, "DeleteStack")
}

func TestDeleteCommand_InvalidContext(t *testing.T) {
	// Test handling of invalid context
	mockDeleter := &MockDeleter{}

	oldDeleter := deleter
	SetDeleter(mockDeleter)
	defer SetDeleter(oldDeleter)

	// Create temporary config and template files
	configContent := `
project: test-project
region: us-east-1

contexts:
  dev:
    account: "123456789012"
    region: us-west-2

stacks:
  - name: vpc
    template: templates/vpc.yaml
    parameters:
      VpcCidr: 10.0.0.0/16
`

	tmpDir := createTempConfigWithTemplates(t, configContent, []string{"vpc.yaml"})

	// Change to temp directory
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	err = os.Chdir(tmpDir)
	require.NoError(t, err)
	defer func() {
		err := os.Chdir(oldWd)
		require.NoError(t, err)
	}()

	// Execute command with invalid context
	rootCmd.SetArgs([]string{"delete", "invalid-context"})
	err = rootCmd.Execute()

	// Assertions
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get stacks for context")
	mockDeleter.AssertNotCalled(t, "DeleteStack")
}

func TestDeleteCommand_StackNotFound(t *testing.T) {
	// Test handling when requested stack doesn't exist
	mockDeleter := &MockDeleter{}

	oldDeleter := deleter
	SetDeleter(mockDeleter)
	defer SetDeleter(oldDeleter)

	// Create temporary config and template files
	configContent := `
project: test-project
region: us-east-1

contexts:
  dev:
    account: "123456789012"
    region: us-west-2

stacks:
  - name: vpc
    template: templates/vpc.yaml
    parameters:
      VpcCidr: 10.0.0.0/16
`

	tmpDir := createTempConfigWithTemplates(t, configContent, []string{"vpc.yaml"})

	// Change to temp directory
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	err = os.Chdir(tmpDir)
	require.NoError(t, err)
	defer func() {
		err := os.Chdir(oldWd)
		require.NoError(t, err)
	}()

	// Execute command with non-existent stack
	rootCmd.SetArgs([]string{"delete", "dev", "non-existent-stack"})
	err = rootCmd.Execute()

	// Assertions
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to resolve stack dependencies")
	mockDeleter.AssertNotCalled(t, "DeleteStack")
}

func TestDeleteCommand_ComplexDependencyOrder(t *testing.T) {
	// Test deletion with complex dependency chain
	mockDeleter := &MockDeleter{}

	oldDeleter := deleter
	SetDeleter(mockDeleter)
	defer SetDeleter(oldDeleter)

	// Create config with complex dependencies: app -> database -> vpc
	configContent := `
project: test-project
region: us-east-1

contexts:
  dev:
    account: "123456789012"
    region: us-east-1

stacks:
  - name: vpc
    template: templates/vpc.yaml

  - name: database
    template: templates/database.yaml
    depends_on: [vpc]

  - name: app
    template: templates/app.yaml
    depends_on: [database]
`

	tmpDir := createTempConfigWithTemplates(t, configContent, []string{"vpc.yaml", "database.yaml", "app.yaml"})

	// Change to temp directory
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	err = os.Chdir(tmpDir)
	require.NoError(t, err)
	defer func() {
		err := os.Chdir(oldWd)
		require.NoError(t, err)
	}()

	// Set up expectations for reverse order: app -> database -> vpc
	deleteOrder := []string{"app", "database", "vpc"}
	for _, stackName := range deleteOrder {
		mockDeleter.On("DeleteStack", mock.Anything, mock.MatchedBy(func(stack *model.Stack) bool {
			return stack.Name == stackName && stack.Context == "dev"
		})).Return(nil).Once()
	}

	// Execute command to delete all stacks
	rootCmd.SetArgs([]string{"delete", "dev"})
	err = rootCmd.Execute()

	// Assertions
	require.NoError(t, err)
	mockDeleter.AssertExpectations(t)
}

func TestGetDeleter(t *testing.T) {
	// Clear any existing deleter
	originalDeleter := deleter
	deleter = nil
	defer func() {
		deleter = originalDeleter
	}()

	// Test that getDeleter creates a default deleter
	result := getDeleter()
	assert.NotNil(t, result)

	// Test that getDeleter returns the same instance
	result2 := getDeleter()
	assert.Equal(t, result, result2)
}

func TestSetDeleter(t *testing.T) {
	// Set up mock deleter
	mockDeleter := &MockDeleter{}
	originalDeleter := deleter

	// Test setting the deleter
	SetDeleter(mockDeleter)
	assert.Equal(t, mockDeleter, deleter)

	// Restore original deleter
	deleter = originalDeleter
}

// Helper function to create temporary config with templates for testing
func createTempConfigWithTemplates(t *testing.T, configContent string, templateNames []string) string {
	tmpDir := t.TempDir()

	// Write config file
	configPath := filepath.Join(tmpDir, "stackaroo.yaml")
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	// Create templates directory if we have templates
	if len(templateNames) > 0 {
		templatesDir := filepath.Join(tmpDir, "templates")
		err = os.MkdirAll(templatesDir, 0755)
		require.NoError(t, err)

		// Create template files with minimal valid CloudFormation content
		templateContent := `{"AWSTemplateFormatVersion": "2010-09-09", "Resources": {}}`
		for _, templateName := range templateNames {
			templatePath := filepath.Join(templatesDir, templateName)
			err = os.WriteFile(templatePath, []byte(templateContent), 0644)
			require.NoError(t, err)
		}
	}

	return tmpDir
}
