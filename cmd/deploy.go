/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	templateFile string
)

// deployCmd represents the deploy command
var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Deploy CloudFormation stacks",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		stackName := args[0]
		err := DeployStack(stackName, templateFile)
		if err != nil {
			fmt.Printf("Error deploying stack %s: %v\n", stackName, err)
			os.Exit(1)
		}
		fmt.Printf("Successfully deployed stack %s\n", stackName)
	},
}

// GetTemplateFile returns the current template file path
func GetTemplateFile() string {
	return templateFile
}

// ReadTemplateFile reads the content of a template file
func ReadTemplateFile(filename string) (string, error) {
	content, err := os.ReadFile(filename)
	if err != nil {
		return "", fmt.Errorf("failed to read template file %s: %w", filename, err)
	}
	return string(content), nil
}

// DeployStack deploys a CloudFormation stack
func DeployStack(stackName, templateFile string) error {
	// TODO: Implement actual CloudFormation deployment
	return nil
}



func init() {
	rootCmd.AddCommand(deployCmd)
	deployCmd.Flags().StringVarP(&templateFile, "template", "t", "", "CloudFormation template file")
}