/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/

package file

import (
	"context"
	"fmt"
	"os"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// ExampleParameterValue_parsing demonstrates how to parse different parameter value types
func ExampleParameterValue_parsing() {
	// Example YAML configuration showing different parameter types
	yamlConfig := `
project: ecommerce-platform
region: us-west-2

stacks:
  - name: networking
    template: networking.yml
    parameters:
      # Literal values (backwards compatible)
      Environment: production
      VpcCidr: "10.0.0.0/16"
      
  - name: application
    template: application.yml
    parameters:
      # Literal values
      AppName: my-ecommerce-app
      
      # Stack output resolver
      VpcId:
        type: stack-output
        stack_name: networking
        output_key: VpcId
        
      # Cross-region stack output
      SharedBucketArn:
        type: stack-output
        stack_name: shared-resources
        output_key: BucketArn
        region: us-east-1
`

	var config Config
	err := yaml.Unmarshal([]byte(yamlConfig), &config)
	if err != nil {
		fmt.Printf("Error parsing YAML: %v\n", err)
		return
	}

	fmt.Printf("Project: %s\n", config.Project)
	fmt.Printf("Region: %s\n", config.Region)
	fmt.Printf("Stacks: %d\n", len(config.Stacks))

	// Examine the networking stack (literal parameters only)
	networkingStack := config.Stacks[0]
	fmt.Printf("\nNetworking Stack: %s\n", networkingStack.Name)
	for paramName, paramValue := range networkingStack.Parameters {
		if paramValue.IsLiteral() {
			fmt.Printf("  %s = %q (literal)\n", paramName, paramValue.Literal)
		}
	}

	// Examine the application stack (mixed parameter types)
	appStack := config.Stacks[1]
	fmt.Printf("\nApplication Stack: %s\n", appStack.Name)

	// Sort parameter names for deterministic output
	var paramNames []string
	for paramName := range appStack.Parameters {
		paramNames = append(paramNames, paramName)
	}
	sort.Strings(paramNames)

	for _, paramName := range paramNames {
		paramValue := appStack.Parameters[paramName]
		if paramValue.IsLiteral() {
			fmt.Printf("  %s = %q (literal)\n", paramName, paramValue.Literal)
		} else if paramValue.IsResolver() {
			fmt.Printf("  %s -> resolver(type=%s)\n", paramName, paramValue.Resolver.Type)
			if paramValue.Resolver.Type == "stack-output" {
				stackName := paramValue.Resolver.Config["stack_name"]
				outputKey := paramValue.Resolver.Config["output_key"]
				fmt.Printf("    references: %s.%s\n", stackName, outputKey)
			}
		}
	}

	// Output:
	// Project: ecommerce-platform
	// Region: us-west-2
	// Stacks: 2
	//
	// Networking Stack: networking
	//   Environment = "production" (literal)
	//   VpcCidr = "10.0.0.0/16" (literal)
	//
	// Application Stack: application
	//   AppName = "my-ecommerce-app" (literal)
	//   SharedBucketArn -> resolver(type=stack-output)
	//     references: shared-resources.BucketArn
	//   VpcId -> resolver(type=stack-output)
	//     references: networking.VpcId
}

// TestExample_CompleteWorkflow demonstrates a complete workflow from YAML parsing to inspection
func TestExample_CompleteWorkflow(t *testing.T) {
	yamlContent := `
project: demo-project
region: us-west-2

contexts:
  dev:
    account: "111111111111"
    region: us-west-2
  prod:
    account: "222222222222"
    region: us-east-1

stacks:
  - name: database
    template: rds.yml
    parameters:
      # Literal parameters
      Environment: dev
      Engine: postgres
      
      # Cross-stack reference
      VpcId:
        type: output
        stack_name: networking
        output_key: VpcId
        
      # Environment-specific overrides
    contexts:
      prod:
        parameters:
          Environment: production
          MultiAZ: "true"
          BackupRetention: "30"
`

	// Parse the YAML
	var config Config
	err := yaml.Unmarshal([]byte(yamlContent), &config)
	require.NoError(t, err)

	// Verify basic structure
	assert.Equal(t, "demo-project", config.Project)
	assert.Equal(t, "us-west-2", config.Region)
	assert.Len(t, config.Contexts, 2)
	assert.Len(t, config.Stacks, 1)

	// Examine the database stack
	dbStack := config.Stacks[0]
	assert.Equal(t, "database", dbStack.Name)
	assert.Equal(t, "rds.yml", dbStack.Template)

	// Check literal parameters
	envParam := dbStack.Parameters["Environment"]
	assert.True(t, envParam.IsLiteral())
	assert.Equal(t, "dev", envParam.Literal)

	engineParam := dbStack.Parameters["Engine"]
	assert.True(t, engineParam.IsLiteral())
	assert.Equal(t, "postgres", engineParam.Literal)

	// Check resolver parameter
	vpcIdParam := dbStack.Parameters["VpcId"]
	assert.True(t, vpcIdParam.IsResolver())
	assert.Equal(t, "output", vpcIdParam.Resolver.Type)
	assert.Equal(t, "networking", vpcIdParam.Resolver.Config["stack_name"])
	assert.Equal(t, "VpcId", vpcIdParam.Resolver.Config["output_key"])

	// Check context overrides
	prodOverride := dbStack.Contexts["prod"]
	require.NotNil(t, prodOverride)

	prodEnvParam := prodOverride.Parameters["Environment"]
	assert.True(t, prodEnvParam.IsLiteral())
	assert.Equal(t, "production", prodEnvParam.Literal)

	multiAZParam := prodOverride.Parameters["MultiAZ"]
	assert.True(t, multiAZParam.IsLiteral())
	assert.Equal(t, "true", multiAZParam.Literal)

	fmt.Printf("Successfully parsed configuration with %d contexts and %d stacks\n",
		len(config.Contexts), len(config.Stacks))
	fmt.Printf("Database stack has %d base parameters and %d prod overrides\n",
		len(dbStack.Parameters), len(prodOverride.Parameters))
}

// TestExample_BackwardsCompatibility shows that old string-based configurations still work
func TestExample_BackwardsCompatibility(t *testing.T) {
	// This YAML only uses literal string values (old format)
	oldFormatYAML := `
project: legacy-project
region: us-east-1

stacks:
  - name: web-app
    template: webapp.yml
    parameters:
      Environment: production
      InstanceType: t3.medium
      MinSize: "2"
      MaxSize: "10"
    tags:
      Owner: platform-team
      Cost: webapp
`

	// Parse with new ParameterValue system
	var config Config
	err := yaml.Unmarshal([]byte(oldFormatYAML), &config)
	require.NoError(t, err)

	webAppStack := config.Stacks[0]
	assert.Equal(t, "web-app", webAppStack.Name)

	// All parameters should be parsed as literals
	for paramName, paramValue := range webAppStack.Parameters {
		assert.True(t, paramValue.IsLiteral(), "Parameter %s should be literal", paramName)
		assert.False(t, paramValue.IsResolver(), "Parameter %s should not be resolver", paramName)
	}

	// Verify specific values
	assert.Equal(t, "production", webAppStack.Parameters["Environment"].Literal)
	assert.Equal(t, "t3.medium", webAppStack.Parameters["InstanceType"].Literal)
	assert.Equal(t, "2", webAppStack.Parameters["MinSize"].Literal)
	assert.Equal(t, "10", webAppStack.Parameters["MaxSize"].Literal)

	fmt.Printf("Successfully parsed legacy configuration with %d literal parameters\n",
		len(webAppStack.Parameters))
}

// TestExample_FileProviderWithLiterals demonstrates using the FileConfigProvider with literal parameters
func TestExample_FileProviderWithLiterals(t *testing.T) {
	yamlContent := `
project: example-project
region: us-west-2

contexts:
  dev:
    account: "123456789012"
    region: us-west-2

stacks:
  - name: simple-stack
    template: simple.yml
    parameters:
      Environment: dev
      Debug: "true"
      Port: "8080"
    contexts:
      dev:
        parameters:
          Debug: "false"
          LogLevel: info
`

	// Create temporary file
	tmpFile, err := os.CreateTemp("", "stackaroo-example-*.yml")
	require.NoError(t, err)
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	_, err = tmpFile.WriteString(yamlContent)
	require.NoError(t, err)
	_ = tmpFile.Close()

	// Use FileConfigProvider to parse and resolve
	provider := NewFileConfigProvider(tmpFile.Name())

	// Load overall config
	config, err := provider.LoadConfig(context.TODO(), "dev")
	require.NoError(t, err)

	assert.Equal(t, "example-project", config.Project)
	assert.Equal(t, "us-west-2", config.Region)

	// Get specific stack (this will resolve literal parameters and merge context overrides)
	stackConfig, err := provider.GetStack("simple-stack", "dev")
	require.NoError(t, err)

	assert.Equal(t, "simple-stack", stackConfig.Name)
	assert.Contains(t, stackConfig.Template, "simple.yml") // Template URI is resolved to full path

	// Verify resolved parameters (literals + context overrides)
	expectedParams := map[string]string{
		"Environment": "dev",   // from base
		"Debug":       "false", // overridden by context
		"Port":        "8080",  // from base
		"LogLevel":    "info",  // added by context
	}

	// Extract literal values from ParameterValue objects for comparison
	actualParams := make(map[string]string)
	for key, paramValue := range stackConfig.Parameters {
		if paramValue.ResolutionType == "literal" {
			actualParams[key] = paramValue.ResolutionConfig["value"]
		}
	}

	assert.Equal(t, expectedParams, actualParams)

	fmt.Printf("Successfully loaded stack configuration with %d resolved parameters\n",
		len(stackConfig.Parameters))
}

// TestExample_ErrorHandlingWithResolvers shows what happens when resolver parameters are encountered
func TestExample_ErrorHandlingWithResolvers(t *testing.T) {
	yamlWithResolvers := `
project: resolver-project
region: us-west-2

stacks:
  - name: app-stack
    template: app.yml
    parameters:
      # This works fine
      AppName: my-app
      
      # This should be parsed successfully by the provider
      VpcId:
        type: stack-output
        stack_name: vpc-stack
        output_key: VpcId
`

	// Create temporary file
	tmpFile, err := os.CreateTemp("", "stackaroo-resolver-*.yml")
	require.NoError(t, err)
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	_, err = tmpFile.WriteString(yamlWithResolvers)
	require.NoError(t, err)
	_ = tmpFile.Close()

	provider := NewFileConfigProvider(tmpFile.Name())

	// This should succeed - provider parses resolver parameters into ParameterValue objects
	stackConfig, err := provider.GetStack("app-stack", "dev")
	require.NoError(t, err)

	// Verify literal parameter
	appNameParam := stackConfig.Parameters["AppName"]
	assert.True(t, appNameParam.ResolutionType == "literal")
	assert.Equal(t, "my-app", appNameParam.ResolutionConfig["value"])

	// Verify resolver parameter is parsed correctly
	vpcIdParam := stackConfig.Parameters["VpcId"]
	assert.True(t, vpcIdParam.ResolutionType == "stack-output")
	assert.Equal(t, "vpc-stack", vpcIdParam.ResolutionConfig["stack_name"])
	assert.Equal(t, "VpcId", vpcIdParam.ResolutionConfig["output_key"])

	fmt.Printf("Successfully parsed resolver parameters into ParameterValue objects\n")

	// However, we can still parse the YAML directly to access raw parameter data
	var rawConfig Config
	err = yaml.Unmarshal([]byte(yamlWithResolvers), &rawConfig)
	require.NoError(t, err) // This works and preserves ParameterValue types

	rawStack := rawConfig.Stacks[0] // Access raw stack data with yamlParameterValue types
	rawVpcIdParam := rawStack.Parameters["VpcId"]
	assert.True(t, rawVpcIdParam.IsResolver())
	assert.Equal(t, "stack-output", rawVpcIdParam.Resolver.Type)

	fmt.Printf("Raw YAML parsing works fine, resolver info preserved for higher-level processing\n")
}

// TestExample_ListParameters demonstrates the full list parameter workflow
func TestExample_ListParameters(t *testing.T) {
	// Example YAML configuration with list parameters
	yamlConfig := `
project: ecommerce-platform
region: us-east-1
contexts:
  prod:
    account: "123456789012"
    region: us-east-1
stacks:
  - name: web-application
    template: webapp.yml
    parameters:
      # Simple literal list
      AllowedPorts:
        - "80"
        - "443"
        - "8080"
      
      # Mixed list with literals and stack outputs  
      SecurityGroupIds:
        - sg-baseline123
        - type: stack-output
          stack_name: security-stack
          output_key: WebServerSGId
        - type: stack-output
          stack_name: database-stack
          output_key: DatabaseSGId
        - sg-additional456
      
      # All stack outputs from different stacks
      SubnetIds:
        - type: stack-output
          stack_name: vpc-stack
          output_key: PublicSubnet1Id
        - type: stack-output
          stack_name: vpc-stack
          output_key: PublicSubnet2Id
        - type: stack-output
          stack_name: additional-vpc
          output_key: ExtraSubnetId
`

	// Parse YAML into raw config structure
	var rawConfig Config
	err := yaml.Unmarshal([]byte(yamlConfig), &rawConfig)
	if err != nil {
		fmt.Printf("Failed to parse YAML: %v\n", err)
		return
	}

	// Get the web application stack
	webAppStack := rawConfig.Stacks[0]
	fmt.Printf("Stack: %s\n", webAppStack.Name)
	fmt.Printf("Parameters parsed:\n")

	// Examine each parameter type
	for paramName, paramValue := range webAppStack.Parameters {
		if paramValue.IsLiteral() {
			fmt.Printf("  %s = \"%s\" (literal)\n", paramName, paramValue.Literal)
		} else if paramValue.IsList() {
			fmt.Printf("  %s = [list with %d items]\n", paramName, len(paramValue.ListItems))

			// Show details of list items
			for i, item := range paramValue.ListItems {
				if item.IsLiteral() {
					fmt.Printf("    [%d] = \"%s\" (literal)\n", i, item.Literal)
				} else if item.IsResolver() {
					fmt.Printf("    [%d] = resolver(type=%s", i, item.Resolver.Type)
					if stackName, exists := item.Resolver.Config["stack_name"]; exists {
						outputKey := item.Resolver.Config["output_key"]
						fmt.Printf(", %s.%s", stackName, outputKey)
					}
					fmt.Printf(")\n")
				}
			}
		} else if paramValue.IsResolver() {
			fmt.Printf("  %s = resolver(type=%s)\n", paramName, paramValue.Resolver.Type)
		}
	}

	// Convert to config.ParameterValue format
	fmt.Printf("\nConverted to config.ParameterValue:\n")
	for paramName, paramValue := range webAppStack.Parameters {
		configParam := paramValue.ToConfigParameterValue()
		if configParam != nil {
			fmt.Printf("  %s: ResolutionType=%s", paramName, configParam.ResolutionType)
			if configParam.ResolutionType == "list" {
				fmt.Printf(" (list with %d items)", len(configParam.ListItems))
			}
			fmt.Printf("\n")
		}
	}

	// Output:
	// Stack: web-application
	// Parameters parsed:
	//   AllowedPorts = [list with 3 items]
	//     [0] = "80" (literal)
	//     [1] = "443" (literal)
	//     [2] = "8080" (literal)
	//   SecurityGroupIds = [list with 4 items]
	//     [0] = "sg-baseline123" (literal)
	//     [1] = resolver(type=stack-output, security-stack.WebServerSGId)
	//     [2] = resolver(type=stack-output, database-stack.DatabaseSGId)
	//     [3] = "sg-additional456" (literal)
	//   SubnetIds = [list with 3 items]
	//     [0] = resolver(type=stack-output, vpc-stack.PublicSubnet1Id)
	//     [1] = resolver(type=stack-output, vpc-stack.PublicSubnet2Id)
	//     [2] = resolver(type=stack-output, additional-vpc.ExtraSubnetId)
	//
	// Converted to config.ParameterValue:
	//   AllowedPorts: ResolutionType=list (list with 3 items)
	//   SecurityGroupIds: ResolutionType=list (list with 4 items)
	//   SubnetIds: ResolutionType=list (list with 3 items)
}
