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
	environmentName string
	// deployer can be injected for testing
	deployer deploy.Deployer
)

// deployCmd represents the deploy command
var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Deploy CloudFormation stacks",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		stackName := args[0]
		ctx := context.Background()

		// Environment must be provided
		if environmentName == "" {
			return fmt.Errorf("--environment must be specified")
		}

		return deployWithConfig(ctx, stackName, environmentName)
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
func deployWithConfig(ctx context.Context, stackName, environmentName string) error {
	// Create configuration provider and resolver
	provider := file.NewDefaultProvider()
	resolver := resolve.NewStackResolver(provider)

	// Resolve stack and all its dependencies
	resolved, err := resolver.Resolve(ctx, environmentName, []string{stackName})
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

		fmt.Printf("Successfully deployed stack %s in environment %s\n", stackName, environmentName)
	}

	return nil
}

func init() {
	rootCmd.AddCommand(deployCmd)
	deployCmd.Flags().StringVar(&environmentName, "environment", "", "deployment environment")
	if err := deployCmd.MarkFlagRequired("environment"); err != nil {
		panic(fmt.Sprintf("failed to mark environment flag as required: %v", err))
	}
}
