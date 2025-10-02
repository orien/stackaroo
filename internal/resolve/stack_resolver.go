/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package resolve

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/orien/stackaroo/internal/aws"
	"github.com/orien/stackaroo/internal/config"
	"github.com/orien/stackaroo/internal/model"
)

// Resolver defines the interface for stack resolution operations
type Resolver interface {
	ResolveStack(ctx context.Context, context string, stackName string) (*model.Stack, error)
	GetDependencyOrder(context string, stackNames []string) ([]string, error)
}

// StackResolver resolves configuration into deployment-ready artifacts
type StackResolver struct {
	configProvider     config.ConfigProvider
	fileSystemResolver FileSystemResolver
	clientFactory      aws.ClientFactory
	templateProcessor  TemplateProcessor
}

// NewStackResolver creates a new stack resolver instance with the given config provider and client factory
func NewStackResolver(configProvider config.ConfigProvider, clientFactory aws.ClientFactory) *StackResolver {
	return &StackResolver{
		configProvider:     configProvider,
		fileSystemResolver: &DefaultFileSystemResolver{},
		clientFactory:      clientFactory,
		templateProcessor:  NewCfnTemplateProcessor(),
	}
}

// SetFileSystemResolver allows injecting a custom file system resolver (for testing)
func (r *StackResolver) SetFileSystemResolver(fileSystemResolver FileSystemResolver) {
	r.fileSystemResolver = fileSystemResolver
}

// SetTemplateProcessor allows injecting a custom template processor (for testing)
func (r *StackResolver) SetTemplateProcessor(templateProcessor TemplateProcessor) {
	r.templateProcessor = templateProcessor
}

// ResolveStack resolves a single stack configuration
func (r *StackResolver) ResolveStack(ctx context.Context, context string, stackName string) (*model.Stack, error) {
	// Load configuration
	cfg, err := r.configProvider.LoadConfig(ctx, context)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Get stack configuration
	stackConfig, err := r.configProvider.GetStack(stackName, context)
	if err != nil {
		return nil, fmt.Errorf("failed to get stack %s: %w", stackName, err)
	}

	// Read raw template content
	rawTemplate, err := r.fileSystemResolver.Resolve(stackConfig.Template)
	if err != nil {
		return nil, fmt.Errorf("failed to read template: %w", err)
	}

	// Process template with variables (parameters and context)
	templateVars := r.buildTemplateVariables(stackConfig, context)
	templateBody, err := r.templateProcessor.Process(rawTemplate, templateVars)
	if err != nil {
		return nil, fmt.Errorf("failed to process template: %w", err)
	}

	// Resolve parameters with new system, passing region for cross-region stack outputs
	parameters, err := r.resolveParameters(ctx, stackConfig.Parameters, cfg.Context.Region)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve parameters for stack %s: %w", stackName, err)
	}

	// Merge tags: global + context + stack (stack takes precedence)
	globalAndContextTags := r.mergeTags(cfg.Tags, cfg.Context.Tags)
	tags := r.mergeTags(globalAndContextTags, stackConfig.Tags)

	// Create context info from resolved configuration
	stackContext := &model.Context{
		Name:    cfg.Context.Name,
		Region:  cfg.Context.Region,
		Account: cfg.Context.Account,
	}

	return &model.Stack{
		Name:         stackConfig.Name,
		Context:      stackContext,
		TemplateBody: templateBody,
		Parameters:   parameters,
		Tags:         tags,
		Capabilities: stackConfig.Capabilities,
		Dependencies: stackConfig.Dependencies,
	}, nil
}

// GetDependencyOrder calculates the dependency order for stacks without resolving them
func (r *StackResolver) GetDependencyOrder(context string, stackNames []string) ([]string, error) {
	// Get stack configurations
	var stackConfigs []*config.StackConfig

	for _, stackName := range stackNames {
		stackConfig, err := r.configProvider.GetStack(stackName, context)
		if err != nil {
			return nil, fmt.Errorf("failed to get stack config %s: %w", stackName, err)
		}

		stackConfigs = append(stackConfigs, stackConfig)
	}

	// Calculate deployment order using topological sort
	// Build name to stack config map
	stackMap := make(map[string]*config.StackConfig)
	for _, stackConfig := range stackConfigs {
		stackMap[stackConfig.Name] = stackConfig
	}

	// Build dependency graph
	inDegree := make(map[string]int)
	adjList := make(map[string][]string)

	// Initialize
	for _, stackConfig := range stackConfigs {
		inDegree[stackConfig.Name] = 0
		adjList[stackConfig.Name] = []string{}
	}

	// Build graph
	for _, stackConfig := range stackConfigs {
		for _, dep := range stackConfig.Dependencies {
			if _, exists := stackMap[dep]; exists {
				adjList[dep] = append(adjList[dep], stackConfig.Name)
				inDegree[stackConfig.Name]++
			}
		}
	}

	// Topological sort using Kahn's algorithm
	var queue []string
	var result []string

	// Find all nodes with no incoming edges
	for name, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, name)
		}
	}

	// Sort queue for deterministic results
	sort.Strings(queue)

	for len(queue) > 0 {
		// Remove node from queue
		current := queue[0]
		queue = queue[1:]
		result = append(result, current)

		// For each neighbor, reduce in-degree
		neighbors := adjList[current]
		sort.Strings(neighbors) // For deterministic ordering
		for _, neighbor := range neighbors {
			inDegree[neighbor]--
			if inDegree[neighbor] == 0 {
				queue = append(queue, neighbor)
				sort.Strings(queue) // Keep queue sorted
			}
		}
	}

	// Check for cycles
	if len(result) != len(stackConfigs) {
		return nil, fmt.Errorf("circular dependency detected in stacks")
	}

	return result, nil
}

// resolveParameters resolves parameters from ParameterValue objects to final string values
func (r *StackResolver) resolveParameters(ctx context.Context, params map[string]*config.ParameterValue, contextRegion string) (map[string]string, error) {
	if params == nil {
		return nil, nil
	}

	result := make(map[string]string, len(params))

	for key, paramValue := range params {
		if paramValue == nil {
			continue
		}

		resolvedValue, err := r.resolveSingleParameter(ctx, paramValue, contextRegion)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve parameter '%s': %w", key, err)
		}
		result[key] = resolvedValue
	}

	return result, nil
}

// resolveStackOutput resolves a stack output reference to its actual value
func (r *StackResolver) resolveStackOutput(ctx context.Context, outputConfig map[string]string, contextRegion string) (string, error) {
	stackName, exists := outputConfig["stack_name"]
	if !exists {
		return "", fmt.Errorf("stack output resolver missing required 'stack_name'")
	}

	outputKey, exists := outputConfig["output_key"]
	if !exists {
		return "", fmt.Errorf("stack output resolver missing required 'output_key'")
	}

	// Determine which region to use for the stack lookup
	region := contextRegion
	if configRegion, exists := outputConfig["region"]; exists && configRegion != "" {
		region = configRegion
	}

	// Get region-specific CloudFormation operations
	cfnOps, err := r.clientFactory.GetCloudFormationOperations(ctx, region)
	if err != nil {
		return "", fmt.Errorf("failed to get CloudFormation operations for region %s: %w", region, err)
	}

	// Fetch stack information from CloudFormation
	stack, err := cfnOps.GetStack(ctx, stackName)
	if err != nil {
		return "", fmt.Errorf("failed to get stack '%s' in region %s: %w", stackName, region, err)
	}

	value, exists := stack.Outputs[outputKey]
	if !exists {
		return "", fmt.Errorf("stack '%s' does not have output '%s'", stackName, outputKey)
	}

	return value, nil
}

// resolveSingleParameter resolves a single parameter value to a string
func (r *StackResolver) resolveSingleParameter(ctx context.Context, paramValue *config.ParameterValue, contextRegion string) (string, error) {
	switch paramValue.ResolutionType {
	case "literal":
		if value, exists := paramValue.ResolutionConfig["value"]; exists {
			return value, nil
		} else {
			return "", fmt.Errorf("literal parameter missing 'value' config")
		}

	case "stack-output":
		return r.resolveStackOutput(ctx, paramValue.ResolutionConfig, contextRegion)

	case "list":
		return r.resolveParameterList(ctx, paramValue.ListItems, contextRegion)

	default:
		return "", fmt.Errorf("unsupported resolution type '%s'", paramValue.ResolutionType)
	}
}

// resolveParameterList resolves lists with mixed resolution types
func (r *StackResolver) resolveParameterList(ctx context.Context, listItems []*config.ParameterValue, contextRegion string) (string, error) {
	if len(listItems) == 0 {
		return "", nil // Empty list becomes empty string
	}

	var resolvedValues []string

	for i, item := range listItems {
		if item == nil {
			return "", fmt.Errorf("list item %d is nil", i)
		}

		var resolvedValue string
		var err error

		resolvedValue, err = r.resolveSingleParameter(ctx, item, contextRegion)
		if err != nil {
			return "", fmt.Errorf("failed to resolve list item %d: %w", i, err)
		}

		// Handle empty resolved values
		if resolvedValue != "" {
			resolvedValues = append(resolvedValues, resolvedValue)
		}
	}

	// Join all resolved values with commas (CloudFormation list format)
	return strings.Join(resolvedValues, ","), nil
}

// mergeTags merges tags with inheritance
func (r *StackResolver) mergeTags(globalTags, stackTags map[string]string) map[string]string {
	result := make(map[string]string)

	// Add global tags first
	for k, v := range globalTags {
		result[k] = v
	}

	// Add stack tags (overriding global)
	for k, v := range stackTags {
		result[k] = v
	}

	return result
}

// buildTemplateVariables creates the variable map for template processing
func (r *StackResolver) buildTemplateVariables(stackConfig *config.StackConfig, context string) map[string]interface{} {
	variables := make(map[string]interface{})

	// Add context information
	variables["Context"] = context
	variables["StackName"] = stackConfig.Name

	return variables
}
