/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package resolve

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/orien/stackaroo/internal/aws"
	"github.com/orien/stackaroo/internal/config"
)

// Helper function to convert string maps to ParameterValue maps for tests
func convertStringMapToParameterValues(stringMap map[string]string) map[string]*config.ParameterValue {
	if stringMap == nil {
		return nil
	}
	result := make(map[string]*config.ParameterValue, len(stringMap))
	for key, value := range stringMap {
		result[key] = &config.ParameterValue{
			ResolutionType: "literal",
			ResolutionConfig: map[string]string{
				"value": value,
			},
		}
	}
	return result
}

func TestNewStackResolver(t *testing.T) {
	// Test that we can create a new stack resolver
	mockConfigProvider := &config.MockConfigProvider{}
	mockFileSystemResolver := &MockFileSystemResolver{}
	mockCfnOperations := &aws.MockCloudFormationOperations{}

	stackResolver := NewStackResolver(mockConfigProvider, mockCfnOperations)
	stackResolver.SetFileSystemResolver(mockFileSystemResolver)

	assert.NotNil(t, stackResolver, "stack resolver should not be nil")
}

func TestStackResolver_ResolveStack_Success(t *testing.T) {
	// Test successful resolution of a single stack
	ctx := context.Background()

	// Set up mocks
	mockConfigProvider := &config.MockConfigProvider{}
	mockFileSystemResolver := &MockFileSystemResolver{}
	mockCfnOperations := &aws.MockCloudFormationOperations{}

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
		Parameters: convertStringMapToParameterValues(map[string]string{
			"VpcCidr": "10.0.0.0/16",
		}),
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
	stackResolver := NewStackResolver(mockConfigProvider, mockCfnOperations)
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

func TestStackResolver_ResolveParameters_LiteralValues(t *testing.T) {
	// Test resolution of literal parameter values
	ctx := context.Background()

	mockConfigProvider := &config.MockConfigProvider{}
	mockCfnOperations := &aws.MockCloudFormationOperations{}
	resolver := NewStackResolver(mockConfigProvider, mockCfnOperations)

	// Test literal parameters only
	params := map[string]*config.ParameterValue{
		"Environment": {
			ResolutionType: "literal",
			ResolutionConfig: map[string]string{
				"value": "production",
			},
		},
		"InstanceType": {
			ResolutionType: "literal",
			ResolutionConfig: map[string]string{
				"value": "t3.medium",
			},
		},
	}

	resolved, err := resolver.resolveParameters(ctx, params, "prod")

	require.NoError(t, err)
	assert.Len(t, resolved, 2)
	assert.Equal(t, "production", resolved["Environment"])
	assert.Equal(t, "t3.medium", resolved["InstanceType"])
}

func TestStackResolver_ResolveParameters_StackOutputs(t *testing.T) {
	// Test resolution of stack output parameters
	ctx := context.Background()

	mockConfigProvider := &config.MockConfigProvider{}
	mockCfnOperations := &aws.MockCloudFormationOperations{}
	resolver := NewStackResolver(mockConfigProvider, mockCfnOperations)

	// Mock CloudFormation stack with outputs
	mockStack := &aws.Stack{
		Name: "vpc-stack",
		Outputs: map[string]string{
			"VpcId":    "vpc-12345",
			"SubnetId": "subnet-67890",
		},
	}

	mockCfnOperations.On("GetStack", ctx, "vpc-stack").Return(mockStack, nil)

	// Test stack output parameters
	params := map[string]*config.ParameterValue{
		"VpcId": {
			ResolutionType: "stack-output",
			ResolutionConfig: map[string]string{
				"stack_name": "vpc-stack",
				"output_key": "VpcId",
			},
		},
		"SubnetId": {
			ResolutionType: "stack-output",
			ResolutionConfig: map[string]string{
				"stack_name": "vpc-stack",
				"output_key": "SubnetId",
			},
		},
	}

	resolved, err := resolver.resolveParameters(ctx, params, "prod")

	require.NoError(t, err)
	assert.Len(t, resolved, 2)
	assert.Equal(t, "vpc-12345", resolved["VpcId"])
	assert.Equal(t, "subnet-67890", resolved["SubnetId"])

	mockCfnOperations.AssertExpectations(t)
}

func TestStackResolver_ResolveParameters_MixedTypes(t *testing.T) {
	// Test resolution of mixed literal and output parameters
	ctx := context.Background()

	mockConfigProvider := &config.MockConfigProvider{}
	mockCfnOperations := &aws.MockCloudFormationOperations{}
	resolver := NewStackResolver(mockConfigProvider, mockCfnOperations)

	// Mock CloudFormation stack
	mockStack := &aws.Stack{
		Name: "networking",
		Outputs: map[string]string{
			"VpcId": "vpc-abcdef",
		},
	}

	mockCfnOperations.On("GetStack", ctx, "networking").Return(mockStack, nil)

	// Test mixed parameter types
	params := map[string]*config.ParameterValue{
		"Environment": {
			ResolutionType: "literal",
			ResolutionConfig: map[string]string{
				"value": "staging",
			},
		},
		"VpcId": {
			ResolutionType: "stack-output",
			ResolutionConfig: map[string]string{
				"stack_name": "networking",
				"output_key": "VpcId",
			},
		},
	}

	resolved, err := resolver.resolveParameters(ctx, params, "staging")

	require.NoError(t, err)
	assert.Len(t, resolved, 2)
	assert.Equal(t, "staging", resolved["Environment"])
	assert.Equal(t, "vpc-abcdef", resolved["VpcId"])

	mockCfnOperations.AssertExpectations(t)
}

func TestStackResolver_ResolveParameters_ErrorCases(t *testing.T) {
	ctx := context.Background()

	t.Run("unsupported resolution type", func(t *testing.T) {
		mockConfigProvider := &config.MockConfigProvider{}
		mockCfnOperations := &aws.MockCloudFormationOperations{}
		resolver := NewStackResolver(mockConfigProvider, mockCfnOperations)

		params := map[string]*config.ParameterValue{
			"BadParam": {
				ResolutionType: "unsupported",
				ResolutionConfig: map[string]string{
					"something": "value",
				},
			},
		}

		_, err := resolver.resolveParameters(ctx, params, "dev")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported parameter resolution type 'unsupported'")
	})

	t.Run("literal missing value", func(t *testing.T) {
		mockConfigProvider := &config.MockConfigProvider{}
		mockCfnOperations := &aws.MockCloudFormationOperations{}
		resolver := NewStackResolver(mockConfigProvider, mockCfnOperations)

		params := map[string]*config.ParameterValue{
			"BadLiteral": {
				ResolutionType:   "literal",
				ResolutionConfig: map[string]string{
					// Missing "value" key
				},
			},
		}

		_, err := resolver.resolveParameters(ctx, params, "dev")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "parameter 'BadLiteral' is literal but missing 'value' config")
	})

	t.Run("stack not found", func(t *testing.T) {
		mockConfigProvider := &config.MockConfigProvider{}
		mockCfnOperations := &aws.MockCloudFormationOperations{}
		resolver := NewStackResolver(mockConfigProvider, mockCfnOperations)

		mockCfnOperations.On("GetStack", ctx, "missing-stack").Return(nil, fmt.Errorf("stack not found"))

		params := map[string]*config.ParameterValue{
			"VpcId": {
				ResolutionType: "stack-output",
				ResolutionConfig: map[string]string{
					"stack_name": "missing-stack",
					"output_key": "VpcId",
				},
			},
		}

		_, err := resolver.resolveParameters(ctx, params, "dev")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to resolve stack output parameter 'VpcId'")
		assert.Contains(t, err.Error(), "failed to get stack 'missing-stack'")

		mockCfnOperations.AssertExpectations(t)
	})

	t.Run("output key not found", func(t *testing.T) {
		mockConfigProvider := &config.MockConfigProvider{}
		mockCfnOperations := &aws.MockCloudFormationOperations{}
		resolver := NewStackResolver(mockConfigProvider, mockCfnOperations)

		mockStack := &aws.Stack{
			Name: "vpc-stack",
			Outputs: map[string]string{
				"VpcId": "vpc-12345",
				// Missing "MissingOutput"
			},
		}

		mockCfnOperations.On("GetStack", ctx, "vpc-stack").Return(mockStack, nil)

		params := map[string]*config.ParameterValue{
			"BadOutput": {
				ResolutionType: "stack-output",
				ResolutionConfig: map[string]string{
					"stack_name": "vpc-stack",
					"output_key": "MissingOutput",
				},
			},
		}

		_, err := resolver.resolveParameters(ctx, params, "dev")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to resolve stack output parameter 'BadOutput'")
		assert.Contains(t, err.Error(), "stack 'vpc-stack' does not have output 'MissingOutput'")

		mockCfnOperations.AssertExpectations(t)
	})
}

func TestStackResolver_ResolveStackOutput_MissingConfig(t *testing.T) {
	ctx := context.Background()

	mockConfigProvider := &config.MockConfigProvider{}
	mockCfnOperations := &aws.MockCloudFormationOperations{}
	resolver := NewStackResolver(mockConfigProvider, mockCfnOperations)

	t.Run("missing stack_name", func(t *testing.T) {
		outputConfig := map[string]string{
			"output_key": "VpcId",
			// Missing stack_name
		}

		_, err := resolver.resolveStackOutput(ctx, outputConfig, "us-west-2")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "stack output resolver missing required 'stack_name'")
	})

	t.Run("missing output_key", func(t *testing.T) {
		outputConfig := map[string]string{
			"stack_name": "vpc-stack",
			// Missing output_key
		}

		_, err := resolver.resolveStackOutput(ctx, outputConfig, "us-west-2")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "stack output resolver missing required 'output_key'")
	})
}

func TestStackResolver_ResolveStack_ConfigLoadError(t *testing.T) {
	// Test error handling when config loading fails
	ctx := context.Background()

	mockConfigProvider := &config.MockConfigProvider{}
	mockFileSystemResolver := &MockFileSystemResolver{}
	mockCfnOperations := &aws.MockCloudFormationOperations{}

	// Set expectation for config load failure
	mockConfigProvider.On("LoadConfig", ctx, "dev").Return(nil, assert.AnError)

	stackResolver := NewStackResolver(mockConfigProvider, mockCfnOperations)
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

	mockConfigProvider := &config.MockConfigProvider{}
	mockFileSystemResolver := &MockFileSystemResolver{}

	cfg := &config.Config{Project: "test-project"}

	mockConfigProvider.On("LoadConfig", ctx, "dev").Return(cfg, nil)
	mockConfigProvider.On("GetStack", "nonexistent", "dev").Return(nil, assert.AnError)

	mockCfnOperations := &aws.MockCloudFormationOperations{}
	stackResolver := NewStackResolver(mockConfigProvider, mockCfnOperations)
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

	mockConfigProvider := &config.MockConfigProvider{}
	mockFileSystemResolver := &MockFileSystemResolver{}

	cfg := &config.Config{Project: "test-project"}
	stackConfig := &config.StackConfig{
		Name:     "vpc",
		Template: "templates/missing.yaml",
	}

	mockConfigProvider.On("LoadConfig", ctx, "dev").Return(cfg, nil)
	mockConfigProvider.On("GetStack", "vpc", "dev").Return(stackConfig, nil)
	mockFileSystemResolver.On("Resolve", "templates/missing.yaml").Return("", assert.AnError)

	mockCfnOperations := &aws.MockCloudFormationOperations{}
	stackResolver := NewStackResolver(mockConfigProvider, mockCfnOperations)
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

	mockConfigProvider := &config.MockConfigProvider{}
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

	mockCfnOperations := &aws.MockCloudFormationOperations{}
	stackResolver := NewStackResolver(mockConfigProvider, mockCfnOperations)
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

	mockConfigProvider := &config.MockConfigProvider{}
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

	mockCfnOperations := &aws.MockCloudFormationOperations{}
	stackResolver := NewStackResolver(mockConfigProvider, mockCfnOperations)
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

	mockConfigProvider := &config.MockConfigProvider{}
	mockFileSystemResolver := &MockFileSystemResolver{}

	mockCfnOperations := &aws.MockCloudFormationOperations{}
	stackResolver := NewStackResolver(mockConfigProvider, mockCfnOperations)
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

	mockConfigProvider := &config.MockConfigProvider{}
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

	mockCfnOperations := &aws.MockCloudFormationOperations{}
	stackResolver := NewStackResolver(mockConfigProvider, mockCfnOperations)
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

	mockConfigProvider := &config.MockConfigProvider{}
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
		Parameters: convertStringMapToParameterValues(map[string]string{
			"InstanceType": "t3.medium",
		}),
		Tags: map[string]string{
			"Component": "web-server",
			"Project":   "overridden-project", // Should override global
		},
	}

	mockConfigProvider.On("LoadConfig", ctx, "staging").Return(cfg, nil)
	mockConfigProvider.On("GetStack", "web", "staging").Return(stackConfig, nil)
	mockFileSystemResolver.On("Resolve", "templates/web.yaml").Return("{}", nil)

	mockCfnOperations := &aws.MockCloudFormationOperations{}
	stackResolver := NewStackResolver(mockConfigProvider, mockCfnOperations)
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

	mockConfigProvider := &config.MockConfigProvider{}
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

	mockCfnOperations := &aws.MockCloudFormationOperations{}
	stackResolver := NewStackResolver(mockConfigProvider, mockCfnOperations)
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

func TestStackResolver_GetDependencyOrder_Success(t *testing.T) {
	// Test successful dependency order calculation without full resolution
	mockConfigProvider := &config.MockConfigProvider{}
	mockCfnOperations := &aws.MockCloudFormationOperations{}

	vpcConfig := &config.StackConfig{
		Name:         "vpc",
		Dependencies: []string{},
	}

	appConfig := &config.StackConfig{
		Name:         "app",
		Dependencies: []string{"vpc"},
	}

	mockConfigProvider.On("GetStack", "vpc", "dev").Return(vpcConfig, nil)
	mockConfigProvider.On("GetStack", "app", "dev").Return(appConfig, nil)

	stackResolver := NewStackResolver(mockConfigProvider, mockCfnOperations)

	order, err := stackResolver.GetDependencyOrder("dev", []string{"app", "vpc"})

	require.NoError(t, err)
	assert.Equal(t, []string{"vpc", "app"}, order)

	mockConfigProvider.AssertExpectations(t)
}

func TestStackResolver_GetDependencyOrder_EmptyList(t *testing.T) {
	// Test empty stack list
	mockConfigProvider := &config.MockConfigProvider{}
	mockCfnOperations := &aws.MockCloudFormationOperations{}

	stackResolver := NewStackResolver(mockConfigProvider, mockCfnOperations)

	order, err := stackResolver.GetDependencyOrder("dev", []string{})

	require.NoError(t, err)
	assert.Empty(t, order)

	mockConfigProvider.AssertExpectations(t)
}

func TestStackResolver_GetDependencyOrder_NoDependencies(t *testing.T) {
	// Test stacks with no dependencies
	mockConfigProvider := &config.MockConfigProvider{}
	mockCfnOperations := &aws.MockCloudFormationOperations{}

	vpcConfig := &config.StackConfig{
		Name:         "vpc",
		Dependencies: []string{},
	}

	appConfig := &config.StackConfig{
		Name:         "app",
		Dependencies: []string{},
	}

	dbConfig := &config.StackConfig{
		Name:         "database",
		Dependencies: []string{},
	}

	mockConfigProvider.On("GetStack", "vpc", "dev").Return(vpcConfig, nil)
	mockConfigProvider.On("GetStack", "app", "dev").Return(appConfig, nil)
	mockConfigProvider.On("GetStack", "database", "dev").Return(dbConfig, nil)

	stackResolver := NewStackResolver(mockConfigProvider, mockCfnOperations)

	order, err := stackResolver.GetDependencyOrder("dev", []string{"database", "app", "vpc"})

	require.NoError(t, err)
	// Should be alphabetical since no dependencies
	assert.Equal(t, []string{"app", "database", "vpc"}, order)

	mockConfigProvider.AssertExpectations(t)
}

func TestStackResolver_GetDependencyOrder_ComplexChain(t *testing.T) {
	// Test complex dependency chain: vpc -> security -> database -> app
	mockConfigProvider := &config.MockConfigProvider{}
	mockCfnOperations := &aws.MockCloudFormationOperations{}

	vpcConfig := &config.StackConfig{
		Name:         "vpc",
		Dependencies: []string{},
	}

	securityConfig := &config.StackConfig{
		Name:         "security",
		Dependencies: []string{"vpc"},
	}

	databaseConfig := &config.StackConfig{
		Name:         "database",
		Dependencies: []string{"security"},
	}

	appConfig := &config.StackConfig{
		Name:         "app",
		Dependencies: []string{"database"},
	}

	mockConfigProvider.On("GetStack", "vpc", "prod").Return(vpcConfig, nil)
	mockConfigProvider.On("GetStack", "security", "prod").Return(securityConfig, nil)
	mockConfigProvider.On("GetStack", "database", "prod").Return(databaseConfig, nil)
	mockConfigProvider.On("GetStack", "app", "prod").Return(appConfig, nil)

	stackResolver := NewStackResolver(mockConfigProvider, mockCfnOperations)

	// Request stacks in random order
	order, err := stackResolver.GetDependencyOrder("prod", []string{"app", "vpc", "database", "security"})

	require.NoError(t, err)
	expectedOrder := []string{"vpc", "security", "database", "app"}
	assert.Equal(t, expectedOrder, order)

	mockConfigProvider.AssertExpectations(t)
}

func TestStackResolver_GetDependencyOrder_CircularDependency(t *testing.T) {
	// Test detection of circular dependencies
	mockConfigProvider := &config.MockConfigProvider{}
	mockCfnOperations := &aws.MockCloudFormationOperations{}

	// Create circular dependency: stack-a depends on stack-b, stack-b depends on stack-a
	stackAConfig := &config.StackConfig{
		Name:         "stack-a",
		Dependencies: []string{"stack-b"},
	}

	stackBConfig := &config.StackConfig{
		Name:         "stack-b",
		Dependencies: []string{"stack-a"},
	}

	mockConfigProvider.On("GetStack", "stack-a", "dev").Return(stackAConfig, nil)
	mockConfigProvider.On("GetStack", "stack-b", "dev").Return(stackBConfig, nil)

	stackResolver := NewStackResolver(mockConfigProvider, mockCfnOperations)

	order, err := stackResolver.GetDependencyOrder("dev", []string{"stack-a", "stack-b"})

	assert.Error(t, err)
	assert.Nil(t, order)
	assert.Contains(t, err.Error(), "circular dependency detected")

	mockConfigProvider.AssertExpectations(t)
}

func TestStackResolver_GetDependencyOrder_StackNotFound(t *testing.T) {
	// Test error handling when stack config is not found
	mockConfigProvider := &config.MockConfigProvider{}
	mockCfnOperations := &aws.MockCloudFormationOperations{}

	mockConfigProvider.On("GetStack", "nonexistent", "dev").Return(nil, assert.AnError)

	stackResolver := NewStackResolver(mockConfigProvider, mockCfnOperations)

	order, err := stackResolver.GetDependencyOrder("dev", []string{"nonexistent"})

	assert.Error(t, err)
	assert.Nil(t, order)
	assert.Contains(t, err.Error(), "failed to get stack config")

	mockConfigProvider.AssertExpectations(t)
}

func TestStackResolver_GetDependencyOrder_MissingDependency(t *testing.T) {
	// Test handling of missing dependency (dependency not in resolution list)
	mockConfigProvider := &config.MockConfigProvider{}
	mockCfnOperations := &aws.MockCloudFormationOperations{}

	// App depends on database, but we're only calculating order for app
	appConfig := &config.StackConfig{
		Name:         "app",
		Dependencies: []string{"database"}, // database not in resolution list
	}

	mockConfigProvider.On("GetStack", "app", "dev").Return(appConfig, nil)

	stackResolver := NewStackResolver(mockConfigProvider, mockCfnOperations)

	// Only calculate order for app, not its dependency
	order, err := stackResolver.GetDependencyOrder("dev", []string{"app"})

	// Should succeed - missing dependencies are ignored for dependency ordering
	// (they might be deployed separately)
	require.NoError(t, err)
	assert.Equal(t, []string{"app"}, order)

	mockConfigProvider.AssertExpectations(t)
}

func TestStackResolver_GetDependencyOrder_MultipleDependenciesPerStack(t *testing.T) {
	// Test stack with multiple dependencies
	mockConfigProvider := &config.MockConfigProvider{}
	mockCfnOperations := &aws.MockCloudFormationOperations{}

	vpcConfig := &config.StackConfig{
		Name:         "vpc",
		Dependencies: []string{},
	}

	securityConfig := &config.StackConfig{
		Name:         "security",
		Dependencies: []string{},
	}

	appConfig := &config.StackConfig{
		Name:         "app",
		Dependencies: []string{"vpc", "security"}, // Multiple dependencies
	}

	mockConfigProvider.On("GetStack", "vpc", "dev").Return(vpcConfig, nil)
	mockConfigProvider.On("GetStack", "security", "dev").Return(securityConfig, nil)
	mockConfigProvider.On("GetStack", "app", "dev").Return(appConfig, nil)

	stackResolver := NewStackResolver(mockConfigProvider, mockCfnOperations)

	order, err := stackResolver.GetDependencyOrder("dev", []string{"app", "vpc", "security"})

	require.NoError(t, err)
	assert.Len(t, order, 3)

	// App should be last
	assert.Equal(t, "app", order[2])

	// VPC and security should be before app (order between them doesn't matter since they're independent)
	assert.Contains(t, order[:2], "vpc")
	assert.Contains(t, order[:2], "security")

	mockConfigProvider.AssertExpectations(t)
}
