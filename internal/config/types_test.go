/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package config

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// Helper function to convert string maps to ParameterValue maps for tests
func convertStringMapToParameterValues(stringMap map[string]string) map[string]*ParameterValue {
	if stringMap == nil {
		return nil
	}
	result := make(map[string]*ParameterValue, len(stringMap))
	for key, value := range stringMap {
		result[key] = &ParameterValue{
			ResolutionType: "literal",
			ResolutionConfig: map[string]string{
				"value": value,
			},
		}
	}
	return result
}

func TestConfig_DefaultValues(t *testing.T) {
	// Test default zero values
	config := Config{}

	assert.Equal(t, "", config.Project)
	assert.Equal(t, "", config.Region)
	assert.Nil(t, config.Tags)
	assert.Nil(t, config.Context)
	assert.Nil(t, config.Stacks)
}

func TestConfig_FieldAssignment(t *testing.T) {
	// Test that Config fields can be set and retrieved
	tags := map[string]string{"Environment": "test", "Owner": "team"}
	context := &ContextConfig{Name: "dev", Account: "123456789012"}
	stacks := []*StackConfig{
		{Name: "vpc", Template: "templates/vpc.yaml"},
		{Name: "app", Template: "templates/app.yaml"},
	}

	config := Config{
		Project: "test-project",
		Region:  "us-west-2",
		Tags:    tags,
		Context: context,
		Stacks:  stacks,
	}

	assert.Equal(t, "test-project", config.Project)
	assert.Equal(t, "us-west-2", config.Region)
	assert.Equal(t, tags, config.Tags)
	assert.Equal(t, context, config.Context)
	assert.Equal(t, stacks, config.Stacks)
}

func TestConfig_JSONMarshaling(t *testing.T) {
	// Test JSON marshaling and unmarshaling
	config := Config{
		Project: "json-test",
		Region:  "eu-west-1",
		Tags: map[string]string{
			"Environment": "test",
			"Project":     "json-test",
		},
		Context: &ContextConfig{
			Name:    "test",
			Account: "123456789012",
			Region:  "eu-west-1",
		},
		Stacks: []*StackConfig{
			{
				Name:     "test-stack",
				Template: "file://test.yaml",
				Parameters: convertStringMapToParameterValues(map[string]string{
					"Param1": "value1",
				}),
			},
		},
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(&config)
	require.NoError(t, err)
	assert.NotEmpty(t, jsonData)

	// Unmarshal from JSON
	var unmarshaledConfig Config
	err = json.Unmarshal(jsonData, &unmarshaledConfig)
	require.NoError(t, err)

	// Verify the unmarshaled data
	assert.Equal(t, config.Project, unmarshaledConfig.Project)
	assert.Equal(t, config.Region, unmarshaledConfig.Region)
	assert.Equal(t, config.Tags, unmarshaledConfig.Tags)
	assert.NotNil(t, unmarshaledConfig.Context)
	assert.Equal(t, config.Context.Name, unmarshaledConfig.Context.Name)
	assert.Len(t, unmarshaledConfig.Stacks, 1)
	assert.Equal(t, config.Stacks[0].Name, unmarshaledConfig.Stacks[0].Name)
}

func TestContextConfig_DefaultValues(t *testing.T) {
	// Test default zero values
	context := ContextConfig{}

	assert.Equal(t, "", context.Name)
	assert.Equal(t, "", context.Account)
	assert.Equal(t, "", context.Region)
	assert.Nil(t, context.Tags)
}

func TestContextConfig_FieldAssignment(t *testing.T) {
	// Test that ContextConfig fields can be set and retrieved
	tags := map[string]string{
		"Environment": "production",
		"CostCenter":  "engineering",
	}

	context := ContextConfig{
		Name:    "prod",
		Account: "987654321098",
		Region:  "us-east-1",
		Tags:    tags,
	}

	assert.Equal(t, "prod", context.Name)
	assert.Equal(t, "987654321098", context.Account)
	assert.Equal(t, "us-east-1", context.Region)
	assert.Equal(t, tags, context.Tags)
}

func TestContextConfig_JSONMarshaling(t *testing.T) {
	// Test JSON marshaling
	context := ContextConfig{
		Name:    "staging",
		Account: "555666777888",
		Region:  "ap-southeast-2",
		Tags: map[string]string{
			"Environment": "staging",
			"Team":        "platform",
		},
	}

	jsonData, err := json.Marshal(&context)
	require.NoError(t, err)

	var unmarshaledContext ContextConfig
	err = json.Unmarshal(jsonData, &unmarshaledContext)
	require.NoError(t, err)

	assert.Equal(t, context.Name, unmarshaledContext.Name)
	assert.Equal(t, context.Account, unmarshaledContext.Account)
	assert.Equal(t, context.Region, unmarshaledContext.Region)
	assert.Equal(t, context.Tags, unmarshaledContext.Tags)
}

func TestStackConfig_DefaultValues(t *testing.T) {
	// Test default zero values
	stack := StackConfig{}

	assert.Equal(t, "", stack.Name)
	assert.Equal(t, "", stack.Template)
	assert.Nil(t, stack.Parameters)
	assert.Nil(t, stack.Tags)
	assert.Nil(t, stack.Dependencies)
	assert.Nil(t, stack.Capabilities)
}

func TestStackConfig_FieldAssignment(t *testing.T) {
	// Test that StackConfig fields can be set and retrieved
	parameters := convertStringMapToParameterValues(map[string]string{
		"InstanceType": "t3.medium",
		"KeyName":      "my-key",
	})
	tags := map[string]string{
		"Component": "application",
		"Layer":     "compute",
	}
	dependencies := []string{"vpc", "security-groups"}
	capabilities := []string{"CAPABILITY_IAM", "CAPABILITY_NAMED_IAM"}

	stack := StackConfig{
		Name:         "app-stack",
		Template:     "s3://my-bucket/templates/app.yaml",
		Parameters:   parameters,
		Tags:         tags,
		Dependencies: dependencies,
		Capabilities: capabilities,
	}

	assert.Equal(t, "app-stack", stack.Name)
	assert.Equal(t, "s3://my-bucket/templates/app.yaml", stack.Template)
	assert.Equal(t, parameters, stack.Parameters)
	assert.Equal(t, tags, stack.Tags)
	assert.Equal(t, dependencies, stack.Dependencies)
	assert.Equal(t, capabilities, stack.Capabilities)
}

func TestStackConfig_JSONMarshaling(t *testing.T) {
	// Test JSON marshaling
	stack := StackConfig{
		Name:     "database",
		Template: "git://github.com/org/repo.git//templates/rds.yaml",
		Parameters: convertStringMapToParameterValues(map[string]string{
			"DBInstanceClass": "db.t3.micro",
			"Engine":          "postgres",
		}),
		Tags: map[string]string{
			"Component": "database",
			"Backup":    "enabled",
		},
		Dependencies: []string{"vpc", "subnet-group"},
		Capabilities: []string{"CAPABILITY_IAM"},
	}

	jsonData, err := json.Marshal(&stack)
	require.NoError(t, err)

	var unmarshaledStack StackConfig
	err = json.Unmarshal(jsonData, &unmarshaledStack)
	require.NoError(t, err)

	assert.Equal(t, stack.Name, unmarshaledStack.Name)
	assert.Equal(t, stack.Template, unmarshaledStack.Template)
	assert.Equal(t, stack.Parameters, unmarshaledStack.Parameters)
	assert.Equal(t, stack.Tags, unmarshaledStack.Tags)
	assert.Equal(t, stack.Dependencies, unmarshaledStack.Dependencies)
	assert.Equal(t, stack.Capabilities, unmarshaledStack.Capabilities)
}

func TestStackConfig_TemplateURIs(t *testing.T) {
	// Test different template URI formats
	tests := []struct {
		name     string
		template string
	}{
		{
			name:     "file URI",
			template: "file://./templates/vpc.yaml",
		},
		{
			name:     "S3 URI",
			template: "s3://my-bucket/templates/app.yaml",
		},
		{
			name:     "Git URI",
			template: "git://github.com/org/repo.git//path/to/template.yaml",
		},
		{
			name:     "HTTPS URI",
			template: "https://raw.githubusercontent.com/org/repo/main/template.yaml",
		},
		{
			name:     "relative path",
			template: "templates/local.yaml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stack := StackConfig{
				Name:     "test-stack",
				Template: tt.template,
			}

			assert.Equal(t, tt.template, stack.Template)
		})
	}
}

func TestStackConfig_Capabilities(t *testing.T) {
	// Test CloudFormation capabilities
	tests := []struct {
		name         string
		capabilities []string
	}{
		{
			name:         "no capabilities",
			capabilities: nil,
		},
		{
			name:         "empty capabilities",
			capabilities: []string{},
		},
		{
			name:         "IAM capability",
			capabilities: []string{"CAPABILITY_IAM"},
		},
		{
			name:         "named IAM capability",
			capabilities: []string{"CAPABILITY_NAMED_IAM"},
		},
		{
			name:         "auto expand capability",
			capabilities: []string{"CAPABILITY_AUTO_EXPAND"},
		},
		{
			name:         "multiple capabilities",
			capabilities: []string{"CAPABILITY_IAM", "CAPABILITY_NAMED_IAM", "CAPABILITY_AUTO_EXPAND"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stack := StackConfig{
				Name:         "test-stack",
				Capabilities: tt.capabilities,
			}

			assert.Equal(t, tt.capabilities, stack.Capabilities)
		})
	}
}

func TestMockConfigProvider_Interface(t *testing.T) {
	// Test that MockConfigProvider implements ConfigProvider interface
	var provider ConfigProvider = &MockConfigProvider{}
	assert.NotNil(t, provider)
}

func TestMockConfigProvider_LoadConfig(t *testing.T) {
	// Test MockConfigProvider LoadConfig method
	mockProvider := &MockConfigProvider{}

	expectedConfig := &Config{
		Project: "test-project",
		Region:  "us-east-1",
	}

	mockProvider.On("LoadConfig", mock.Anything, "dev").Return(expectedConfig, nil)

	config, err := mockProvider.LoadConfig(context.Background(), "dev")

	assert.NoError(t, err)
	assert.Equal(t, expectedConfig, config)
	mockProvider.AssertExpectations(t)
}

func TestMockConfigProvider_ListContexts(t *testing.T) {
	// Test MockConfigProvider ListContexts method
	mockProvider := &MockConfigProvider{}

	expectedContexts := []string{"dev", "staging", "prod"}
	mockProvider.On("ListContexts").Return(expectedContexts, nil)

	contexts, err := mockProvider.ListContexts()

	assert.NoError(t, err)
	assert.Equal(t, expectedContexts, contexts)
	mockProvider.AssertExpectations(t)
}

func TestMockConfigProvider_GetStack(t *testing.T) {
	// Test MockConfigProvider GetStack method
	mockProvider := &MockConfigProvider{}

	expectedStack := &StackConfig{
		Name:     "vpc",
		Template: "templates/vpc.yaml",
	}

	mockProvider.On("GetStack", "vpc", "dev").Return(expectedStack, nil)

	stack, err := mockProvider.GetStack("vpc", "dev")

	assert.NoError(t, err)
	assert.Equal(t, expectedStack, stack)
	mockProvider.AssertExpectations(t)
}

func TestMockConfigProvider_ListStacks(t *testing.T) {
	// Test MockConfigProvider ListStacks method
	mockProvider := &MockConfigProvider{}

	expectedStacks := []string{"vpc", "app", "database"}
	mockProvider.On("ListStacks", "dev").Return(expectedStacks, nil)

	stacks, err := mockProvider.ListStacks("dev")

	assert.NoError(t, err)
	assert.Equal(t, expectedStacks, stacks)
	mockProvider.AssertExpectations(t)
}

func TestMockConfigProvider_Validate(t *testing.T) {
	// Test MockConfigProvider Validate method
	mockProvider := &MockConfigProvider{}

	mockProvider.On("Validate").Return(nil)

	err := mockProvider.Validate()

	assert.NoError(t, err)
	mockProvider.AssertExpectations(t)
}

func TestConfig_EmptyMapsVsNilMaps(t *testing.T) {
	// Test behavior difference between empty maps and nil maps
	config1 := Config{
		Tags: map[string]string{},
	}

	config2 := Config{
		Tags: nil,
	}

	// Empty map should not be nil
	assert.NotNil(t, config1.Tags)
	assert.Len(t, config1.Tags, 0)

	// Nil map should be nil
	assert.Nil(t, config2.Tags)
}

func TestStackConfig_Dependencies(t *testing.T) {
	// Test dependency handling
	tests := []struct {
		name         string
		dependencies []string
		valid        bool
	}{
		{
			name:         "no dependencies",
			dependencies: nil,
			valid:        true,
		},
		{
			name:         "single dependency",
			dependencies: []string{"vpc"},
			valid:        true,
		},
		{
			name:         "multiple dependencies",
			dependencies: []string{"vpc", "subnet-group", "security-group"},
			valid:        true,
		},
		{
			name:         "empty dependencies list",
			dependencies: []string{},
			valid:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stack := StackConfig{
				Name:         "dependent-stack",
				Dependencies: tt.dependencies,
			}

			assert.Equal(t, tt.dependencies, stack.Dependencies)
			if tt.dependencies == nil {
				assert.Nil(t, stack.Dependencies)
			} else {
				assert.Equal(t, len(tt.dependencies), len(stack.Dependencies))
			}
		})
	}
}

func TestConfig_CompleteStructure(t *testing.T) {
	// Test a complete Config structure with all fields populated
	config := Config{
		Project: "complete-project",
		Region:  "ap-southeast-1",
		Tags: map[string]string{
			"ManagedBy":   "stackaroo",
			"Environment": "test",
			"Owner":       "platform-team",
		},
		Context: &ContextConfig{
			Name:    "integration",
			Account: "444555666777",
			Region:  "ap-southeast-1",
			Tags: map[string]string{
				"Context":    "integration",
				"TestSuite":  "enabled",
				"Monitoring": "basic",
			},
		},
		Stacks: []*StackConfig{
			{
				Name:     "infrastructure",
				Template: "file://./templates/infra.yaml",
				Parameters: convertStringMapToParameterValues(map[string]string{
					"Environment":  "integration",
					"InstanceType": "t3.small",
					"MinSize":      "1",
					"MaxSize":      "3",
				}),
				Tags: map[string]string{
					"Component": "infrastructure",
					"Layer":     "foundation",
				},
				Dependencies: []string{},
				Capabilities: []string{"CAPABILITY_IAM"},
			},
			{
				Name:     "application",
				Template: "s3://templates-bucket/app-template.yaml",
				Parameters: convertStringMapToParameterValues(map[string]string{
					"ImageTag":      "latest",
					"DesiredCount":  "2",
					"ContainerPort": "8080",
				}),
				Tags: map[string]string{
					"Component": "application",
					"Layer":     "service",
				},
				Dependencies: []string{"infrastructure"},
				Capabilities: []string{"CAPABILITY_IAM", "CAPABILITY_NAMED_IAM"},
			},
		},
	}

	// Verify all fields are properly set
	assert.Equal(t, "complete-project", config.Project)
	assert.Equal(t, "ap-southeast-1", config.Region)
	assert.Len(t, config.Tags, 3)
	assert.Equal(t, "stackaroo", config.Tags["ManagedBy"])

	assert.NotNil(t, config.Context)
	assert.Equal(t, "integration", config.Context.Name)
	assert.Len(t, config.Context.Tags, 3)

	assert.Len(t, config.Stacks, 2)

	infraStack := config.Stacks[0]
	assert.Equal(t, "infrastructure", infraStack.Name)
	assert.Len(t, infraStack.Parameters, 4)
	assert.Len(t, infraStack.Dependencies, 0)

	appStack := config.Stacks[1]
	assert.Equal(t, "application", appStack.Name)
	assert.Contains(t, appStack.Dependencies, "infrastructure")
	assert.Len(t, appStack.Capabilities, 2)
}
