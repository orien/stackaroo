/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package cmd

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/orien/stackaroo/internal/deploy"
	"github.com/orien/stackaroo/internal/config/file"
)

var (
	templateFile string
	contextName  string
	// deployer can be injected for testing
	deployer Deployer
)

// Deployer defines the interface for stack deployment operations
type Deployer interface {
	DeployStack(ctx context.Context, stackName, templateFile string) error
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
		
		err := d.DeployStack(ctx, stackName, templateFile)
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
	// Load configuration from default file
	provider := file.NewProvider("stackaroo.yaml")
	
	// Get stack configuration for the specified context
	stackConfig, err := provider.GetStack(stackName, contextName)
	if err != nil {
		return fmt.Errorf("failed to load stack configuration: %w", err)
	}
	
	// Resolve template path relative to config file directory
	templatePath := stackConfig.Template
	if !filepath.IsAbs(templatePath) {
		templatePath = filepath.Join(filepath.Dir("stackaroo.yaml"), templatePath)
	}
	
	// Get or create deployer
	d := getDeployer()
	
	// Deploy using the resolved template path
	err = d.DeployStack(ctx, stackName, templatePath)
	if err != nil {
		return fmt.Errorf("error deploying stack %s: %w", stackName, err)
	}
	
	fmt.Printf("Successfully deployed stack %s in context %s\n", stackName, contextName)
	return nil
}

func init() {
	rootCmd.AddCommand(deployCmd)
	deployCmd.Flags().StringVarP(&templateFile, "template", "t", "", "CloudFormation template file")
	deployCmd.Flags().StringVar(&contextName, "context", "", "deployment context (environment)")
}
