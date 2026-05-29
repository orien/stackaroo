/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package file

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"codeberg.org/orien/stackaroo/internal/config"
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
	rawStack, exists := fp.rawConfig.Stacks[stackName]
	if !exists {
		return nil, fmt.Errorf("stack '%s' not found in configuration", stackName)
	}

	// Resolve the stack for the given context
	stack, err := fp.resolveStack(stackName, rawStack, context)
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
	for stackName := range fp.rawConfig.Stacks {
		stackNames = append(stackNames, stackName)
	}

	return stackNames, nil
}

// Validate checks the configuration for consistency and errors
func (fp *FileConfigProvider) Validate() error {
	if err := fp.ensureLoaded(); err != nil {
		return err
	}

	// Check that all stack context references exist
	for stackName, stack := range fp.rawConfig.Stacks {
		for contextName := range stack.Contexts {
			if _, exists := fp.rawConfig.Contexts[contextName]; !exists {
				return fmt.Errorf("stack '%s' references undefined context '%s'", stackName, contextName)
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
	for stackName, stack := range fp.rawConfig.Stacks {
		if stack.Template != "" {
			templatePath, err := fp.resolveTemplatePath(stack.Template)
			if err != nil {
				return fmt.Errorf("invalid template path for stack '%s': %w", stackName, err)
			}
			if _, err := os.Stat(templatePath); err != nil && os.IsNotExist(err) {
				return fmt.Errorf("template file not found for stack '%s': %s", stackName, templatePath)
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

	for stackName, rawStack := range fp.rawConfig.Stacks {
		stack, err := fp.resolveStack(stackName, rawStack, context)
		if err != nil {
			return nil, err
		}
		resolved = append(resolved, stack)
	}

	return resolved, nil
}

// resolveStack resolves a single stack configuration for the given context
func (fp *FileConfigProvider) resolveStack(stackName string, rawStack *Stack, context string) (*config.StackConfig, error) {
	// Convert parameters to string map (only literal values supported)
	parameters, err := fp.convertParameters(rawStack.Parameters)
	if err != nil {
		return nil, fmt.Errorf("failed to convert parameters for stack '%s': %w", stackName, err)
	}

	templateURI, err := fp.resolveTemplateURI(rawStack.Template)
	if err != nil {
		return nil, fmt.Errorf("invalid template path for stack '%s': %w", stackName, err)
	}

	resolved := &config.StackConfig{
		Name:         stackName,
		Template:     templateURI,
		Parameters:   parameters,
		Tags:         fp.copyStringMap(rawStack.Tags),
		Dependencies: fp.copyStringSlice(rawStack.Dependencies),
		Capabilities: fp.copyStringSlice(rawStack.Capabilities),
	}

	// Apply context-specific overrides if they exist
	if contextOverride, exists := rawStack.Contexts[context]; exists {
		// Merge parameters (context overrides take precedence)
		if contextOverride.Parameters != nil {
			contextParams, err := fp.convertParameters(contextOverride.Parameters)
			if err != nil {
				return nil, fmt.Errorf("failed to convert context parameters for stack '%s': %w", stackName, err)
			}

			if resolved.Parameters == nil {
				resolved.Parameters = make(map[string]*config.ParameterValue)
			}
			for k, v := range contextParams {
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

// resolveTemplatePath resolves a relative template path against the allowed root
// (templates.directory if set, otherwise the config file's directory).
// Absolute paths and traversal outside the root are rejected.
func (fp *FileConfigProvider) resolveTemplatePath(templatePath string) (string, error) {
	if filepath.IsAbs(templatePath) {
		return "", fmt.Errorf("template path must be relative: %s", templatePath)
	}

	// Ensure configDir is absolute so candidate paths are always absolute.
	configDir, err := filepath.Abs(filepath.Dir(fp.filename))
	if err != nil {
		return "", fmt.Errorf("cannot resolve config directory: %w", err)
	}
	var root string
	if fp.rawConfig != nil && fp.rawConfig.Templates != nil && fp.rawConfig.Templates.Directory != "" {
		templateDir := fp.rawConfig.Templates.Directory
		if !filepath.IsAbs(templateDir) {
			templateDir = filepath.Join(configDir, templateDir)
		}
		root = filepath.Clean(templateDir)

		// Validate that templates.directory is confined to configDir — it sets the
		// root that all template path confinement is anchored to, so it must be
		// within the config tree.
		rootRel, rootRelErr := filepath.Rel(configDir, root)
		if rootRelErr != nil || rootRel == ".." || strings.HasPrefix(rootRel, ".."+string(filepath.Separator)) {
			return "", fmt.Errorf("templates directory escapes config directory: %s", fp.rawConfig.Templates.Directory)
		}
		if realRoot, sErr := filepath.EvalSymlinks(root); sErr == nil {
			realConfigDir := configDir
			if r, sErr := filepath.EvalSymlinks(configDir); sErr == nil {
				realConfigDir = r
			}
			rootRel, rootRelErr := filepath.Rel(realConfigDir, realRoot)
			if rootRelErr != nil || rootRel == ".." || strings.HasPrefix(rootRel, ".."+string(filepath.Separator)) {
				return "", fmt.Errorf("templates directory escapes config directory via symlink: %s", fp.rawConfig.Templates.Directory)
			}
		}
	} else {
		root = filepath.Clean(configDir)
	}

	candidate := filepath.Clean(filepath.Join(root, templatePath))

	rel, err := filepath.Rel(root, candidate)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("template path escapes allowed directory: %s", templatePath)
	}

	// If the file already exists, resolve symlinks and re-verify confinement.
	if real, err := filepath.EvalSymlinks(candidate); err == nil {
		realRoot := root
		if r, err := filepath.EvalSymlinks(root); err == nil {
			realRoot = r
		}
		rel, err := filepath.Rel(realRoot, real)
		if err != nil || strings.HasPrefix(rel, "..") {
			return "", fmt.Errorf("template path escapes allowed directory via symlink: %s", templatePath)
		}
		return real, nil
	}

	return candidate, nil
}

// resolveTemplateURI resolves template path to file:// URI relative to the allowed root.
func (fp *FileConfigProvider) resolveTemplateURI(templatePath string) (string, error) {
	resolvedPath, err := fp.resolveTemplatePath(templatePath)
	if err != nil {
		return "", err
	}
	return (&url.URL{Scheme: "file", Path: resolvedPath}).String(), nil
}

// Helper methods for copying maps and slices to avoid shared references

func (fp *FileConfigProvider) copyStringMap(source map[string]string) map[string]string {
	if source == nil {
		return nil
	}

	result := make(map[string]string, len(source))
	for k, v := range source {
		result[k] = v
	}
	return result
}

// convertParameters converts yamlParameterValue map to config.ParameterValue map
func (fp *FileConfigProvider) convertParameters(params map[string]*yamlParameterValue) (map[string]*config.ParameterValue, error) {
	if params == nil {
		return nil, nil
	}

	result := make(map[string]*config.ParameterValue, len(params))

	for key, paramValue := range params {
		if paramValue == nil {
			continue
		}

		// Convert YAML parameter to generic config parameter
		configParam := paramValue.ToConfigParameterValue()
		if configParam == nil {
			return nil, fmt.Errorf("failed to convert parameter '%s' to config parameter value", key)
		}

		result[key] = configParam
	}

	return result, nil
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
