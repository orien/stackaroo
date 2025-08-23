/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/stackaroo/stackaroo/internal/deploy"
)

var (
	templateFile string
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
		
		// Get or create deployer
		d := getDeployer()
		
		ctx := context.Background()
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



func init() {
	rootCmd.AddCommand(deployCmd)
	deployCmd.Flags().StringVarP(&templateFile, "template", "t", "", "CloudFormation template file")
}