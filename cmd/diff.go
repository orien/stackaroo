/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package cmd

import (
	"context"
	"fmt"

	"github.com/orien/stackaroo/internal/diff"
	"github.com/spf13/cobra"
)

var (
	diffTemplateOnly   bool
	diffParametersOnly bool
	diffTagsOnly       bool

	// differ can be injected for testing
	differ diff.Differ
)

// diffCmd represents the diff command
var diffCmd = &cobra.Command{
	Use:   "diff <context> <stack-name>",
	Short: "Show differences between deployed stack and local configuration",
	Long: `Compare the currently deployed CloudFormation stack with your local configuration.

This command shows what changes would be made if you ran 'stackaroo deploy' with
the current configuration. It compares:

• Template differences (deployed vs. local template)
• Parameter differences (current vs. resolved parameters)
• Tag differences (current vs. resolved tags)
• Resource-level changes (when possible via AWS ChangeSets)

Examples:
  stackaroo diff dev vpc                        # Show all changes
  stackaroo diff prod vpc --template            # Template diff only
  stackaroo diff dev vpc --parameters           # Parameter diff only`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		contextName := args[0]
		stackName := args[1]
		ctx := context.Background()

		configFile, _ := cmd.Flags().GetString("config")

		return diffSingleStack(ctx, stackName, contextName, configFile)
	},
}

// getDiffer returns the differ instance, creating a default one if none is set
func getDiffer() diff.Differ {
	if differ != nil {
		return differ
	}

	clientFactory := getClientFactory()
	differ = diff.NewStackDiffer(clientFactory)
	return differ
}

// SetDiffer allows injection of a differ (for testing)
func SetDiffer(d diff.Differ) {
	differ = d
}

// diffSingleStack handles diff using configuration file
func diffSingleStack(ctx context.Context, stackName, contextName, configFile string) error {
	_, resolver := createResolver(configFile)

	// Resolve the target stack
	targetStack, err := resolver.ResolveStack(ctx, contextName, stackName)
	if err != nil {
		return fmt.Errorf("failed to resolve stack %s: %w", stackName, err)
	}

	// Create diff options based on command flags
	options := diff.Options{
		TemplateOnly:   diffTemplateOnly,
		ParametersOnly: diffParametersOnly,
		TagsOnly:       diffTagsOnly,
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
		fmt.Printf("\nChanges detected for stack %s in context %s\n", stackName, contextName)
	} else {
		fmt.Printf("\nNo changes detected for stack %s in context %s\n", stackName, contextName)
	}

	return nil
}

func init() {
	rootCmd.AddCommand(diffCmd)

	// Optional flags for filtering diff output
	diffCmd.Flags().BoolVar(&diffTemplateOnly, "template", false, "show only template differences")
	diffCmd.Flags().BoolVar(&diffParametersOnly, "parameters", false, "show only parameter differences")
	diffCmd.Flags().BoolVar(&diffTagsOnly, "tags", false, "show only tag differences")

}
