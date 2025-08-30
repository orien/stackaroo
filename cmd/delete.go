/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package cmd

import (
	"context"
	"fmt"

	"github.com/orien/stackaroo/internal/delete"
	"github.com/orien/stackaroo/internal/model"

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

		if len(args) > 1 {
			stackName := args[1]
			return deleteSingleStack(ctx, stackName, contextName, configFile)
		}
		return deleteAllStacks(ctx, contextName, configFile)
	},
}

// getDeleter returns the deleter instance, creating a default one if none is set
func getDeleter() delete.Deleter {
	if deleter != nil {
		return deleter
	}

	cfOpts := getCloudFormationOperations()
	deleter = delete.NewStackDeleter(cfOpts)
	return deleter
}

// SetDeleter allows injection of a deleter (for testing)
func SetDeleter(d delete.Deleter) {
	deleter = d
}

// deleteStackWithFeedback deletes a stack and provides feedback
func deleteStackWithFeedback(ctx context.Context, stack *model.Stack, contextName string) error {
	d := getDeleter()

	err := d.DeleteStack(ctx, stack)
	if err != nil {
		return fmt.Errorf("error deleting stack %s: %w", stack.Name, err)
	}

	fmt.Printf("Successfully deleted stack %s in context %s\n", stack.Name, contextName)
	return nil
}

// deleteSingleStack handles deletion of a single stack
func deleteSingleStack(ctx context.Context, stackName, contextName, configFile string) error {
	_, resolver := createResolver(configFile)

	// Resolve single stack
	stack, err := resolver.ResolveStack(ctx, contextName, stackName)
	if err != nil {
		return fmt.Errorf("failed to resolve stack dependencies: %w", err)
	}

	return deleteStackWithFeedback(ctx, stack, contextName)
}

// deleteAllStacks handles deletion of all stacks in a context using configuration file
func deleteAllStacks(ctx context.Context, contextName, configFile string) error {
	provider, resolver := createResolver(configFile)

	// Get list of stacks to delete
	stackNames, err := provider.ListStacks(contextName)
	if err != nil {
		return fmt.Errorf("failed to get stacks for context %s: %w", contextName, err)
	}
	if len(stackNames) == 0 {
		fmt.Printf("No stacks found in context %s\n", contextName)
		return nil
	}

	// Resolve all stacks and their dependencies
	resolved, err := resolver.ResolveStacks(ctx, contextName, stackNames)
	if err != nil {
		return fmt.Errorf("failed to resolve stack dependencies: %w", err)
	}

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

		err = deleteStackWithFeedback(ctx, stackToDelete, contextName)
		if err != nil {
			return err
		}
	}

	return nil
}

func init() {
	rootCmd.AddCommand(deleteCmd)
}
