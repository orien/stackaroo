/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package cmd

import (
	"github.com/orien/stackaroo/internal/config/file"
	"github.com/orien/stackaroo/internal/resolve"
)

// createResolver creates a configuration provider and resolver
func createResolver() (*file.Provider, *resolve.StackResolver) {
	provider := file.NewDefaultProvider()
	resolver := resolve.NewStackResolver(provider)
	return provider, resolver
}
