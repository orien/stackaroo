/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package cmd

import (
	"context"
	"fmt"

	"github.com/orien/stackaroo/internal/deploy"
	"github.com/orien/stackaroo/internal/model"

	"github.com/spf13/cobra"
)

var (
	// deployer can be injected for testing
	deployer deploy.Deployer
)

// deployCmd represents the deploy command
var deployCmd = &cobra.Command{
	Use:   "deploy <context> [stack-name]",
	Short: "Deploy CloudFormation stacks",
	Long: `Deploy CloudFormation stacks with integrated change preview and confirmation.

This command shows you exactly what changes will be made and prompts for
confirmation before applying them to your infrastructure. For existing stacks,
it uses AWS CloudFormation ChangeSets to provide accurate previews including:

• Template changes (resources added, modified, or removed)
• Parameter changes (current vs new values)
• Tag changes (added, modified, or removed tags)
• Resource-level impact analysis with replacement warnings

After displaying the changes, you will be prompted to confirm before the
deployment proceeds. For new stacks, the command prompts for confirmation
before proceeding with stack creation.

If no stack name is provided, all stacks in the context will be deployed in
dependency order.

Examples:
  stackaroo deploy dev            # Deploy all stacks with confirmation prompts
  stackaroo deploy dev vpc        # Deploy single stack with confirmation prompt
  stackaroo deploy prod app       # Deploy stack after confirming changes

The preview shows the same detailed diff information as 'stackaroo diff' and
waits for your confirmation before applying the changes.`,
	Args: cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		contextName := args[0]
		ctx := context.Background()

		if len(args) > 1 {
			stackName := args[1]
			return deploySingleStack(ctx, stackName, contextName)
		}
		return deployAllStacks(ctx, contextName)
	},
}

// getDeployer returns the deployer instance, creating a default one if none is set
func getDeployer() deploy.Deployer {
	if deployer != nil {
		return deployer
	}

	return deploy.NewStackDeployer(createCloudFormationOperations())
}

// SetDeployer allows injection of a deployer (for testing)
func SetDeployer(d deploy.Deployer) {
	deployer = d
}

// deployStackWithFeedback deploys a stack and provides feedback
func deployStackWithFeedback(ctx context.Context, stack *model.Stack, contextName string) error {
	d := getDeployer()

	err := d.DeployStack(ctx, stack)
	if err != nil {
		return fmt.Errorf("error deploying stack %s: %w", stack.Name, err)
	}

	fmt.Printf("Successfully deployed stack %s in context %s\n", stack.Name, contextName)
	return nil
}

// deploySingleStack handles deployment of a single stack
func deploySingleStack(ctx context.Context, stackName, contextName string) error {
	_, resolver := createResolver()

	// Resolve single stack
	stack, err := resolver.ResolveStack(ctx, contextName, stackName)
	if err != nil {
		return fmt.Errorf("failed to resolve stack dependencies: %w", err)
	}

	return deployStackWithFeedback(ctx, stack, contextName)
}

// deployAllStacks handles deployment of all stacks in a context using configuration file
func deployAllStacks(ctx context.Context, contextName string) error {
	provider, resolver := createResolver()

	// Get list of stacks to deploy
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

	// Deploy all stacks in dependency order
	for _, stackName := range resolved.DeploymentOrder {
		// Find the resolved stack
		var stackToDeploy *model.Stack
		for _, stack := range resolved.Stacks {
			if stack.Name == stackName {
				stackToDeploy = stack
				break
			}
		}

		if stackToDeploy == nil {
			return fmt.Errorf("resolved stack %s not found", stackName)
		}

		err = deployStackWithFeedback(ctx, stackToDeploy, contextName)
		if err != nil {
			return err
		}
	}

	return nil
}

func init() {
	rootCmd.AddCommand(deployCmd)
}
