/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "stackaroo",
	Short: "A command-line tool for managing AWS CloudFormation stacks as code",
	Long: `Stackaroo is a CLI tool that simplifies CloudFormation stack management by providing:

• Declarative configuration in YAML files
• Environment-specific parameter management
• Stack dependency resolution
• Template validation and deployment
• Rich terminal output with progress indicators

Use stackaroo to deploy, update, delete, and monitor your CloudFormation stacks
across multiple environments with consistent, repeatable configurations.`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().StringP("config", "c", "stackaroo.yaml", "config file (default is stackaroo.yaml)")
	rootCmd.PersistentFlags().StringP("profile", "p", "", "AWS profile (overrides config)")
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().Bool("dry-run", false, "show what would be done without executing")
}
