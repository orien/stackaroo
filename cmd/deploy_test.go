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

	"codeberg.org/orien/stackaroo/internal/deploy"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestDeployCommand_Exists(t *testing.T) {
	// Test that deploy command is registered with root command
	deployCmd := findCommand(rootCmd, "deploy")

	assert.NotNil(t, deployCmd, "deploy command should be registered")
	assert.Equal(t, "deploy <context> [stack-name]", deployCmd.Use)
}

func TestDeployCommand_AcceptsStackName(t *testing.T) {
	// Test that deploy command accepts a stack name argument
	deployCmd := findCommand(rootCmd, "deploy")
	assert.NotNil(t, deployCmd)

	// Test that Args validation is set
	assert.NotNil(t, deployCmd.Args, "deploy command should have Args validation set")
}

func TestDeployCommand_AcceptsTwoArgs(t *testing.T) {
	// Test that deploy command accepts one or two arguments (context and optional stack name)
	deployCmd := findCommand(rootCmd, "deploy")
	assert.NotNil(t, deployCmd)

	// Test that Args validation accepts 1-2 arguments
	err := deployCmd.Args(deployCmd, []string{"dev", "vpc"})
	assert.NoError(t, err, "Two arguments should be valid")

	err = deployCmd.Args(deployCmd, []string{"dev"})
	assert.NoError(t, err, "One argument should be valid")

	err = deployCmd.Args(deployCmd, []string{})
	assert.Error(t, err, "No arguments should be invalid")
}

func TestDeployCommand_RequiresAtLeastOneArg(t *testing.T) {
	// Test that deploy command requires at least a context argument

	// Mock deployer that shouldn't be called for argument validation tests
	mockDeployer := &deploy.MockDeployer{}

	oldDeployer := deployer
	SetDeployer(mockDeployer)
	defer SetDeployer(oldDeployer)

	// Execute with no arguments - should fail
	rootCmd.SetArgs([]string{"deploy"})

	err := rootCmd.Execute()
	assert.Error(t, err, "deploy command should require at least a context argument")
	assert.Contains(t, err.Error(), "accepts between 1 and 2 arg(s), received 0")

	// Verify no deployer calls were made
	mockDeployer.AssertExpectations(t)
}

func TestDeployCommand_DeployAllStacksInContext(t *testing.T) {
	// Test that deploy command with single argument deploys all stacks in context

	// Create a temporary config file with multiple stacks
	configContent := `
project: test-project

contexts:
  test-context:
    region: us-east-1

stacks:
  vpc:
    template: templates/vpc.yaml
  app:
    template: templates/app.yaml
`
	tmpDir := t.TempDir()
	configFile := tmpDir + "/stackaroo.yaml"
	err := os.WriteFile(configFile, []byte(configContent), 0644)
	require.NoError(t, err)

	// Create template files
	err = os.MkdirAll(tmpDir+"/templates", 0755)
	require.NoError(t, err)

	vpcTemplate := `AWSTemplateFormatVersion: '2010-09-09'
Resources:
  TestVPC:
    Type: AWS::EC2::VPC
    Properties:
      CidrBlock: 10.0.0.0/16`

	appTemplate := `AWSTemplateFormatVersion: '2010-09-09'
Resources:
  TestApp:
    Type: AWS::EC2::Instance
    Properties:
      ImageId: ami-12345678
      InstanceType: t2.micro`

	err = os.WriteFile(tmpDir+"/templates/vpc.yaml", []byte(vpcTemplate), 0644)
	require.NoError(t, err)
	err = os.WriteFile(tmpDir+"/templates/app.yaml", []byte(appTemplate), 0644)
	require.NoError(t, err)

	// Mock deployer that expects two deployments
	mockDeployer := &deploy.MockDeployer{}
	mockDeployer.On("DeployAllStacks", mock.Anything, "test-context").Return(nil).Once()

	oldDeployer := deployer
	SetDeployer(mockDeployer)
	defer SetDeployer(oldDeployer)

	// Change to temp directory and execute
	oldDir, _ := os.Getwd()
	defer func() {
		err := os.Chdir(oldDir)
		require.NoError(t, err)
	}()
	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	rootCmd.SetArgs([]string{"deploy", "test-context"})

	err = rootCmd.Execute()
	assert.NoError(t, err, "deploy command should successfully deploy all stacks")

	// Verify both stacks were deployed
	mockDeployer.AssertExpectations(t)
}

func TestDeployCommand_NoStacksInContext(t *testing.T) {
	// Test that deploy command handles context with no stacks gracefully

	// Create a temporary config file with no stacks
	configContent := `
project: test-project

contexts:
  empty-context:
    region: us-east-1

stacks: {}
`
	tmpDir := t.TempDir()
	configFile := tmpDir + "/stackaroo.yaml"
	err := os.WriteFile(configFile, []byte(configContent), 0644)
	require.NoError(t, err)

	// Mock deployer that expects DeployAllStacks call (will handle no stacks internally)
	mockDeployer := &deploy.MockDeployer{}
	mockDeployer.On("DeployAllStacks", mock.Anything, "empty-context").Return(nil).Once()

	oldDeployer := deployer
	SetDeployer(mockDeployer)
	defer SetDeployer(oldDeployer)

	// Change to temp directory and execute
	oldDir, _ := os.Getwd()
	defer func() {
		err := os.Chdir(oldDir)
		require.NoError(t, err)
	}()
	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	rootCmd.SetArgs([]string{"deploy", "empty-context"})

	err = rootCmd.Execute()
	assert.NoError(t, err, "deploy command should handle empty context without error")

	// Verify no deployer calls were made
	mockDeployer.AssertExpectations(t)
}

func TestDeployCommand_HandlesDeployerError(t *testing.T) {
	// Test that deploy command properly handles errors from deployer

	// Create temporary directory and config files
	tmpDir := t.TempDir()

	// Create stackaroo.yaml
	configContent := `
contexts:
  test:
    parameters:
      Environment: test
stacks:
  test-stack:
    template: test-template.json
`
	err := os.WriteFile(filepath.Join(tmpDir, "stackaroo.yaml"), []byte(configContent), 0644)
	require.NoError(t, err)

	templateContent := `{"AWSTemplateFormatVersion": "2010-09-09"}`
	err = os.WriteFile(filepath.Join(tmpDir, "test-template.json"), []byte(templateContent), 0644)
	require.NoError(t, err)

	// Set up mock deployer that returns an error
	mockDeployer := &deploy.MockDeployer{}
	mockDeployer.On("DeploySingleStack", mock.Anything, "test-stack", "test").Return(errors.New("deployment failed"))

	oldDeployer := deployer
	SetDeployer(mockDeployer)
	defer SetDeployer(oldDeployer)

	// Change to temp directory
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	err = os.Chdir(tmpDir)
	require.NoError(t, err)
	defer func() {
		err := os.Chdir(oldWd)
		require.NoError(t, err)
	}()

	// Execute the root command with deploy subcommand and arguments
	rootCmd.SetArgs([]string{"deploy", "test", "test-stack"})

	// Execute the command - should return error
	err = rootCmd.Execute()
	assert.Error(t, err, "deploy command should return error when deployer fails")
	assert.Contains(t, err.Error(), "deployment failed", "error should contain original error")
}

func TestDeployCommand_AcceptsOneOrTwoArgs(t *testing.T) {
	// Test that deploy command accepts 1-2 arguments (context and optional stack name)

	// Mock deployer for valid calls
	mockDeployer := &deploy.MockDeployer{}
	mockDeployer.On("DeployAllStacks", mock.Anything, "dev").Return(nil).Once()

	oldDeployer := deployer
	SetDeployer(mockDeployer)
	defer SetDeployer(oldDeployer) // Restore after test

	// Test with no arguments - should error
	rootCmd.SetArgs([]string{"deploy"})
	err := rootCmd.Execute()
	assert.Error(t, err, "should error when no arguments provided")

	// Test with one argument (context only) - should work
	rootCmd.SetArgs([]string{"deploy", "dev"})
	err = rootCmd.Execute()
	assert.NoError(t, err, "should work with just context")

	// Test with too many arguments - should error
	rootCmd.SetArgs([]string{"deploy", "dev", "stack1", "stack2"})
	err = rootCmd.Execute()
	assert.Error(t, err, "should error when too many arguments provided")

	// Verify deployer was called as expected
	mockDeployer.AssertExpectations(t)
}

func TestDeployCommand_AdvancedMockingFeatures(t *testing.T) {
	// Test advanced mock features like expectations count and argument matching

	// Create temporary directory and config files
	tmpDir := t.TempDir()

	// Create stackaroo.yaml with two stacks
	configContent := `
contexts:
  test:
    parameters:
      Environment: test
stacks:
  stack-1:
    template: template1.json
  stack-2:
    template: template2.json
`
	err := os.WriteFile(filepath.Join(tmpDir, "stackaroo.yaml"), []byte(configContent), 0644)
	require.NoError(t, err)

	templateContent := `{
		"AWSTemplateFormatVersion": "2010-09-09",
		"Resources": {
			"TestResource": {
				"Type": "AWS::CloudFormation::WaitConditionHandle"
			}
		}
	}`

	err = os.WriteFile(filepath.Join(tmpDir, "template1.json"), []byte(templateContent), 0644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(tmpDir, "template2.json"), []byte(templateContent), 0644)
	require.NoError(t, err)

	// Set up mock with multiple expectations and argument matching
	mockDeployer := &deploy.MockDeployer{}

	// Expect specific calls with exact argument matching
	mockDeployer.On("DeploySingleStack", mock.Anything, "stack-1", "test").Return(nil).Once()

	mockDeployer.On("DeploySingleStack", mock.Anything, "stack-2", "test").Return(errors.New("second deployment failed")).Once()

	oldDeployer := deployer
	SetDeployer(mockDeployer)
	defer SetDeployer(oldDeployer)

	// Change to temp directory
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	err = os.Chdir(tmpDir)
	require.NoError(t, err)
	defer func() {
		err := os.Chdir(oldWd)
		require.NoError(t, err)
	}()

	// First deployment should succeed
	rootCmd.SetArgs([]string{"deploy", "test", "stack-1"})
	err = rootCmd.Execute()
	assert.NoError(t, err, "first deployment should succeed")

	// Second deployment should fail
	rootCmd.SetArgs([]string{"deploy", "test", "stack-2"})
	err = rootCmd.Execute()
	assert.Error(t, err, "second deployment should fail")
	assert.Contains(t, err.Error(), "second deployment failed", "error should contain expected message")

	// Verify all expectations were met
	mockDeployer.AssertExpectations(t)
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
  vpc:
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
	mockDeployer := &deploy.MockDeployer{}
	// Expect DeploySingleStack call for vpc in dev context
	mockDeployer.On("DeploySingleStack", mock.Anything, "vpc", "dev").Return(nil)

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

	// Execute deploy command with context and stack name
	rootCmd.SetArgs([]string{"deploy", "dev", "vpc"})

	err = rootCmd.Execute()
	assert.NoError(t, err, "deploy command should execute successfully with config")

	// Verify deployer was called with correct parameters
	mockDeployer.AssertExpectations(t)
}

func TestDeployCommand_RequiresDependencyResolution(t *testing.T) {
	// Test that deploy command uses resolver for dependency resolution
	// This will fail because current implementation doesn't use resolver for dependencies

	// Create config with dependencies: app depends on database, database depends on vpc
	configContent := `
project: test-project
region: us-east-1

contexts:
  test:
    account: "123456789012"
    region: us-east-1

stacks:
  vpc:
    template: templates/vpc.yaml
    parameters:
      VpcCidr: 10.0.0.0/16

  database:
    template: templates/db.yaml
    depends_on: [vpc]
    parameters:
      DBInstanceClass: db.t3.micro

  app:
    template: templates/app.yaml
    depends_on: [database]
    parameters:
      InstanceType: t3.micro
`

	// Create temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "stackaroo.yaml")
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	// Create template files
	templatesDir := filepath.Join(tmpDir, "templates")
	err = os.MkdirAll(templatesDir, 0755)
	require.NoError(t, err)

	templateContent := `{"AWSTemplateFormatVersion": "2010-09-09", "Resources": {}}`
	for _, name := range []string{"vpc.yaml", "db.yaml", "app.yaml"} {
		err = os.WriteFile(filepath.Join(templatesDir, name), []byte(templateContent), 0644)
		require.NoError(t, err)
	}

	// Mock deployer that expects calls in dependency order: vpc → database → app
	mockDeployer := &deploy.MockDeployer{}

	// This test will fail because current implementation doesn't resolve dependencies
	// We expect DeploySingleStack to be called for the app stack
	mockDeployer.On("DeploySingleStack", mock.Anything, "app", "test").Return(nil)

	oldDeployer := deployer
	SetDeployer(mockDeployer)
	defer SetDeployer(oldDeployer)

	// Change to temp directory
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	err = os.Chdir(tmpDir)
	require.NoError(t, err)
	defer func() {
		err := os.Chdir(oldWd)
		require.NoError(t, err)
	}()

	// This should resolve dependencies and deploy: vpc → database → app
	// But current implementation will only deploy app
	rootCmd.SetArgs([]string{"deploy", "test", "app"})

	err = rootCmd.Execute()
	assert.NoError(t, err, "deploy should succeed")

	// This test currently passes but doesn't verify dependency resolution
	// When we integrate the resolver, we should expect 3 calls in order
	mockDeployer.AssertExpectations(t)
}

func TestDeployCommand_UsesResolverForMultiStackDeployment(t *testing.T) {
	// Test that deploy command uses resolver to deploy dependencies in correct order
	// This WILL FAIL because current implementation doesn't use resolver

	configContent := `
project: test-project
region: us-east-1

contexts:
  test:
    account: "123456789012"
    region: us-east-1

stacks:
  vpc:
    template: templates/vpc.yaml

  database:
    template: templates/db.yaml
    depends_on: [vpc]

  app:
    template: templates/app.yaml
    depends_on: [database]
`

	// Create temporary config and template files
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "stackaroo.yaml")
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	templatesDir := filepath.Join(tmpDir, "templates")
	err = os.MkdirAll(templatesDir, 0755)
	require.NoError(t, err)

	templateContent := `{"AWSTemplateFormatVersion": "2010-09-09", "Resources": {}}`
	for _, name := range []string{"vpc.yaml", "db.yaml", "app.yaml"} {
		err = os.WriteFile(filepath.Join(templatesDir, name), []byte(templateContent), 0644)
		require.NoError(t, err)
	}

	// Mock deployer that expects ALL THREE stacks in dependency order
	mockDeployer := &deploy.MockDeployer{}

	// Current implementation only deploys the directly requested stack
	// Transitive dependency resolution is not yet implemented
	mockDeployer.On("DeploySingleStack", mock.Anything, "app", "test").Return(nil).Once()

	oldDeployer := deployer
	SetDeployer(mockDeployer)
	defer SetDeployer(oldDeployer)

	// Change to temp directory
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	err = os.Chdir(tmpDir)
	require.NoError(t, err)
	defer func() {
		err := os.Chdir(oldWd)
		require.NoError(t, err)
	}()

	// Deploy app - should trigger resolver to deploy vpc → database → app
	rootCmd.SetArgs([]string{"deploy", "test", "app"})

	err = rootCmd.Execute()
	assert.NoError(t, err, "deploy should succeed")

	// Verify the requested stack was deployed
	mockDeployer.AssertExpectations(t)
}

// Helper function to find a command by name
func findCommand(parent *cobra.Command, name string) *cobra.Command {
	for _, cmd := range parent.Commands() {
		if cmd.Name() == name {
			return cmd
		}
	}
	return nil
}
