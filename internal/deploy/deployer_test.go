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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/orien/stackaroo/internal/aws"
	"github.com/orien/stackaroo/internal/config"
)

// MockAWSClient is a mock implementation of aws.ClientInterface
type MockAWSClient struct {
	mock.Mock
}

func (m *MockAWSClient) NewCloudFormationOperations() aws.CloudFormationOperationsInterface {
	args := m.Called()
	return args.Get(0).(aws.CloudFormationOperationsInterface)
}

// MockCloudFormationOperations is a mock implementation of aws.CloudFormationOperationsInterface
type MockCloudFormationOperations struct {
	mock.Mock
}

func (m *MockCloudFormationOperations) DeployStack(ctx context.Context, input aws.DeployStackInput) error {
	args := m.Called(ctx, input)
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
	// Set up mock expectations - now expecting resolved parameters and tags from StackConfig
	mockCfnOps.On("DeployStack", mock.Anything, mock.MatchedBy(func(input aws.DeployStackInput) bool {
		return input.StackName == "test-stack" &&
			len(input.Parameters) == 1 &&
			input.Parameters[0].Key == "Param1" &&
			input.Parameters[0].Value == "value1" &&
			len(input.Tags) == 1 &&
			input.Tags["Environment"] == "test" &&
			len(input.Capabilities) == 1 &&
			input.Capabilities[0] == "CAPABILITY_IAM"
	})).Return(nil)

	// Create deployer with mock client
	deployer := NewAWSDeployer(mockClient)

	// Create stack config
	stackConfig := &config.StackConfig{
		Name:         "test-stack",
		Template:     templateFile,
		Parameters:   map[string]string{"Param1": "value1"},
		Tags:         map[string]string{"Environment": "test"},
		Dependencies: []string{},
		Capabilities: []string{"CAPABILITY_IAM"},
	}

	// Execute
	err = deployer.DeployStack(ctx, stackConfig)

	// Verify
	assert.NoError(t, err)
	mockClient.AssertExpectations(t)
	mockCfnOps.AssertExpectations(t)
}

func TestAWSDeployer_DeployStack_FileNotFound(t *testing.T) {
	// Test deploy stack with non-existent template file
	ctx := context.Background()

	mockClient := &MockAWSClient{}
	deployer := NewAWSDeployer(mockClient)

	// Create stack config with non-existent template file
	stackConfig := &config.StackConfig{
		Name:         "test-stack",
		Template:     "/nonexistent/template.json",
		Parameters:   map[string]string{},
		Tags:         map[string]string{},
		Dependencies: []string{},
		Capabilities: []string{"CAPABILITY_IAM"},
	}

	// Execute with non-existent file
	err := deployer.DeployStack(ctx, stackConfig)

	// Verify
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read template")
	assert.Contains(t, err.Error(), "no such file or directory")

	// AWS client should not be called
	mockClient.AssertExpectations(t)
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
	mockCfnOps.On("DeployStack", ctx, mock.Anything).Return(errors.New("AWS deployment error"))

	// Create deployer with mock client
	deployer := NewAWSDeployer(mockClient)

	// Create stack config with valid template file
	stackConfig := &config.StackConfig{
		Name:         "test-stack",
		Template:     templateFile,
		Parameters:   map[string]string{},
		Tags:         map[string]string{},
		Dependencies: []string{},
		Capabilities: []string{"CAPABILITY_IAM"},
	}

	// Execute
	err = deployer.DeployStack(ctx, stackConfig)

	// Verify
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to deploy stack")
	assert.Contains(t, err.Error(), "AWS deployment error")

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

func TestAWSDeployer_ReadTemplateFile_Success(t *testing.T) {
	// Test reading template file through DeployStack (since readTemplateFile is private)
	ctx := context.Background()

	// Create temporary template file with specific content
	tmpDir := t.TempDir()
	templateFile := filepath.Join(tmpDir, "test-template.yaml")
	templateContent := `AWSTemplateFormatVersion: '2010-09-09'
Resources:
  MyBucket:
    Type: AWS::S3::Bucket
    Properties:
      BucketName: my-test-bucket`

	err := os.WriteFile(templateFile, []byte(templateContent), 0644)
	require.NoError(t, err)

	// Set up mocks to capture the template content
	mockCfnOps := &MockCloudFormationOperations{}
	mockClient := &MockAWSClient{}

	mockClient.On("NewCloudFormationOperations").Return(mockCfnOps)
	mockCfnOps.On("DeployStack", ctx, mock.MatchedBy(func(input aws.DeployStackInput) bool {
		// Verify the template content was read correctly
		return input.TemplateBody == templateContent
	})).Return(nil)

	// Create deployer with mock client
	deployer := NewAWSDeployer(mockClient)

	// Create stack config
	stackConfig := &config.StackConfig{
		Name:         "test-stack",
		Template:     templateFile,
		Parameters:   map[string]string{},
		Tags:         map[string]string{},
		Dependencies: []string{},
		Capabilities: []string{"CAPABILITY_IAM"},
	}

	// Execute
	err = deployer.DeployStack(ctx, stackConfig)

	// Verify
	assert.NoError(t, err)
	mockClient.AssertExpectations(t)
	mockCfnOps.AssertExpectations(t)
}

func TestAWSDeployer_ReadTemplateFile_EmptyFile(t *testing.T) {
	// Test reading empty template file
	ctx := context.Background()

	// Create empty template file
	tmpDir := t.TempDir()
	templateFile := filepath.Join(tmpDir, "empty-template.json")

	err := os.WriteFile(templateFile, []byte(""), 0644)
	require.NoError(t, err)

	// Set up mocks
	mockCfnOps := &MockCloudFormationOperations{}
	mockClient := &MockAWSClient{}

	mockClient.On("NewCloudFormationOperations").Return(mockCfnOps)
	mockCfnOps.On("DeployStack", ctx, mock.MatchedBy(func(input aws.DeployStackInput) bool {
		return input.TemplateBody == ""
	})).Return(nil)

	// Create deployer with mock client
	deployer := NewAWSDeployer(mockClient)

	// Create stack config with parameters and tags
	stackConfig := &config.StackConfig{
		Name:         "test-stack",
		Template:     templateFile,
		Parameters:   map[string]string{"Environment": "test", "InstanceType": "t3.micro"},
		Tags:         map[string]string{"Project": "stackaroo", "Environment": "test"},
		Dependencies: []string{},
		Capabilities: []string{"CAPABILITY_IAM", "CAPABILITY_NAMED_IAM"},
	}

	// Execute
	err = deployer.DeployStack(ctx, stackConfig)

	// Verify
	assert.NoError(t, err)
	mockClient.AssertExpectations(t)
	mockCfnOps.AssertExpectations(t)
}

func TestAWSDeployer_ReadTemplateFile_PermissionDenied(t *testing.T) {
	// Test reading template file with permission denied
	// Skip on Windows as file permissions work differently
	if os.Getenv("GOOS") == "windows" {
		t.Skip("Skipping permission test on Windows")
	}

	ctx := context.Background()

	// Create temporary template file and remove read permissions
	tmpDir := t.TempDir()
	templateFile := filepath.Join(tmpDir, "no-read-template.json")
	templateContent := `{"AWSTemplateFormatVersion": "2010-09-09"}`

	err := os.WriteFile(templateFile, []byte(templateContent), 0644)
	require.NoError(t, err)

	// Remove read permissions
	err = os.Chmod(templateFile, 0000)
	require.NoError(t, err)

	// Restore permissions for cleanup
	defer func() {
		_ = os.Chmod(templateFile, 0644)
	}()

	mockClient := &MockAWSClient{}
	// Create deployer with mock client
	deployer := NewAWSDeployer(mockClient)

	// Create stack config
	stackConfig := &config.StackConfig{
		Name:         "test-stack",
		Template:     templateFile,
		Parameters:   map[string]string{"Param1": "value1"},
		Tags:         map[string]string{"Environment": "test"},
		Dependencies: []string{},
		Capabilities: []string{"CAPABILITY_IAM"},
	}

	// Execute
	err = deployer.DeployStack(ctx, stackConfig)

	// Verify
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read template")
	assert.Contains(t, err.Error(), "permission denied")

	// AWS client should not be called
	mockClient.AssertExpectations(t)
}
