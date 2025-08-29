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
	"github.com/orien/stackaroo/internal/model"
	"github.com/orien/stackaroo/internal/prompt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockAWSClient is a mock implementation of aws.Client
type MockAWSClient struct {
	mock.Mock
}

func (m *MockAWSClient) NewCloudFormationOperations() aws.CloudFormationOperations {
	args := m.Called()
	return args.Get(0).(aws.CloudFormationOperations)
}

// MockPrompter is a mock implementation of the Prompter interface for testing
type MockPrompter struct {
	mock.Mock
}

// Confirm mock implementation
func (m *MockPrompter) Confirm(message string) (bool, error) {
	args := m.Called(message)
	return args.Bool(0), args.Error(1)
}

// MockCloudFormationOperations is a mock implementation of CloudFormationOperations
type MockCloudFormationOperations struct {
	mock.Mock
}

func (m *MockCloudFormationOperations) DeployStack(ctx context.Context, input aws.DeployStackInput) error {
	args := m.Called(ctx, input)
	return args.Error(0)
}

func (m *MockCloudFormationOperations) DeployStackWithCallback(ctx context.Context, input aws.DeployStackInput, eventCallback func(aws.StackEvent)) error {
	args := m.Called(ctx, input, eventCallback)
	return args.Error(0)
}

func (m *MockCloudFormationOperations) UpdateStack(ctx context.Context, input aws.UpdateStackInput) error {
	args := m.Called(ctx, input)
	return args.Error(0)
}

func (m *MockCloudFormationOperations) DeleteStack(ctx context.Context, input aws.DeleteStackInput) error {
	args := m.Called(ctx, input)
	return args.Error(0)
}

func (m *MockCloudFormationOperations) GetStack(ctx context.Context, stackName string) (*aws.Stack, error) {
	args := m.Called(ctx, stackName)
	return args.Get(0).(*aws.Stack), args.Error(1)
}

func (m *MockCloudFormationOperations) ListStacks(ctx context.Context) ([]*aws.Stack, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*aws.Stack), args.Error(1)
}

func (m *MockCloudFormationOperations) ValidateTemplate(ctx context.Context, templateBody string) error {
	args := m.Called(ctx, templateBody)
	return args.Error(0)
}

func (m *MockCloudFormationOperations) StackExists(ctx context.Context, stackName string) (bool, error) {
	args := m.Called(ctx, stackName)
	return args.Bool(0), args.Error(1)
}

func (m *MockCloudFormationOperations) GetTemplate(ctx context.Context, stackName string) (string, error) {
	args := m.Called(ctx, stackName)
	return args.String(0), args.Error(1)
}

func (m *MockCloudFormationOperations) DescribeStack(ctx context.Context, stackName string) (*aws.StackInfo, error) {
	args := m.Called(ctx, stackName)
	return args.Get(0).(*aws.StackInfo), args.Error(1)
}

func (m *MockCloudFormationOperations) DeleteChangeSet(ctx context.Context, changeSetID string) error {
	args := m.Called(ctx, changeSetID)
	return args.Error(0)
}

// ExecuteChangeSet executes a changeset by ID (abstracted method)
func (m *MockCloudFormationOperations) ExecuteChangeSet(ctx context.Context, changeSetID string) error {
	args := m.Called(ctx, changeSetID)
	return args.Error(0)
}

func (m *MockCloudFormationOperations) DescribeStackEvents(ctx context.Context, stackName string) ([]aws.StackEvent, error) {
	args := m.Called(ctx, stackName)
	return args.Get(0).([]aws.StackEvent), args.Error(1)
}

func (m *MockCloudFormationOperations) WaitForStackOperation(ctx context.Context, stackName string, eventCallback func(aws.StackEvent)) error {
	args := m.Called(ctx, stackName, eventCallback)
	return args.Error(0)
}

func (m *MockCloudFormationOperations) CreateChangeSetPreview(ctx context.Context, stackName string, template string, parameters map[string]string) (*aws.ChangeSetInfo, error) {
	args := m.Called(ctx, stackName, template, parameters)
	return args.Get(0).(*aws.ChangeSetInfo), args.Error(1)
}

func (m *MockCloudFormationOperations) CreateChangeSetForDeployment(ctx context.Context, stackName string, template string, parameters map[string]string, capabilities []string, tags map[string]string) (*aws.ChangeSetInfo, error) {
	args := m.Called(ctx, stackName, template, parameters, capabilities, tags)
	return args.Get(0).(*aws.ChangeSetInfo), args.Error(1)
}

func TestNewStackDeployer(t *testing.T) {
	// Test that NewStackDeployer creates a deployer with the provided client
	mockClient := &MockAWSClient{}

	deployer := NewStackDeployer(mockClient)

	assert.NotNil(t, deployer)
	// We can't directly test the internal client field, but we can test behavior
}

func TestStackDeployer_DeployStack_Success(t *testing.T) {
	// Test successful stack deployment
	ctx := context.Background()

	// Set up mock prompter for confirmation
	mockPrompter := &MockPrompter{}
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
	mockCfnOps := &MockCloudFormationOperations{}
	mockClient := &MockAWSClient{}

	mockClient.On("NewCloudFormationOperations").Return(mockCfnOps)

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

	// Create deployer with mock client
	deployer := NewStackDeployer(mockClient)

	// Create resolved stack
	stack := &model.Stack{
		Name:         "test-stack",
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
	mockClient.AssertExpectations(t)
	mockCfnOps.AssertExpectations(t)
	mockPrompter.AssertExpectations(t)
}

func TestStackDeployer_DeployStack_WithEmptyTemplate(t *testing.T) {
	// Set up mock prompter for confirmation
	mockPrompter := &MockPrompter{}
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
	mockCfnOps := &MockCloudFormationOperations{}
	mockClient := &MockAWSClient{}

	mockClient.On("NewCloudFormationOperations").Return(mockCfnOps)

	// Mock StackExists call (new stack)
	mockCfnOps.On("StackExists", mock.Anything, "test-stack").Return(false, nil)

	mockCfnOps.On("DeployStackWithCallback", mock.Anything, mock.MatchedBy(func(input aws.DeployStackInput) bool {
		return input.StackName == "test-stack" && input.TemplateBody == ""
	}), mock.AnythingOfType("func(aws.StackEvent)")).Return(nil)

	deployer := NewStackDeployer(mockClient)

	// Create resolved stack with empty template body
	stack := &model.Stack{
		Name:         "test-stack",
		TemplateBody: "",
		Parameters:   map[string]string{},
		Tags:         map[string]string{},
		Dependencies: []string{},
		Capabilities: []string{"CAPABILITY_IAM"},
	}

	// Execute
	err := deployer.DeployStack(ctx, stack)

	// Verify
	assert.NoError(t, err)
	mockClient.AssertExpectations(t)
	mockCfnOps.AssertExpectations(t)
	mockPrompter.AssertExpectations(t)
}

func TestStackDeployer_DeployStack_AWSError(t *testing.T) {
	// Set up mock prompter for confirmation
	mockPrompter := &MockPrompter{}
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
	mockCfnOps := &MockCloudFormationOperations{}
	mockClient := &MockAWSClient{}

	mockClient.On("NewCloudFormationOperations").Return(mockCfnOps)

	// Mock StackExists call (new stack)
	mockCfnOps.On("StackExists", mock.Anything, "test-stack").Return(false, nil)

	mockCfnOps.On("DeployStackWithCallback", mock.Anything, mock.MatchedBy(func(input aws.DeployStackInput) bool {
		return input.StackName == "test-stack" && input.TemplateBody == templateContent
	}), mock.AnythingOfType("func(aws.StackEvent)")).Return(errors.New("AWS deployment error"))

	// Create deployer with mock client
	deployer := NewStackDeployer(mockClient)

	// Create resolved stack with template content
	stack := &model.Stack{
		Name:         "test-stack",
		TemplateBody: templateContent,
		Parameters:   map[string]string{},
		Tags:         map[string]string{},
		Dependencies: []string{},
		Capabilities: []string{"CAPABILITY_IAM"},
	}

	// Execute
	err = deployer.DeployStack(ctx, stack)

	// Verify
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create stack")
	assert.Contains(t, err.Error(), "AWS deployment error")

	mockClient.AssertExpectations(t)
	mockCfnOps.AssertExpectations(t)
	mockPrompter.AssertExpectations(t)
}

func TestStackDeployer_DeployStack_NoChanges(t *testing.T) {
	// Test deploy stack when there are no changes to deploy
	ctx := context.Background()

	templateContent := `{"AWSTemplateFormatVersion": "2010-09-09"}`

	// Set up mocks
	mockCfnOps := &MockCloudFormationOperations{}
	mockClient := &MockAWSClient{}

	mockClient.On("NewCloudFormationOperations").Return(mockCfnOps)

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

	// Create deployer with mock client
	deployer := NewStackDeployer(mockClient)

	// Create resolved stack
	stack := &model.Stack{
		Name:         "test-stack",
		TemplateBody: templateContent,
		Parameters:   map[string]string{},
		Tags:         map[string]string{},
		Dependencies: []string{},
		Capabilities: []string{"CAPABILITY_IAM"},
	}

	// Execute
	err := deployer.DeployStack(ctx, stack)

	// Verify - should succeed with no error when no changes detected
	assert.NoError(t, err)
	mockClient.AssertExpectations(t)
	mockCfnOps.AssertExpectations(t)
}

func TestStackDeployer_DeployStack_WithChanges(t *testing.T) {
	// Test successful deployment with changes
	ctx := context.Background()

	// Set up mock prompter to auto-confirm deployment
	mockPrompter := &MockPrompter{}
	// Business logic sends core message, prompter adds formatting
	expectedMessage := "Do you want to apply these changes to stack test-stack?"
	mockPrompter.On("Confirm", expectedMessage).Return(true, nil).Once()

	originalPrompter := prompt.GetDefaultPrompter()
	prompt.SetPrompter(mockPrompter)
	defer prompt.SetPrompter(originalPrompter)

	templateContent := `{"AWSTemplateFormatVersion": "2010-09-09", "Resources": {"NewBucket": {"Type": "AWS::S3::Bucket"}}}`

	// Set up mocks
	mockCfnOps := &MockCloudFormationOperations{}
	mockClient := &MockAWSClient{}

	mockClient.On("NewCloudFormationOperations").Return(mockCfnOps)

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
	mockCfnOps.On("WaitForStackOperation", mock.Anything, "test-stack", mock.AnythingOfType("func(aws.StackEvent)")).Return(nil)

	// Mock delete changeset (cleanup after successful deployment - both differ and deployer delete changesets)
	mockCfnOps.On("DeleteChangeSet", mock.Anything, "test-changeset-id").Return(nil)

	// Create deployer with mock client
	deployer := NewStackDeployer(mockClient)

	// Create resolved stack
	stack := &model.Stack{
		Name:         "test-stack",
		TemplateBody: templateContent,
		Parameters:   map[string]string{},
		Tags:         map[string]string{},
		Dependencies: []string{},
		Capabilities: []string{"CAPABILITY_IAM"},
	}

	// Execute
	err := deployer.DeployStack(ctx, stack)

	// Verify - should succeed
	assert.NoError(t, err)
	mockClient.AssertExpectations(t)
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
	mockCfnOps := &MockCloudFormationOperations{}
	mockClient := &MockAWSClient{}

	mockClient.On("NewCloudFormationOperations").Return(mockCfnOps)
	mockCfnOps.On("ValidateTemplate", ctx, templateContent).Return(nil)

	// Create deployer with mock client
	deployer := NewStackDeployer(mockClient)

	// Execute
	err = deployer.ValidateTemplate(ctx, templateFile)

	// Verify
	assert.NoError(t, err)
	mockClient.AssertExpectations(t)
	mockCfnOps.AssertExpectations(t)
}

func TestStackDeployer_ValidateTemplate_FileNotFound(t *testing.T) {
	// Test validate template with non-existent file
	ctx := context.Background()

	mockClient := &MockAWSClient{}
	deployer := NewStackDeployer(mockClient)

	// Execute with non-existent file
	err := deployer.ValidateTemplate(ctx, "/nonexistent/template.json")

	// Verify
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read template")
	assert.Contains(t, err.Error(), "no such file or directory")

	// AWS client should not be called
	mockClient.AssertExpectations(t)
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
	mockCfnOps := &MockCloudFormationOperations{}
	mockClient := &MockAWSClient{}

	mockClient.On("NewCloudFormationOperations").Return(mockCfnOps)
	mockCfnOps.On("ValidateTemplate", ctx, templateContent).Return(errors.New("template validation failed"))

	// Create deployer with mock client
	deployer := NewStackDeployer(mockClient)

	// Execute ValidateTemplate instead - this test is for template validation
	err = deployer.ValidateTemplate(ctx, templateFile)

	// Verify
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "template validation failed")

	mockClient.AssertExpectations(t)
	mockCfnOps.AssertExpectations(t)
}

func TestStackDeployer_DeployStack_WithYAMLTemplate(t *testing.T) {
	// Set up mock prompter for confirmation
	mockPrompter := &MockPrompter{}
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
	mockCfnOps := &MockCloudFormationOperations{}
	mockClient := &MockAWSClient{}

	mockClient.On("NewCloudFormationOperations").Return(mockCfnOps)

	// Mock StackExists call (new stack)
	mockCfnOps.On("StackExists", mock.Anything, "test-stack").Return(false, nil)

	mockCfnOps.On("DeployStackWithCallback", mock.Anything, mock.MatchedBy(func(input aws.DeployStackInput) bool {
		// Verify the template content was passed correctly
		return input.TemplateBody == templateContent &&
			input.StackName == "test-stack"
	}), mock.AnythingOfType("func(aws.StackEvent)")).Return(nil)

	// Create deployer with mock client
	deployer := NewStackDeployer(mockClient)

	// Create resolved stack
	stack := &model.Stack{
		Name:         "test-stack",
		TemplateBody: templateContent,
		Parameters:   map[string]string{},
		Tags:         map[string]string{},
		Dependencies: []string{},
		Capabilities: []string{"CAPABILITY_IAM"},
	}

	// Execute
	err := deployer.DeployStack(ctx, stack)

	// Verify
	assert.NoError(t, err)
	mockClient.AssertExpectations(t)
	mockCfnOps.AssertExpectations(t)
	mockPrompter.AssertExpectations(t)
}

func TestStackDeployer_DeployStack_WithMultipleParametersAndTags(t *testing.T) {
	// Test deployment with multiple parameters and tags
	ctx := context.Background()

	// Set up mock prompter for confirmation
	mockPrompter := &MockPrompter{}
	originalPrompter := prompt.GetDefaultPrompter()
	prompt.SetPrompter(mockPrompter)
	defer prompt.SetPrompter(originalPrompter)

	// Mock user confirmation (new stack creation requires confirmation)
	// Business logic sends core message, prompter adds formatting
	expectedMessage := "Do you want to apply these changes to stack test-stack?"
	mockPrompter.On("Confirm", expectedMessage).Return(true, nil)

	// Set up mocks
	mockCfnOps := &MockCloudFormationOperations{}
	mockClient := &MockAWSClient{}

	mockClient.On("NewCloudFormationOperations").Return(mockCfnOps)

	// Mock StackExists call (new stack)
	mockCfnOps.On("StackExists", mock.Anything, "test-stack").Return(false, nil)

	mockCfnOps.On("DeployStackWithCallback", mock.Anything, mock.MatchedBy(func(input aws.DeployStackInput) bool {
		return input.TemplateBody == "" &&
			input.StackName == "test-stack" &&
			len(input.Parameters) == 2 &&
			len(input.Tags) == 2 &&
			len(input.Capabilities) == 2
	}), mock.AnythingOfType("func(aws.StackEvent)")).Return(nil)

	// Create deployer with mock client
	deployer := NewStackDeployer(mockClient)

	// Create resolved stack with parameters and tags
	stack := &model.Stack{
		Name:         "test-stack",
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
	mockClient.AssertExpectations(t)
	mockCfnOps.AssertExpectations(t)
	mockPrompter.AssertExpectations(t)
}
