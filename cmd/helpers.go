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
func createResolver(configFile string) (*file.FileConfigProvider, *resolve.StackResolver) {
	provider := file.NewFileConfigProvider(configFile)
	clientFactory := getClientFactory()
	resolver := resolve.NewStackResolver(provider, clientFactory)
	return provider, resolver
}

// Global factory instance (created once per command execution)
var clientFactory aws.ClientFactory

// getClientFactory creates or returns the shared AWS client factory
func getClientFactory() aws.ClientFactory {
	if clientFactory != nil {
		return clientFactory
	}

	ctx := context.Background()

	factory, err := aws.NewClientFactory(ctx)
	if err != nil {
		panic(fmt.Sprintf("failed to create AWS client factory: %v", err))
	}

	clientFactory = factory
	return clientFactory
}
