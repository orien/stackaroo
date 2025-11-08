/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package cmd

import (
	"context"

	"github.com/orien/stackaroo/internal/validate"
	"github.com/spf13/cobra"
)

var (
	// validator can be injected for testing
	validator validate.Validator
)

// validateCmd represents the validate command
var validateCmd = &cobra.Command{
	Use:   "validate <context> [stack-name]",
	Short: "Validate CloudFormation templates",
	Long: `Validate CloudFormation templates using the AWS CloudFormation API.

This command validates templates for syntax errors, valid resource types,
parameter definitions, and other AWS-specific requirements. It provides
fast feedback during development without requiring deployment.

If no stack name is provided, all stacks in the context will be validated.

Examples:
  stackaroo validate dev            # Validate all stacks in dev context
  stackaroo validate dev vpc        # Validate single stack
  stackaroo validate prod           # Validate all stacks in production`,
	Args: cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		contextName := args[0]
		ctx := context.Background()

		configFile, _ := cmd.Flags().GetString("config")
		v := getValidator(configFile)

		if len(args) > 1 {
			stackName := args[1]
			return v.ValidateSingleStack(ctx, stackName, contextName)
		}
		return v.ValidateAllStacks(ctx, contextName)
	},
}

// getValidator returns the validator instance, creating a default one if none is set
func getValidator(configFile string) validate.Validator {
	if validator != nil {
		return validator
	}

	provider, resolver := createResolver(configFile)
	clientFactory := getClientFactory()
	validator = validate.NewTemplateValidator(clientFactory, provider, resolver)
	return validator
}

// SetValidator allows injection of a validator (for testing)
func SetValidator(v validate.Validator) {
	validator = v
}

func init() {
	rootCmd.AddCommand(validateCmd)
}
