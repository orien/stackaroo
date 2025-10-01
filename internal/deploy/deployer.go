/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package deploy

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/orien/stackaroo/internal/aws"
	"github.com/orien/stackaroo/internal/config"
	"github.com/orien/stackaroo/internal/diff"
	"github.com/orien/stackaroo/internal/model"
	"github.com/orien/stackaroo/internal/prompt"
	"github.com/orien/stackaroo/internal/resolve"
)

// CancellationError indicates that a stack operation was cancelled by the user
type CancellationError struct {
	StackName string
}

func (e CancellationError) Error() string {
	return fmt.Sprintf("deployment of stack %s was cancelled by user", e.StackName)
}

// Deployer defines the interface for stack deployment operations
type Deployer interface {
	DeployStack(ctx context.Context, stack *model.Stack) error
	DeploySingleStack(ctx context.Context, stackName, contextName string) error
	DeployAllStacks(ctx context.Context, contextName string) error
	ValidateTemplate(ctx context.Context, templateFile string) error
}

// StackDeployer implements Deployer using AWS CloudFormation
type StackDeployer struct {
	clientFactory aws.ClientFactory
	provider      config.ConfigProvider
	resolver      resolve.Resolver
}

// NewStackDeployer creates a new StackDeployer
func NewStackDeployer(clientFactory aws.ClientFactory, provider config.ConfigProvider, resolver resolve.Resolver) *StackDeployer {
	return &StackDeployer{
		clientFactory: clientFactory,
		provider:      provider,
		resolver:      resolver,
	}
}

// DeployStack deploys a CloudFormation stack using changesets for preview and deployment
func (d *StackDeployer) DeployStack(ctx context.Context, stack *model.Stack) error {
	// Get region-specific CloudFormation operations
	cfnOps, err := d.clientFactory.GetCloudFormationOperations(ctx, stack.Context.Region)
	if err != nil {
		return fmt.Errorf("failed to get CloudFormation operations for region %s: %w", stack.Context.Region, err)
	}

	// Check if stack exists to determine deployment approach
	exists, err := cfnOps.StackExists(ctx, stack.Name)
	if err != nil {
		return fmt.Errorf("failed to check if stack exists: %w", err)
	}

	if !exists {
		// For new stacks, use direct creation (changesets are less useful)
		return d.deployNewStack(ctx, stack, cfnOps)
	}

	// For existing stacks, use changeset approach for preview + deployment
	return d.deployWithChangeSet(ctx, stack, cfnOps)
}

// deployNewStack handles deployment of new stacks using direct creation
func (d *StackDeployer) deployNewStack(ctx context.Context, stack *model.Stack, cfnOps aws.CloudFormationOperations) error {
	fmt.Printf("=== Creating new stack %s ===\n", stack.Name)

	// Show what will be created
	fmt.Printf("This will create a new CloudFormation stack with:\n")
	fmt.Printf("- Name: %s\n", stack.Name)
	if len(stack.Parameters) > 0 {
		fmt.Printf("- Parameters: %d\n", len(stack.Parameters))
	}
	if len(stack.Tags) > 0 {
		fmt.Printf("- Tags: %d\n", len(stack.Tags))
	}
	fmt.Println()

	// Prompt for confirmation before creating new resources
	message := fmt.Sprintf("Do you want to apply these changes to stack %s?", stack.Name)
	confirmed, err := prompt.Confirm(message)
	if err != nil {
		return fmt.Errorf("failed to get user confirmation: %w", err)
	}
	if !confirmed {
		fmt.Printf("Stack creation cancelled for %s\n", stack.Name)
		return CancellationError{StackName: stack.Name}
	}

	// Convert parameters to AWS format
	awsParams := make([]aws.Parameter, 0, len(stack.Parameters))
	for key, value := range stack.Parameters {
		awsParams = append(awsParams, aws.Parameter{
			Key:   key,
			Value: value,
		})
	}

	// Use capabilities from resolved stack, with default fallback
	capabilities := stack.Capabilities
	if len(capabilities) == 0 {
		capabilities = []string{"CAPABILITY_IAM"} // Default capability
	}

	// Set up event callback for user feedback
	eventCallback := func(event aws.StackEvent) {
		timestamp := event.Timestamp.Format("2006-01-02 15:04:05")
		fmt.Printf("[%s] %-20s %-40s %s %s\n",
			timestamp,
			event.ResourceStatus,
			event.ResourceType,
			event.LogicalResourceId,
			event.ResourceStatusReason,
		)
	}

	deployInput := aws.DeployStackInput{
		StackName:    stack.Name,
		TemplateBody: stack.TemplateBody,
		Parameters:   awsParams,
		Tags:         stack.Tags,
		Capabilities: capabilities,
	}

	// Deploy the stack with event streaming
	err = cfnOps.DeployStackWithCallback(ctx, deployInput, eventCallback)
	if err != nil {
		return fmt.Errorf("failed to create stack: %w", err)
	}

	fmt.Printf("Stack %s create completed successfully\n", stack.Name)
	return nil
}

// deployWithChangeSet handles deployment using changeset preview + execution
func (d *StackDeployer) deployWithChangeSet(ctx context.Context, stack *model.Stack, cfnOps aws.CloudFormationOperations) error {
	// Create differ for consistent change display
	fmt.Printf("=== Calculating changes for stack %s ===\n", stack.Name)

	differ := diff.NewStackDiffer(d.clientFactory)

	// Generate diff result using the same system as 'stackaroo diff'
	// Keep changeset alive for deployment use
	diffOptions := diff.Options{KeepChangeSet: true}
	diffResult, err := differ.DiffStack(ctx, stack, diffOptions)
	if err != nil {
		return fmt.Errorf("failed to calculate changes: %w", err)
	}

	// Show preview using consistent formatting
	if diffResult.HasChanges() {
		fmt.Printf("Changes to be applied to stack %s:\n\n", stack.Name)
		fmt.Print(diffResult.String())
		fmt.Println()

		// Prompt for user confirmation
		message := fmt.Sprintf("Do you want to apply these changes to stack %s?", stack.Name)
		confirmed, err := prompt.Confirm(message)
		if err != nil {
			// Clean up changeset on error
			if diffResult.ChangeSet != nil {
				_ = cfnOps.DeleteChangeSet(ctx, diffResult.ChangeSet.ChangeSetID)
			}
			return fmt.Errorf("failed to get user confirmation: %w", err)
		}

		if !confirmed {
			// Clean up changeset when user cancels
			if diffResult.ChangeSet != nil {
				_ = cfnOps.DeleteChangeSet(ctx, diffResult.ChangeSet.ChangeSetID)
			}
			fmt.Printf("Deployment cancelled for stack %s\n", stack.Name)
			return CancellationError{StackName: stack.Name}
		}
	} else {
		fmt.Printf("No changes detected for stack %s\n", stack.Name)
		return nil
	}

	// Get changeset from diff result (kept alive for deployment)
	if diffResult.ChangeSet == nil {
		return fmt.Errorf("no changeset available for deployment")
	}
	changeSetInfo := diffResult.ChangeSet

	// Execute the changeset
	fmt.Printf("=== Deploying stack %s ===\n", stack.Name)

	err = cfnOps.ExecuteChangeSet(ctx, changeSetInfo.ChangeSetID)
	if err != nil {
		// Clean up changeset on failure
		_ = cfnOps.DeleteChangeSet(ctx, changeSetInfo.ChangeSetID)
		return fmt.Errorf("failed to execute changeset: %w", err)
	}

	// Wait for deployment to complete with progress updates
	eventCallback := func(event aws.StackEvent) {
		timestamp := event.Timestamp.Format("2006-01-02 15:04:05")
		fmt.Printf("[%s] %-20s %-40s %s %s\n",
			timestamp,
			event.ResourceStatus,
			event.ResourceType,
			event.LogicalResourceId,
			event.ResourceStatusReason,
		)
	}

	err = cfnOps.WaitForStackOperation(ctx, stack.Name, eventCallback)
	if err != nil {
		return fmt.Errorf("stack deployment failed: %w", err)
	}

	// Clean up changeset after successful deployment
	_ = cfnOps.DeleteChangeSet(ctx, changeSetInfo.ChangeSetID)

	fmt.Printf("Stack %s update completed successfully\n", stack.Name)
	return nil
}

// ValidateTemplate validates a CloudFormation template
// Note: This method requires region information - consider updating interface to accept region
func (d *StackDeployer) ValidateTemplate(ctx context.Context, templateFile string) error {
	// Read the template file
	templateContent, err := d.readTemplateFile(templateFile)
	if err != nil {
		return fmt.Errorf("failed to read template: %w", err)
	}

	// For template validation, we need a region. Use default from base config.
	// TODO: Consider updating interface to accept region parameter
	baseConfig := d.clientFactory.GetBaseConfig()
	if baseConfig.Region == "" {
		return fmt.Errorf("no default region configured for template validation")
	}

	cfnOps, err := d.clientFactory.GetCloudFormationOperations(ctx, baseConfig.Region)
	if err != nil {
		return fmt.Errorf("failed to get CloudFormation operations for template validation: %w", err)
	}

	// Validate the template
	err = cfnOps.ValidateTemplate(ctx, templateContent)
	if err != nil {
		return fmt.Errorf("template validation failed: %w", err)
	}

	return nil
}

// readTemplateFile reads and returns the contents of a template file
func (d *StackDeployer) readTemplateFile(filename string) (string, error) {
	content, err := os.ReadFile(filename)
	if err != nil {
		return "", fmt.Errorf("failed to read template file %s: %w", filename, err)
	}
	return string(content), nil
}

// deployStackWithFeedback deploys a stack and provides feedback
func (d *StackDeployer) deployStackWithFeedback(ctx context.Context, stack *model.Stack, contextName string) error {
	err := d.DeployStack(ctx, stack)
	if err != nil {
		// Handle cancellation - don't treat it as an error for the caller
		var cancellationErr CancellationError
		if errors.As(err, &cancellationErr) {
			return nil
		}
		return fmt.Errorf("error deploying stack %s: %w", stack.Name, err)
	}

	fmt.Printf("Successfully deployed stack %s in context %s\n", stack.Name, contextName)
	return nil
}

// DeploySingleStack handles deployment of a single stack
func (d *StackDeployer) DeploySingleStack(ctx context.Context, stackName, contextName string) error {
	// Resolve single stack
	stack, err := d.resolver.ResolveStack(ctx, contextName, stackName)
	if err != nil {
		return fmt.Errorf("failed to resolve stack dependencies: %w", err)
	}

	return d.deployStackWithFeedback(ctx, stack, contextName)
}

// DeployAllStacks handles deployment of all stacks in a context
func (d *StackDeployer) DeployAllStacks(ctx context.Context, contextName string) error {
	// Get list of stacks to deploy
	stackNames, err := d.provider.ListStacks(contextName)
	if err != nil {
		return fmt.Errorf("failed to get stacks for context %s: %w", contextName, err)
	}
	if len(stackNames) == 0 {
		fmt.Printf("No stacks found in context %s\n", contextName)
		return nil
	}

	// Get dependency order without resolving stacks
	deploymentOrder, err := d.resolver.GetDependencyOrder(contextName, stackNames)
	if err != nil {
		return fmt.Errorf("failed to calculate dependency order: %w", err)
	}

	// Deploy each stack in dependency order, resolving individually to get fresh parameters
	for _, stackName := range deploymentOrder {
		// Resolve this specific stack to get fresh parameter values
		stack, err := d.resolver.ResolveStack(ctx, contextName, stackName)
		if err != nil {
			return fmt.Errorf("failed to resolve stack %s: %w", stackName, err)
		}

		err = d.deployStackWithFeedback(ctx, stack, contextName)
		if err != nil {
			return err
		}
	}

	return nil
}
