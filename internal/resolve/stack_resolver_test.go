/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package resolve

import (
	"context"
	"testing"

	"github.com/orien/stackaroo/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockConfigProvider is a mock implementation of config.ConfigProvider
type MockConfigProvider struct {
	mock.Mock
}

// Ensure MockConfigProvider implements config.ConfigProvider
var _ config.ConfigProvider = (*MockConfigProvider)(nil)

func (m *MockConfigProvider) LoadConfig(ctx context.Context, context string) (*config.Config, error) {
	args := m.Called(ctx, context)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*config.Config), args.Error(1)
}

func (m *MockConfigProvider) GetStack(stackName, context string) (*config.StackConfig, error) {
	args := m.Called(stackName, context)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*config.StackConfig), args.Error(1)
}

func (m *MockConfigProvider) ListStacks(context string) ([]string, error) {
	args := m.Called(context)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockConfigProvider) ListContexts() ([]string, error) {
	args := m.Called()
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockConfigProvider) Validate() error {
	args := m.Called()
	return args.Error(0)
}

// MockFileSystemResolver is a mock implementation of the FileSystemResolver interface
type MockFileSystemResolver struct {
	mock.Mock
}

func (m *MockFileSystemResolver) Resolve(templateURI string) (string, error) {
	args := m.Called(templateURI)
	return args.String(0), args.Error(1)
}

func TestNewStackResolver(t *testing.T) {
	// Test that we can create a new stack resolver
	mockConfigProvider := &MockConfigProvider{}
	mockFileSystemResolver := &MockFileSystemResolver{}

	stackResolver := NewStackResolver(mockConfigProvider)
	stackResolver.SetFileSystemResolver(mockFileSystemResolver)

	assert.NotNil(t, stackResolver, "stack resolver should not be nil")
}

func TestStackResolver_ResolveStack_Success(t *testing.T) {
	// Test successful resolution of a single stack
	ctx := context.Background()

	// Set up mocks
	mockConfigProvider := &MockConfigProvider{}
	mockFileSystemResolver := &MockFileSystemResolver{}

	// Mock data
	cfg := &config.Config{
		Project: "test-project",
		Tags: map[string]string{
			"Project": "test-project",
		},
	}

	stackConfig := &config.StackConfig{
		Name:     "vpc",
		Template: "templates/vpc.yaml",
		Parameters: map[string]string{
			"VpcCidr": "10.0.0.0/16",
		},
		Tags: map[string]string{
			"Stack": "vpc",
		},
		Capabilities: []string{"CAPABILITY_IAM"},
		Dependencies: []string{},
	}

	templateContent := `{
		"AWSTemplateFormatVersion": "2010-09-09",
		"Resources": {
			"VPC": {
				"Type": "AWS::EC2::VPC"
			}
		}
	}`

	// Set expectations
	mockConfigProvider.On("LoadConfig", ctx, "dev").Return(cfg, nil)
	mockConfigProvider.On("GetStack", "vpc", "dev").Return(stackConfig, nil)
	mockFileSystemResolver.On("Resolve", "templates/vpc.yaml").Return(templateContent, nil)

	// Create stack resolver
	stackResolver := NewStackResolver(mockConfigProvider)
	stackResolver.SetFileSystemResolver(mockFileSystemResolver)

	// Execute
	resolved, err := stackResolver.ResolveStack(ctx, "dev", "vpc")

	// Verify
	require.NoError(t, err)
	assert.NotNil(t, resolved)
	assert.Equal(t, "vpc", resolved.Name)
	assert.Equal(t, templateContent, resolved.TemplateBody)
	assert.Equal(t, "10.0.0.0/16", resolved.Parameters["VpcCidr"])
	assert.Equal(t, "test-project", resolved.Tags["Project"])
	assert.Equal(t, "vpc", resolved.Tags["Stack"])
	assert.Contains(t, resolved.Capabilities, "CAPABILITY_IAM")
	assert.Empty(t, resolved.Dependencies)

	// Verify all expectations were met
	mockConfigProvider.AssertExpectations(t)
	mockFileSystemResolver.AssertExpectations(t)
}

func TestStackResolver_ResolveStack_ConfigLoadError(t *testing.T) {
	// Test error handling when config loading fails
	ctx := context.Background()

	mockConfigProvider := &MockConfigProvider{}
	mockFileSystemResolver := &MockFileSystemResolver{}

	// Set expectation for config load failure
	mockConfigProvider.On("LoadConfig", ctx, "dev").Return(nil, assert.AnError)

	stackResolver := NewStackResolver(mockConfigProvider)
	stackResolver.SetFileSystemResolver(mockFileSystemResolver)

	resolved, err := stackResolver.ResolveStack(ctx, "dev", "vpc")

	assert.Error(t, err)
	assert.Nil(t, resolved)
	assert.Contains(t, err.Error(), "failed to load config")

	mockConfigProvider.AssertExpectations(t)
	mockFileSystemResolver.AssertExpectations(t)
}

func TestStackResolver_ResolveStack_StackNotFoundError(t *testing.T) {
	// Test error handling when stack is not found
	ctx := context.Background()

	mockConfigProvider := &MockConfigProvider{}
	mockFileSystemResolver := &MockFileSystemResolver{}

	cfg := &config.Config{Project: "test-project"}

	mockConfigProvider.On("LoadConfig", ctx, "dev").Return(cfg, nil)
	mockConfigProvider.On("GetStack", "nonexistent", "dev").Return(nil, assert.AnError)

	stackResolver := NewStackResolver(mockConfigProvider)
	stackResolver.SetFileSystemResolver(mockFileSystemResolver)

	resolved, err := stackResolver.ResolveStack(ctx, "dev", "nonexistent")

	assert.Error(t, err)
	assert.Nil(t, resolved)
	assert.Contains(t, err.Error(), "failed to get stack")

	mockConfigProvider.AssertExpectations(t)
	mockFileSystemResolver.AssertExpectations(t)
}

func TestStackResolver_ResolveStack_TemplateReadError(t *testing.T) {
	// Test error handling when template reading fails
	ctx := context.Background()

	mockConfigProvider := &MockConfigProvider{}
	mockFileSystemResolver := &MockFileSystemResolver{}

	cfg := &config.Config{Project: "test-project"}
	stackConfig := &config.StackConfig{
		Name:     "vpc",
		Template: "templates/missing.yaml",
	}

	mockConfigProvider.On("LoadConfig", ctx, "dev").Return(cfg, nil)
	mockConfigProvider.On("GetStack", "vpc", "dev").Return(stackConfig, nil)
	mockFileSystemResolver.On("Resolve", "templates/missing.yaml").Return("", assert.AnError)

	stackResolver := NewStackResolver(mockConfigProvider)
	stackResolver.SetFileSystemResolver(mockFileSystemResolver)

	resolved, err := stackResolver.ResolveStack(ctx, "dev", "vpc")

	assert.Error(t, err)
	assert.Nil(t, resolved)
	assert.Contains(t, err.Error(), "failed to read template")

	mockConfigProvider.AssertExpectations(t)
	mockFileSystemResolver.AssertExpectations(t)
}

func TestStackResolver_Resolve_MultipleStacks(t *testing.T) {
	// Test resolving multiple stacks
	ctx := context.Background()

	mockConfigProvider := &MockConfigProvider{}
	mockFileSystemResolver := &MockFileSystemResolver{}

	cfg := &config.Config{Project: "test-project"}

	vpcConfig := &config.StackConfig{
		Name:         "vpc",
		Template:     "templates/vpc.yaml",
		Dependencies: []string{},
	}

	appConfig := &config.StackConfig{
		Name:         "app",
		Template:     "templates/app.yaml",
		Dependencies: []string{"vpc"},
	}

	// Set expectations
	mockConfigProvider.On("LoadConfig", ctx, "dev").Return(cfg, nil).Times(2)
	mockConfigProvider.On("GetStack", "vpc", "dev").Return(vpcConfig, nil)
	mockConfigProvider.On("GetStack", "app", "dev").Return(appConfig, nil)
	mockFileSystemResolver.On("Resolve", "templates/vpc.yaml").Return("{}", nil)
	mockFileSystemResolver.On("Resolve", "templates/app.yaml").Return("{}", nil)

	stackResolver := NewStackResolver(mockConfigProvider)
	stackResolver.SetFileSystemResolver(mockFileSystemResolver)

	resolved, err := stackResolver.ResolveStacks(ctx, "dev", []string{"vpc", "app"})

	require.NoError(t, err)
	assert.NotNil(t, resolved)
	assert.Equal(t, "dev", resolved.Context)
	assert.Len(t, resolved.Stacks, 2)

	// Check deployment order - vpc should come before app due to dependency
	assert.Equal(t, []string{"vpc", "app"}, resolved.DeploymentOrder)

	mockConfigProvider.AssertExpectations(t)
	mockFileSystemResolver.AssertExpectations(t)
}

func TestStackResolver_Resolve_CircularDependency(t *testing.T) {
	// Test detection of circular dependencies
	ctx := context.Background()

	mockConfigProvider := &MockConfigProvider{}
	mockFileSystemResolver := &MockFileSystemResolver{}

	cfg := &config.Config{Project: "test-project"}

	// Create circular dependency: stack-a depends on stack-b, stack-b depends on stack-a
	stackAConfig := &config.StackConfig{
		Name:         "stack-a",
		Template:     "templates/a.yaml",
		Dependencies: []string{"stack-b"},
	}

	stackBConfig := &config.StackConfig{
		Name:         "stack-b",
		Template:     "templates/b.yaml",
		Dependencies: []string{"stack-a"},
	}

	mockConfigProvider.On("LoadConfig", ctx, "dev").Return(cfg, nil).Times(2)
	mockConfigProvider.On("GetStack", "stack-a", "dev").Return(stackAConfig, nil)
	mockConfigProvider.On("GetStack", "stack-b", "dev").Return(stackBConfig, nil)
	mockFileSystemResolver.On("Resolve", "templates/a.yaml").Return("{}", nil)
	mockFileSystemResolver.On("Resolve", "templates/b.yaml").Return("{}", nil)

	stackResolver := NewStackResolver(mockConfigProvider)
	stackResolver.SetFileSystemResolver(mockFileSystemResolver)

	resolved, err := stackResolver.ResolveStacks(ctx, "dev", []string{"stack-a", "stack-b"})

	assert.Error(t, err)
	assert.Nil(t, resolved)
	assert.Contains(t, err.Error(), "circular dependency detected")

	mockConfigProvider.AssertExpectations(t)
	mockFileSystemResolver.AssertExpectations(t)
}

func TestStackResolver_Resolve_EmptyStackList(t *testing.T) {
	// Test resolving empty stack list
	ctx := context.Background()

	mockConfigProvider := &MockConfigProvider{}
	mockFileSystemResolver := &MockFileSystemResolver{}

	stackResolver := NewStackResolver(mockConfigProvider)
	stackResolver.SetFileSystemResolver(mockFileSystemResolver)

	resolved, err := stackResolver.ResolveStacks(ctx, "dev", []string{})

	require.NoError(t, err)
	assert.NotNil(t, resolved)
	assert.Equal(t, "dev", resolved.Context)
	assert.Empty(t, resolved.Stacks)
	assert.Empty(t, resolved.DeploymentOrder)

	mockConfigProvider.AssertExpectations(t)
	mockFileSystemResolver.AssertExpectations(t)
}

func TestStackResolver_Resolve_ComplexDependencyChain(t *testing.T) {
	// Test complex dependency chain: vpc -> security -> database -> app
	ctx := context.Background()

	mockConfigProvider := &MockConfigProvider{}
	mockFileSystemResolver := &MockFileSystemResolver{}

	cfg := &config.Config{Project: "test-project"}

	vpcConfig := &config.StackConfig{
		Name:         "vpc",
		Template:     "templates/vpc.yaml",
		Dependencies: []string{},
	}

	securityConfig := &config.StackConfig{
		Name:         "security",
		Template:     "templates/security.yaml",
		Dependencies: []string{"vpc"},
	}

	databaseConfig := &config.StackConfig{
		Name:         "database",
		Template:     "templates/database.yaml",
		Dependencies: []string{"security"},
	}

	appConfig := &config.StackConfig{
		Name:         "app",
		Template:     "templates/app.yaml",
		Dependencies: []string{"database"},
	}

	// Set expectations
	mockConfigProvider.On("LoadConfig", ctx, "prod").Return(cfg, nil).Times(4)
	mockConfigProvider.On("GetStack", "vpc", "prod").Return(vpcConfig, nil)
	mockConfigProvider.On("GetStack", "security", "prod").Return(securityConfig, nil)
	mockConfigProvider.On("GetStack", "database", "prod").Return(databaseConfig, nil)
	mockConfigProvider.On("GetStack", "app", "prod").Return(appConfig, nil)
	mockFileSystemResolver.On("Resolve", "templates/vpc.yaml").Return("{}", nil)
	mockFileSystemResolver.On("Resolve", "templates/security.yaml").Return("{}", nil)
	mockFileSystemResolver.On("Resolve", "templates/database.yaml").Return("{}", nil)
	mockFileSystemResolver.On("Resolve", "templates/app.yaml").Return("{}", nil)

	stackResolver := NewStackResolver(mockConfigProvider)
	stackResolver.SetFileSystemResolver(mockFileSystemResolver)

	// Request stacks in random order
	resolved, err := stackResolver.ResolveStacks(ctx, "prod", []string{"app", "vpc", "database", "security"})

	require.NoError(t, err)
	assert.NotNil(t, resolved)
	assert.Equal(t, "prod", resolved.Context)
	assert.Len(t, resolved.Stacks, 4)

	// Verify correct dependency order
	expectedOrder := []string{"vpc", "security", "database", "app"}
	assert.Equal(t, expectedOrder, resolved.DeploymentOrder)

	mockConfigProvider.AssertExpectations(t)
	mockFileSystemResolver.AssertExpectations(t)
}

func TestStackResolver_ResolveStack_ParameterInheritance(t *testing.T) {
	// Test parameter inheritance from global config
	ctx := context.Background()

	mockConfigProvider := &MockConfigProvider{}
	mockFileSystemResolver := &MockFileSystemResolver{}

	cfg := &config.Config{
		Project: "test-project",
		Tags: map[string]string{
			"Project":     "test-project",
			"Environment": "staging",
		},
	}

	stackConfig := &config.StackConfig{
		Name:     "web",
		Template: "templates/web.yaml",
		Parameters: map[string]string{
			"InstanceType": "t3.medium",
		},
		Tags: map[string]string{
			"Component": "web-server",
			"Project":   "overridden-project", // Should override global
		},
	}

	mockConfigProvider.On("LoadConfig", ctx, "staging").Return(cfg, nil)
	mockConfigProvider.On("GetStack", "web", "staging").Return(stackConfig, nil)
	mockFileSystemResolver.On("Resolve", "templates/web.yaml").Return("{}", nil)

	stackResolver := NewStackResolver(mockConfigProvider)
	stackResolver.SetFileSystemResolver(mockFileSystemResolver)

	resolved, err := stackResolver.ResolveStack(ctx, "staging", "web")

	require.NoError(t, err)
	assert.NotNil(t, resolved)

	// Verify parameter inheritance
	assert.Equal(t, "t3.medium", resolved.Parameters["InstanceType"])

	// Verify tag inheritance and override
	assert.Equal(t, "overridden-project", resolved.Tags["Project"]) // Stack overrides global
	assert.Equal(t, "staging", resolved.Tags["Environment"])        // From global
	assert.Equal(t, "web-server", resolved.Tags["Component"])       // From stack only

	mockConfigProvider.AssertExpectations(t)
	mockFileSystemResolver.AssertExpectations(t)
}

func TestStackResolver_Resolve_MissingDependency(t *testing.T) {
	// Test handling of missing dependency (dependency not in resolved stack list)
	ctx := context.Background()

	mockConfigProvider := &MockConfigProvider{}
	mockFileSystemResolver := &MockFileSystemResolver{}

	cfg := &config.Config{Project: "test-project"}

	// App depends on database, but we're only resolving app
	appConfig := &config.StackConfig{
		Name:         "app",
		Template:     "templates/app.yaml",
		Dependencies: []string{"database"}, // database not in resolution list
	}

	mockConfigProvider.On("LoadConfig", ctx, "dev").Return(cfg, nil)
	mockConfigProvider.On("GetStack", "app", "dev").Return(appConfig, nil)
	mockFileSystemResolver.On("Resolve", "templates/app.yaml").Return("{}", nil)

	stackResolver := NewStackResolver(mockConfigProvider)
	stackResolver.SetFileSystemResolver(mockFileSystemResolver)

	// Only resolve app, not its dependency
	resolved, err := stackResolver.ResolveStacks(ctx, "dev", []string{"app"})

	// Should succeed - missing dependencies are ignored for dependency ordering
	// (they might be deployed separately)
	require.NoError(t, err)
	assert.NotNil(t, resolved)
	assert.Len(t, resolved.Stacks, 1)
	assert.Equal(t, []string{"app"}, resolved.DeploymentOrder)
	assert.Equal(t, []string{"database"}, resolved.Stacks[0].Dependencies)

	mockConfigProvider.AssertExpectations(t)
	mockFileSystemResolver.AssertExpectations(t)
}
