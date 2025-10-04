/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/orien/stackaroo/internal/deploy"
	"github.com/spf13/cobra"
)

var (
	deployPlain bool

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

		// Set STACKAROO_PLAIN environment variable if --plain flag is set
		if deployPlain {
			if err := os.Setenv("STACKAROO_PLAIN", "1"); err != nil {
				return fmt.Errorf("failed to set STACKAROO_PLAIN environment variable: %w", err)
			}
		}

		configFile, _ := cmd.Flags().GetString("config")
		d := getDeployer(configFile)

		if len(args) > 1 {
			stackName := args[1]
			return d.DeploySingleStack(ctx, stackName, contextName)
		}
		return d.DeployAllStacks(ctx, contextName)
	},
}

// getDeployer returns the deployer instance, creating a default one if none is set
func getDeployer(configFile string) deploy.Deployer {
	if deployer != nil {
		return deployer
	}

	provider, resolver := createResolver(configFile)
	clientFactory := getClientFactory()
	deployer = deploy.NewStackDeployer(clientFactory, provider, resolver)
	return deployer
}

// SetDeployer allows injection of a deployer (for testing)
func SetDeployer(d deploy.Deployer) {
	deployer = d
}

func init() {
	rootCmd.AddCommand(deployCmd)

	deployCmd.Flags().BoolVar(&deployPlain, "plain", false, "use plain text output instead of interactive viewer")
}
