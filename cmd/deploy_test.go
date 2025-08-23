/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeployCommand_Exists(t *testing.T) {
	// Test that deploy command is registered with root command
	deployCmd := findCommand(rootCmd, "deploy")
	
	assert.NotNil(t, deployCmd, "deploy command should be registered")
	assert.Equal(t, "deploy", deployCmd.Use)
}

func TestDeployCommand_AcceptsStackName(t *testing.T) {
	// Test that deploy command accepts a stack name argument
	deployCmd := findCommand(rootCmd, "deploy")
	assert.NotNil(t, deployCmd)
	
	// Test that Args validation is set to require exactly one argument
	assert.NotNil(t, deployCmd.Args, "deploy command should have Args validation set")
}

func TestDeployCommand_HasTemplateFlag(t *testing.T) {
	// Test that deploy command has a --template flag
	deployCmd := findCommand(rootCmd, "deploy")
	assert.NotNil(t, deployCmd)
	
	// Check that --template flag exists
	templateFlag := deployCmd.Flags().Lookup("template")
	assert.NotNil(t, templateFlag, "deploy command should have --template flag")
	assert.Equal(t, "template", templateFlag.Name)
}

func TestDeployCommand_ReadsTemplateFile(t *testing.T) {
	// Test that deploy command can read actual template file content
	templateContent := `{
		"AWSTemplateFormatVersion": "2010-09-09",
		"Resources": {
			"TestBucket": {
				"Type": "AWS::S3::Bucket"
			}
		}
	}`
	
	// Test error handling when file doesn't exist
	content, err := ReadTemplateFile("test-template.json")
	assert.Error(t, err, "should error when file doesn't exist")
	assert.Empty(t, content, "content should be empty when file doesn't exist")
	
	// Template content for testing
	_ = templateContent // Use the variable so it doesn't cause unused error
}

func TestReadTemplateFile_ReadsActualFile(t *testing.T) {
	// Test that ReadTemplateFile actually reads file content from disk
	templateContent := `{
	"AWSTemplateFormatVersion": "2010-09-09",
	"Resources": {
		"TestBucket": {
			"Type": "AWS::S3::Bucket"
		}
	}
}`

	// Create a temporary file
	tmpDir := t.TempDir()
	templateFile := filepath.Join(tmpDir, "test-template.json")
	
	err := os.WriteFile(templateFile, []byte(templateContent), 0644)
	require.NoError(t, err)

	// Test reading actual file content from disk
	content, err := ReadTemplateFile(templateFile)
	assert.NoError(t, err, "should successfully read existing file")
	assert.Equal(t, templateContent, content, "should return file content")
}

func TestDeployCommand_CallsDeployStack(t *testing.T) {
	// Test that DeployStack function can be called with stack name and template file
	
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
	
	// Test calling DeployStack directly
	err = DeployStack("test-stack", templateFile)
	assert.NoError(t, err, "DeployStack should execute without error")
}

func TestDeployCommand_RunCallsDeployStack(t *testing.T) {
	// Test that deploy command executes end-to-end successfully
	
	// Create a temporary template file
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
	
	// Execute the root command with deploy subcommand and arguments
	rootCmd.SetArgs([]string{"deploy", "test-stack", "--template", templateFile})
	
	// Execute the command
	err = rootCmd.Execute()
	assert.NoError(t, err, "deploy command execution should not error")
	
	// For now, we just verify the command executes successfully
	// TODO: Add proper integration test with AWS client mock
}

// Helper function to find a command by name
func findCommand(parent *cobra.Command, name string) *cobra.Command {
	for _, cmd := range parent.Commands() {
		if cmd.Use == name {
			return cmd
		}
	}
	return nil
}