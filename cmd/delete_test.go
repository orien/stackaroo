/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package cmd

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/orien/stackaroo/internal/delete"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

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
	mockDeleter := &delete.MockDeleter{}

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
	mockDeleter := &delete.MockDeleter{}

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
  vpc:
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

	// Set up mock expectations for the new DeleteSingleStack method
	mockDeleter.On("DeleteSingleStack", mock.Anything, "vpc", "dev").Return(nil)

	// Execute command
	rootCmd.SetArgs([]string{"delete", "dev", "vpc"})
	err = rootCmd.Execute()

	// Assertions
	require.NoError(t, err)
	mockDeleter.AssertExpectations(t)
}

func TestDeleteCommand_DeleteAllStacksInContext(t *testing.T) {
	// Test deleting all stacks in a context with dependency ordering
	mockDeleter := &delete.MockDeleter{}

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
  vpc:
    template: templates/vpc.yaml
    parameters:
      VpcCidr: 10.0.0.0/16
  app:
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
	// Set up mock expectations
	mockDeleter.On("DeleteAllStacks", mock.Anything, "dev").Return(nil)

	// Execute command
	rootCmd.SetArgs([]string{"delete", "dev"})
	err = rootCmd.Execute()

	// Assertions
	require.NoError(t, err)
	mockDeleter.AssertExpectations(t)
}

func TestDeleteCommand_DeletionFails(t *testing.T) {
	// Test handling of deletion failure
	mockDeleter := &delete.MockDeleter{}

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
  vpc:
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
	mockDeleter.On("DeleteSingleStack", mock.Anything, "vpc", "dev").Return(errors.New("deletion failed"))

	// Execute command
	rootCmd.SetArgs([]string{"delete", "dev", "vpc"})
	err = rootCmd.Execute()

	// Assertions
	require.Error(t, err)
	assert.Contains(t, err.Error(), "deletion failed")
	mockDeleter.AssertExpectations(t)
}

func TestDeleteCommand_NoStacksInContext(t *testing.T) {
	// Test handling when no stacks exist in context
	mockDeleter := &delete.MockDeleter{}

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

	// Set up mock expectations for DeleteAllStacks
	mockDeleter.On("DeleteAllStacks", mock.Anything, "dev").Return(nil)

	// Execute command
	rootCmd.SetArgs([]string{"delete", "dev"})
	err = rootCmd.Execute()

	// Should succeed with no stacks to delete
	require.NoError(t, err)
	mockDeleter.AssertExpectations(t)
}

func TestDeleteCommand_InvalidContext(t *testing.T) {
	// Test handling of invalid context
	mockDeleter := &delete.MockDeleter{}

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
  vpc:
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

	// Set up mock expectations for DeleteAllStacks with error
	mockDeleter.On("DeleteAllStacks", mock.Anything, "invalid-context").Return(errors.New("failed to get stacks for context invalid-context"))

	// Execute command with invalid context
	rootCmd.SetArgs([]string{"delete", "invalid-context"})
	err = rootCmd.Execute()

	// Assertions
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get stacks for context")
	mockDeleter.AssertExpectations(t)
}

func TestDeleteCommand_StackNotFound(t *testing.T) {
	// Test handling when requested stack doesn't exist
	mockDeleter := &delete.MockDeleter{}

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
  vpc:
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

	// Set up mock expectations for DeleteSingleStack to return error
	mockDeleter.On("DeleteSingleStack", mock.Anything, "non-existent-stack", "dev").Return(errors.New("failed to resolve stack dependencies"))

	// Execute command with non-existent stack
	rootCmd.SetArgs([]string{"delete", "dev", "non-existent-stack"})
	err = rootCmd.Execute()

	// Assertions
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to resolve stack dependencies")
	mockDeleter.AssertExpectations(t)
}

func TestDeleteCommand_ComplexDependencyOrder(t *testing.T) {
	// Test deletion with complex dependency chain
	mockDeleter := &delete.MockDeleter{}

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
  vpc:
    template: templates/vpc.yaml

  database:
    template: templates/database.yaml
    depends_on: [vpc]

  app:
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

	// Set up mock expectations for DeleteAllStacks
	mockDeleter.On("DeleteAllStacks", mock.Anything, "dev").Return(nil)

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
	result := getDeleter("stackaroo.yaml")
	assert.NotNil(t, result)

	// Test that getDeleter returns the same instance
	result2 := getDeleter("stackaroo.yaml")
	assert.Equal(t, result, result2)
}

func TestSetDeleter(t *testing.T) {
	// Set up mock deleter
	mockDeleter := &delete.MockDeleter{}
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
