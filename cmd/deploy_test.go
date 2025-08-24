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

	"github.com/orien/stackaroo/internal/config"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockDeployer is a mock implementation of the Deployer interface
type MockDeployer struct {
	mock.Mock
}

func (m *MockDeployer) DeployStack(ctx context.Context, stackConfig *config.StackConfig) error {
	args := m.Called(ctx, stackConfig)
	return args.Error(0)
}

func TestDeployCommand_Exists(t *testing.T) {
	// Test that deploy command is registered with root command
	deployCmd := findCommand(rootCmd, "deploy")

	assert.NotNil(t, deployCmd, "deploy command should be registered")
	assert.Equal(t, "deploy", deployCmd.Use)
}

func TestDeployCommand_AcceptsStackName(t *testing.T) {
	// Test that deploy command accepts a stack name argument
	deployCmd := findCommand(rootCmd, "deploy")
	assert.NotNil(t, deployCmd)

	// Test that Args validation is set
	assert.NotNil(t, deployCmd.Args, "deploy command should have Args validation set")
}

func TestDeployCommand_HasTemplateFlag(t *testing.T) {
	// Test that deploy command has a --template flag
	deployCmd := findCommand(rootCmd, "deploy")
	assert.NotNil(t, deployCmd)

	// Check that --template flag exists
	templateFlag := deployCmd.Flags().Lookup("template")
	assert.NotNil(t, templateFlag, "deploy command should have --template flag")
	assert.Equal(t, "template", templateFlag.Name)
}

func TestDeployCommand_CallsDeployerCorrectly(t *testing.T) {
	// Test that deploy command calls the deployer with correct parameters

	// Create a temporary template file
	tmpDir := t.TempDir()
	templateFile := filepath.Join(tmpDir, "test-template.json")
	templateContent := `{
		"AWSTemplateFormatVersion": "2010-09-09",
		"Resources": {
			"TestBucket": {
				"Type": "AWS::S3::Bucket"
			}
		}
	}`

	err := os.WriteFile(templateFile, []byte(templateContent), 0644)
	require.NoError(t, err)

	// Set up mock deployer
	mockDeployer := &MockDeployer{}
	expectedStackConfig := &config.StackConfig{
		Name:         "test-stack",
		Template:     templateFile,
		Parameters:   make(map[string]string),
		Tags:         make(map[string]string),
		Dependencies: []string{},
		Capabilities: []string{"CAPABILITY_IAM"},
	}
	mockDeployer.On("DeployStack", mock.Anything, expectedStackConfig).Return(nil)

	oldDeployer := deployer
	SetDeployer(mockDeployer)
	defer SetDeployer(oldDeployer) // Restore after test

	// Execute the root command with deploy subcommand and arguments
	rootCmd.SetArgs([]string{"deploy", "test-stack", "--template", templateFile})

	// Execute the command
	err = rootCmd.Execute()
	assert.NoError(t, err, "deploy command should execute successfully with mock")

	// Verify that all expected calls were made
	mockDeployer.AssertExpectations(t)
}

func TestDeployCommand_HandlesDeployerError(t *testing.T) {
	// Test that deploy command properly handles errors from deployer

	// Create a temporary template file
	tmpDir := t.TempDir()
	templateFile := filepath.Join(tmpDir, "test-template.json")
	templateContent := `{"AWSTemplateFormatVersion": "2010-09-09"}`

	err := os.WriteFile(templateFile, []byte(templateContent), 0644)
	require.NoError(t, err)

	// Set up mock deployer that returns an error
	mockDeployer := &MockDeployer{}
	expectedStackConfig := &config.StackConfig{
		Name:         "test-stack",
		Template:     templateFile,
		Parameters:   make(map[string]string),
		Tags:         make(map[string]string),
		Dependencies: []string{},
		Capabilities: []string{"CAPABILITY_IAM"},
	}
	mockDeployer.On("DeployStack", mock.Anything, expectedStackConfig).Return(errors.New("deployment failed"))

	oldDeployer := deployer
	SetDeployer(mockDeployer)
	defer SetDeployer(oldDeployer) // Restore after test

	// Execute the root command with deploy subcommand and arguments
	rootCmd.SetArgs([]string{"deploy", "test-stack", "--template", templateFile})

	// Execute the command - should return error
	err = rootCmd.Execute()
	assert.Error(t, err, "deploy command should return error when deployer fails")
	assert.Contains(t, err.Error(), "error deploying stack test-stack", "error should contain stack name")
	assert.Contains(t, err.Error(), "deployment failed", "error should contain original error")
}

func TestDeployCommand_RequiresStackName(t *testing.T) {
	// Test that deploy command requires exactly one argument (stack name)

	// Mock deployer that shouldn't be called (no expectations set)
	mockDeployer := &MockDeployer{}

	oldDeployer := deployer
	SetDeployer(mockDeployer)
	defer SetDeployer(oldDeployer) // Restore after test

	// Test with no arguments
	rootCmd.SetArgs([]string{"deploy"})
	err := rootCmd.Execute()
	assert.Error(t, err, "should error when no stack name provided")

	// Test with too many arguments
	rootCmd.SetArgs([]string{"deploy", "stack1", "stack2"})
	err = rootCmd.Execute()
	assert.Error(t, err, "should error when too many arguments provided")

	// Verify deployer was not called
	mockDeployer.AssertExpectations(t)
}

func TestDeployCommand_AdvancedMockingFeatures(t *testing.T) {
	// Test demonstrating advanced testify/mock features

	// Create temporary template files
	tmpDir := t.TempDir()
	templateFile1 := filepath.Join(tmpDir, "template1.json")
	templateFile2 := filepath.Join(tmpDir, "template2.json")

	templateContent := `{"AWSTemplateFormatVersion": "2010-09-09"}`
	err := os.WriteFile(templateFile1, []byte(templateContent), 0644)
	require.NoError(t, err)
	err = os.WriteFile(templateFile2, []byte(templateContent), 0644)
	require.NoError(t, err)

	// Set up mock with multiple expectations and argument matching
	mockDeployer := &MockDeployer{}

	// Expect specific calls with exact argument matching
	expectedStackConfig1 := &config.StackConfig{
		Name:         "stack-1",
		Template:     templateFile1,
		Parameters:   make(map[string]string),
		Tags:         make(map[string]string),
		Dependencies: []string{},
		Capabilities: []string{"CAPABILITY_IAM"},
	}
	expectedStackConfig2 := &config.StackConfig{
		Name:         "stack-2",
		Template:     templateFile2,
		Parameters:   make(map[string]string),
		Tags:         make(map[string]string),
		Dependencies: []string{},
		Capabilities: []string{"CAPABILITY_IAM"},
	}
	mockDeployer.On("DeployStack", mock.Anything, expectedStackConfig1).Return(nil).Once()
	mockDeployer.On("DeployStack", mock.Anything, expectedStackConfig2).Return(errors.New("second deployment failed")).Once()

	oldDeployer := deployer
	SetDeployer(mockDeployer)
	defer SetDeployer(oldDeployer)

	// First deployment should succeed
	rootCmd.SetArgs([]string{"deploy", "stack-1", "--template", templateFile1})
	err = rootCmd.Execute()
	assert.NoError(t, err, "first deployment should succeed")

	// Second deployment should fail
	rootCmd.SetArgs([]string{"deploy", "stack-2", "--template", templateFile2})
	err = rootCmd.Execute()
	assert.Error(t, err, "second deployment should fail")
	assert.Contains(t, err.Error(), "second deployment failed")

	// Verify all expectations were met exactly once
	mockDeployer.AssertExpectations(t)

	// Verify specific methods were called the expected number of times
	mockDeployer.AssertNumberOfCalls(t, "DeployStack", 2)
}

func TestDeployCommand_WithConfigurationFile(t *testing.T) {
	// Test that deploy command can use configuration from stackaroo.yaml

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
  - name: vpc
    template: templates/vpc.yaml
    parameters:
      VpcCidr: 10.0.0.0/16
    contexts:
      dev:
        parameters:
          VpcCidr: 10.1.0.0/16
`

	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "stackaroo.yaml")
	templateFile := filepath.Join(tmpDir, "templates", "vpc.yaml")

	// Create config file
	err := os.WriteFile(configFile, []byte(configContent), 0644)
	require.NoError(t, err)

	// Create template directory and file
	err = os.MkdirAll(filepath.Dir(templateFile), 0755)
	require.NoError(t, err)
	templateContent := `{
		"AWSTemplateFormatVersion": "2010-09-09",
		"Resources": {
			"VPC": {
				"Type": "AWS::EC2::VPC",
				"Properties": {
					"CidrBlock": {"Ref": "VpcCidr"}
				}
			}
		},
		"Parameters": {
			"VpcCidr": {"Type": "String"}
		}
	}`
	err = os.WriteFile(templateFile, []byte(templateContent), 0644)
	require.NoError(t, err)

	// Set up mock deployer that expects config-resolved values
	mockDeployer := &MockDeployer{}
	// Expect StackConfig with resolved parameters from dev context
	mockDeployer.On("DeployStack", mock.Anything, mock.MatchedBy(func(stackConfig *config.StackConfig) bool {
		return stackConfig.Name == "vpc" &&
			stackConfig.Parameters["VpcCidr"] == "10.1.0.0/16" &&
			(stackConfig.Template == "templates/vpc.yaml" ||
				filepath.Base(filepath.Dir(stackConfig.Template)) == "templates" && filepath.Base(stackConfig.Template) == "vpc.yaml")
	})).Return(nil)

	oldDeployer := deployer
	SetDeployer(mockDeployer)
	defer SetDeployer(oldDeployer)

	// Change to temp directory so config file is found
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	err = os.Chdir(tmpDir)
	require.NoError(t, err)
	defer func() {
		err := os.Chdir(oldWd)
		require.NoError(t, err)
	}()

	// Execute deploy command with context flag
	rootCmd.SetArgs([]string{"deploy", "vpc", "--context", "dev"})

	err = rootCmd.Execute()
	assert.NoError(t, err, "deploy command should execute successfully with config")

	// Verify deployer was called with correct parameters
	mockDeployer.AssertExpectations(t)
}

// Helper function to find a command by name
func findCommand(parent *cobra.Command, name string) *cobra.Command {
	for _, cmd := range parent.Commands() {
		if cmd.Use == name {
			return cmd
		}
	}
	return nil
}
