/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package cmd

import (
	"context"
	"fmt"

	"github.com/orien/stackaroo/internal/describe"
	"github.com/spf13/cobra"
)

var (
	// describer can be injected for testing
	describer describe.Describer
)

// describeCmd represents the describe command
var describeCmd = &cobra.Command{
	Use:   "describe <context> <stack-name>",
	Short: "Display detailed information about a CloudFormation stack",
	Long: `Display comprehensive information about a deployed CloudFormation stack.

This command shows detailed information about a stack including:

• Stack status and metadata (creation time, last update, etc.)
• Stack parameters and their current values
• Stack outputs (if any)
• Stack tags
• Stack description

The command retrieves information from the currently deployed stack in AWS
and displays it in a human-readable format.

Examples:
  stackaroo describe dev vpc        # Show information about VPC stack in dev context
  stackaroo describe prod app       # Show information about app stack in prod context`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		contextName := args[0]
		stackName := args[1]
		ctx := context.Background()

		return describeSingleStack(ctx, stackName, contextName)
	},
}

// getDescriber returns the describer instance, creating a default one if none is set
func getDescriber() describe.Describer {
	if describer != nil {
		return describer
	}

	cfOps := getCloudFormationOperations()
	describer = describe.NewStackDescriber(cfOps)
	return describer
}

// SetDescriber allows injection of a describer (for testing)
func SetDescriber(d describe.Describer) {
	describer = d
}

// describeSingleStack handles describing a single stack using configuration file
func describeSingleStack(ctx context.Context, stackName, contextName string) error {
	_, resolver := createResolver()

	// Resolve the target stack configuration
	stack, err := resolver.ResolveStack(ctx, contextName, stackName)
	if err != nil {
		return fmt.Errorf("failed to resolve stack %s: %w", stackName, err)
	}

	// Get describer instance
	d := getDescriber()

	// Retrieve stack information from AWS
	stackDesc, err := d.DescribeStack(ctx, stack)
	if err != nil {
		return fmt.Errorf("failed to describe stack %s: %w", stackName, err)
	}

	// Format and display the information
	fmt.Print(describe.FormatStackDescription(stackDesc))

	return nil
}

func init() {
	rootCmd.AddCommand(describeCmd)
}
