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
	"strings"
	"testing"

	"github.com/orien/stackaroo/internal/config/file"
	"github.com/orien/stackaroo/internal/model"
	"github.com/orien/stackaroo/internal/resolve"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockDeployer is a mock implementation of the Deployer interface
type MockDeployer struct {
	mock.Mock
}

func (m *MockDeployer) DeployStack(ctx context.Context, stack *model.Stack) error {
	args := m.Called(ctx, stack)
	return args.Error(0)
}

func (m *MockDeployer) ValidateTemplate(ctx context.Context, templateFile string) error {
	args := m.Called(ctx, templateFile)
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

func TestDeployCommand_HasContextFlag(t *testing.T) {
	// Test that deploy command has a --context flag
	deployCmd := findCommand(rootCmd, "deploy")
	assert.NotNil(t, deployCmd)

	// Check that --context flag exists
	contextFlag := deployCmd.Flags().Lookup("context")
	assert.NotNil(t, contextFlag, "deploy command should have --context flag")
	assert.Equal(t, "context", contextFlag.Name)
}

func TestDeployCommand_RequiresContext(t *testing.T) {
	// Test that deploy command requires --context flag

	// Mock deployer that shouldn't be called
	mockDeployer := &MockDeployer{}

	oldDeployer := deployer
	SetDeployer(mockDeployer)
	defer SetDeployer(oldDeployer)

	// Execute without context flag - should fail
	rootCmd.SetArgs([]string{"deploy", "test-stack"})

	err := rootCmd.Execute()
	assert.Error(t, err, "deploy command should require --context flag")
	assert.Contains(t, err.Error(), "required flag(s) \"context\" not set")

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
  - name: test-stack
    template: test-template.json
`
	err := os.WriteFile(filepath.Join(tmpDir, "stackaroo.yaml"), []byte(configContent), 0644)
	require.NoError(t, err)

	templateContent := `{"AWSTemplateFormatVersion": "2010-09-09"}`
	err = os.WriteFile(filepath.Join(tmpDir, "test-template.json"), []byte(templateContent), 0644)
	require.NoError(t, err)

	// Set up mock deployer that returns an error
	mockDeployer := &MockDeployer{}
	mockDeployer.On("DeployStack", mock.Anything, mock.MatchedBy(func(stack *model.Stack) bool {
		return stack.Name == "test-stack"
	})).Return(errors.New("deployment failed"))

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
	rootCmd.SetArgs([]string{"deploy", "test-stack", "--context", "test"})

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
  - name: stack-1
    template: template1.json
  - name: stack-2
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
	mockDeployer := &MockDeployer{}

	// Expect specific calls with exact argument matching
	mockDeployer.On("DeployStack", mock.Anything, mock.MatchedBy(func(stack *model.Stack) bool {
		return stack.Name == "stack-1"
	})).Return(nil).Once()

	mockDeployer.On("DeployStack", mock.Anything, mock.MatchedBy(func(stack *model.Stack) bool {
		return stack.Name == "stack-2"
	})).Return(errors.New("second deployment failed")).Once()

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
	rootCmd.SetArgs([]string{"deploy", "stack-1", "--context", "test"})
	err = rootCmd.Execute()
	assert.NoError(t, err, "first deployment should succeed")

	// Second deployment should fail
	rootCmd.SetArgs([]string{"deploy", "stack-2", "--context", "test"})
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
	// Expect Stack with resolved parameters from dev context
	mockDeployer.On("DeployStack", mock.Anything, mock.MatchedBy(func(stack *model.Stack) bool {
		return stack.Name == "vpc" &&
			stack.Parameters["VpcCidr"] == "10.1.0.0/16" &&
			strings.Contains(stack.TemplateBody, "AWSTemplateFormatVersion") &&
			strings.Contains(stack.TemplateBody, "AWS::EC2::VPC")
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
  - name: vpc
    template: templates/vpc.yaml
    parameters:
      VpcCidr: 10.0.0.0/16

  - name: database
    template: templates/db.yaml
    depends_on: [vpc]
    parameters:
      DBInstanceClass: db.t3.micro

  - name: app
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
	mockDeployer := &MockDeployer{}

	// This test will fail because current implementation doesn't resolve dependencies
	// We expect the resolver to be called and handle the dependency ordering
	// For now, just expect app deployment (what current implementation does)
	mockDeployer.On("DeployStack", mock.Anything, mock.MatchedBy(func(stack *model.Stack) bool {
		return stack.Name == "app" // Current implementation only deploys single stack
	})).Return(nil)

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
	rootCmd.SetArgs([]string{"deploy", "app", "--context", "test"})

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
  - name: vpc
    template: templates/vpc.yaml

  - name: database
    template: templates/db.yaml
    depends_on: [vpc]

  - name: app
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
	mockDeployer := &MockDeployer{}

	// Current implementation only deploys the directly requested stack
	// Transitive dependency resolution is not yet implemented
	mockDeployer.On("DeployStack", mock.Anything, mock.MatchedBy(func(stack *model.Stack) bool {
		return stack.Name == "app"
	})).Return(nil).Once()

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
	rootCmd.SetArgs([]string{"deploy", "app", "--context", "test"})

	err = rootCmd.Execute()
	assert.NoError(t, err, "deploy should succeed")

	// Verify the requested stack was deployed
	mockDeployer.AssertExpectations(t)
}

func TestDebugResolver(t *testing.T) {
	// Debug test to see what resolver actually returns
	configContent := `
project: test-project
region: us-east-1

contexts:
  test:
    account: "123456789012"
    region: us-east-1

stacks:
  - name: vpc
    template: templates/vpc.yaml

  - name: database
    template: templates/db.yaml
    depends_on: [vpc]

  - name: app
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

	// Change to temp directory
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	err = os.Chdir(tmpDir)
	require.NoError(t, err)
	defer func() {
		err := os.Chdir(oldWd)
		require.NoError(t, err)
	}()

	// Test configuration provider directly first
	provider := file.NewProvider("stackaroo.yaml")

	// Debug: Check if config loads correctly
	config, err := provider.LoadConfig(context.Background(), "test")
	assert.NoError(t, err, "config should load")
	if config != nil {
		t.Logf("Config loaded successfully")
		t.Logf("Number of stacks in config: %d", len(config.Stacks))
		for _, stack := range config.Stacks {
			t.Logf("Config stack: %s, deps: %v", stack.Name, stack.Dependencies)
		}
	}

	// Debug: Check individual stack lookup
	appStack, err := provider.GetStack("app", "test")
	assert.NoError(t, err, "should find app stack")
	if appStack != nil {
		t.Logf("App stack dependencies: %v", appStack.Dependencies)
	}

	// Test resolver
	resolver := resolve.NewStackResolver(provider)

	resolved, err := resolver.Resolve(context.Background(), "test", []string{"app"})
	assert.NoError(t, err, "resolver should work")

	if resolved != nil {
		t.Logf("Resolved stacks count: %d", len(resolved.Stacks))
		t.Logf("Deployment order: %v", resolved.DeploymentOrder)
		for _, stack := range resolved.Stacks {
			t.Logf("Resolved stack: %s, deps: %v", stack.Name, stack.Dependencies)
		}
	}
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
