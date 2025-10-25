/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/

// Package file contains types and structures specific to file-based configuration providers.
// These types represent the raw YAML structure before context resolution and inheritance.
package file

import (
	"fmt"

	"github.com/orien/stackaroo/internal/config"
	"gopkg.in/yaml.v3"
)

// Config represents the raw YAML configuration file structure
// Used for parsing the stackaroo.yaml file before context resolution
type Config struct {
	Project   string              `yaml:"project"`
	Region    string              `yaml:"region"`
	Tags      map[string]string   `yaml:"tags"`
	Templates *Templates          `yaml:"templates"`
	Contexts  map[string]*Context `yaml:"contexts"`
	Stacks    map[string]*Stack   `yaml:"stacks"`
}

// Templates represents global template configuration
type Templates struct {
	Directory string `yaml:"directory"`
}

// Context represents context configuration as it appears in YAML
type Context struct {
	Account string            `yaml:"account"`
	Region  string            `yaml:"region"`
	Tags    map[string]string `yaml:"tags"`
}

// Stack represents stack configuration as it appears in YAML before context resolution
type Stack struct {
	Template     string                         `yaml:"template"`
	Parameters   map[string]*yamlParameterValue `yaml:"parameters"`
	Tags         map[string]string              `yaml:"tags"`
	Dependencies []string                       `yaml:"depends_on"`
	Capabilities []string                       `yaml:"capabilities"`
	Contexts     map[string]*ContextOverride    `yaml:"contexts"`
}

// ContextOverride represents context-specific overrides for a stack
type ContextOverride struct {
	Parameters   map[string]*yamlParameterValue `yaml:"parameters"`
	Tags         map[string]string              `yaml:"tags"`
	Dependencies []string                       `yaml:"depends_on"`
	Capabilities []string                       `yaml:"capabilities"`
}

// yamlParameterValue represents either a literal value, complex resolution object, or list (YAML-specific)
type yamlParameterValue struct {
	// For literal values
	Literal        string
	IsLiteralValue bool // Tracks if this is a literal (needed for empty string literals)

	// For complex resolution
	Resolver *yamlParameterResolver

	// For list parameters - detected automatically from YAML array structure
	ListItems   []*yamlParameterValue
	IsListValue bool // Tracks if this is a list parameter
}

// yamlParameterResolver defines how to resolve a parameter dynamically (YAML-specific)
type yamlParameterResolver struct {
	Type   string                 `yaml:"type"`    // "literal", "output"
	Config map[string]interface{} `yaml:",inline"` // Type-specific configuration
}

// StackOutputConfig represents configuration for resolving stack output values
type StackOutputConfig struct {
	StackName string `yaml:"stack_name"`
	OutputKey string `yaml:"output_key"`
	Region    string `yaml:"region,omitempty"` // Optional, defaults to current region
}

// UnmarshalYAML implements custom YAML unmarshalling for yamlParameterValue
func (pv *yamlParameterValue) UnmarshalYAML(node *yaml.Node) error {
	switch node.Kind {
	case yaml.ScalarNode:
		// Handle literal string values
		pv.Literal = node.Value
		pv.IsLiteralValue = true
		return nil

	case yaml.MappingNode:
		// Handle complex resolver objects
		pv.Resolver = &yamlParameterResolver{}
		return node.Decode(pv.Resolver)

	case yaml.SequenceNode:
		// Handle array/list parameters
		pv.IsListValue = true
		pv.ListItems = make([]*yamlParameterValue, len(node.Content))

		for i, itemNode := range node.Content {
			pv.ListItems[i] = &yamlParameterValue{}
			if err := pv.ListItems[i].UnmarshalYAML(itemNode); err != nil {
				return fmt.Errorf("failed to parse list item %d: %w", i, err)
			}
		}
		return nil

	default:
		return fmt.Errorf("parameter value must be a string literal, resolver object, or array")
	}
}

// MarshalYAML implements custom YAML marshalling for yamlParameterValue
func (pv *yamlParameterValue) MarshalYAML() (interface{}, error) {
	if pv.IsLiteralValue {
		return pv.Literal, nil
	}

	if pv.IsListValue {
		// Return the list items directly as a YAML sequence
		return pv.ListItems, nil
	}

	if pv.Resolver != nil {
		return pv.Resolver, nil
	}

	return nil, fmt.Errorf("parameter value has no valid content")
}

// ToConfigParameterValue converts YAML parameter value to generic config parameter value
func (pv *yamlParameterValue) ToConfigParameterValue() *config.ParameterValue {
	if pv.IsLiteralValue {
		return &config.ParameterValue{
			ResolutionType: "literal",
			ResolutionConfig: map[string]string{
				"value": pv.Literal,
			},
		}
	}

	if pv.IsListValue {
		// Convert list items to config parameter values
		configListItems := make([]*config.ParameterValue, len(pv.ListItems))
		for i, item := range pv.ListItems {
			configListItems[i] = item.ToConfigParameterValue()
		}

		return &config.ParameterValue{
			ResolutionType:   "list",
			ResolutionConfig: make(map[string]string), // List metadata if needed
			ListItems:        configListItems,
		}
	}

	if pv.Resolver != nil {
		// Convert interface{} values to strings
		stringConfig := make(map[string]string)
		for key, value := range pv.Resolver.Config {
			if strValue, ok := value.(string); ok {
				stringConfig[key] = strValue
			} else {
				stringConfig[key] = fmt.Sprintf("%v", value)
			}
		}

		return &config.ParameterValue{
			ResolutionType:   pv.Resolver.Type,
			ResolutionConfig: stringConfig,
		}
	}

	return nil
}

// IsLiteral returns true if this parameter value is a literal string
func (pv *yamlParameterValue) IsLiteral() bool {
	return pv.IsLiteralValue
}

// IsResolver returns true if this parameter value uses a resolver
func (pv *yamlParameterValue) IsResolver() bool {
	return pv.Resolver != nil
}

// IsList returns true if this parameter value is a list
func (pv *yamlParameterValue) IsList() bool {
	return pv.IsListValue
}

// ConvertStringMap converts a map[string]string to map[string]*config.ParameterValue for backwards compatibility
func ConvertStringMap(stringMap map[string]string) map[string]*config.ParameterValue {
	if stringMap == nil {
		return nil
	}

	result := make(map[string]*config.ParameterValue, len(stringMap))
	for key, value := range stringMap {
		result[key] = &config.ParameterValue{
			ResolutionType: "literal",
			ResolutionConfig: map[string]string{
				"value": value,
			},
		}
	}
	return result
}
