/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package cmd

import (
	"context"
	"fmt"

	"github.com/orien/stackaroo/internal/config/file"
	"github.com/orien/stackaroo/internal/diff"
	"github.com/orien/stackaroo/internal/model"
	"github.com/orien/stackaroo/internal/resolve"
	"github.com/spf13/cobra"
)

var (
	diffEnvironmentName string
	diffTemplateOnly    bool
	diffParametersOnly  bool
	diffTagsOnly        bool
	diffFormat          string
	// differ can be injected for testing
	differ Differ
)

// Differ defines the interface for stack diff operations
type Differ interface {
	DiffStack(ctx context.Context, resolvedStack *model.Stack, options diff.Options) (*diff.Result, error)
}

// diffCmd represents the diff command
var diffCmd = &cobra.Command{
	Use:   "diff [stack-name]",
	Short: "Show differences between deployed stack and local configuration",
	Long: `Compare the currently deployed CloudFormation stack with your local configuration.

This command shows what changes would be made if you ran 'stackaroo deploy' with
the current configuration. It compares:

• Template differences (deployed vs. local template)
• Parameter differences (current vs. resolved parameters)  
• Tag differences (current vs. resolved tags)
• Resource-level changes (when possible via AWS ChangeSets)

Examples:
  stackaroo diff vpc --environment dev              # Show all changes
  stackaroo diff vpc --environment prod --template  # Template diff only
  stackaroo diff vpc --environment dev --parameters # Parameter diff only
  stackaroo diff vpc --environment dev --format json # JSON output`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		stackName := args[0]
		ctx := context.Background()

		// Environment must be provided
		if diffEnvironmentName == "" {
			return fmt.Errorf("--environment must be specified")
		}

		// Validate format option
		if diffFormat != "text" && diffFormat != "json" {
			return fmt.Errorf("--format must be 'text' or 'json'")
		}

		return diffWithConfig(ctx, stackName, diffEnvironmentName)
	},
}

// getDiffer returns the differ instance, creating a default one if none is set
func getDiffer() Differ {
	if differ != nil {
		return differ
	}

	// Create default differ
	ctx := context.Background()
	d, err := diff.NewDefaultDiffer(ctx)
	if err != nil {
		// This shouldn't happen in normal operation, but if it does,
		// we'll handle it in the command execution
		panic(fmt.Sprintf("failed to create default differ: %v", err))
	}

	return d
}

// SetDiffer allows injection of a differ (for testing)
func SetDiffer(d Differ) {
	differ = d
}

// diffWithConfig handles diff using configuration file
func diffWithConfig(ctx context.Context, stackName, environmentName string) error {
	// Create configuration provider and resolver
	provider := file.NewDefaultProvider()
	resolver := resolve.NewStackResolver(provider)

	// Resolve stack and all its dependencies
	resolved, err := resolver.Resolve(ctx, environmentName, []string{stackName})
	if err != nil {
		return fmt.Errorf("failed to resolve stack dependencies: %w", err)
	}

	// Find the target stack in resolved stacks
	var targetStack *model.Stack
	for _, stack := range resolved.Stacks {
		if stack.Name == stackName {
			targetStack = stack
			break
		}
	}

	if targetStack == nil {
		return fmt.Errorf("stack %s not found in resolved configuration", stackName)
	}

	// Create diff options based on command flags
	options := diff.Options{
		TemplateOnly:   diffTemplateOnly,
		ParametersOnly: diffParametersOnly,
		TagsOnly:       diffTagsOnly,
		Format:         diffFormat,
	}

	// Get or create differ
	d := getDiffer()

	// Perform the diff
	result, err := d.DiffStack(ctx, targetStack, options)
	if err != nil {
		return fmt.Errorf("failed to diff stack %s: %w", stackName, err)
	}

	// Output the results
	fmt.Print(result.String())

	// Set exit code based on whether changes were found
	if result.HasChanges() {
		// Exit with code 1 if changes detected (similar to git diff)
		fmt.Printf("\nChanges detected for stack %s in environment %s\n", stackName, environmentName)
	} else {
		fmt.Printf("\nNo changes detected for stack %s in environment %s\n", stackName, environmentName)
	}

	return nil
}

func init() {
	rootCmd.AddCommand(diffCmd)

	// Required flags
	diffCmd.Flags().StringVar(&diffEnvironmentName, "environment", "", "deployment environment")
	if err := diffCmd.MarkFlagRequired("environment"); err != nil {
		panic(fmt.Sprintf("failed to mark environment flag as required: %v", err))
	}

	// Optional flags for filtering diff output
	diffCmd.Flags().BoolVar(&diffTemplateOnly, "template", false, "show only template differences")
	diffCmd.Flags().BoolVar(&diffParametersOnly, "parameters", false, "show only parameter differences")
	diffCmd.Flags().BoolVar(&diffTagsOnly, "tags", false, "show only tag differences")

	// Output format flag
	diffCmd.Flags().StringVar(&diffFormat, "format", "text", "output format: text, json")
}
