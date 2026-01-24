/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package cmd

import (
	"context"

	"codeberg.org/orien/stackaroo/internal/delete"
	"github.com/spf13/cobra"
)

var (
	// deleter can be injected for testing
	deleter delete.Deleter
)

// deleteCmd represents the delete command
var deleteCmd = &cobra.Command{
	Use:   "delete <context> [stack-name]",
	Short: "Delete CloudFormation stacks",
	Long: `Delete CloudFormation stacks with dependency-aware ordering and confirmation prompts.

This command safely deletes CloudFormation stacks by:

• Resolving stack dependencies and deleting in reverse order
• Showing detailed information about what will be deleted
• Prompting for confirmation before deletion

When deleting multiple stacks, they are processed in reverse dependency order
to ensure dependent stacks are deleted before their dependencies.

Examples:
  stackaroo delete dev vpc        # Delete single stack with confirmation
  stackaroo delete dev            # Delete all stacks in context with confirmation

CAUTION: Deletion is destructive and cannot be undone. Always verify what
will be deleted before confirming.`,
	Args: cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		contextName := args[0]
		ctx := context.Background()

		configFile, _ := cmd.Flags().GetString("config")
		d := getDeleter(configFile)

		if len(args) > 1 {
			stackName := args[1]
			return d.DeleteSingleStack(ctx, stackName, contextName)
		}
		return d.DeleteAllStacks(ctx, contextName)
	},
}

// getDeleter returns the deleter instance, creating a default one if none is set
func getDeleter(configFile string) delete.Deleter {
	if deleter != nil {
		return deleter
	}

	clientFactory := getClientFactory()
	provider, resolver := createResolver(configFile)
	deleter = delete.NewStackDeleter(clientFactory, provider, resolver)
	return deleter
}

// SetDeleter allows injection of a deleter (for testing)
func SetDeleter(d delete.Deleter) {
	deleter = d
}

func init() {
	rootCmd.AddCommand(deleteCmd)
}
