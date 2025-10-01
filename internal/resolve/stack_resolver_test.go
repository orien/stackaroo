/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package resolve

import (
	"context"
	"fmt"
	"testing"

	"github.com/orien/stackaroo/internal/aws"
	"github.com/orien/stackaroo/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
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
	mockFactory, _ := aws.NewMockClientFactoryForRegion("us-east-1")

	stackResolver := NewStackResolver(mockConfigProvider, mockFactory)
	stackResolver.SetFileSystemResolver(mockFileSystemResolver)

	assert.NotNil(t, stackResolver, "stack resolver should not be nil")
}

func TestStackResolver_ResolveStack_Success(t *testing.T) {
	// Test successful resolution of a single stack
	ctx := context.Background()

	// Set up mocks
	mockConfigProvider := &config.MockConfigProvider{}
	mockFileSystemResolver := &MockFileSystemResolver{}
	mockTemplateProcessor := &MockTemplateProcessor{}
	mockFactory, _ := aws.NewMockClientFactoryForRegion("us-east-1")

	// Mock data
	cfg := &config.Config{
		Project: "test-project",
		Tags: map[string]string{
			"Project": "test-project",
		},
		Context: &config.ContextConfig{
			Name:    "dev",
			Account: "123456789012",
			Region:  "us-east-1",
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
	mockTemplateProcessor.On("Process", templateContent, mock.AnythingOfType("map[string]interface {}")).Return(templateContent, nil)

	// Create stack resolver
	stackResolver := NewStackResolver(mockConfigProvider, mockFactory)
	stackResolver.SetFileSystemResolver(mockFileSystemResolver)
	stackResolver.SetTemplateProcessor(mockTemplateProcessor)

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
	mockTemplateProcessor.AssertExpectations(t)
}

func TestStackResolver_ResolveParameters_LiteralValues(t *testing.T) {
	// Test resolution of literal parameter values
	ctx := context.Background()

	mockConfigProvider := &config.MockConfigProvider{}
	mockFactory, _ := aws.NewMockClientFactoryForRegion("us-east-1")
	resolver := NewStackResolver(mockConfigProvider, mockFactory)

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
	mockFactory, mockCfnOps := aws.NewMockClientFactoryForRegion("us-east-1")
	resolver := NewStackResolver(mockConfigProvider, mockFactory)

	// Mock CloudFormation stack with outputs
	mockStack := &aws.Stack{
		Name: "vpc-stack",
		Outputs: map[string]string{
			"VpcId":    "vpc-12345",
			"SubnetId": "subnet-67890",
		},
	}

	mockCfnOps.On("GetStack", ctx, "vpc-stack").Return(mockStack, nil)

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

	resolved, err := resolver.resolveParameters(ctx, params, "us-east-1")

	require.NoError(t, err)
	assert.Len(t, resolved, 2)
	assert.Equal(t, "vpc-12345", resolved["VpcId"])
	assert.Equal(t, "subnet-67890", resolved["SubnetId"])

	mockCfnOps.AssertExpectations(t)
}

func TestStackResolver_ResolveParameters_MixedTypes(t *testing.T) {
	// Test resolution of mixed literal and output parameters
	ctx := context.Background()

	mockConfigProvider := &config.MockConfigProvider{}
	mockFactory, mockCfnOps := aws.NewMockClientFactoryForRegion("us-east-1")
	resolver := NewStackResolver(mockConfigProvider, mockFactory)

	// Mock CloudFormation stack
	mockStack := &aws.Stack{
		Name: "networking",
		Outputs: map[string]string{
			"VpcId": "vpc-abcdef",
		},
	}

	mockCfnOps.On("GetStack", ctx, "networking").Return(mockStack, nil)

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

	resolved, err := resolver.resolveParameters(ctx, params, "us-east-1")

	require.NoError(t, err)
	assert.Len(t, resolved, 2)
	assert.Equal(t, "staging", resolved["Environment"])
	assert.Equal(t, "vpc-abcdef", resolved["VpcId"])

	mockCfnOps.AssertExpectations(t)
}

func TestStackResolver_ResolveParameters_ErrorCases(t *testing.T) {
	ctx := context.Background()

	t.Run("unsupported resolution type", func(t *testing.T) {
		mockConfigProvider := &config.MockConfigProvider{}
		mockFactory, _ := aws.NewMockClientFactoryForRegion("us-east-1")
		resolver := NewStackResolver(mockConfigProvider, mockFactory)

		params := map[string]*config.ParameterValue{
			"BadParam": {
				ResolutionType: "unsupported",
				ResolutionConfig: map[string]string{
					"something": "value",
				},
			},
		}

		_, err := resolver.resolveParameters(ctx, params, "us-east-1")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported resolution type 'unsupported'")
	})

	t.Run("literal missing value", func(t *testing.T) {
		mockConfigProvider := &config.MockConfigProvider{}
		mockFactory, _ := aws.NewMockClientFactoryForRegion("us-east-1")
		resolver := NewStackResolver(mockConfigProvider, mockFactory)

		params := map[string]*config.ParameterValue{
			"BadLiteral": {
				ResolutionType:   "literal",
				ResolutionConfig: map[string]string{
					// Missing "value" key
				},
			},
		}

		_, err := resolver.resolveParameters(ctx, params, "us-east-1")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "literal parameter missing 'value' config")
	})

	t.Run("stack not found", func(t *testing.T) {
		mockConfigProvider := &config.MockConfigProvider{}
		mockFactory, mockCfnOps := aws.NewMockClientFactoryForRegion("us-east-1")
		resolver := NewStackResolver(mockConfigProvider, mockFactory)

		mockCfnOps.On("GetStack", ctx, "missing-stack").Return(nil, fmt.Errorf("stack not found"))

		params := map[string]*config.ParameterValue{
			"VpcId": {
				ResolutionType: "stack-output",
				ResolutionConfig: map[string]string{
					"stack_name": "missing-stack",
					"output_key": "VpcId",
				},
			},
		}

		_, err := resolver.resolveParameters(ctx, params, "us-east-1")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get stack 'missing-stack'")
		assert.Contains(t, err.Error(), "failed to get stack 'missing-stack'")

		mockCfnOps.AssertExpectations(t)
	})

	t.Run("output key not found", func(t *testing.T) {
		mockConfigProvider := &config.MockConfigProvider{}
		mockFactory, mockCfnOps := aws.NewMockClientFactoryForRegion("us-east-1")
		resolver := NewStackResolver(mockConfigProvider, mockFactory)

		mockStack := &aws.Stack{
			Name: "vpc-stack",
			Outputs: map[string]string{
				"VpcId": "vpc-12345",
			},
		}
		mockCfnOps.On("GetStack", ctx, "vpc-stack").Return(mockStack, nil)

		params := map[string]*config.ParameterValue{
			"BadOutput": {
				ResolutionType: "stack-output",
				ResolutionConfig: map[string]string{
					"stack_name": "vpc-stack",
					"output_key": "MissingOutput",
				},
			},
		}

		_, err := resolver.resolveParameters(ctx, params, "us-east-1")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "stack 'vpc-stack' does not have output 'MissingOutput'")
		assert.Contains(t, err.Error(), "stack 'vpc-stack' does not have output 'MissingOutput'")

		mockCfnOps.AssertExpectations(t)
	})
}

func TestStackResolver_ResolveStackOutput_MissingConfig(t *testing.T) {
	ctx := context.Background()

	mockConfigProvider := &config.MockConfigProvider{}
	mockFactory, _ := aws.NewMockClientFactoryForRegion("us-east-1")
	resolver := NewStackResolver(mockConfigProvider, mockFactory)

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
	mockFactory, _ := aws.NewMockClientFactoryForRegion("us-east-1")

	// Set expectation for config load failure
	mockConfigProvider.On("LoadConfig", ctx, "dev").Return(nil, assert.AnError)

	stackResolver := NewStackResolver(mockConfigProvider, mockFactory)
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

	mockFactory, _ := aws.NewMockClientFactoryForRegion("us-east-1")
	stackResolver := NewStackResolver(mockConfigProvider, mockFactory)
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

	mockFactory, _ := aws.NewMockClientFactoryForRegion("us-east-1")
	stackResolver := NewStackResolver(mockConfigProvider, mockFactory)
	stackResolver.SetFileSystemResolver(mockFileSystemResolver)

	resolved, err := stackResolver.ResolveStack(ctx, "dev", "vpc")

	assert.Error(t, err)
	assert.Nil(t, resolved)
	assert.Contains(t, err.Error(), "failed to read template")

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
		Context: &config.ContextConfig{
			Name:    "staging",
			Account: "123456789012",
			Region:  "us-east-1",
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

	mockFactory, _ := aws.NewMockClientFactoryForRegion("us-east-1")
	stackResolver := NewStackResolver(mockConfigProvider, mockFactory)
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

func TestStackResolver_GetDependencyOrder_Success(t *testing.T) {
	// Test successful dependency order calculation without full resolution
	mockConfigProvider := &config.MockConfigProvider{}
	mockFactory, _ := aws.NewMockClientFactoryForRegion("us-east-1")

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

	stackResolver := NewStackResolver(mockConfigProvider, mockFactory)

	order, err := stackResolver.GetDependencyOrder("dev", []string{"app", "vpc"})

	require.NoError(t, err)
	assert.Equal(t, []string{"vpc", "app"}, order)

	mockConfigProvider.AssertExpectations(t)
}

func TestStackResolver_GetDependencyOrder_EmptyList(t *testing.T) {
	// Test empty stack list
	mockConfigProvider := &config.MockConfigProvider{}
	mockFactory, _ := aws.NewMockClientFactoryForRegion("us-east-1")

	stackResolver := NewStackResolver(mockConfigProvider, mockFactory)

	order, err := stackResolver.GetDependencyOrder("dev", []string{})

	require.NoError(t, err)
	assert.Empty(t, order)

	mockConfigProvider.AssertExpectations(t)
}

func TestStackResolver_GetDependencyOrder_NoDependencies(t *testing.T) {
	// Test stacks with no dependencies
	mockConfigProvider := &config.MockConfigProvider{}
	mockFactory, _ := aws.NewMockClientFactoryForRegion("us-east-1")

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

	stackResolver := NewStackResolver(mockConfigProvider, mockFactory)

	order, err := stackResolver.GetDependencyOrder("dev", []string{"database", "app", "vpc"})

	require.NoError(t, err)
	// Should be alphabetical since no dependencies
	assert.Equal(t, []string{"app", "database", "vpc"}, order)

	mockConfigProvider.AssertExpectations(t)
}

func TestStackResolver_GetDependencyOrder_ComplexChain(t *testing.T) {
	// Test complex dependency chain: vpc -> security -> database -> app
	mockConfigProvider := &config.MockConfigProvider{}
	mockFactory, _ := aws.NewMockClientFactoryForRegion("us-east-1")

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

	stackResolver := NewStackResolver(mockConfigProvider, mockFactory)

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
	mockFactory, _ := aws.NewMockClientFactoryForRegion("us-east-1")

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

	stackResolver := NewStackResolver(mockConfigProvider, mockFactory)

	order, err := stackResolver.GetDependencyOrder("dev", []string{"stack-a", "stack-b"})

	assert.Error(t, err)
	assert.Nil(t, order)
	assert.Contains(t, err.Error(), "circular dependency detected")

	mockConfigProvider.AssertExpectations(t)
}

func TestStackResolver_GetDependencyOrder_StackNotFound(t *testing.T) {
	// Test error handling when stack config is not found
	mockConfigProvider := &config.MockConfigProvider{}
	mockFactory, _ := aws.NewMockClientFactoryForRegion("us-east-1")

	mockConfigProvider.On("GetStack", "nonexistent", "dev").Return(nil, assert.AnError)

	stackResolver := NewStackResolver(mockConfigProvider, mockFactory)

	order, err := stackResolver.GetDependencyOrder("dev", []string{"nonexistent"})

	assert.Error(t, err)
	assert.Nil(t, order)
	assert.Contains(t, err.Error(), "failed to get stack config")

	mockConfigProvider.AssertExpectations(t)
}

func TestStackResolver_GetDependencyOrder_MissingDependency(t *testing.T) {
	// Test handling of missing dependency (dependency not in resolution list)
	mockConfigProvider := &config.MockConfigProvider{}
	mockFactory, _ := aws.NewMockClientFactoryForRegion("us-east-1")

	// App depends on database, but we're only calculating order for app
	appConfig := &config.StackConfig{
		Name:         "app",
		Dependencies: []string{"database"}, // database not in resolution list
	}

	mockConfigProvider.On("GetStack", "app", "dev").Return(appConfig, nil)

	stackResolver := NewStackResolver(mockConfigProvider, mockFactory)

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
	mockFactory, _ := aws.NewMockClientFactoryForRegion("us-east-1")

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

	stackResolver := NewStackResolver(mockConfigProvider, mockFactory)

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

func TestStackResolver_ResolveParameters_LiteralList(t *testing.T) {
	// Test resolution of simple literal lists
	mockConfigProvider := &config.MockConfigProvider{}
	mockFactory, _ := aws.NewMockClientFactoryForRegion("us-east-1")
	resolver := NewStackResolver(mockConfigProvider, mockFactory)

	parameters := map[string]*config.ParameterValue{
		"Ports": {
			ResolutionType: "list",
			ListItems: []*config.ParameterValue{
				{
					ResolutionType:   "literal",
					ResolutionConfig: map[string]string{"value": "80"},
				},
				{
					ResolutionType:   "literal",
					ResolutionConfig: map[string]string{"value": "443"},
				},
				{
					ResolutionType:   "literal",
					ResolutionConfig: map[string]string{"value": "8080"},
				},
			},
		},
	}

	result, err := resolver.resolveParameters(context.Background(), parameters, "dev")
	require.NoError(t, err)

	assert.Equal(t, "80,443,8080", result["Ports"])
}

func TestStackResolver_ResolveParameters_MixedList(t *testing.T) {
	// Test resolution of mixed literal + stack output lists
	mockConfigProvider := &config.MockConfigProvider{}
	mockFactory, mockCfnOps := aws.NewMockClientFactoryForRegion("us-east-1")
	resolver := NewStackResolver(mockConfigProvider, mockFactory)

	// Mock stack outputs
	mockCfnOps.On("GetStack", mock.Anything, "security-stack").Return(&aws.Stack{
		Outputs: map[string]string{
			"WebSGId": "sg-web123",
		},
	}, nil)

	mockCfnOps.On("GetStack", mock.Anything, "database-stack").Return(&aws.Stack{
		Outputs: map[string]string{
			"DatabaseSGId": "sg-db456",
		},
	}, nil)

	parameters := map[string]*config.ParameterValue{
		"SecurityGroupIds": {
			ResolutionType: "list",
			ListItems: []*config.ParameterValue{
				{
					ResolutionType:   "literal",
					ResolutionConfig: map[string]string{"value": "sg-baseline123"},
				},
				{
					ResolutionType: "stack-output",
					ResolutionConfig: map[string]string{
						"stack_name": "security-stack",
						"output_key": "WebSGId",
					},
				},
				{
					ResolutionType: "stack-output",
					ResolutionConfig: map[string]string{
						"stack_name": "database-stack",
						"output_key": "DatabaseSGId",
					},
				},
				{
					ResolutionType:   "literal",
					ResolutionConfig: map[string]string{"value": "sg-additional789"},
				},
			},
		},
	}

	result, err := resolver.resolveParameters(context.Background(), parameters, "us-east-1")
	require.NoError(t, err)

	assert.Equal(t, "sg-baseline123,sg-web123,sg-db456,sg-additional789", result["SecurityGroupIds"])
	mockCfnOps.AssertExpectations(t)
}

func TestStackResolver_ResolveParameters_EmptyList(t *testing.T) {
	// Test resolution of empty lists
	mockConfigProvider := &config.MockConfigProvider{}
	mockFactory, _ := aws.NewMockClientFactoryForRegion("us-east-1")
	resolver := NewStackResolver(mockConfigProvider, mockFactory)

	parameters := map[string]*config.ParameterValue{
		"EmptyList": {
			ResolutionType: "list",
			ListItems:      []*config.ParameterValue{},
		},
	}

	result, err := resolver.resolveParameters(context.Background(), parameters, "dev")
	require.NoError(t, err)

	assert.Equal(t, "", result["EmptyList"])
}

func TestStackResolver_ResolveParameters_ListWithEmptyValues(t *testing.T) {
	// Test handling of empty values in lists (should be filtered out)
	mockConfigProvider := &config.MockConfigProvider{}
	mockFactory, _ := aws.NewMockClientFactoryForRegion("us-east-1")
	resolver := NewStackResolver(mockConfigProvider, mockFactory)

	parameters := map[string]*config.ParameterValue{
		"FilteredList": {
			ResolutionType: "list",
			ListItems: []*config.ParameterValue{
				{
					ResolutionType:   "literal",
					ResolutionConfig: map[string]string{"value": "value1"},
				},
				{
					ResolutionType:   "literal",
					ResolutionConfig: map[string]string{"value": ""}, // Empty value
				},
				{
					ResolutionType:   "literal",
					ResolutionConfig: map[string]string{"value": "value2"},
				},
			},
		},
	}

	result, err := resolver.resolveParameters(context.Background(), parameters, "dev")
	require.NoError(t, err)

	// Empty values should be filtered out
	assert.Equal(t, "value1,value2", result["FilteredList"])
}

func TestStackResolver_ResolveParameters_NestedList(t *testing.T) {
	// Test resolution of nested lists (lists within lists)
	mockConfigProvider := &config.MockConfigProvider{}
	mockFactory, _ := aws.NewMockClientFactoryForRegion("us-east-1")
	resolver := NewStackResolver(mockConfigProvider, mockFactory)

	parameters := map[string]*config.ParameterValue{
		"NestedList": {
			ResolutionType: "list",
			ListItems: []*config.ParameterValue{
				{
					ResolutionType:   "literal",
					ResolutionConfig: map[string]string{"value": "outer1"},
				},
				{
					ResolutionType: "list",
					ListItems: []*config.ParameterValue{
						{
							ResolutionType:   "literal",
							ResolutionConfig: map[string]string{"value": "inner1"},
						},
						{
							ResolutionType:   "literal",
							ResolutionConfig: map[string]string{"value": "inner2"},
						},
					},
				},
				{
					ResolutionType:   "literal",
					ResolutionConfig: map[string]string{"value": "outer2"},
				},
			},
		},
	}

	result, err := resolver.resolveParameters(context.Background(), parameters, "dev")
	require.NoError(t, err)

	assert.Equal(t, "outer1,inner1,inner2,outer2", result["NestedList"])
}

func TestStackResolver_ResolveParameters_ListErrorCases(t *testing.T) {
	mockConfigProvider := &config.MockConfigProvider{}
	mockFactory, mockCfnOps := aws.NewMockClientFactoryForRegion("us-east-1")
	resolver := NewStackResolver(mockConfigProvider, mockFactory)

	t.Run("nil list item", func(t *testing.T) {
		parameters := map[string]*config.ParameterValue{
			"BadList": {
				ResolutionType: "list",
				ListItems: []*config.ParameterValue{
					{
						ResolutionType:   "literal",
						ResolutionConfig: map[string]string{"value": "value1"},
					},
					nil, // Nil item
				},
			},
		}

		_, err := resolver.resolveParameters(context.Background(), parameters, "dev")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "list item 1 is nil")
	})

	t.Run("invalid resolution type in list", func(t *testing.T) {
		parameters := map[string]*config.ParameterValue{
			"BadList": {
				ResolutionType: "list",
				ListItems: []*config.ParameterValue{
					{
						ResolutionType:   "invalid-type",
						ResolutionConfig: map[string]string{},
					},
				},
			},
		}

		_, err := resolver.resolveParameters(context.Background(), parameters, "dev")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported resolution type 'invalid-type'")
	})

	t.Run("literal list item missing value", func(t *testing.T) {
		parameters := map[string]*config.ParameterValue{
			"BadList": {
				ResolutionType: "list",
				ListItems: []*config.ParameterValue{
					{
						ResolutionType:   "literal",
						ResolutionConfig: map[string]string{}, // Missing value
					},
				},
			},
		}

		_, err := resolver.resolveParameters(context.Background(), parameters, "dev")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "literal parameter missing 'value' config")
	})

	t.Run("stack output list item error", func(t *testing.T) {
		// Mock stack output failure
		mockCfnOps.On("GetStack", mock.Anything, "missing-stack").Return(
			(*aws.Stack)(nil), fmt.Errorf("stack not found"))

		parameters := map[string]*config.ParameterValue{
			"BadList": {
				ResolutionType: "list",
				ListItems: []*config.ParameterValue{
					{
						ResolutionType: "stack-output",
						ResolutionConfig: map[string]string{
							"stack_name": "missing-stack",
							"output_key": "SomeOutput",
						},
					},
				},
			},
		}

		_, err := resolver.resolveParameters(context.Background(), parameters, "us-east-1")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get stack 'missing-stack'")
		assert.Contains(t, err.Error(), "stack not found")

		mockCfnOps.AssertExpectations(t)
	})
}
