/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/

// Package file contains types and structures specific to file-based configuration providers.
// These types represent the raw YAML structure before context resolution and inheritance.
package file

// Config represents the raw YAML configuration file structure
// Used for parsing the stackaroo.yaml file before context resolution
type Config struct {
	Project  string              `yaml:"project"`
	Region   string              `yaml:"region"`
	Tags     map[string]string   `yaml:"tags"`
	Contexts map[string]*Context `yaml:"contexts"`
	Stacks   []*Stack            `yaml:"stacks"`
}

// Context represents context configuration as it appears in YAML
type Context struct {
	Account string            `yaml:"account"`
	Region  string            `yaml:"region"`
	Tags    map[string]string `yaml:"tags"`
}

// Stack represents stack configuration as it appears in YAML before context resolution
type Stack struct {
	Name         string                      `yaml:"name"`
	Template     string                      `yaml:"template"`
	Parameters   map[string]string           `yaml:"parameters"`
	Tags         map[string]string           `yaml:"tags"`
	Dependencies []string                    `yaml:"depends_on"`
	Capabilities []string                    `yaml:"capabilities"`
	Contexts     map[string]*ContextOverride `yaml:"contexts"`
}

// ContextOverride represents context-specific overrides for a stack
type ContextOverride struct {
	Parameters   map[string]string `yaml:"parameters"`
	Tags         map[string]string `yaml:"tags"`
	Dependencies []string          `yaml:"depends_on"`
	Capabilities []string          `yaml:"capabilities"`
}
