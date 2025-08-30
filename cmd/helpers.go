/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package cmd

import (
	"context"
	"fmt"

	"github.com/orien/stackaroo/internal/aws"
	"github.com/orien/stackaroo/internal/config/file"
	"github.com/orien/stackaroo/internal/resolve"
)

// createResolver creates a configuration provider and resolver
func createResolver() (*file.FileConfigProvider, *resolve.StackResolver) {
	provider := file.NewDefaultFileConfigProvider()
	resolver := resolve.NewStackResolver(provider)
	return provider, resolver
}

// getAWSClient creates a default AWS client with panic on error
func getAWSClient() aws.Client {
	ctx := context.Background()
	client, err := aws.NewDefaultClient(ctx, aws.Config{})
	if err != nil {
		// This shouldn't happen in normal operation, but if it does,
		// we'll handle it in the command execution
		panic(fmt.Sprintf("failed to create AWS client: %v", err))
	}
	return client
}

// getCloudFormationOperations creates CloudFormation operations with panic on error
func getCloudFormationOperations() aws.CloudFormationOperations {
	client := getAWSClient()
	return client.NewCloudFormationOperations()
}
