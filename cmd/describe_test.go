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
	"time"

	"codeberg.org/orien/stackaroo/internal/describe"
	"codeberg.org/orien/stackaroo/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockDescriber implements the describe.Describer interface for testing
type MockDescriber struct {
	mock.Mock
}

func (m *MockDescriber) DescribeStack(ctx context.Context, stack *model.Stack) (*describe.StackDescription, error) {
	args := m.Called(ctx, stack)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*describe.StackDescription), args.Error(1)
}

func TestDescribeCommand_Exists(t *testing.T) {
	// Test that describe command is registered with root command
	describeCmd := findCommand(rootCmd, "describe")

	assert.NotNil(t, describeCmd, "describe command should be registered")
	assert.Equal(t, "describe <context> <stack-name>", describeCmd.Use)
}

func TestDescribeCommand_RequiresExactlyTwoArgs(t *testing.T) {
	// Test that describe command requires exactly two arguments
	describeCmd := findCommand(rootCmd, "describe")
	assert.NotNil(t, describeCmd)

	// Test that Args validation accepts exactly 2 arguments
	err := describeCmd.Args(describeCmd, []string{"dev", "vpc"})
	assert.NoError(t, err, "Two arguments should be valid")

	// Test that 1 argument is invalid
	err = describeCmd.Args(describeCmd, []string{"dev"})
	assert.Error(t, err, "One argument should be invalid")

	// Test that 3 arguments is invalid
	err = describeCmd.Args(describeCmd, []string{"dev", "vpc", "extra"})
	assert.Error(t, err, "Three arguments should be invalid")

	// Test that no arguments is invalid
	err = describeCmd.Args(describeCmd, []string{})
	assert.Error(t, err, "No arguments should be invalid")
}

func TestDescribeCommand_WithMockDescriber_Success(t *testing.T) {
	// Test successful describe operation with mock describer

	// Create temporary config file
	configContent := `
project: test-project
region: us-east-1

contexts:
  dev:
    account: "123456789012"
    region: us-west-2
    tags:
      Environment: dev

stacks:
  test-stack:
    template: templates/test-stack.yaml
    parameters:
      Environment: dev
    contexts:
      dev:
        parameters:
          Environment: dev
`

	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "stackaroo.yaml")
	templateFile := filepath.Join(tmpDir, "templates", "test-stack.yaml")

	// Create config file
	err := os.WriteFile(configFile, []byte(configContent), 0644)
	require.NoError(t, err)

	// Create template directory and file
	err = os.MkdirAll(filepath.Dir(templateFile), 0755)
	require.NoError(t, err)
	templateContent := `{
		"AWSTemplateFormatVersion": "2010-09-09",
		"Resources": {
			"TestResource": {
				"Type": "AWS::CloudFormation::WaitConditionHandle"
			}
		}
	}`
	err = os.WriteFile(templateFile, []byte(templateContent), 0644)
	require.NoError(t, err)

	// Create mock describer
	mockDescriber := &MockDescriber{}

	// Expected stack description
	expectedDesc := &describe.StackDescription{
		Name:        "test-stack",
		Status:      "CREATE_COMPLETE",
		CreatedTime: time.Now(),
		Parameters: map[string]string{
			"Environment": "dev",
		},
		Outputs: map[string]string{
			"VpcId": "vpc-12345678",
		},
		Tags: map[string]string{
			"Environment": "dev",
		},
	}

	// Set up expectations
	mockDescriber.On("DescribeStack", mock.Anything, mock.AnythingOfType("*model.Stack")).Return(expectedDesc, nil)

	// Inject mock describer
	oldDescriber := describer
	SetDescriber(mockDescriber)
	defer SetDescriber(oldDescriber)

	// Change to temp directory so config file is found
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	err = os.Chdir(tmpDir)
	require.NoError(t, err)
	defer func() {
		err := os.Chdir(oldWd)
		require.NoError(t, err)
	}()

	// Execute command
	rootCmd.SetArgs([]string{"describe", "dev", "test-stack"})
	err = rootCmd.Execute()

	// Verify success
	assert.NoError(t, err, "describe command should execute successfully")

	// Verify mock expectations
	mockDescriber.AssertExpectations(t)
}

func TestDescribeCommand_HandlesDescriberError(t *testing.T) {
	// Test error handling when describer fails

	// Create temporary config file
	configContent := `
project: test-project
region: us-east-1

contexts:
  dev:
    account: "123456789012"
    region: us-west-2
    tags:
      Environment: dev

stacks:
  nonexistent-stack:
    template: templates/nonexistent-stack.yaml
    parameters:
      Environment: dev
    contexts:
      dev:
        parameters:
          Environment: dev
`

	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "stackaroo.yaml")
	templateFile := filepath.Join(tmpDir, "templates", "nonexistent-stack.yaml")

	// Create config file
	err := os.WriteFile(configFile, []byte(configContent), 0644)
	require.NoError(t, err)

	// Create template directory and file
	err = os.MkdirAll(filepath.Dir(templateFile), 0755)
	require.NoError(t, err)
	templateContent := `{
		"AWSTemplateFormatVersion": "2010-09-09",
		"Resources": {
			"TestResource": {
				"Type": "AWS::CloudFormation::WaitConditionHandle"
			}
		}
	}`
	err = os.WriteFile(templateFile, []byte(templateContent), 0644)
	require.NoError(t, err)

	// Create mock describer that returns an error
	mockDescriber := &MockDescriber{}
	expectedError := errors.New("AWS API error: stack not found")

	// Set up expectations
	mockDescriber.On("DescribeStack", mock.Anything, mock.AnythingOfType("*model.Stack")).Return(nil, expectedError)

	// Inject mock describer
	oldDescriber := describer
	SetDescriber(mockDescriber)
	defer SetDescriber(oldDescriber)

	// Change to temp directory so config file is found
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	err = os.Chdir(tmpDir)
	require.NoError(t, err)
	defer func() {
		err := os.Chdir(oldWd)
		require.NoError(t, err)
	}()

	// Execute command
	rootCmd.SetArgs([]string{"describe", "dev", "nonexistent-stack"})
	err = rootCmd.Execute()

	// Verify error handling
	assert.Error(t, err, "describe command should return error when describer fails")
	assert.Contains(t, err.Error(), "stack not found")

	// Verify mock expectations
	mockDescriber.AssertExpectations(t)
}

func TestDescribeCommand_HandlesResolverError(t *testing.T) {
	// Test error handling when stack resolution fails

	// Create mock describer that shouldn't be called
	mockDescriber := &MockDescriber{}

	// Inject mock describer
	oldDescriber := describer
	SetDescriber(mockDescriber)
	defer SetDescriber(oldDescriber)

	// Execute command with invalid context/stack combination
	rootCmd.SetArgs([]string{"describe", "nonexistent-context", "nonexistent-stack"})
	err := rootCmd.Execute()

	// Verify error handling
	assert.Error(t, err, "describe command should return error when resolver fails")
	assert.Contains(t, err.Error(), "failed to read config file")

	// Verify no describer calls were made since resolver failed
	mockDescriber.AssertNotCalled(t, "DescribeStack")
}

func TestDescribeCommand_RequiresBothContextAndStackName(t *testing.T) {
	// Test that describe command requires both context and stack name

	// Mock describer that shouldn't be called
	mockDescriber := &MockDescriber{}

	oldDescriber := describer
	SetDescriber(mockDescriber)
	defer SetDescriber(oldDescriber)

	// Execute with only context - should fail
	rootCmd.SetArgs([]string{"describe", "dev"})
	err := rootCmd.Execute()
	assert.Error(t, err, "describe command should require both context and stack name")
	assert.Contains(t, err.Error(), "accepts 2 arg(s), received 1")

	// Execute with no arguments - should fail
	rootCmd.SetArgs([]string{"describe"})
	err = rootCmd.Execute()
	assert.Error(t, err, "describe command should require both arguments")
	assert.Contains(t, err.Error(), "accepts 2 arg(s), received 0")

	// Verify no describer calls were made
	mockDescriber.AssertExpectations(t)
}

func TestDescribeCommand_PassesCorrectStackToDescriber(t *testing.T) {
	// Test that the correct stack object is passed to the describer

	// Create temporary config file
	configContent := `
project: test-project
region: us-east-1

contexts:
  production:
    account: "123456789012"
    region: us-east-1
    tags:
      Environment: production

stacks:
  my-stack:
    template: templates/my-stack.yaml
    parameters:
      Environment: production
    contexts:
      production:
        parameters:
          Environment: production
`

	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "stackaroo.yaml")
	templateFile := filepath.Join(tmpDir, "templates", "my-stack.yaml")

	// Create config file
	err := os.WriteFile(configFile, []byte(configContent), 0644)
	require.NoError(t, err)

	// Create template directory and file
	err = os.MkdirAll(filepath.Dir(templateFile), 0755)
	require.NoError(t, err)
	templateContent := `{
		"AWSTemplateFormatVersion": "2010-09-09",
		"Resources": {
			"TestResource": {
				"Type": "AWS::CloudFormation::WaitConditionHandle"
			}
		}
	}`
	err = os.WriteFile(templateFile, []byte(templateContent), 0644)
	require.NoError(t, err)

	// Create mock describer
	mockDescriber := &MockDescriber{}

	// Set up expectations with specific stack name verification
	mockDescriber.On("DescribeStack", mock.Anything, mock.MatchedBy(func(stack *model.Stack) bool {
		return stack.Name == "my-stack" && stack.Context.Name == "production"
	})).Return(&describe.StackDescription{
		Name:   "my-stack",
		Status: "CREATE_COMPLETE",
	}, nil)

	// Inject mock describer
	oldDescriber := describer
	SetDescriber(mockDescriber)
	defer SetDescriber(oldDescriber)

	// Change to temp directory so config file is found
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	err = os.Chdir(tmpDir)
	require.NoError(t, err)
	defer func() {
		err := os.Chdir(oldWd)
		require.NoError(t, err)
	}()

	// Execute command
	rootCmd.SetArgs([]string{"describe", "production", "my-stack"})
	err = rootCmd.Execute()

	// Verify success
	require.NoError(t, err, "describe command should execute successfully")

	// Verify mock expectations (including the stack name and context check)
	mockDescriber.AssertExpectations(t)
}

func TestSetDescriber_AllowsInjection(t *testing.T) {
	// Test that SetDescriber allows proper dependency injection

	// Create mock describer
	mockDescriber := &MockDescriber{}

	// Store original describer
	oldDescriber := describer

	// Inject mock
	SetDescriber(mockDescriber)

	// Verify injection worked
	assert.Equal(t, mockDescriber, getDescriber(), "SetDescriber should allow dependency injection")

	// Restore original
	SetDescriber(oldDescriber)
}
