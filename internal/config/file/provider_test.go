/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package file

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileProvider_LoadConfig_ReturnsErrorWhenFileNotFound(t *testing.T) {
	// Test that FileProvider returns an appropriate error when config file doesn't exist
	provider := NewProvider("nonexistent-config.yaml")

	ctx := context.Background()
	cfg, err := provider.LoadConfig(ctx, "dev")

	assert.Error(t, err, "should return error when config file doesn't exist")
	assert.Nil(t, cfg, "should return nil config when file doesn't exist")
	assert.Contains(t, err.Error(), "nonexistent-config.yaml", "error should mention the file name")
}

func TestFileProvider_LoadConfig_ParsesBasicConfiguration(t *testing.T) {
	// Test that FileProvider can parse a basic stackaroo.yaml configuration
	configContent := `
project: test-project
region: us-east-1

contexts:
  dev:
    region: us-west-2
    account: "123456789012"
    tags:
      Environment: dev
  prod:
    region: us-east-1  
    account: "987654321098"
    tags:
      Environment: prod

stacks:
  - name: vpc
    template: templates/vpc.yaml
    parameters:
      VpcCidr: 10.0.0.0/16
    contexts:
      dev:
        parameters:
          VpcCidr: 10.1.0.0/16
`

	// Create temporary config file
	tmpFile := createTempConfigFile(t, configContent)

	provider := NewProvider(tmpFile)
	ctx := context.Background()

	cfg, err := provider.LoadConfig(ctx, "dev")
	require.NoError(t, err, "should successfully load valid config file")
	require.NotNil(t, cfg, "should return config object")

	// Verify global config
	assert.Equal(t, "test-project", cfg.Project)
	assert.Equal(t, "us-east-1", cfg.Region) // Global default

	// Verify context-specific config
	assert.Equal(t, "us-west-2", cfg.Context.Region) // Context override
	assert.Equal(t, "123456789012", cfg.Context.Account)
	assert.Equal(t, "dev", cfg.Context.Tags["Environment"])

	// Verify stacks
	require.Len(t, cfg.Stacks, 1)
	stack := cfg.Stacks[0]
	assert.Equal(t, "vpc", stack.Name)
	assert.True(t, strings.HasPrefix(stack.Template, "file://"), "template should be a file:// URI")
	assert.True(t, strings.HasSuffix(stack.Template, "templates/vpc.yaml"), "template should end with templates/vpc.yaml")
	assert.Equal(t, "10.1.0.0/16", stack.Parameters["VpcCidr"]) // Context-specific parameter
}

func TestFileProvider_ListContexts_ReturnsAvailableContexts(t *testing.T) {
	// Test that FileProvider can list available contexts from config file
	configContent := `
project: test-project

contexts:
  dev:
    region: us-west-2
  staging:
    region: us-east-1
  prod:
    region: us-east-1

stacks:
  - name: vpc
    template: templates/vpc.yaml
`

	tmpFile := createTempConfigFile(t, configContent)
	provider := NewProvider(tmpFile)

	contexts, err := provider.ListContexts()
	require.NoError(t, err, "should successfully list contexts")

	expected := []string{"dev", "staging", "prod"}
	assert.ElementsMatch(t, expected, contexts, "should return all defined contexts")
}

func TestFileProvider_GetStack_ReturnsStackWithContextOverrides(t *testing.T) {
	// Test that GetStack returns stack configuration with context-specific overrides applied
	configContent := `
project: test-project

contexts:
  dev:
    region: us-west-2
  prod:
    region: us-east-1

stacks:
  - name: database
    template: templates/rds.yaml
    parameters:
      DBInstanceClass: db.t3.micro
      MultiAZ: false
    tags:
      Component: database
    contexts:
      prod:
        parameters:
          DBInstanceClass: db.t3.small
          MultiAZ: true
        tags:
          Component: production-database
`

	tmpFile := createTempConfigFile(t, configContent)
	provider := NewProvider(tmpFile)

	// Test dev context (uses defaults)
	devStack, err := provider.GetStack("database", "dev")
	require.NoError(t, err)
	require.NotNil(t, devStack)
	assert.Equal(t, "database", devStack.Name)
	assert.Equal(t, "db.t3.micro", devStack.Parameters["DBInstanceClass"])
	assert.Equal(t, "false", devStack.Parameters["MultiAZ"]) // Boolean as string in YAML
	assert.Equal(t, "database", devStack.Tags["Component"])

	// Test prod context (uses overrides)
	prodStack, err := provider.GetStack("database", "prod")
	require.NoError(t, err)
	require.NotNil(t, prodStack)
	assert.Equal(t, "database", prodStack.Name)
	assert.Equal(t, "db.t3.small", prodStack.Parameters["DBInstanceClass"]) // Overridden
	assert.Equal(t, "true", prodStack.Parameters["MultiAZ"])                // Overridden
	assert.Equal(t, "production-database", prodStack.Tags["Component"])     // Overridden
}

func TestFileProvider_Validate_DetectsInvalidConfiguration(t *testing.T) {
	// Test that Validate catches common configuration errors
	invalidConfigContent := `
project: test-project

contexts:
  dev:
    region: us-west-2
  
stacks:
  - name: vpc
    template: nonexistent/template.yaml  # Invalid template path
    contexts:
      nonexistent-context:  # References context that doesn't exist
        parameters:
          VpcCidr: 10.0.0.0/16
`

	tmpFile := createTempConfigFile(t, invalidConfigContent)
	provider := NewProvider(tmpFile)

	err := provider.Validate()
	assert.Error(t, err, "should detect invalid configuration")
	// Could test for specific validation errors, but keeping it simple for now
}

// Helper function to create a temporary config file for testing
func createTempConfigFile(t *testing.T, content string) string {
	tmpDir := t.TempDir()
	tmpFile := tmpDir + "/stackaroo.yaml"

	err := os.WriteFile(tmpFile, []byte(content), 0644)
	require.NoError(t, err, "should create temporary config file")

	return tmpFile
}
