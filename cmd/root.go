/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package cmd

import (
	"context"
	"os"

	"github.com/charmbracelet/fang"
	"github.com/orien/stackaroo/internal/version"
	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:     "stackaroo",
	Version: version.Short(),
	Short:   "A command-line tool for managing AWS CloudFormation stacks as code",
	Long: `Stackaroo is a CLI tool that simplifies CloudFormation stack management by providing:

• Declarative configuration in YAML files
• Context-specific parameter management
• Stack dependency resolution
• Change preview before deployment
• Template validation and deployment
• Rich terminal output with progress indicators

Use stackaroo to deploy, update, delete, diff, and monitor your CloudFormation stacks
across multiple contexts with consistent, repeatable configurations.`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := fang.Execute(context.Background(), rootCmd); err != nil {
		os.Exit(1)
	}
}

func init() {
	// Set custom version template to show detailed build information
	rootCmd.SetVersionTemplate(version.Info() + "\n")

	// Global flags
	rootCmd.PersistentFlags().StringP("config", "c", "stackaroo.yaml", "config file (default is stackaroo.yaml)")
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "verbose output")
}
