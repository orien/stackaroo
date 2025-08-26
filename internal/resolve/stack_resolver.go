/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package resolve

import (
	"context"
	"fmt"
	"sort"

	"github.com/orien/stackaroo/internal/config"
	"github.com/orien/stackaroo/internal/model"
)

// StackResolver resolves configuration into deployment-ready artifacts
type StackResolver struct {
	configProvider     config.ConfigProvider
	fileSystemResolver FileSystemResolver
}

// NewStackResolver creates a new stack resolver instance with the given config provider
func NewStackResolver(configProvider config.ConfigProvider) *StackResolver {
	return &StackResolver{
		configProvider:     configProvider,
		fileSystemResolver: &DefaultFileSystemResolver{},
	}
}

// SetFileSystemResolver allows injecting a custom file system resolver (for testing)
func (r *StackResolver) SetFileSystemResolver(fileSystemResolver FileSystemResolver) {
	r.fileSystemResolver = fileSystemResolver
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

	// Read template
	templateBody, err := r.fileSystemResolver.Resolve(stackConfig.Template)
	if err != nil {
		return nil, fmt.Errorf("failed to read template: %w", err)
	}

	// Merge parameters and tags
	parameters := r.mergeParameters(stackConfig.Parameters)
	tags := r.mergeTags(cfg.Tags, stackConfig.Tags)

	return &model.Stack{
		Name:         stackConfig.Name,
		Environment:  context,
		TemplateBody: templateBody,
		Parameters:   parameters,
		Tags:         tags,
		Capabilities: stackConfig.Capabilities,
		Dependencies: stackConfig.Dependencies,
	}, nil
}

// Resolve resolves multiple stacks and calculates deployment order
func (r *StackResolver) Resolve(ctx context.Context, context string, stackNames []string) (*model.ResolvedStacks, error) {
	var stacks []*model.Stack

	// Resolve each stack
	for _, stackName := range stackNames {
		resolved, err := r.ResolveStack(ctx, context, stackName)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve stack %s: %w", stackName, err)
		}
		stacks = append(stacks, resolved)
	}

	// Calculate deployment order
	deploymentOrder, err := r.calculateDependencyOrder(stacks)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate dependency order: %w", err)
	}

	return &model.ResolvedStacks{
		Context:         context,
		Stacks:          stacks,
		DeploymentOrder: deploymentOrder,
	}, nil
}

// mergeParameters merges parameters with inheritance
func (r *StackResolver) mergeParameters(stackParams map[string]string) map[string]string {
	// Simple implementation - just return stack parameters for now
	result := make(map[string]string)
	for k, v := range stackParams {
		result[k] = v
	}
	return result
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

// calculateDependencyOrder calculates the deployment order based on dependencies
func (r *StackResolver) calculateDependencyOrder(stacks []*model.Stack) ([]string, error) {
	// Simple topological sort implementation
	// Build name to stack map
	stackMap := make(map[string]*model.Stack)
	for _, stack := range stacks {
		stackMap[stack.Name] = stack
	}

	// Build dependency graph
	inDegree := make(map[string]int)
	adjList := make(map[string][]string)

	// Initialize
	for _, stack := range stacks {
		inDegree[stack.Name] = 0
		adjList[stack.Name] = []string{}
	}

	// Build graph
	for _, stack := range stacks {
		for _, dep := range stack.Dependencies {
			if _, exists := stackMap[dep]; exists {
				adjList[dep] = append(adjList[dep], stack.Name)
				inDegree[stack.Name]++
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
	if len(result) != len(stacks) {
		return nil, fmt.Errorf("circular dependency detected in stacks")
	}

	return result, nil
}
