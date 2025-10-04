/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package deploy

import (
	"context"
	"errors"
	"os"
	"path/filepath"
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

// createMockDeployer creates a StackDeployer with mock dependencies for testing DeployStack method
func createMockDeployer(mockFactory aws.ClientFactory) *StackDeployer {
	// Create minimal mock provider and resolver (won't be called in DeployStack tests)
	mockProvider := &config.MockConfigProvider{}
	mockResolver := resolve.NewStackResolver(mockProvider, mockFactory)
	return NewStackDeployer(mockFactory, mockProvider, mockResolver)
}

func TestNewStackDeployer(t *testing.T) {
	// Test that NewStackDeployer creates a deployer with the provided client factory
	mockFactory, _ := aws.NewMockClientFactoryForRegion("us-east-1")

	deployer := createMockDeployer(mockFactory)

	assert.NotNil(t, deployer)
	// We can't directly test the internal clientFactory field, but we can test behavior
}

func TestStackDeployer_DeployStack_Success(t *testing.T) {
	// Test successful stack deployment
	ctx := context.Background()

	// Set up mock prompter for confirmation
	mockPrompter := &prompt.MockPrompter{}
	originalPrompter := prompt.GetDefaultPrompter()
	prompt.SetPrompter(mockPrompter)
	defer prompt.SetPrompter(originalPrompter)

	// Mock user confirmation (new stack creation requires confirmation)
	// Business logic sends core message, prompter adds formatting
	expectedMessage := "Do you want to apply these changes to stack test-stack?"
	mockPrompter.On("Confirm", expectedMessage).Return(true, nil)

	// Create temporary template file
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

	// Set up mocks
	mockFactory, mockCfnOps := aws.NewMockClientFactoryForRegion("us-east-1")

	// Mock StackExists call (new stack)
	mockCfnOps.On("StackExists", mock.Anything, "test-stack").Return(false, nil)

	// Set up mock expectations - now expecting DeployStackWithCallback
	mockCfnOps.On("DeployStackWithCallback", mock.Anything, mock.MatchedBy(func(input aws.DeployStackInput) bool {
		return input.StackName == "test-stack" &&
			input.TemplateBody == templateContent &&
			len(input.Parameters) == 1 &&
			input.Parameters[0].Key == "Param1" &&
			input.Parameters[0].Value == "value1" &&
			len(input.Tags) == 1 &&
			input.Tags["Environment"] == "test" &&
			len(input.Capabilities) == 1 &&
			input.Capabilities[0] == "CAPABILITY_IAM"
	}), mock.AnythingOfType("func(aws.StackEvent)")).Return(nil)

	// Create deployer with mock CloudFormation operations
	deployer := createMockDeployer(mockFactory)

	// Create resolved stack
	stack := &model.Stack{
		Name:         "test-stack",
		Context:      model.NewTestContext("dev", "us-east-1", "123456789012"),
		TemplateBody: templateContent,
		Parameters:   map[string]string{"Param1": "value1"},
		Tags:         map[string]string{"Environment": "test"},
		Dependencies: []string{},
		Capabilities: []string{"CAPABILITY_IAM"},
	}

	// Execute
	err = deployer.DeployStack(ctx, stack)

	// Verify
	assert.NoError(t, err)
	mockCfnOps.AssertExpectations(t)
	mockPrompter.AssertExpectations(t)
}

func TestStackDeployer_DeployStack_WithEmptyTemplate(t *testing.T) {
	// Set up mock prompter for confirmation
	mockPrompter := &prompt.MockPrompter{}
	originalPrompter := prompt.GetDefaultPrompter()
	prompt.SetPrompter(mockPrompter)
	defer prompt.SetPrompter(originalPrompter)

	// Mock user confirmation (new stack creation requires confirmation)
	// Business logic sends core message, prompter adds formatting
	expectedMessage := "Do you want to apply these changes to stack test-stack?"
	mockPrompter.On("Confirm", expectedMessage).Return(true, nil)
	// Test deploy stack with empty template body
	ctx := context.Background()

	// Set up mocks
	mockFactory, mockCfnOps := aws.NewMockClientFactoryForRegion("us-east-1")

	// Mock StackExists call (new stack)
	mockCfnOps.On("StackExists", mock.Anything, "test-stack").Return(false, nil)

	mockCfnOps.On("DeployStackWithCallback", mock.Anything, mock.MatchedBy(func(input aws.DeployStackInput) bool {
		return input.StackName == "test-stack" && input.TemplateBody == ""
	}), mock.AnythingOfType("func(aws.StackEvent)")).Return(nil)

	deployer := createMockDeployer(mockFactory)

	// Create resolved stack with empty template body
	stack := &model.Stack{
		Name:         "test-stack",
		Context:      model.NewTestContext("dev", "us-east-1", "123456789012"),
		TemplateBody: "",
		Parameters:   map[string]string{},
		Tags:         map[string]string{},
		Capabilities: []string{"CAPABILITY_IAM"},
	}

	// Execute
	err := deployer.DeployStack(ctx, stack)

	// Verify
	assert.NoError(t, err)
	mockCfnOps.AssertExpectations(t)
	mockPrompter.AssertExpectations(t)
}

// TestDeploySingleStack_ResolverError tests error handling when resolver fails
func TestDeploySingleStack_ResolverError(t *testing.T) {
	ctx := context.Background()

	// Create mock dependencies
	mockFactory, _ := aws.NewMockClientFactoryForRegion("us-east-1")
	mockProvider := &config.MockConfigProvider{}
	mockResolver := resolve.NewStackResolver(mockProvider, mockFactory)

	// Create deployer
	deployer := NewStackDeployer(mockFactory, mockProvider, mockResolver)

	// Mock provider to return error when resolver tries to load config
	expectedError := errors.New("config load failed")
	mockProvider.On("LoadConfig", ctx, "test-context").Return((*config.Config)(nil), expectedError)

	// Test execution - should propagate resolver error
	err := deployer.DeploySingleStack(ctx, "test-stack", "test-context")

	// Verify error is propagated correctly
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to resolve stack dependencies")
	assert.Contains(t, err.Error(), "config load failed")

	mockProvider.AssertExpectations(t)
}

// TestDeployAllStacks_ConfigLoadError tests error handling when config loading fails
func TestDeployAllStacks_ConfigLoadError(t *testing.T) {
	ctx := context.Background()

	// Create mock dependencies
	mockFactory, _ := aws.NewMockClientFactoryForRegion("us-east-1")
	mockProvider := &config.MockConfigProvider{}
	mockResolver := resolve.NewStackResolver(mockProvider, mockFactory)

	// Create deployer
	deployer := NewStackDeployer(mockFactory, mockProvider, mockResolver)

	// Mock provider to return stack list
	stackNames := []string{"stack1", "stack2"}
	mockProvider.On("ListStacks", "test-context").Return(stackNames, nil)

	// Mock GetStack calls for GetDependencyOrder
	mockStackConfig1 := &config.StackConfig{Name: "stack1", Dependencies: []string{}}
	mockStackConfig2 := &config.StackConfig{Name: "stack2", Dependencies: []string{}}
	mockProvider.On("GetStack", "stack1", "test-context").Return(mockStackConfig1, nil)
	mockProvider.On("GetStack", "stack2", "test-context").Return(mockStackConfig2, nil)

	// Mock LoadConfig call that resolver will make - return error to test error handling
	expectedError := errors.New("config resolution failed")
	mockProvider.On("LoadConfig", ctx, "test-context").Return((*config.Config)(nil), expectedError)

	// Test execution - will fail when resolver tries to load config for individual stack resolution
	err := deployer.DeployAllStacks(ctx, "test-context")

	// Should fail during config loading for individual stack resolution
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to resolve stack")
	assert.Contains(t, err.Error(), "config resolution failed")

	mockProvider.AssertExpectations(t)
}

// TestDeployAllStacks_EmptyContext tests deploying to context with no stacks
func TestDeployAllStacks_EmptyContext(t *testing.T) {
	ctx := context.Background()

	// Create mock dependencies
	mockFactory, _ := aws.NewMockClientFactoryForRegion("us-east-1")
	mockProvider := &config.MockConfigProvider{}
	mockResolver := resolve.NewStackResolver(mockProvider, mockFactory)

	// Create deployer
	deployer := NewStackDeployer(mockFactory, mockProvider, mockResolver)

	// Mock provider to return empty stack list
	mockProvider.On("ListStacks", "empty-context").Return([]string{}, nil)

	// Execute - should handle empty context gracefully
	err := deployer.DeployAllStacks(ctx, "empty-context")
	assert.NoError(t, err, "Should handle empty context without error")

	mockProvider.AssertExpectations(t)
}

// TestDeployAllStacks_ProviderError tests error handling when provider fails
func TestDeployAllStacks_ProviderError(t *testing.T) {
	ctx := context.Background()

	// Create mock dependencies
	mockFactory, _ := aws.NewMockClientFactoryForRegion("us-east-1")
	mockProvider := &config.MockConfigProvider{}
	mockResolver := resolve.NewStackResolver(mockProvider, mockFactory)

	// Create deployer
	deployer := NewStackDeployer(mockFactory, mockProvider, mockResolver)

	// Mock provider to return error
	expectedError := errors.New("failed to list stacks")
	mockProvider.On("ListStacks", "error-context").Return([]string(nil), expectedError)

	// Execute - should propagate provider error
	err := deployer.DeployAllStacks(ctx, "error-context")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get stacks for context error-context")
	assert.Contains(t, err.Error(), "failed to list stacks")

	mockProvider.AssertExpectations(t)
}

func TestStackDeployer_DeployStack_AWSError(t *testing.T) {
	// Set up mock prompter for confirmation
	mockPrompter := &prompt.MockPrompter{}
	originalPrompter := prompt.GetDefaultPrompter()
	prompt.SetPrompter(mockPrompter)
	defer prompt.SetPrompter(originalPrompter)

	// Mock user confirmation (new stack creation requires confirmation)
	// Business logic sends core message, prompter adds formatting
	expectedMessage := "Do you want to apply these changes to stack test-stack?"
	mockPrompter.On("Confirm", expectedMessage).Return(true, nil)
	// Test deploy stack when AWS returns an error
	ctx := context.Background()

	// Create temporary template file
	tmpDir := t.TempDir()
	templateFile := filepath.Join(tmpDir, "test-template.json")
	templateContent := `{"AWSTemplateFormatVersion": "2010-09-09"}`

	err := os.WriteFile(templateFile, []byte(templateContent), 0644)
	require.NoError(t, err)

	// Set up mocks
	mockFactory, mockCfnOps := aws.NewMockClientFactoryForRegion("us-east-1")

	// Mock StackExists call (new stack)
	mockCfnOps.On("StackExists", mock.Anything, "test-stack").Return(false, nil)

	mockCfnOps.On("DeployStackWithCallback", mock.Anything, mock.MatchedBy(func(input aws.DeployStackInput) bool {
		return input.StackName == "test-stack" && input.TemplateBody == templateContent
	}), mock.AnythingOfType("func(aws.StackEvent)")).Return(errors.New("AWS deployment error"))

	// Create deployer with mock CloudFormation operations
	deployer := createMockDeployer(mockFactory)

	// Create resolved stack with template content
	stack := &model.Stack{
		Name:         "test-stack",
		Context:      model.NewTestContext("dev", "us-east-1", "123456789012"),
		TemplateBody: templateContent,
		Parameters:   map[string]string{},
		Tags:         map[string]string{},
		Capabilities: []string{"CAPABILITY_IAM"},
	}

	// Execute
	err = deployer.DeployStack(ctx, stack)

	// Verify
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create stack")
	assert.Contains(t, err.Error(), "AWS deployment error")

	mockCfnOps.AssertExpectations(t)
	mockPrompter.AssertExpectations(t)
}

func TestStackDeployer_DeployStack_NoChanges(t *testing.T) {
	// Test deploy stack when there are no changes to deploy
	ctx := context.Background()

	templateContent := `{"AWSTemplateFormatVersion": "2010-09-09"}`

	// Set up mocks
	mockFactory, mockCfnOps := aws.NewMockClientFactoryForRegion("us-east-1")

	// Mock StackExists call (existing stack)
	mockCfnOps.On("StackExists", mock.Anything, "test-stack").Return(true, nil)

	// Mock differ operations
	currentStackInfo := &aws.StackInfo{
		Name:       "test-stack",
		Status:     "UPDATE_COMPLETE",
		Parameters: map[string]string{},
		Tags:       map[string]string{},
	}
	mockCfnOps.On("DescribeStack", mock.Anything, "test-stack").Return(currentStackInfo, nil)
	mockCfnOps.On("GetTemplate", mock.Anything, "test-stack").Return(`{"AWSTemplateFormatVersion": "2010-09-09"}`, nil)

	// No changeset operations expected for no-changes scenario
	// The deployer should return early when no changes are detected

	// Create deployer with mock CloudFormation operations
	deployer := createMockDeployer(mockFactory)

	// Create resolved stack
	stack := &model.Stack{
		Name:         "test-stack",
		Context:      model.NewTestContext("dev", "us-east-1", "123456789012"),
		TemplateBody: templateContent,
		Parameters:   map[string]string{},
		Tags:         map[string]string{},
		Capabilities: []string{"CAPABILITY_IAM"},
	}

	// Execute
	err := deployer.DeployStack(ctx, stack)

	// Verify - should succeed with no error when no changes detected
	assert.NoError(t, err)

	mockCfnOps.AssertExpectations(t)
}

func TestStackDeployer_DeployStack_WithChanges(t *testing.T) {
	// Test successful deployment with changes
	ctx := context.Background()

	// Set up mock prompter to auto-confirm deployment
	mockPrompter := &prompt.MockPrompter{}
	// Business logic sends core message, prompter adds formatting
	expectedMessage := "Do you want to apply these changes to stack test-stack?"
	mockPrompter.On("Confirm", expectedMessage).Return(true, nil).Once()

	originalPrompter := prompt.GetDefaultPrompter()
	prompt.SetPrompter(mockPrompter)
	defer prompt.SetPrompter(originalPrompter)

	templateContent := `{"AWSTemplateFormatVersion": "2010-09-09", "Resources": {"NewBucket": {"Type": "AWS::S3::Bucket"}}}`

	// Set up mocks
	mockFactory, mockCfnOps := aws.NewMockClientFactoryForRegion("us-east-1")

	// Mock StackExists call (existing stack)
	mockCfnOps.On("StackExists", mock.Anything, "test-stack").Return(true, nil)

	// Mock differ operations
	currentStackInfo := &aws.StackInfo{
		Name:       "test-stack",
		Status:     "UPDATE_COMPLETE",
		Parameters: map[string]string{},
		Tags:       map[string]string{},
	}
	mockCfnOps.On("DescribeStack", mock.Anything, "test-stack").Return(currentStackInfo, nil)
	mockCfnOps.On("GetTemplate", mock.Anything, "test-stack").Return(`{"AWSTemplateFormatVersion": "2010-09-09", "Resources": {"OldBucket": {"Type": "AWS::S3::Bucket"}}}`, nil)

	// Mock changeset operations for the differ
	changeSetInfo := &aws.ChangeSetInfo{
		ChangeSetID: "test-changeset-id",
		Status:      "CREATE_COMPLETE",
		Changes: []aws.ResourceChange{
			{
				Action:       "Modify",
				ResourceType: "AWS::S3::Bucket",
				LogicalID:    "TestBucket",
				PhysicalID:   "test-bucket-123",
				Replacement:  "False",
				Details:      []string{},
			},
		},
	}
	mockCfnOps.On("CreateChangeSetForDeployment", mock.Anything, "test-stack", templateContent, map[string]string{}, []string{"CAPABILITY_IAM"}, map[string]string{}).Return(changeSetInfo, nil)

	// Mock execute changeset using abstracted method
	mockCfnOps.On("ExecuteChangeSet", mock.Anything, "test-changeset-id").Return(nil)

	// Mock wait for stack operation
	mockCfnOps.On("WaitForStackOperation", mock.Anything, "test-stack", mock.AnythingOfType("time.Time"), mock.AnythingOfType("func(aws.StackEvent)")).Return(nil)

	// Mock delete changeset (cleanup after successful deployment - both differ and deployer delete changesets)
	mockCfnOps.On("DeleteChangeSet", mock.Anything, "test-changeset-id").Return(nil)

	// Create deployer with mock CloudFormation operations
	deployer := createMockDeployer(mockFactory)

	// Create resolved stack
	stack := &model.Stack{
		Name:         "test-stack",
		Context:      model.NewTestContext("dev", "us-east-1", "123456789012"),
		TemplateBody: templateContent,
		Parameters:   map[string]string{},
		Tags:         map[string]string{},
		Capabilities: []string{"CAPABILITY_IAM"},
	}

	// Execute
	err := deployer.DeployStack(ctx, stack)

	// Verify - should succeed
	assert.NoError(t, err)

	mockCfnOps.AssertExpectations(t)
	mockPrompter.AssertExpectations(t)
}

func TestStackDeployer_ValidateTemplate_Success(t *testing.T) {
	// Test successful template validation
	ctx := context.Background()

	// Create temporary template file
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

	// Set up mocks
	mockFactory, mockCfnOps := aws.NewMockClientFactoryForRegion("us-east-1")
	mockCfnOps.On("ValidateTemplate", ctx, templateContent).Return(nil)

	// Create deployer with mock CloudFormation operations
	deployer := createMockDeployer(mockFactory)

	// Execute
	err = deployer.ValidateTemplate(ctx, templateFile)

	// Verify
	assert.NoError(t, err)
	mockCfnOps.AssertExpectations(t)
}

func TestStackDeployer_ValidateTemplate_FileNotFound(t *testing.T) {
	// Test validate template with non-existent file
	ctx := context.Background()

	mockFactory, _ := aws.NewMockClientFactoryForRegion("us-east-1")
	deployer := createMockDeployer(mockFactory)

	// Execute with non-existent file
	err := deployer.ValidateTemplate(ctx, "/nonexistent/template.json")

	// Verify
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read template")
	assert.Contains(t, err.Error(), "no such file or directory")

	// CloudFormation operations should not be called for file not found
}

func TestStackDeployer_ValidateTemplate_ValidationError(t *testing.T) {
	// Test validate template when AWS returns validation error
	ctx := context.Background()

	// Create temporary template file
	tmpDir := t.TempDir()
	templateFile := filepath.Join(tmpDir, "test-template.json")
	templateContent := `{"invalid": "template"}`

	err := os.WriteFile(templateFile, []byte(templateContent), 0644)
	require.NoError(t, err)

	// Set up mocks
	mockFactory, mockCfnOps := aws.NewMockClientFactoryForRegion("us-east-1")
	mockCfnOps.On("ValidateTemplate", ctx, templateContent).Return(errors.New("template validation failed"))

	// Create deployer with mock CloudFormation operations
	deployer := createMockDeployer(mockFactory)

	// Execute ValidateTemplate instead - this test is for template validation
	err = deployer.ValidateTemplate(ctx, templateFile)

	// Verify
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "template validation failed")

	mockCfnOps.AssertExpectations(t)
}

func TestStackDeployer_DeployStack_WithYAMLTemplate(t *testing.T) {
	// Set up mock prompter for confirmation
	mockPrompter := &prompt.MockPrompter{}
	originalPrompter := prompt.GetDefaultPrompter()
	prompt.SetPrompter(mockPrompter)
	defer prompt.SetPrompter(originalPrompter)

	// Mock user confirmation (new stack creation requires confirmation)
	// Business logic sends core message, prompter adds formatting
	expectedMessage := "Do you want to apply these changes to stack test-stack?"
	mockPrompter.On("Confirm", expectedMessage).Return(true, nil)
	// Test deploying stack with YAML template content
	ctx := context.Background()

	templateContent := `AWSTemplateFormatVersion: '2010-09-09'
Resources:
  MyBucket:
    Type: AWS::S3::Bucket
    Properties:
      BucketName: my-test-bucket`

	// Set up mocks to capture the template content
	mockFactory, mockCfnOps := aws.NewMockClientFactoryForRegion("us-east-1")

	// Mock StackExists call (new stack)
	mockCfnOps.On("StackExists", mock.Anything, "test-stack").Return(false, nil)

	mockCfnOps.On("DeployStackWithCallback", mock.Anything, mock.MatchedBy(func(input aws.DeployStackInput) bool {
		// Verify the template content was passed correctly
		return input.TemplateBody == templateContent &&
			input.StackName == "test-stack"
	}), mock.AnythingOfType("func(aws.StackEvent)")).Return(nil)

	// Create deployer with mock CloudFormation operations
	deployer := createMockDeployer(mockFactory)

	// Create resolved stack
	stack := &model.Stack{
		Name:         "test-stack",
		Context:      model.NewTestContext("dev", "us-east-1", "123456789012"),
		TemplateBody: templateContent,
		Parameters:   map[string]string{},
		Tags:         map[string]string{},
		Capabilities: []string{"CAPABILITY_IAM"},
	}

	// Execute
	err := deployer.DeployStack(ctx, stack)

	// Verify
	assert.NoError(t, err)
	mockCfnOps.AssertExpectations(t)
	mockPrompter.AssertExpectations(t)
}

func TestStackDeployer_DeployStack_WithMultipleParametersAndTags(t *testing.T) {
	// Test deployment with multiple parameters and tags
	ctx := context.Background()

	// Set up mock prompter for confirmation
	mockPrompter := &prompt.MockPrompter{}
	originalPrompter := prompt.GetDefaultPrompter()
	prompt.SetPrompter(mockPrompter)
	defer prompt.SetPrompter(originalPrompter)

	// Mock user confirmation (new stack creation requires confirmation)
	// Business logic sends core message, prompter adds formatting
	expectedMessage := "Do you want to apply these changes to stack test-stack?"
	mockPrompter.On("Confirm", expectedMessage).Return(true, nil)

	// Set up mocks
	mockFactory, mockCfnOps := aws.NewMockClientFactoryForRegion("us-east-1")

	// Mock StackExists call (new stack)
	mockCfnOps.On("StackExists", mock.Anything, "test-stack").Return(false, nil)

	mockCfnOps.On("DeployStackWithCallback", mock.Anything, mock.MatchedBy(func(input aws.DeployStackInput) bool {
		return input.TemplateBody == "" &&
			input.StackName == "test-stack" &&
			len(input.Parameters) == 2 &&
			len(input.Tags) == 2 &&
			len(input.Capabilities) == 2
	}), mock.AnythingOfType("func(aws.StackEvent)")).Return(nil)

	// Create deployer with mock CloudFormation operations
	deployer := createMockDeployer(mockFactory)

	// Create resolved stack with parameters and tags
	stack := &model.Stack{
		Name:         "test-stack",
		Context:      model.NewTestContext("dev", "us-east-1", "123456789012"),
		TemplateBody: "",
		Parameters:   map[string]string{"Environment": "test", "InstanceType": "t3.micro"},
		Tags:         map[string]string{"Project": "stackaroo", "Environment": "test"},
		Dependencies: []string{},
		Capabilities: []string{"CAPABILITY_IAM", "CAPABILITY_NAMED_IAM"},
	}

	// Execute
	err := deployer.DeployStack(ctx, stack)

	// Verify
	assert.NoError(t, err)
	mockCfnOps.AssertExpectations(t)
	mockPrompter.AssertExpectations(t)
}

// TestDeployStack_NewStack_UserCancels tests cancellation during new stack creation
func TestDeployStack_NewStack_UserCancels(t *testing.T) {
	ctx := context.Background()

	// Set up mock prompter for cancellation
	mockPrompter := &prompt.MockPrompter{}
	originalPrompter := prompt.GetDefaultPrompter()
	prompt.SetPrompter(mockPrompter)
	defer prompt.SetPrompter(originalPrompter)

	// Mock user confirmation (user cancels)
	expectedMessage := "Do you want to apply these changes to stack test-stack?"
	mockPrompter.On("Confirm", expectedMessage).Return(false, nil)

	// Set up mocks
	mockFactory, mockCfnOps := aws.NewMockClientFactoryForRegion("us-east-1")

	// Mock StackExists call (new stack)
	mockCfnOps.On("StackExists", mock.Anything, "test-stack").Return(false, nil)

	// Create deployer with mock CloudFormation operations
	deployer := createMockDeployer(mockFactory)

	// Create resolved stack
	stack := &model.Stack{
		Name:         "test-stack",
		Context:      model.NewTestContext("dev", "us-east-1", "123456789012"),
		TemplateBody: "template content",
		Parameters:   map[string]string{"Environment": "test"},
		Tags:         map[string]string{"Project": "stackaroo"},
		Dependencies: []string{},
		Capabilities: []string{"CAPABILITY_IAM"},
	}

	// Execute
	err := deployer.DeployStack(ctx, stack)

	// Verify that CancellationError is returned
	assert.Error(t, err)
	var cancellationErr CancellationError
	assert.ErrorAs(t, err, &cancellationErr)
	assert.Equal(t, "test-stack", cancellationErr.StackName)
	mockCfnOps.AssertExpectations(t)
	mockPrompter.AssertExpectations(t)
}

// TestDeployStackWithFeedback_CancellationHandling tests that deployStackWithFeedback handles cancellation correctly
func TestDeployStackWithFeedback_CancellationHandling(t *testing.T) {
	ctx := context.Background()

	// Set up mock prompter for cancellation
	mockPrompter := &prompt.MockPrompter{}
	originalPrompter := prompt.GetDefaultPrompter()
	prompt.SetPrompter(mockPrompter)
	defer prompt.SetPrompter(originalPrompter)

	// Mock user confirmation (user cancels)
	expectedMessage := "Do you want to apply these changes to stack test-stack?"
	mockPrompter.On("Confirm", expectedMessage).Return(false, nil)

	// Set up mocks
	mockFactory, mockCfnOps := aws.NewMockClientFactoryForRegion("us-east-1")
	mockCfnOps.On("StackExists", mock.Anything, "test-stack").Return(false, nil)

	// Create deployer with mock CloudFormation operations
	deployer := createMockDeployer(mockFactory)

	// Create resolved stack
	stack := &model.Stack{
		Name:         "test-stack",
		Context:      model.NewTestContext("dev", "us-east-1", "123456789012"),
		TemplateBody: "template content",
		Parameters:   map[string]string{"Environment": "test"},
		Tags:         map[string]string{"Project": "stackaroo"},
		Dependencies: []string{},
		Capabilities: []string{"CAPABILITY_IAM"},
	}

	// Execute deployStackWithFeedback directly
	err := deployer.deployStackWithFeedback(ctx, stack, "test-context")

	// Verify that no error is returned (cancellation is handled gracefully)
	assert.NoError(t, err)
	mockCfnOps.AssertExpectations(t)
	mockPrompter.AssertExpectations(t)
}

// TestDeployStack_ExistingStack_UserCancelsChangeset tests cancellation during changeset deployment
func TestDeployStack_ExistingStack_UserCancelsChangeset(t *testing.T) {
	ctx := context.Background()

	// Set up mock prompter for cancellation
	mockPrompter := &prompt.MockPrompter{}
	originalPrompter := prompt.GetDefaultPrompter()
	prompt.SetPrompter(mockPrompter)
	defer prompt.SetPrompter(originalPrompter)

	// Mock user confirmation (user cancels changeset)
	expectedMessage := "Do you want to apply these changes to stack test-stack?"
	mockPrompter.On("Confirm", expectedMessage).Return(false, nil)

	// Set up mocks
	mockFactory, mockCfnOps := aws.NewMockClientFactoryForRegion("us-east-1")

	// Mock StackExists call (existing stack)
	mockCfnOps.On("StackExists", mock.Anything, "test-stack").Return(true, nil)

	// Mock differ operations (required for changeset approach)
	currentStackInfo := &aws.StackInfo{
		Name:       "test-stack",
		Status:     "UPDATE_COMPLETE",
		Parameters: map[string]string{"Environment": "test"},
		Tags:       map[string]string{"Project": "stackaroo"},
	}
	mockCfnOps.On("DescribeStack", mock.Anything, "test-stack").Return(currentStackInfo, nil)
	mockCfnOps.On("GetTemplate", mock.Anything, "test-stack").Return(`{"AWSTemplateFormatVersion": "2010-09-09", "Resources": {"OldBucket": {"Type": "AWS::S3::Bucket"}}}`, nil)

	// Mock changeset creation for deployment
	changeSetInfo := &aws.ChangeSetInfo{
		ChangeSetID: "changeset-123",
		Status:      "CREATE_COMPLETE",
		Changes: []aws.ResourceChange{
			{
				Action:       "Add",
				ResourceType: "AWS::S3::Bucket",
				LogicalID:    "TestBucket",
				PhysicalID:   "test-bucket-123",
				Replacement:  "False",
				Details:      []string{},
			},
		},
	}
	mockCfnOps.On("CreateChangeSetForDeployment", mock.Anything, "test-stack", `{"AWSTemplateFormatVersion": "2010-09-09", "Resources": {"NewBucket": {"Type": "AWS::S3::Bucket"}}}`, map[string]string{"Environment": "test"}, []string{"CAPABILITY_IAM"}, map[string]string{"Project": "stackaroo"}).Return(changeSetInfo, nil)

	// Mock changeset deletion (cleanup after cancellation)
	mockCfnOps.On("DeleteChangeSet", mock.Anything, "changeset-123").Return(nil)

	// Create deployer with mock CloudFormation operations
	deployer := createMockDeployer(mockFactory)

	// Create resolved stack
	stack := &model.Stack{
		Name:         "test-stack",
		Context:      model.NewTestContext("dev", "us-east-1", "123456789012"),
		TemplateBody: `{"AWSTemplateFormatVersion": "2010-09-09", "Resources": {"NewBucket": {"Type": "AWS::S3::Bucket"}}}`,
		Parameters:   map[string]string{"Environment": "test"},
		Tags:         map[string]string{"Project": "stackaroo"},
		Dependencies: []string{},
		Capabilities: []string{"CAPABILITY_IAM"},
	}

	// Execute
	err := deployer.DeployStack(ctx, stack)

	// Verify that CancellationError is returned
	assert.Error(t, err)
	var cancellationErr CancellationError
	assert.ErrorAs(t, err, &cancellationErr)
	assert.Equal(t, "test-stack", cancellationErr.StackName)
	mockCfnOps.AssertExpectations(t)
	mockPrompter.AssertExpectations(t)
}
