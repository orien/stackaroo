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
	
	// Validate checks the configuration for consistency and errors
	Validate() error
}

// Config represents the resolved configuration for a specific context
// Based on ADR 0010 (File provider configuration structure)
type Config struct {
	Project string            `yaml:"project"`
	Region  string            `yaml:"region"`
	Tags    map[string]string `yaml:"tags"`
	Context *ContextConfig    `yaml:"-"` // Resolved context, not from YAML
	Stacks  []*StackConfig    `yaml:"-"` // Resolved stacks, not from YAML
}

// ContextConfig represents resolved context-specific configuration
type ContextConfig struct {
	Name    string            `yaml:"-"`
	Account string            `yaml:"account"`
	Region  string            `yaml:"region"`
	Tags    map[string]string `yaml:"tags"`
}

// StackConfig represents resolved stack configuration with context overrides applied
type StackConfig struct {
	Name         string            `yaml:"name"`
	Template     string            `yaml:"template"`
	Parameters   map[string]string `yaml:"parameters"`
	Tags         map[string]string `yaml:"tags"`
	Dependencies []string          `yaml:"depends_on"`
	Capabilities []string          `yaml:"capabilities"`
}

