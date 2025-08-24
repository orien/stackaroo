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
	"github.com/orien/stackaroo/internal/resolve"
	"github.com/spf13/cobra"
)

var (
	contextName string
	// deployer can be injected for testing
	deployer Deployer
)

// Deployer defines the interface for stack deployment operations
type Deployer interface {
	DeployStack(ctx context.Context, resolvedStack *resolve.ResolvedStack) error
}

// deployCmd represents the deploy command
var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Deploy CloudFormation stacks",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		stackName := args[0]
		ctx := context.Background()

		// Context must be provided
		if contextName == "" {
			return fmt.Errorf("--context must be specified")
		}

		return deployWithConfig(ctx, stackName, contextName)
	},
}

// getDeployer returns the deployer instance, creating a default one if none is set
func getDeployer() Deployer {
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
func SetDeployer(d Deployer) {
	deployer = d
}

// deployWithConfig handles deployment using configuration file
func deployWithConfig(ctx context.Context, stackName, contextName string) error {
	// Create configuration provider and resolver
	provider := file.NewDefaultProvider()
	resolver := resolve.NewStackResolver(provider)

	// Resolve stack and all its dependencies
	resolved, err := resolver.Resolve(ctx, contextName, []string{stackName})
	if err != nil {
		return fmt.Errorf("failed to resolve stack dependencies: %w", err)
	}

	// Get or create deployer
	d := getDeployer()

	// Deploy all stacks in dependency order
	for _, stackName := range resolved.DeploymentOrder {
		// Find the resolved stack
		var stackToDeploy *resolve.ResolvedStack
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
	deployCmd.Flags().StringVar(&contextName, "context", "", "deployment context (environment)")
	if err := deployCmd.MarkFlagRequired("context"); err != nil {
		panic(fmt.Sprintf("failed to mark context flag as required: %v", err))
	}
}
