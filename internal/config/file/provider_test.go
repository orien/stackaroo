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
	provider := NewFileConfigProvider("nonexistent-config.yaml")

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

	provider := NewFileConfigProvider(tmpFile)
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
	provider := NewFileConfigProvider(tmpFile)

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
	provider := NewFileConfigProvider(tmpFile)

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
	provider := NewFileConfigProvider(tmpFile)

	err := provider.Validate()
	assert.Error(t, err, "should detect invalid configuration")
	// Could test for specific validation errors, but keeping it simple for now
}

func TestFileProvider_ListStacks_ReturnsAllStackNames(t *testing.T) {
	// Test that ListStacks returns all available stack names for a context
	configContent := `
project: test-project

contexts:
  dev:
    region: us-west-2
  prod:
    region: us-east-1

stacks:
  - name: vpc
    template: templates/vpc.yaml
  - name: app
    template: templates/app.yaml
  - name: database
    template: templates/db.yaml
`

	tmpFile := createTempConfigFile(t, configContent)
	provider := NewFileConfigProvider(tmpFile)

	// Test valid context
	stackNames, err := provider.ListStacks("dev")
	require.NoError(t, err, "should successfully list stacks for valid context")

	expectedStacks := []string{"vpc", "app", "database"}
	assert.ElementsMatch(t, expectedStacks, stackNames, "should return all stack names")

	// Test another valid context
	stackNames, err = provider.ListStacks("prod")
	require.NoError(t, err, "should successfully list stacks for another valid context")
	assert.ElementsMatch(t, expectedStacks, stackNames, "should return same stacks for different context")

	// Test invalid context
	stackNames, err = provider.ListStacks("nonexistent")
	assert.Error(t, err, "should return error for nonexistent context")
	assert.Nil(t, stackNames, "should return nil stack names on error")
	assert.Contains(t, err.Error(), "context 'nonexistent' not found", "error should mention the missing context")
}

func TestFileProvider_ListStacks_HandlesEmptyConfiguration(t *testing.T) {
	// Test that ListStacks handles configuration with no stacks
	configContent := `
project: test-project

contexts:
  dev:
    region: us-west-2

stacks: []
`

	tmpFile := createTempConfigFile(t, configContent)
	provider := NewFileConfigProvider(tmpFile)

	stackNames, err := provider.ListStacks("dev")
	require.NoError(t, err, "should successfully handle empty stacks list")
	assert.Empty(t, stackNames, "should return empty list when no stacks are defined")
}

// Helper function to create a temporary config file for testing
func createTempConfigFile(t *testing.T, content string) string {
	tmpDir := t.TempDir()
	tmpFile := tmpDir + "/stackaroo.yaml"

	err := os.WriteFile(tmpFile, []byte(content), 0644)
	require.NoError(t, err, "should create temporary config file")

	return tmpFile
}

func TestFileProvider_LoadConfig_WithGlobalTemplateDirectory(t *testing.T) {
	// Test that global template directory resolves template paths correctly
	configContent := `
project: test-project
region: us-east-1

templates:
  directory: "templates/"

contexts:
  dev:
    region: us-west-2
    account: "123456789012"

stacks:
  - name: vpc
    template: vpc.yaml
  - name: app
    template: subdirectory/app.yaml
`

	// Create temporary config file and template directory structure
	tmpDir := t.TempDir()
	tmpFile := tmpDir + "/stackaroo.yaml"

	err := os.WriteFile(tmpFile, []byte(configContent), 0644)
	require.NoError(t, err)

	// Create template directory and files
	templatesDir := tmpDir + "/templates"
	err = os.MkdirAll(templatesDir+"/subdirectory", 0755)
	require.NoError(t, err)

	err = os.WriteFile(templatesDir+"/vpc.yaml", []byte("template content"), 0644)
	require.NoError(t, err)

	err = os.WriteFile(templatesDir+"/subdirectory/app.yaml", []byte("template content"), 0644)
	require.NoError(t, err)

	provider := NewFileConfigProvider(tmpFile)
	ctx := context.Background()

	cfg, err := provider.LoadConfig(ctx, "dev")
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Verify stacks use global template directory
	require.Len(t, cfg.Stacks, 2)

	vpcStack := cfg.Stacks[0]
	assert.Equal(t, "vpc", vpcStack.Name)
	assert.True(t, strings.HasPrefix(vpcStack.Template, "file://"))
	assert.True(t, strings.Contains(vpcStack.Template, "templates/vpc.yaml"))

	appStack := cfg.Stacks[1]
	assert.Equal(t, "app", appStack.Name)
	assert.True(t, strings.HasPrefix(appStack.Template, "file://"))
	assert.True(t, strings.Contains(appStack.Template, "templates/subdirectory/app.yaml"))
}

func TestFileProvider_LoadConfig_FallbackWithoutGlobalTemplateDirectory(t *testing.T) {
	// Test that without global template directory, behavior remains the same (backward compatibility)
	configContent := `
project: test-project
region: us-east-1

contexts:
  dev:
    region: us-west-2
    account: "123456789012"

stacks:
  - name: vpc
    template: templates/vpc.yaml
`

	tmpFile := createTempConfigFile(t, configContent)
	provider := NewFileConfigProvider(tmpFile)
	ctx := context.Background()

	cfg, err := provider.LoadConfig(ctx, "dev")
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Verify stack template path resolves relative to config directory (current behavior)
	require.Len(t, cfg.Stacks, 1)
	stack := cfg.Stacks[0]
	assert.Equal(t, "vpc", stack.Name)
	assert.True(t, strings.HasPrefix(stack.Template, "file://"))
	assert.True(t, strings.Contains(stack.Template, "templates/vpc.yaml"))
}

func TestFileProvider_LoadConfig_AbsolutePathsBypassGlobalDirectory(t *testing.T) {
	// Test that absolute template paths bypass global template directory
	configContent := `
project: test-project
region: us-east-1

templates:
  directory: "templates/"

contexts:
  dev:
    region: us-west-2
    account: "123456789012"

stacks:
  - name: vpc
    template: /absolute/path/vpc.yaml
`

	tmpFile := createTempConfigFile(t, configContent)
	provider := NewFileConfigProvider(tmpFile)
	ctx := context.Background()

	cfg, err := provider.LoadConfig(ctx, "dev")
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Verify absolute path bypasses global template directory
	require.Len(t, cfg.Stacks, 1)
	stack := cfg.Stacks[0]
	assert.Equal(t, "vpc", stack.Name)
	assert.Equal(t, "file:///absolute/path/vpc.yaml", stack.Template)
}

func TestFileProvider_Validate_ChecksGlobalTemplateDirectoryExists(t *testing.T) {
	// Test that validation fails if global template directory doesn't exist
	configContent := `
project: test-project
region: us-east-1

templates:
  directory: "nonexistent-templates/"

contexts:
  dev:
    region: us-west-2

stacks:
  - name: vpc
    template: vpc.yaml
`

	tmpFile := createTempConfigFile(t, configContent)
	provider := NewFileConfigProvider(tmpFile)

	err := provider.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "global template directory not found")
	assert.Contains(t, err.Error(), "nonexistent-templates")
}

func TestFileProvider_Validate_PassesWithValidGlobalTemplateDirectory(t *testing.T) {
	// Test that validation passes when global template directory exists
	tmpDir := t.TempDir()
	tmpFile := tmpDir + "/stackaroo.yaml"

	configContent := `
project: test-project
region: us-east-1

templates:
  directory: "templates/"

contexts:
  dev:
    region: us-west-2

stacks:
  - name: vpc
    template: vpc.yaml
`

	err := os.WriteFile(tmpFile, []byte(configContent), 0644)
	require.NoError(t, err)

	// Create template directory and file
	templatesDir := tmpDir + "/templates"
	err = os.MkdirAll(templatesDir, 0755)
	require.NoError(t, err)

	err = os.WriteFile(templatesDir+"/vpc.yaml", []byte("template content"), 0644)
	require.NoError(t, err)

	provider := NewFileConfigProvider(tmpFile)

	err = provider.Validate()
	assert.NoError(t, err)
}

func TestFileProvider_Validate_AbsoluteGlobalTemplateDirectory(t *testing.T) {
	// Test that global template directory works with absolute paths
	tmpDir := t.TempDir()
	tmpFile := tmpDir + "/stackaroo.yaml"
	templatesDir := tmpDir + "/absolute-templates"

	configContent := `
project: test-project
region: us-east-1

templates:
  directory: "` + templatesDir + `"

contexts:
  dev:
    region: us-west-2

stacks:
  - name: vpc
    template: vpc.yaml
`

	err := os.WriteFile(tmpFile, []byte(configContent), 0644)
	require.NoError(t, err)

	// Create absolute template directory and file
	err = os.MkdirAll(templatesDir, 0755)
	require.NoError(t, err)

	err = os.WriteFile(templatesDir+"/vpc.yaml", []byte("template content"), 0644)
	require.NoError(t, err)

	provider := NewFileConfigProvider(tmpFile)
	ctx := context.Background()

	// Test loading config
	cfg, err := provider.LoadConfig(ctx, "dev")
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Verify template resolves to absolute directory
	require.Len(t, cfg.Stacks, 1)
	stack := cfg.Stacks[0]
	assert.Equal(t, "file://"+templatesDir+"/vpc.yaml", stack.Template)

	// Test validation
	err = provider.Validate()
	assert.NoError(t, err)
}
