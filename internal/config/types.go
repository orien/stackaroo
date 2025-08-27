/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package config

import (
	"context"
)

// ConfigProvider defines the interface for loading and managing configuration
// Based on ADR 0008 (Configuration abstraction) and ADR 0009 (Context abstraction)
type ConfigProvider interface {
	// LoadConfig loads configuration for a specific context
	LoadConfig(ctx context.Context, context string) (*Config, error)

	// ListContexts returns all available contexts in the configuration
	ListContexts() ([]string, error)

	// GetStack returns stack configuration for a specific stack and context
	GetStack(stackName, context string) (*StackConfig, error)

	// ListStacks returns all available stack names for a specific context
	ListStacks(context string) ([]string, error)

	// Validate checks the configuration for consistency and errors
	Validate() error
}

// Config represents the resolved configuration for a specific context
// Based on ADR 0010 (File provider configuration structure)
type Config struct {
	Project string
	Region  string
	Tags    map[string]string
	Context *ContextConfig // Resolved context
	Stacks  []*StackConfig // Resolved stacks
}

// ContextConfig represents resolved context-specific configuration
type ContextConfig struct {
	Name    string
	Account string
	Region  string
	Tags    map[string]string
}

// StackConfig represents resolved stack configuration with context overrides applied
type StackConfig struct {
	Name         string
	Template     string // URI to template (file://, s3://, git://, etc.)
	Parameters   map[string]string
	Tags         map[string]string
	Dependencies []string
	Capabilities []string
}
