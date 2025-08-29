/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package cmd

import (
	"context"
	"fmt"

	"github.com/orien/stackaroo/internal/aws"
	"github.com/orien/stackaroo/internal/config/file"
	"github.com/orien/stackaroo/internal/delete"
	"github.com/orien/stackaroo/internal/model"
	"github.com/orien/stackaroo/internal/resolve"
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
		var stackName string
		if len(args) > 1 {
			stackName = args[1]
		}
		ctx := context.Background()

		return deleteWithConfig(ctx, stackName, contextName)
	},
}

// getDeleter returns the deleter instance, creating a default one if none is set
func getDeleter() delete.Deleter {
	if deleter != nil {
		return deleter
	}

	// Create default deleter
	ctx := context.Background()
	client, err := aws.NewDefaultClient(ctx, aws.Config{})
	if err != nil {
		// This shouldn't happen in normal operation, but if it does,
		// we'll handle it in the command execution
		panic(fmt.Sprintf("failed to create AWS client: %v", err))
	}

	deleter = delete.NewStackDeleter(client)
	return deleter
}

// SetDeleter allows injection of a deleter (for testing)
func SetDeleter(d delete.Deleter) {
	deleter = d
}

// deleteWithConfig handles deletion using configuration file
func deleteWithConfig(ctx context.Context, stackName, contextName string) error {
	// Create configuration provider and resolver
	provider := file.NewDefaultProvider()
	resolver := resolve.NewStackResolver(provider)

	// Determine which stacks to delete
	var stackNames []string
	if stackName != "" {
		// Delete single stack
		stackNames = []string{stackName}
	} else {
		// Delete all stacks in context
		var err error
		stackNames, err = provider.ListStacks(contextName)
		if err != nil {
			return fmt.Errorf("failed to get stacks for context %s: %w", contextName, err)
		}
		if len(stackNames) == 0 {
			fmt.Printf("No stacks found in context %s\n", contextName)
			return nil
		}
	}

	// Resolve stack(s) and all their dependencies
	resolved, err := resolver.Resolve(ctx, contextName, stackNames)
	if err != nil {
		return fmt.Errorf("failed to resolve stack dependencies: %w", err)
	}

	// Get or create deleter
	d := getDeleter()

	// Reverse the deployment order for safe deletion
	// Dependencies should be deleted before the stacks that depend on them
	deletionOrder := make([]string, len(resolved.DeploymentOrder))
	for i, stackName := range resolved.DeploymentOrder {
		deletionOrder[len(resolved.DeploymentOrder)-1-i] = stackName
	}

	// Delete all stacks in reverse dependency order
	for _, stackName := range deletionOrder {
		// Find the resolved stack
		var stackToDelete *model.Stack
		for _, stack := range resolved.Stacks {
			if stack.Name == stackName {
				stackToDelete = stack
				break
			}
		}

		if stackToDelete == nil {
			return fmt.Errorf("resolved stack %s not found", stackName)
		}

		// Delete the stack
		err = d.DeleteStack(ctx, stackToDelete)
		if err != nil {
			return fmt.Errorf("error deleting stack %s: %w", stackName, err)
		}

		fmt.Printf("Successfully deleted stack %s in context %s\n", stackName, contextName)
	}

	return nil
}

func init() {
	rootCmd.AddCommand(deleteCmd)
}
