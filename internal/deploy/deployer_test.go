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

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	awsinternal "github.com/orien/stackaroo/internal/aws"
	"github.com/orien/stackaroo/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockAWSClient is a mock implementation of aws.Client
type MockAWSClient struct {
	mock.Mock
}

func (m *MockAWSClient) NewCloudFormationOperations() awsinternal.CloudFormationOperations {
	args := m.Called()
	return args.Get(0).(awsinternal.CloudFormationOperations)
}

// MockCloudFormationOperations is a mock implementation of aws.CloudFormationOperations
type MockCloudFormationOperations struct {
	mock.Mock
}

func (m *MockCloudFormationOperations) DeployStack(ctx context.Context, input awsinternal.DeployStackInput) error {
	args := m.Called(ctx, input)
	return args.Error(0)
}

func (m *MockCloudFormationOperations) DeployStackWithCallback(ctx context.Context, input awsinternal.DeployStackInput, eventCallback func(awsinternal.StackEvent)) error {
	args := m.Called(ctx, input, eventCallback)
	return args.Error(0)
}

func (m *MockCloudFormationOperations) UpdateStack(ctx context.Context, input awsinternal.UpdateStackInput) error {
	args := m.Called(ctx, input)
	return args.Error(0)
}

func (m *MockCloudFormationOperations) DeleteStack(ctx context.Context, input awsinternal.DeleteStackInput) error {
	args := m.Called(ctx, input)
	return args.Error(0)
}

func (m *MockCloudFormationOperations) GetStack(ctx context.Context, stackName string) (*awsinternal.Stack, error) {
	args := m.Called(ctx, stackName)
	return args.Get(0).(*awsinternal.Stack), args.Error(1)
}

func (m *MockCloudFormationOperations) ListStacks(ctx context.Context) ([]*awsinternal.Stack, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*awsinternal.Stack), args.Error(1)
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

func (m *MockCloudFormationOperations) DescribeStack(ctx context.Context, stackName string) (*awsinternal.StackInfo, error) {
	args := m.Called(ctx, stackName)
	return args.Get(0).(*awsinternal.StackInfo), args.Error(1)
}

func (m *MockCloudFormationOperations) CreateChangeSet(ctx context.Context, params *cloudformation.CreateChangeSetInput, optFns ...func(*cloudformation.Options)) (*cloudformation.CreateChangeSetOutput, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(*cloudformation.CreateChangeSetOutput), args.Error(1)
}

func (m *MockCloudFormationOperations) DeleteChangeSet(ctx context.Context, params *cloudformation.DeleteChangeSetInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DeleteChangeSetOutput, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(*cloudformation.DeleteChangeSetOutput), args.Error(1)
}

func (m *MockCloudFormationOperations) DescribeChangeSet(ctx context.Context, params *cloudformation.DescribeChangeSetInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DescribeChangeSetOutput, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(*cloudformation.DescribeChangeSetOutput), args.Error(1)
}

func (m *MockCloudFormationOperations) ExecuteChangeSet(ctx context.Context, params *cloudformation.ExecuteChangeSetInput, optFns ...func(*cloudformation.Options)) (*cloudformation.ExecuteChangeSetOutput, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(*cloudformation.ExecuteChangeSetOutput), args.Error(1)
}

func (m *MockCloudFormationOperations) DescribeStackEvents(ctx context.Context, stackName string) ([]awsinternal.StackEvent, error) {
	args := m.Called(ctx, stackName)
	return args.Get(0).([]awsinternal.StackEvent), args.Error(1)
}

func (m *MockCloudFormationOperations) WaitForStackOperation(ctx context.Context, stackName string, eventCallback func(awsinternal.StackEvent)) error {
	args := m.Called(ctx, stackName, eventCallback)
	return args.Error(0)
}

func TestNewAWSDeployer(t *testing.T) {
	// Test that NewAWSDeployer creates a deployer with the provided client
	mockClient := &MockAWSClient{}

	deployer := NewAWSDeployer(mockClient)

	assert.NotNil(t, deployer)
	// We can't directly test the internal client field, but we can test behavior
}

func TestNewDefaultDeployer_CreatesDeployer(t *testing.T) {
	// Test that NewDefaultDeployer attempts to create a deployer
	// This will fail in CI/testing environments without AWS credentials, which is expected
	ctx := context.Background()

	deployer, err := NewDefaultDeployer(ctx)

	// In environments without AWS credentials, this should fail
	// In environments with credentials, it should succeed
	// Either way, the function should behave predictably
	if err != nil {
		assert.Nil(t, deployer)
		assert.Contains(t, err.Error(), "failed to create AWS client")
	} else {
		assert.NotNil(t, deployer)
	}
}

func TestAWSDeployer_DeployStack_Success(t *testing.T) {
	// Test successful stack deployment
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

	// Mock StackExists call (new stack)
	mockCfnOps.On("StackExists", mock.Anything, "test-stack").Return(false, nil)

	// Set up mock expectations - now expecting DeployStackWithCallback
	mockCfnOps.On("DeployStackWithCallback", mock.Anything, mock.MatchedBy(func(input awsinternal.DeployStackInput) bool {
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
	deployer := NewAWSDeployer(mockClient)

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
}

func TestAWSDeployer_DeployStack_WithEmptyTemplate(t *testing.T) {
	// Test deploy stack with empty template body
	ctx := context.Background()

	// Set up mocks
	mockCfnOps := &MockCloudFormationOperations{}
	mockClient := &MockAWSClient{}

	mockClient.On("NewCloudFormationOperations").Return(mockCfnOps)

	// Mock StackExists call (new stack)
	mockCfnOps.On("StackExists", mock.Anything, "test-stack").Return(false, nil)

	mockCfnOps.On("DeployStackWithCallback", mock.Anything, mock.MatchedBy(func(input awsinternal.DeployStackInput) bool {
		return input.StackName == "test-stack" && input.TemplateBody == ""
	}), mock.AnythingOfType("func(aws.StackEvent)")).Return(nil)

	deployer := NewAWSDeployer(mockClient)

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
}

func TestAWSDeployer_DeployStack_AWSError(t *testing.T) {
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

	mockCfnOps.On("DeployStackWithCallback", mock.Anything, mock.MatchedBy(func(input awsinternal.DeployStackInput) bool {
		return input.StackName == "test-stack" && input.TemplateBody == templateContent
	}), mock.AnythingOfType("func(aws.StackEvent)")).Return(errors.New("AWS deployment error"))

	// Create deployer with mock client
	deployer := NewAWSDeployer(mockClient)

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
}

func TestAWSDeployer_DeployStack_NoChanges(t *testing.T) {
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
	currentStackInfo := &awsinternal.StackInfo{
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
	deployer := NewAWSDeployer(mockClient)

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

	// Verify - should succeed with no error despite NoChangesError
	assert.NoError(t, err)
	mockClient.AssertExpectations(t)
	mockCfnOps.AssertExpectations(t)
}

func TestAWSDeployer_DeployStack_WithChanges(t *testing.T) {
	// Test deploy stack with changeset that has changes
	ctx := context.Background()

	templateContent := `{"AWSTemplateFormatVersion": "2010-09-09", "Resources": {"NewBucket": {"Type": "AWS::S3::Bucket"}}}`

	// Set up mocks
	mockCfnOps := &MockCloudFormationOperations{}
	mockClient := &MockAWSClient{}

	mockClient.On("NewCloudFormationOperations").Return(mockCfnOps)

	// Mock StackExists call (existing stack)
	mockCfnOps.On("StackExists", mock.Anything, "test-stack").Return(true, nil)

	// Mock differ operations
	currentStackInfo := &awsinternal.StackInfo{
		Name:       "test-stack",
		Status:     "UPDATE_COMPLETE",
		Parameters: map[string]string{},
		Tags:       map[string]string{},
	}
	mockCfnOps.On("DescribeStack", mock.Anything, "test-stack").Return(currentStackInfo, nil)
	mockCfnOps.On("GetTemplate", mock.Anything, "test-stack").Return(`{"AWSTemplateFormatVersion": "2010-09-09", "Resources": {"OldBucket": {"Type": "AWS::S3::Bucket"}}}`, nil)

	// Mock changeset operations for changes scenario (both differ and deployer create changesets)
	changeSetOutput := &cloudformation.CreateChangeSetOutput{
		Id: aws.String("test-changeset-id"),
	}
	mockCfnOps.On("CreateChangeSet", mock.Anything, mock.AnythingOfType("*cloudformation.CreateChangeSetInput")).Return(changeSetOutput, nil).Times(2)

	// Mock describe changeset - return complete status with changes
	describeOutput := &cloudformation.DescribeChangeSetOutput{
		Status: types.ChangeSetStatusCreateComplete,
		Changes: []types.Change{
			{
				ResourceChange: &types.ResourceChange{
					Action:             types.ChangeActionModify,
					ResourceType:       aws.String("AWS::S3::Bucket"),
					LogicalResourceId:  aws.String("TestBucket"),
					PhysicalResourceId: aws.String("test-bucket-123"),
					Replacement:        types.ReplacementFalse,
				},
			},
		},
	}
	mockCfnOps.On("DescribeChangeSet", mock.Anything, mock.AnythingOfType("*cloudformation.DescribeChangeSetInput")).Return(describeOutput, nil).Times(4)

	// Mock execute changeset
	mockCfnOps.On("ExecuteChangeSet", mock.Anything, mock.AnythingOfType("*cloudformation.ExecuteChangeSetInput")).Return(&cloudformation.ExecuteChangeSetOutput{}, nil)

	// Mock wait for stack operation
	mockCfnOps.On("WaitForStackOperation", mock.Anything, "test-stack", mock.AnythingOfType("func(aws.StackEvent)")).Return(nil)

	// Mock delete changeset (cleanup after successful deployment - both differ and deployer delete changesets)
	mockCfnOps.On("DeleteChangeSet", mock.Anything, mock.AnythingOfType("*cloudformation.DeleteChangeSetInput")).Return(&cloudformation.DeleteChangeSetOutput{}, nil).Times(2)

	// Create deployer with mock client
	deployer := NewAWSDeployer(mockClient)

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
}

func TestAWSDeployer_ValidateTemplate_Success(t *testing.T) {
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
	deployer := NewAWSDeployer(mockClient)

	// Execute
	err = deployer.ValidateTemplate(ctx, templateFile)

	// Verify
	assert.NoError(t, err)
	mockClient.AssertExpectations(t)
	mockCfnOps.AssertExpectations(t)
}

func TestAWSDeployer_ValidateTemplate_FileNotFound(t *testing.T) {
	// Test validate template with non-existent file
	ctx := context.Background()

	mockClient := &MockAWSClient{}
	deployer := NewAWSDeployer(mockClient)

	// Execute with non-existent file
	err := deployer.ValidateTemplate(ctx, "/nonexistent/template.json")

	// Verify
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read template")
	assert.Contains(t, err.Error(), "no such file or directory")

	// AWS client should not be called
	mockClient.AssertExpectations(t)
}

func TestAWSDeployer_ValidateTemplate_ValidationError(t *testing.T) {
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
	deployer := NewAWSDeployer(mockClient)

	// Execute ValidateTemplate instead - this test is for template validation
	err = deployer.ValidateTemplate(ctx, templateFile)

	// Verify
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "template validation failed")

	mockClient.AssertExpectations(t)
	mockCfnOps.AssertExpectations(t)
}

func TestAWSDeployer_DeployStack_WithYAMLTemplate(t *testing.T) {
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

	mockCfnOps.On("DeployStackWithCallback", mock.Anything, mock.MatchedBy(func(input awsinternal.DeployStackInput) bool {
		// Verify the template content was passed correctly
		return input.TemplateBody == templateContent &&
			input.StackName == "test-stack"
	}), mock.AnythingOfType("func(aws.StackEvent)")).Return(nil)

	// Create deployer with mock client
	deployer := NewAWSDeployer(mockClient)

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
}

func TestAWSDeployer_DeployStack_WithMultipleParametersAndTags(t *testing.T) {
	// Test deploying stack with multiple parameters and tags
	ctx := context.Background()

	// Set up mocks
	mockCfnOps := &MockCloudFormationOperations{}
	mockClient := &MockAWSClient{}

	mockClient.On("NewCloudFormationOperations").Return(mockCfnOps)

	// Mock StackExists call (new stack)
	mockCfnOps.On("StackExists", mock.Anything, "test-stack").Return(false, nil)

	mockCfnOps.On("DeployStackWithCallback", mock.Anything, mock.MatchedBy(func(input awsinternal.DeployStackInput) bool {
		return input.TemplateBody == "" &&
			input.StackName == "test-stack" &&
			len(input.Parameters) == 2 &&
			len(input.Tags) == 2 &&
			len(input.Capabilities) == 2
	}), mock.AnythingOfType("func(aws.StackEvent)")).Return(nil)

	// Create deployer with mock client
	deployer := NewAWSDeployer(mockClient)

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
}
