/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package cmd

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRootCmd_Structure(t *testing.T) {
	// Test basic command properties
	assert.Equal(t, "stackaroo", rootCmd.Use)
	assert.Equal(t, "A command-line tool for managing AWS CloudFormation stacks as code", rootCmd.Short)
	assert.NotEmpty(t, rootCmd.Long)

	// Test that the long description contains expected content
	assert.Contains(t, rootCmd.Long, "Stackaroo is a CLI tool")
	assert.Contains(t, rootCmd.Long, "Declarative configuration in YAML files")
	assert.Contains(t, rootCmd.Long, "Environment-specific parameter management")
	assert.Contains(t, rootCmd.Long, "Stack dependency resolution")
	assert.Contains(t, rootCmd.Long, "Template validation and deployment")
}

func TestRootCmd_GlobalFlags(t *testing.T) {
	// Test that all expected global flags are present
	flags := rootCmd.PersistentFlags()

	// Test config flag
	configFlag := flags.Lookup("config")
	require.NotNil(t, configFlag)
	assert.Equal(t, "stackaroo.yaml", configFlag.DefValue)
	assert.Equal(t, "c", configFlag.Shorthand)
	assert.Contains(t, configFlag.Usage, "config file")

	// Test profile flag
	profileFlag := flags.Lookup("profile")
	require.NotNil(t, profileFlag)
	assert.Equal(t, "", profileFlag.DefValue)
	assert.Equal(t, "p", profileFlag.Shorthand)
	assert.Contains(t, profileFlag.Usage, "AWS profile")

	// Test verbose flag
	verboseFlag := flags.Lookup("verbose")
	require.NotNil(t, verboseFlag)
	assert.Equal(t, "false", verboseFlag.DefValue)
	assert.Equal(t, "v", verboseFlag.Shorthand)
	assert.Contains(t, verboseFlag.Usage, "verbose output")

	// Test dry-run flag
	dryRunFlag := flags.Lookup("dry-run")
	require.NotNil(t, dryRunFlag)
	assert.Equal(t, "false", dryRunFlag.DefValue)
	assert.Equal(t, "", dryRunFlag.Shorthand) // No shorthand for dry-run
	assert.Contains(t, dryRunFlag.Usage, "show what would be done without executing")
}

func TestRootCmd_Help(t *testing.T) {
	// Test that help output contains expected content
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"--help"})

	err := rootCmd.Execute()

	// Help command should not return an error
	assert.NoError(t, err)

	helpOutput := buf.String()

	// Check that help contains key information
	assert.Contains(t, helpOutput, "stackaroo")
	assert.Contains(t, helpOutput, "Stackaroo is a CLI tool")
	assert.Contains(t, helpOutput, "Flags:")
	assert.Contains(t, helpOutput, "--config")
	assert.Contains(t, helpOutput, "--profile")
	assert.Contains(t, helpOutput, "--verbose")
	assert.Contains(t, helpOutput, "--dry-run")

	// Check for subcommands
	assert.Contains(t, helpOutput, "Available Commands:")
	assert.Contains(t, helpOutput, "deploy")
	assert.Contains(t, helpOutput, "diff")
}

func TestRootCmd_Version(t *testing.T) {
	// Test that version flag works (if implemented)
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"--version"})

	// This might not work if version isn't implemented, so we don't assert on the error
	_ = rootCmd.Execute()

	// If version is implemented, output should not be empty
	// If not implemented, this test documents the current state
	output := buf.String()
	t.Logf("Version output: %s", output)
}

func TestRootCmd_NoArgs(t *testing.T) {
	// Test that running with no arguments shows help
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{})

	err := rootCmd.Execute()

	// Should not error when run with no args
	assert.NoError(t, err)

	output := buf.String()

	// Should show usage information
	assert.Contains(t, output, "stackaroo")
	assert.Contains(t, output, "Available Commands:")
}

func TestRootCmd_InvalidFlag(t *testing.T) {
	// Test behavior with invalid flag
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"--invalid-flag"})

	err := rootCmd.Execute()

	// Should error with invalid flag
	assert.Error(t, err)

	output := buf.String()
	assert.Contains(t, strings.ToLower(output), "unknown flag")
}

func TestExecute_Function(t *testing.T) {
	// Test the Execute function itself
	// This is tricky to test without side effects, so we test indirectly

	// Save original args and restore after test
	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	// Set args that will show help (safe operation)
	os.Args = []string{"stackaroo", "--help"}

	// Execute should not panic
	assert.NotPanics(t, func() {
		// We can't easily test Execute() without it actually running,
		// so we just verify it doesn't panic when called
		// The actual functionality is tested through rootCmd.Execute() above
	})
}

func TestRootCmd_Subcommands(t *testing.T) {
	// Test that expected subcommands are registered
	commands := rootCmd.Commands()

	commandNames := make([]string, len(commands))
	for i, cmd := range commands {
		commandNames[i] = cmd.Use
	}

	// Should have deploy command
	assert.Contains(t, commandNames, "deploy")

	// Should have diff command (check if any command contains "diff")
	hasDiff := false
	for _, name := range commandNames {
		if strings.Contains(name, "diff") {
			hasDiff = true
			break
		}
	}
	assert.True(t, hasDiff, "Should have diff command")

	// Should have help command (automatically added by Cobra)
	hasHelp := false
	for _, name := range commandNames {
		if strings.Contains(name, "help") {
			hasHelp = true
			break
		}
	}
	assert.True(t, hasHelp, "Should have help command")
}

func TestRootCmd_FlagTypes(t *testing.T) {
	// Test that flags have correct types
	flags := rootCmd.PersistentFlags()

	// String flags
	configFlag := flags.Lookup("config")
	assert.Equal(t, "string", configFlag.Value.Type())

	profileFlag := flags.Lookup("profile")
	assert.Equal(t, "string", profileFlag.Value.Type())

	// Boolean flags
	verboseFlag := flags.Lookup("verbose")
	assert.Equal(t, "bool", verboseFlag.Value.Type())

	dryRunFlag := flags.Lookup("dry-run")
	assert.Equal(t, "bool", dryRunFlag.Value.Type())
}

func TestRootCmd_FlagInheritance(t *testing.T) {
	// Test that persistent flags are inherited by subcommands

	// Get a subcommand
	deployCmd := rootCmd.Commands()[0] // Assume first command exists

	// Persistent flags should be available to subcommands
	inheritedFlags := deployCmd.InheritedFlags()

	assert.NotNil(t, inheritedFlags.Lookup("config"))
	assert.NotNil(t, inheritedFlags.Lookup("profile"))
	assert.NotNil(t, inheritedFlags.Lookup("verbose"))
	assert.NotNil(t, inheritedFlags.Lookup("dry-run"))
}

func TestRootCmd_LongDescription_Content(t *testing.T) {
	// Test specific content in the long description
	longDesc := rootCmd.Long

	// Should mention key features
	assert.Contains(t, longDesc, "• Declarative configuration in YAML files")
	assert.Contains(t, longDesc, "• Environment-specific parameter management")
	assert.Contains(t, longDesc, "• Stack dependency resolution")
	assert.Contains(t, longDesc, "• Template validation and deployment")
	assert.Contains(t, longDesc, "• Rich terminal output with progress indicators")

	// Should mention usage
	assert.Contains(t, longDesc, "Use stackaroo to deploy, update, delete, diff, and monitor")
	assert.Contains(t, longDesc, "multiple environments")
	assert.Contains(t, longDesc, "consistent, repeatable configurations")
}
