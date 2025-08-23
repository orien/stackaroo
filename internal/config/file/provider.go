/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package file

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
	"github.com/orien/stackaroo/internal/config"
)

// Provider implements config.ConfigProvider by reading from a YAML file
// Based on ADR 0010 (File provider configuration structure)
type Provider struct {
	filename string
	rawConfig *Config
}

// NewProvider creates a new file-based ConfigProvider for the given filename
func NewProvider(filename string) *Provider {
	return &Provider{
		filename: filename,
	}
}

// LoadConfig loads and resolves configuration for the specified context
func (fp *Provider) LoadConfig(ctx context.Context, context string) (*config.Config, error) {
	// Load raw config if not already loaded
	if err := fp.ensureLoaded(); err != nil {
		return nil, err
	}
	
	// Find the requested context
	rawContext, exists := fp.rawConfig.Contexts[context]
	if !exists {
		return nil, fmt.Errorf("context '%s' not found in configuration", context)
	}
	
	// Resolve context configuration with inheritance
	resolvedContext := fp.resolveContext(context, rawContext)
	
	// Resolve all stacks for this context
	resolvedStacks, err := fp.resolveStacks(context)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve stacks for context '%s': %w", context, err)
	}
	
	// Build final config
	cfg := &config.Config{
		Project: fp.rawConfig.Project,
		Region:  fp.rawConfig.Region, // Global default
		Tags:    fp.copyStringMap(fp.rawConfig.Tags),
		Context: resolvedContext,
		Stacks:  resolvedStacks,
	}
	
	return cfg, nil
}

// ListContexts returns all available contexts in the configuration
func (fp *Provider) ListContexts() ([]string, error) {
	if err := fp.ensureLoaded(); err != nil {
		return nil, err
	}
	
	contexts := make([]string, 0, len(fp.rawConfig.Contexts))
	for name := range fp.rawConfig.Contexts {
		contexts = append(contexts, name)
	}
	
	return contexts, nil
}

// GetStack returns stack configuration for a specific stack and context
func (fp *Provider) GetStack(stackName, context string) (*config.StackConfig, error) {
	if err := fp.ensureLoaded(); err != nil {
		return nil, err
	}
	
	// Find the stack in raw config
	var rawStack *Stack
	for _, stack := range fp.rawConfig.Stacks {
		if stack.Name == stackName {
			rawStack = stack
			break
		}
	}
	
	if rawStack == nil {
		return nil, fmt.Errorf("stack '%s' not found in configuration", stackName)
	}
	
	// Resolve the stack for the given context
	resolvedStack, err := fp.resolveStack(rawStack, context)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve stack '%s' for context '%s': %w", stackName, context, err)
	}
	
	return resolvedStack, nil
}

// Validate checks the configuration for consistency and errors
func (fp *Provider) Validate() error {
	if err := fp.ensureLoaded(); err != nil {
		return err
	}
	
	// Check that all stack context references exist
	for _, stack := range fp.rawConfig.Stacks {
		for contextName := range stack.Contexts {
			if _, exists := fp.rawConfig.Contexts[contextName]; !exists {
				return fmt.Errorf("stack '%s' references undefined context '%s'", stack.Name, contextName)
			}
		}
	}
	
	// Check that template files exist (basic validation)
	for _, stack := range fp.rawConfig.Stacks {
		if stack.Template != "" {
			// Make template path relative to config file directory
			templatePath := fp.resolveTemplatePath(stack.Template)
			if _, err := os.Stat(templatePath); err != nil && os.IsNotExist(err) {
				return fmt.Errorf("template file not found for stack '%s': %s", stack.Name, templatePath)
			}
		}
	}
	
	return nil
}

// ensureLoaded loads the raw configuration from file if not already loaded
func (fp *Provider) ensureLoaded() error {
	if fp.rawConfig != nil {
		return nil // Already loaded
	}
	
	// Read file
	data, err := os.ReadFile(fp.filename)
	if err != nil {
		return fmt.Errorf("failed to read config file '%s': %w", fp.filename, err)
	}
	
	// Parse YAML
	var rawConfig Config
	if err := yaml.Unmarshal(data, &rawConfig); err != nil {
		return fmt.Errorf("failed to parse YAML config file '%s': %w", fp.filename, err)
	}
	
	fp.rawConfig = &rawConfig
	return nil
}

// resolveContext creates a resolved context configuration with inheritance
func (fp *Provider) resolveContext(name string, rawContext *Context) *config.ContextConfig {
	resolved := &config.ContextConfig{
		Name:    name,
		Account: rawContext.Account,
		Region:  rawContext.Region,
		Tags:    fp.copyStringMap(rawContext.Tags),
	}
	
	// Apply global defaults if not overridden
	if resolved.Region == "" {
		resolved.Region = fp.rawConfig.Region
	}
	
	// Merge global tags with context tags (context takes precedence)
	if fp.rawConfig.Tags != nil {
		if resolved.Tags == nil {
			resolved.Tags = make(map[string]string)
		}
		for k, v := range fp.rawConfig.Tags {
			if _, exists := resolved.Tags[k]; !exists {
				resolved.Tags[k] = v
			}
		}
	}
	
	return resolved
}

// resolveStacks resolves all stacks for the given context
func (fp *Provider) resolveStacks(context string) ([]*config.StackConfig, error) {
	resolved := make([]*config.StackConfig, 0, len(fp.rawConfig.Stacks))
	
	for _, rawStack := range fp.rawConfig.Stacks {
		resolvedStack, err := fp.resolveStack(rawStack, context)
		if err != nil {
			return nil, err
		}
		resolved = append(resolved, resolvedStack)
	}
	
	return resolved, nil
}

// resolveStack resolves a single stack configuration for the given context
func (fp *Provider) resolveStack(rawStack *Stack, context string) (*config.StackConfig, error) {
	resolved := &config.StackConfig{
		Name:         rawStack.Name,
		Template:     rawStack.Template,
		Parameters:   fp.copyStringMap(rawStack.Parameters),
		Tags:         fp.copyStringMap(rawStack.Tags),
		Dependencies: fp.copyStringSlice(rawStack.Dependencies),
		Capabilities: fp.copyStringSlice(rawStack.Capabilities),
	}
	
	// Apply context-specific overrides if they exist
	if contextOverride, exists := rawStack.Contexts[context]; exists {
		// Merge parameters (context overrides take precedence)
		if contextOverride.Parameters != nil {
			if resolved.Parameters == nil {
				resolved.Parameters = make(map[string]string)
			}
			for k, v := range contextOverride.Parameters {
				resolved.Parameters[k] = v
			}
		}
		
		// Merge tags (context overrides take precedence)
		if contextOverride.Tags != nil {
			if resolved.Tags == nil {
				resolved.Tags = make(map[string]string)
			}
			for k, v := range contextOverride.Tags {
				resolved.Tags[k] = v
			}
		}
		
		// Override dependencies if specified
		if contextOverride.Dependencies != nil {
			resolved.Dependencies = fp.copyStringSlice(contextOverride.Dependencies)
		}
		
		// Override capabilities if specified
		if contextOverride.Capabilities != nil {
			resolved.Capabilities = fp.copyStringSlice(contextOverride.Capabilities)
		}
	}
	
	return resolved, nil
}

// resolveTemplatePath resolves template path relative to config file directory
func (fp *Provider) resolveTemplatePath(templatePath string) string {
	if filepath.IsAbs(templatePath) {
		return templatePath
	}
	
	configDir := filepath.Dir(fp.filename)
	return filepath.Join(configDir, templatePath)
}

// Helper methods for copying maps and slices to avoid shared references

func (fp *Provider) copyStringMap(source map[string]string) map[string]string {
	if source == nil {
		return nil
	}
	
	copy := make(map[string]string, len(source))
	for k, v := range source {
		copy[k] = v
	}
	return copy
}

func (fp *Provider) copyStringSlice(source []string) []string {
	if source == nil {
		return nil
	}
	
	copy := make([]string, len(source))
	for i, v := range source {
		copy[i] = v
	}
	return copy
}