/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package cmd

import (
	"context"
	"fmt"

	"github.com/orien/stackaroo/internal/config"
	"github.com/orien/stackaroo/internal/config/file"
	"github.com/orien/stackaroo/internal/deploy"
	"github.com/orien/stackaroo/internal/resolve"
	"github.com/spf13/cobra"
)

var (
	templateFile string
	contextName  string
	// deployer can be injected for testing
	deployer Deployer
)

// Deployer defines the interface for stack deployment operations
type Deployer interface {
	DeployStack(ctx context.Context, stackConfig *config.StackConfig) error
}

// deployCmd represents the deploy command
var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Deploy CloudFormation stacks",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		stackName := args[0]
		ctx := context.Background()

		// If context is provided, load configuration
		if contextName != "" {
			return deployWithConfig(ctx, stackName, contextName)
		}

		// Fall back to legacy template-based deployment
		if templateFile == "" {
			return fmt.Errorf("either --template or --context must be specified")
		}

		// Get or create deployer
		d := getDeployer()

		// Create a basic stack config for legacy deployment
		stackConfig := &config.StackConfig{
			Name:         stackName,
			Template:     templateFile,
			Parameters:   make(map[string]string),
			Tags:         make(map[string]string),
			Dependencies: []string{},
			Capabilities: []string{"CAPABILITY_IAM"},
		}
		err := d.DeployStack(ctx, stackConfig)
		if err != nil {
			return fmt.Errorf("error deploying stack %s: %w", stackName, err)
		}
		fmt.Printf("Successfully deployed stack %s\n", stackName)
		return nil
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
	// Create configuration provider and template reader
	provider := file.NewDefaultProvider()
	templateReader := &resolve.FileTemplateReader{}

	// Create resolver
	resolver := resolve.NewResolver(provider, templateReader)

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

		// Convert ResolvedStack to StackConfig for deployer
		stackConfig := &config.StackConfig{
			Name:         stackToDeploy.Name,
			Template:     stackToDeploy.TemplateBody, // Use resolved template content
			Parameters:   stackToDeploy.Parameters,
			Tags:         stackToDeploy.Tags,
			Dependencies: stackToDeploy.Dependencies,
			Capabilities: stackToDeploy.Capabilities,
		}

		// Deploy the stack
		err = d.DeployStack(ctx, stackConfig)
		if err != nil {
			return fmt.Errorf("error deploying stack %s: %w", stackName, err)
		}

		fmt.Printf("Successfully deployed stack %s in context %s\n", stackName, contextName)
	}

	return nil
}

func init() {
	rootCmd.AddCommand(deployCmd)
	deployCmd.Flags().StringVarP(&templateFile, "template", "t", "", "CloudFormation template file")
	deployCmd.Flags().StringVar(&contextName, "context", "", "deployment context (environment)")
}
