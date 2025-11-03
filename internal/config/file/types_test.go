/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/

package file

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestParameterValue_UnmarshalYAML_LiteralString(t *testing.T) {
	tests := []struct {
		name     string
		yaml     string
		expected string
	}{
		{
			name:     "simple string",
			yaml:     `"hello world"`,
			expected: "hello world",
		},
		{
			name:     "numeric string",
			yaml:     `"12345"`,
			expected: "12345",
		},
		{
			name:     "empty string",
			yaml:     `""`,
			expected: "",
		},
		{
			name:     "unquoted string",
			yaml:     `production`,
			expected: "production",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var pv yamlParameterValue
			err := yaml.Unmarshal([]byte(tt.yaml), &pv)

			require.NoError(t, err)
			assert.True(t, pv.IsLiteral())
			assert.False(t, pv.IsResolver())
			assert.Equal(t, tt.expected, pv.Literal)
			assert.Nil(t, pv.Resolver)
		})
	}
}

func TestParameterValue_UnmarshalYAML_ComplexObject(t *testing.T) {
	tests := []struct {
		name           string
		yaml           string
		expectedType   string
		expectedConfig map[string]interface{}
	}{
		{
			name: "stack output resolver",
			yaml: `
type: output
stack: vpc-stack
output: VpcId`,
			expectedType: "output",
			expectedConfig: map[string]interface{}{
				"stack":  "vpc-stack",
				"output": "VpcId",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var pv yamlParameterValue
			err := yaml.Unmarshal([]byte(tt.yaml), &pv)

			require.NoError(t, err)
			assert.False(t, pv.IsLiteral())
			assert.True(t, pv.IsResolver())
			assert.Empty(t, pv.Literal)
			assert.NotNil(t, pv.Resolver)

			assert.Equal(t, tt.expectedType, pv.Resolver.Type)
			for key, expectedValue := range tt.expectedConfig {
				assert.Equal(t, expectedValue, pv.Resolver.Config[key], "Config key %s", key)
			}
		})
	}
}

func TestParameterValue_MarshalYAML(t *testing.T) {
	t.Run("marshal literal", func(t *testing.T) {
		pv := yamlParameterValue{Literal: "test-value", IsLiteralValue: true}

		result, err := yaml.Marshal(&pv)
		require.NoError(t, err)

		var unmarshalled string
		err = yaml.Unmarshal(result, &unmarshalled)
		require.NoError(t, err)
		assert.Equal(t, "test-value", unmarshalled)
	})

	t.Run("marshal resolver", func(t *testing.T) {
		pv := yamlParameterValue{
			Resolver: &yamlParameterResolver{
				Type: "output",
				Config: map[string]interface{}{
					"stack":  "vpc-stack",
					"output": "VpcId",
				},
			},
		}

		result, err := yaml.Marshal(&pv)
		require.NoError(t, err)

		var unmarshalled map[string]interface{}
		err = yaml.Unmarshal(result, &unmarshalled)
		require.NoError(t, err)
		assert.Equal(t, "output", unmarshalled["type"])
		assert.Equal(t, "vpc-stack", unmarshalled["stack"])
		assert.Equal(t, "VpcId", unmarshalled["output"])
	})
}

func TestConvertStringMap(t *testing.T) {
	t.Run("convert normal map", func(t *testing.T) {
		input := map[string]string{
			"key1": "value1",
			"key2": "value2",
		}

		result := ConvertStringMap(input)

		require.NotNil(t, result)
		assert.Len(t, result, 2)

		assert.Equal(t, "literal", result["key1"].ResolutionType)
		assert.Equal(t, "value1", result["key1"].ResolutionConfig["value"])

		assert.Equal(t, "literal", result["key2"].ResolutionType)
		assert.Equal(t, "value2", result["key2"].ResolutionConfig["value"])
	})

	t.Run("convert nil map", func(t *testing.T) {
		result := ConvertStringMap(nil)
		assert.Nil(t, result)
	})
}

func TestStack_MixedParameterTypes(t *testing.T) {
	yamlConfig := `
template: test.yaml
parameters:
  # Literal values
  Environment: production
  Region: us-west-2

  # Stack output resolver
  VpcId:
    type: output
    stack: vpc-stack
    output: VpcId
`

	var stack Stack
	err := yaml.Unmarshal([]byte(yamlConfig), &stack)
	require.NoError(t, err)

	// stack.Name removed - name is now the map key
	assert.Equal(t, "test.yaml", stack.Template)
	assert.Len(t, stack.Parameters, 3)

	// Test literal parameters
	assert.True(t, stack.Parameters["Environment"].IsLiteral())
	assert.Equal(t, "production", stack.Parameters["Environment"].Literal)

	assert.True(t, stack.Parameters["Region"].IsLiteral())
	assert.Equal(t, "us-west-2", stack.Parameters["Region"].Literal)

	// Test resolver parameters
	vpcIdParam := stack.Parameters["VpcId"]
	assert.True(t, vpcIdParam.IsResolver())
	assert.Equal(t, "output", vpcIdParam.Resolver.Type)
	assert.Equal(t, "vpc-stack", vpcIdParam.Resolver.Config["stack"])
	assert.Equal(t, "VpcId", vpcIdParam.Resolver.Config["output"])

}

func TestFileProvider_ConvertsResolverParameters(t *testing.T) {
	// Create a temporary config file with resolver parameters
	yamlContent := `
project: test-project
region: us-west-2

stacks:
  test-stack:
    template: test.yaml
    parameters:
      # Literal parameter
      Environment: production

      # Resolver parameter
      VpcId:
        type: stack-output
        stack: vpc-stack
        output: VpcId
`

	// Create temporary file
	tmpFile, err := os.CreateTemp("", "stackaroo-test-*.yaml")
	require.NoError(t, err)
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	_, err = tmpFile.WriteString(yamlContent)
	require.NoError(t, err)
	_ = tmpFile.Close()

	// Create provider and test
	provider := NewFileConfigProvider(tmpFile.Name())

	// This should now succeed and convert resolver parameters to config.ParameterValue
	stackConfig, err := provider.GetStack("test-stack", "dev")
	require.NoError(t, err)

	// Verify literal parameter
	envParam := stackConfig.Parameters["Environment"]
	assert.Equal(t, "literal", envParam.ResolutionType)
	assert.Equal(t, "production", envParam.ResolutionConfig["value"])

	// Verify resolver parameter
	vpcIdParam := stackConfig.Parameters["VpcId"]
	assert.Equal(t, "stack-output", vpcIdParam.ResolutionType)
	assert.Equal(t, "vpc-stack", vpcIdParam.ResolutionConfig["stack"])
	assert.Equal(t, "VpcId", vpcIdParam.ResolutionConfig["output"])
}

func TestFileConfig_DefaultValues(t *testing.T) {
	// Test default zero values
	config := Config{}

	assert.Equal(t, "", config.Project)
	assert.Equal(t, "", config.Region)
	assert.Nil(t, config.Tags)
	assert.Nil(t, config.Contexts)
	assert.Nil(t, config.Stacks)
}

func TestFileConfig_FieldAssignment(t *testing.T) {
	// Test that FileConfig fields can be set and retrieved
	tags := map[string]string{"Environment": "test"}
	contexts := map[string]*Context{"dev": {}}
	stacks := map[string]*Stack{"test-stack": {}}

	config := Config{
		Project:  "test-project",
		Region:   "us-east-1",
		Tags:     tags,
		Contexts: contexts,
		Stacks:   stacks,
	}

	assert.Equal(t, "test-project", config.Project)
	assert.Equal(t, "us-east-1", config.Region)
	assert.Equal(t, tags, config.Tags)
	assert.Equal(t, contexts, config.Contexts)
	assert.Equal(t, stacks, config.Stacks)
}

func TestTemplates_DefaultValues(t *testing.T) {
	// Test default zero values
	templates := Templates{}

	assert.Equal(t, "", templates.Directory)
}

func TestTemplates_FieldAssignment(t *testing.T) {
	// Test that Templates fields can be set and retrieved
	templates := Templates{
		Directory: "templates/",
	}

	assert.Equal(t, "templates/", templates.Directory)
}

func TestTemplates_YAMLMarshaling(t *testing.T) {
	// Test YAML marshaling and unmarshaling
	templates := Templates{
		Directory: "custom-templates/",
	}

	// Marshal to YAML
	yamlData, err := yaml.Marshal(&templates)
	require.NoError(t, err)
	assert.NotEmpty(t, yamlData)

	// Unmarshal from YAML
	var unmarshaledTemplates Templates
	err = yaml.Unmarshal(yamlData, &unmarshaledTemplates)
	require.NoError(t, err)

	// Verify the unmarshaled data
	assert.Equal(t, templates.Directory, unmarshaledTemplates.Directory)
}

func TestFileConfig_YAMLMarshaling(t *testing.T) {
	// Test YAML marshaling and unmarshaling
	config := Config{
		Project: "test-project",
		Region:  "us-west-2",
		Tags: map[string]string{
			"Owner":   "team-a",
			"Project": "webapp",
		},
		Contexts: map[string]*Context{
			"dev": {
				Account: "123456789012",
				Region:  "us-west-2",
				Tags: map[string]string{
					"Environment": "development",
				},
			},
		},
		Templates: &Templates{
			Directory: "templates/",
		},
		Stacks: map[string]*Stack{
			"vpc": {
				Template: "templates/vpc.yaml",
				Parameters: map[string]*yamlParameterValue{
					"VpcCidr": {Literal: "10.0.0.0/16", IsLiteralValue: true},
				},
				Tags: map[string]string{
					"Component": "networking",
				},
				Capabilities: []string{"CAPABILITY_IAM"},
			},
		},
	}

	// Marshal to YAML
	yamlData, err := yaml.Marshal(&config)
	require.NoError(t, err)
	assert.NotEmpty(t, yamlData)

	// Unmarshal from YAML
	var unmarshaledConfig Config
	err = yaml.Unmarshal(yamlData, &unmarshaledConfig)
	require.NoError(t, err)

	// Verify the unmarshaled data
	assert.Equal(t, config.Project, unmarshaledConfig.Project)
	assert.Equal(t, config.Region, unmarshaledConfig.Region)
	assert.Equal(t, config.Tags, unmarshaledConfig.Tags)
	assert.NotNil(t, unmarshaledConfig.Templates)
	assert.Equal(t, "templates/", unmarshaledConfig.Templates.Directory)
	assert.Len(t, unmarshaledConfig.Contexts, 1)
	assert.Len(t, unmarshaledConfig.Stacks, 1)
}

func TestFileConfig_JSONMarshaling(t *testing.T) {
	// Test JSON marshaling (for completeness)
	config := Config{
		Project: "json-test",
		Region:  "eu-west-1",
		Tags: map[string]string{
			"Format": "json",
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
}

func TestContextConfig_DefaultValues(t *testing.T) {
	// Test default zero values
	context := Context{}

	assert.Equal(t, "", context.Account)
	assert.Equal(t, "", context.Region)
	assert.Nil(t, context.Tags)
}

func TestContextConfig_FieldAssignment(t *testing.T) {
	// Test that Context fields can be set and retrieved
	tags := map[string]string{"Environment": "production"}

	context := Context{
		Account: "987654321098",
		Region:  "us-east-1",
		Tags:    tags,
	}
	assert.Equal(t, "987654321098", context.Account)
	assert.Equal(t, "us-east-1", context.Region)
	assert.Equal(t, tags, context.Tags)
}

func TestStackConfig_DefaultValues(t *testing.T) {
	// Test default zero values
	stack := Stack{}

	// stack.Name removed - name is now the map key
	assert.Equal(t, "", stack.Template)
	assert.Nil(t, stack.Parameters)
	assert.Nil(t, stack.Tags)
	assert.Nil(t, stack.Dependencies)
	assert.Nil(t, stack.Capabilities)
	assert.Nil(t, stack.Contexts)
}

func TestStackConfig_FieldAssignment(t *testing.T) {
	// Test that Stack fields can be set and retrieved
	parameters := map[string]*yamlParameterValue{
		"Size": {Literal: "large", IsLiteralValue: true},
	}
	tags := map[string]string{"Component": "database"}
	dependencies := []string{"vpc", "security-groups"}
	capabilities := []string{"CAPABILITY_IAM", "CAPABILITY_NAMED_IAM"}
	contexts := map[string]*ContextOverride{
		"dev": {Parameters: map[string]*yamlParameterValue{
			"Size": {Literal: "small", IsLiteralValue: true},
		}},
	}

	stack := Stack{
		Template:     "templates/rds.yaml",
		Parameters:   parameters,
		Tags:         tags,
		Dependencies: dependencies,
		Capabilities: capabilities,
		Contexts:     contexts,
	}

	// stack.Name removed - name is now the map key
	assert.Equal(t, "templates/rds.yaml", stack.Template)
	assert.Equal(t, parameters, stack.Parameters)
	assert.Equal(t, tags, stack.Tags)
	assert.Equal(t, dependencies, stack.Dependencies)
	assert.Equal(t, capabilities, stack.Capabilities)
	assert.Equal(t, contexts, stack.Contexts)
}

func TestContextOverride_DefaultValues(t *testing.T) {
	// Test default zero values
	contextConfig := ContextOverride{}

	assert.Nil(t, contextConfig.Parameters)
	assert.Nil(t, contextConfig.Tags)
	assert.Nil(t, contextConfig.Dependencies)
	assert.Nil(t, contextConfig.Capabilities)
}

func TestContextOverride_FieldAssignment(t *testing.T) {
	// Test that ContextOverride fields can be set and retrieved
	parameters := map[string]*yamlParameterValue{
		"InstanceType": {Literal: "t3.micro", IsLiteralValue: true},
	}
	tags := map[string]string{"Environment": "development"}
	dependencies := []string{"vpc"}
	capabilities := []string{"CAPABILITY_IAM"}

	contextConfig := ContextOverride{
		Parameters:   parameters,
		Tags:         tags,
		Dependencies: dependencies,
		Capabilities: capabilities,
	}

	assert.Equal(t, parameters, contextConfig.Parameters)
	assert.Equal(t, tags, contextConfig.Tags)
	assert.Equal(t, dependencies, contextConfig.Dependencies)
	assert.Equal(t, capabilities, contextConfig.Capabilities)
}

func TestConfig_ComplexYAMLStructure(t *testing.T) {
	// Test a complex YAML structure that represents a realistic config file
	yamlContent := `
project: complex-app
region: us-east-1
tags:
  ManagedBy: stackaroo
  Project: complex-app

templates:
  directory: "templates/"

contexts:
  dev:
    account: "123456789012"
    region: us-west-2
    tags:
      Environment: dev
      CostCenter: engineering
  prod:
    account: "987654321098"
    region: us-east-1
    tags:
      Environment: prod
      CostCenter: production

stacks:
  vpc:
    template: templates/vpc.yaml
    parameters:
      VpcCidr: "10.0.0.0/16"
      EnableDnsHostnames: "true"
    tags:
      Component: networking
    capabilities:
      - CAPABILITY_IAM
    contexts:
      dev:
        parameters:
          VpcCidr: "10.1.0.0/16"
      prod:
        parameters:
          VpcCidr: "10.3.0.0/16"

  app:
    template: templates/app.yaml
    depends_on:
      - vpc
    parameters:
      InstanceType: "t3.medium"
    contexts:
      dev:
        parameters:
          InstanceType: "t3.micro"
      prod:
        parameters:
          InstanceType: "t3.large"
        tags:
          Monitoring: "enabled"
`

	var config Config
	err := yaml.Unmarshal([]byte(yamlContent), &config)
	require.NoError(t, err)

	// Verify top-level fields
	assert.Equal(t, "complex-app", config.Project)
	assert.Equal(t, "us-east-1", config.Region)
	assert.Equal(t, "stackaroo", config.Tags["ManagedBy"])

	// Verify templates
	assert.NotNil(t, config.Templates)
	assert.Equal(t, "templates/", config.Templates.Directory)

	// Verify contexts
	assert.Len(t, config.Contexts, 2)
	devContext := config.Contexts["dev"]
	assert.NotNil(t, devContext)
	assert.Equal(t, "123456789012", devContext.Account)
	assert.Equal(t, "dev", devContext.Tags["Environment"])

	// Verify stacks
	assert.Len(t, config.Stacks, 2)

	vpcStack := config.Stacks["vpc"]
	assert.NotNil(t, vpcStack)
	assert.Equal(t, "templates/vpc.yaml", vpcStack.Template)
	assert.Equal(t, "10.0.0.0/16", vpcStack.Parameters["VpcCidr"].Literal)
	assert.Contains(t, vpcStack.Capabilities, "CAPABILITY_IAM")
	assert.Equal(t, "10.1.0.0/16", vpcStack.Contexts["dev"].Parameters["VpcCidr"].Literal)

	appStack := config.Stacks["app"]
	assert.NotNil(t, appStack)
	assert.Contains(t, appStack.Dependencies, "vpc")
	assert.Equal(t, "t3.micro", appStack.Contexts["dev"].Parameters["InstanceType"].Literal)
	assert.Equal(t, "enabled", appStack.Contexts["prod"].Tags["Monitoring"])
}

func TestParameterValue_ListParameterIntegration(t *testing.T) {
	// Test complete YAML parsing with list parameters
	yamlConfig := `
template: test.yaml
parameters:
  # Single literal
  Environment: production

  # Single resolver
  VpcId:
    type: stack-output
    stack: vpc-stack
    output: VpcId

  # List parameter with mixed types
  SecurityGroupIds:
    - sg-baseline123
    - type: stack-output
      stack: security-stack
      output: WebSGId
    - type: stack-output
      stack: database-stack
      output: DatabaseSGId
    - sg-additional456

  # Simple literal list
  Ports:
    - "80"
    - "443"
    - "8080"
`

	var stack Stack
	err := yaml.Unmarshal([]byte(yamlConfig), &stack)
	require.NoError(t, err)

	// Test single literal parameter
	envParam := stack.Parameters["Environment"]
	assert.True(t, envParam.IsLiteral())
	assert.Equal(t, "production", envParam.Literal)

	// Test single resolver parameter
	vpcIdParam := stack.Parameters["VpcId"]
	assert.True(t, vpcIdParam.IsResolver())
	assert.Equal(t, "stack-output", vpcIdParam.Resolver.Type)

	// Test mixed list parameter
	sgParam := stack.Parameters["SecurityGroupIds"]
	assert.True(t, sgParam.IsList())
	assert.Len(t, sgParam.ListItems, 4)

	// Verify list items
	assert.True(t, sgParam.ListItems[0].IsLiteral())
	assert.Equal(t, "sg-baseline123", sgParam.ListItems[0].Literal)

	assert.True(t, sgParam.ListItems[1].IsResolver())
	assert.Equal(t, "stack-output", sgParam.ListItems[1].Resolver.Type)
	assert.Equal(t, "security-stack", sgParam.ListItems[1].Resolver.Config["stack"])

	assert.True(t, sgParam.ListItems[2].IsResolver())
	assert.Equal(t, "stack-output", sgParam.ListItems[2].Resolver.Type)
	assert.Equal(t, "database-stack", sgParam.ListItems[2].Resolver.Config["stack"])

	assert.True(t, sgParam.ListItems[3].IsLiteral())
	assert.Equal(t, "sg-additional456", sgParam.ListItems[3].Literal)

	// Test simple literal list
	portsParam := stack.Parameters["Ports"]
	assert.True(t, portsParam.IsList())
	assert.Len(t, portsParam.ListItems, 3)
	assert.True(t, portsParam.ListItems[0].IsLiteral())
	assert.Equal(t, "80", portsParam.ListItems[0].Literal)
	assert.True(t, portsParam.ListItems[1].IsLiteral())
	assert.Equal(t, "443", portsParam.ListItems[1].Literal)
	assert.True(t, portsParam.ListItems[2].IsLiteral())
	assert.Equal(t, "8080", portsParam.ListItems[2].Literal)
}

func TestParameterValue_ToConfigParameterValue_List(t *testing.T) {
	// Test conversion of list parameters to config.ParameterValue
	yamlParam := &yamlParameterValue{
		IsListValue: true,
		ListItems: []*yamlParameterValue{
			{Literal: "sg-literal123", IsLiteralValue: true},
			{
				Resolver: &yamlParameterResolver{
					Type: "stack-output",
					Config: map[string]interface{}{
						"stack":  "security-stack",
						"output": "WebSGId",
					},
				},
			},
			{Literal: "sg-literal456", IsLiteralValue: true},
		},
	}

	configParam := yamlParam.ToConfigParameterValue()
	require.NotNil(t, configParam)

	// Verify list parameter properties
	assert.Equal(t, "list", configParam.ResolutionType)
	assert.NotNil(t, configParam.ListItems)
	assert.Len(t, configParam.ListItems, 3)

	// Verify first item (literal)
	assert.Equal(t, "literal", configParam.ListItems[0].ResolutionType)
	assert.Equal(t, "sg-literal123", configParam.ListItems[0].ResolutionConfig["value"])

	// Verify second item (stack-output)
	assert.Equal(t, "stack-output", configParam.ListItems[1].ResolutionType)
	assert.Equal(t, "security-stack", configParam.ListItems[1].ResolutionConfig["stack"])
	assert.Equal(t, "WebSGId", configParam.ListItems[1].ResolutionConfig["output"])

	// Verify third item (literal)
	assert.Equal(t, "literal", configParam.ListItems[2].ResolutionType)
	assert.Equal(t, "sg-literal456", configParam.ListItems[2].ResolutionConfig["value"])
}

func TestParameterValue_MarshalYAML_List(t *testing.T) {
	// Test marshalling list parameters back to YAML
	yamlParam := &yamlParameterValue{
		IsListValue: true,
		ListItems: []*yamlParameterValue{
			{Literal: "literal-value", IsLiteralValue: true},
			{
				Resolver: &yamlParameterResolver{
					Type: "stack-output",
					Config: map[string]interface{}{
						"stack":  "vpc-stack",
						"output": "VpcId",
					},
				},
			},
		},
	}

	result, err := yaml.Marshal(yamlParam)
	require.NoError(t, err)

	// Parse the result back to verify structure instead of exact string match
	var unmarshalled []*yamlParameterValue
	err = yaml.Unmarshal(result, &unmarshalled)
	require.NoError(t, err)

	// Verify structure
	assert.Len(t, unmarshalled, 2)
	assert.True(t, unmarshalled[0].IsLiteral())
	assert.Equal(t, "literal-value", unmarshalled[0].Literal)

	assert.True(t, unmarshalled[1].IsResolver())
	assert.Equal(t, "stack-output", unmarshalled[1].Resolver.Type)
	assert.Equal(t, "vpc-stack", unmarshalled[1].Resolver.Config["stack"])
	assert.Equal(t, "VpcId", unmarshalled[1].Resolver.Config["output"])
}

func TestConfig_EmptyMaps(t *testing.T) {
	// Test behavior with empty maps vs nil maps
	config1 := Config{
		Tags: map[string]string{},
	}

	config2 := Config{
		Tags: nil,
	}

	// Both should be valid but behave differently
	assert.NotNil(t, config1.Tags)
	assert.Len(t, config1.Tags, 0)

	assert.Nil(t, config2.Tags)
}

func TestConfig_YAMLTags(t *testing.T) {
	// Test that YAML tags work correctly (if they're defined on the struct)
	yamlContent := `
project: yaml-tags-test
region: eu-central-1
`

	var config Config
	err := yaml.Unmarshal([]byte(yamlContent), &config)
	require.NoError(t, err)

	assert.Equal(t, "yaml-tags-test", config.Project)
	assert.Equal(t, "eu-central-1", config.Region)

	// Marshal back to YAML
	yamlData, err := yaml.Marshal(&config)
	require.NoError(t, err)

	yamlString := string(yamlData)
	assert.Contains(t, yamlString, "project: yaml-tags-test")
	assert.Contains(t, yamlString, "region: eu-central-1")
}

func TestStackConfig_Dependencies(t *testing.T) {
	// Test stack dependencies handling
	tests := []struct {
		name         string
		dependencies []string
	}{
		{
			name:         "no dependencies",
			dependencies: nil,
		},
		{
			name:         "empty dependencies",
			dependencies: []string{},
		},
		{
			name:         "single dependency",
			dependencies: []string{"vpc"},
		},
		{
			name:         "multiple dependencies",
			dependencies: []string{"vpc", "security-groups", "iam-roles"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stack := Stack{
				Dependencies: tt.dependencies,
			}

			if tt.dependencies == nil {
				assert.Nil(t, stack.Dependencies)
			} else {
				assert.Equal(t, tt.dependencies, stack.Dependencies)
				assert.Equal(t, len(tt.dependencies), len(stack.Dependencies))
			}
		})
	}
}

func TestStackConfig_Capabilities(t *testing.T) {
	// Test CloudFormation capabilities handling
	tests := []struct {
		name         string
		capabilities []string
	}{
		{
			name:         "no capabilities",
			capabilities: nil,
		},
		{
			name:         "CAPABILITY_IAM",
			capabilities: []string{"CAPABILITY_IAM"},
		},
		{
			name:         "multiple capabilities",
			capabilities: []string{"CAPABILITY_IAM", "CAPABILITY_NAMED_IAM", "CAPABILITY_AUTO_EXPAND"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stack := Stack{
				Capabilities: tt.capabilities,
			}

			assert.Equal(t, tt.capabilities, stack.Capabilities)
		})
	}
}
