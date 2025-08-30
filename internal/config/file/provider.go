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

	"github.com/orien/stackaroo/internal/config"
	"gopkg.in/yaml.v3"
)

// FileConfigProvider implements config.ConfigProvider by reading from a YAML file
// Based on ADR 0010 (File provider configuration structure)
type FileConfigProvider struct {
	filename  string
	rawConfig *Config
}

// NewFileConfigProvider creates a new file-based ConfigProvider for the given filename
func NewFileConfigProvider(filename string) *FileConfigProvider {
	return &FileConfigProvider{
		filename: filename,
	}
}

// NewDefaultProvider creates a new file-based ConfigProvider using the default config filename
func NewDefaultProvider() *FileConfigProvider {
	return NewFileConfigProvider("stackaroo.yaml")
}

// LoadConfig loads and resolves configuration for the specified context
func (fp *FileConfigProvider) LoadConfig(ctx context.Context, context string) (*config.Config, error) {
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
	stacks, err := fp.resolveStacks(context)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve stacks for context '%s': %w", context, err)
	}

	// Build final config
	cfg := &config.Config{
		Project: fp.rawConfig.Project,
		Region:  fp.rawConfig.Region, // Global default
		Tags:    fp.copyStringMap(fp.rawConfig.Tags),
		Context: resolvedContext,
		Stacks:  stacks,
	}

	return cfg, nil
}

// ListContexts returns all available contexts in the configuration
func (fp *FileConfigProvider) ListContexts() ([]string, error) {
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
func (fp *FileConfigProvider) GetStack(stackName, context string) (*config.StackConfig, error) {
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
	stack, err := fp.resolveStack(rawStack, context)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve stack '%s' for context '%s': %w", stackName, context, err)
	}

	return stack, nil
}

// ListStacks returns all available stack names for a specific context
func (fp *FileConfigProvider) ListStacks(context string) ([]string, error) {
	if err := fp.ensureLoaded(); err != nil {
		return nil, err
	}

	// Check if the context exists
	if _, exists := fp.rawConfig.Contexts[context]; !exists {
		return nil, fmt.Errorf("context '%s' not found in configuration", context)
	}

	// Extract stack names
	stackNames := make([]string, 0, len(fp.rawConfig.Stacks))
	for _, stack := range fp.rawConfig.Stacks {
		stackNames = append(stackNames, stack.Name)
	}

	return stackNames, nil
}

// Validate checks the configuration for consistency and errors
func (fp *FileConfigProvider) Validate() error {
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

	// Check that global template directory exists if specified
	if fp.rawConfig.Templates != nil && fp.rawConfig.Templates.Directory != "" {
		templateDir := fp.rawConfig.Templates.Directory
		configDir := filepath.Dir(fp.filename)
		if !filepath.IsAbs(templateDir) {
			templateDir = filepath.Join(configDir, templateDir)
		}
		if _, err := os.Stat(templateDir); err != nil && os.IsNotExist(err) {
			return fmt.Errorf("global template directory not found: %s", templateDir)
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
func (fp *FileConfigProvider) ensureLoaded() error {
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
func (fp *FileConfigProvider) resolveContext(name string, rawContext *Context) *config.ContextConfig {
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
func (fp *FileConfigProvider) resolveStacks(context string) ([]*config.StackConfig, error) {
	resolved := make([]*config.StackConfig, 0, len(fp.rawConfig.Stacks))

	for _, rawStack := range fp.rawConfig.Stacks {
		stack, err := fp.resolveStack(rawStack, context)
		if err != nil {
			return nil, err
		}
		resolved = append(resolved, stack)
	}

	return resolved, nil
}

// resolveStack resolves a single stack configuration for the given context
func (fp *FileConfigProvider) resolveStack(rawStack *Stack, context string) (*config.StackConfig, error) {
	resolved := &config.StackConfig{
		Name:         rawStack.Name,
		Template:     fp.resolveTemplateURI(rawStack.Template),
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

// resolveTemplatePath resolves template path relative to global template directory or config file directory
func (fp *FileConfigProvider) resolveTemplatePath(templatePath string) string {
	if filepath.IsAbs(templatePath) {
		return templatePath
	}

	configDir := filepath.Dir(fp.filename)

	// Use global template directory if specified
	if fp.rawConfig != nil && fp.rawConfig.Templates != nil && fp.rawConfig.Templates.Directory != "" {
		templateDir := fp.rawConfig.Templates.Directory
		if !filepath.IsAbs(templateDir) {
			templateDir = filepath.Join(configDir, templateDir)
		}
		return filepath.Join(templateDir, templatePath)
	}

	// Fall back to config directory (current behaviour)
	return filepath.Join(configDir, templatePath)
}

// resolveTemplateURI resolves template path to file:// URI relative to global template directory or config file directory
func (fp *FileConfigProvider) resolveTemplateURI(templatePath string) string {
	resolvedPath := fp.resolveTemplatePath(templatePath)
	return "file://" + resolvedPath
}

// Helper methods for copying maps and slices to avoid shared references

func (fp *FileConfigProvider) copyStringMap(source map[string]string) map[string]string {
	if source == nil {
		return nil
	}

	copy := make(map[string]string, len(source))
	for k, v := range source {
		copy[k] = v
	}
	return copy
}

func (fp *FileConfigProvider) copyStringSlice(source []string) []string {
	if source == nil {
		return nil
	}

	copy := make([]string, len(source))
	for i, v := range source {
		copy[i] = v
	}
	return copy
}
