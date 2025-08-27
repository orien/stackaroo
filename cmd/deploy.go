/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package cmd

import (
	"context"
	"fmt"

	"github.com/orien/stackaroo/internal/config/file"
	"github.com/orien/stackaroo/internal/deploy"
	"github.com/orien/stackaroo/internal/model"
	"github.com/orien/stackaroo/internal/resolve"
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
		var stackName string
		if len(args) > 1 {
			stackName = args[1]
		}
		ctx := context.Background()

		return deployWithConfig(ctx, stackName, contextName)
	},
}

// getDeployer returns the deployer instance, creating a default one if none is set
func getDeployer() deploy.Deployer {
	if deployer != nil {
		return deployer
	}

	// Create default deployer
	ctx := context.Background()
	d, err := deploy.NewDefaultDeployer(ctx)
	if err != nil {
		// This shouldn't happen in normal operation, but if it does,
		// we'll handle it in the command execution
		panic(fmt.Sprintf("failed to create default deployer: %v", err))
	}

	return d
}

// SetDeployer allows injection of a deployer (for testing)
func SetDeployer(d deploy.Deployer) {
	deployer = d
}

// deployWithConfig handles deployment using configuration file
func deployWithConfig(ctx context.Context, stackName, contextName string) error {
	// Create configuration provider and resolver
	provider := file.NewDefaultProvider()
	resolver := resolve.NewStackResolver(provider)

	// Determine which stacks to deploy
	var stackNames []string
	if stackName != "" {
		// Deploy single stack
		stackNames = []string{stackName}
	} else {
		// Deploy all stacks in context
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

	// Get or create deployer
	d := getDeployer()

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

		// Deploy the stack
		err = d.DeployStack(ctx, stackToDeploy)
		if err != nil {
			return fmt.Errorf("error deploying stack %s: %w", stackName, err)
		}

		fmt.Printf("Successfully deployed stack %s in context %s\n", stackName, contextName)
	}

	return nil
}

func init() {
	rootCmd.AddCommand(deployCmd)
}
