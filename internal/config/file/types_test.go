/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package file

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

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
	stacks := []*Stack{{Name: "test-stack"}}

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
		Stacks: []*Stack{
			{
				Name:     "vpc",
				Template: "templates/vpc.yaml",
				Parameters: map[string]string{
					"VpcCidr": "10.0.0.0/16",
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

	assert.Equal(t, "", stack.Name)
	assert.Equal(t, "", stack.Template)
	assert.Nil(t, stack.Parameters)
	assert.Nil(t, stack.Tags)
	assert.Nil(t, stack.Dependencies)
	assert.Nil(t, stack.Capabilities)
	assert.Nil(t, stack.Contexts)
}

func TestStackConfig_FieldAssignment(t *testing.T) {
	// Test that Stack fields can be set and retrieved
	parameters := map[string]string{"Size": "large"}
	tags := map[string]string{"Component": "database"}
	dependencies := []string{"vpc", "security-groups"}
	capabilities := []string{"CAPABILITY_IAM", "CAPABILITY_NAMED_IAM"}
	contexts := map[string]*ContextOverride{
		"dev": {Parameters: map[string]string{"Size": "small"}},
	}

	stack := Stack{
		Name:         "database",
		Template:     "templates/rds.yaml",
		Parameters:   parameters,
		Tags:         tags,
		Dependencies: dependencies,
		Capabilities: capabilities,
		Contexts:     contexts,
	}

	assert.Equal(t, "database", stack.Name)
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
	parameters := map[string]string{"InstanceType": "t3.micro"}
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
  - name: vpc
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
          
  - name: app
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

	// Verify contexts
	assert.Len(t, config.Contexts, 2)
	devContext := config.Contexts["dev"]
	assert.NotNil(t, devContext)
	assert.Equal(t, "123456789012", devContext.Account)
	assert.Equal(t, "dev", devContext.Tags["Environment"])

	// Verify stacks
	assert.Len(t, config.Stacks, 2)

	vpcStack := config.Stacks[0]
	assert.Equal(t, "vpc", vpcStack.Name)
	assert.Equal(t, "templates/vpc.yaml", vpcStack.Template)
	assert.Equal(t, "10.0.0.0/16", vpcStack.Parameters["VpcCidr"])
	assert.Contains(t, vpcStack.Capabilities, "CAPABILITY_IAM")
	assert.Equal(t, "10.1.0.0/16", vpcStack.Contexts["dev"].Parameters["VpcCidr"])

	appStack := config.Stacks[1]
	assert.Equal(t, "app", appStack.Name)
	assert.Contains(t, appStack.Dependencies, "vpc")
	assert.Equal(t, "t3.micro", appStack.Contexts["dev"].Parameters["InstanceType"])
	assert.Equal(t, "enabled", appStack.Contexts["prod"].Tags["Monitoring"])
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
				Name:         "test-stack",
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
				Name:         "test-stack",
				Capabilities: tt.capabilities,
			}

			assert.Equal(t, tt.capabilities, stack.Capabilities)
		})
	}
}
