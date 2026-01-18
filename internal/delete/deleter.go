/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package delete

import (
	"context"
	"fmt"
	"time"

	"github.com/orien/stackaroo/internal/aws"
	"github.com/orien/stackaroo/internal/config"
	"github.com/orien/stackaroo/internal/model"
	"github.com/orien/stackaroo/internal/prompt"
	"github.com/orien/stackaroo/internal/resolve"
)

// Deleter defines the interface for stack deletion operations
type Deleter interface {
	DeleteStack(ctx context.Context, stack *model.Stack) error
	DeleteSingleStack(ctx context.Context, stackName, contextName string) error
	DeleteAllStacks(ctx context.Context, contextName string) error
}

// StackDeleter implements Deleter using AWS CloudFormation
type StackDeleter struct {
	clientFactory  aws.ClientFactory
	configProvider config.ConfigProvider
	resolver       resolve.Resolver
}

// NewStackDeleter creates a new StackDeleter
func NewStackDeleter(clientFactory aws.ClientFactory, configProvider config.ConfigProvider, resolver resolve.Resolver) *StackDeleter {
	return &StackDeleter{
		clientFactory:  clientFactory,
		configProvider: configProvider,
		resolver:       resolver,
	}
}

// DeleteStack deletes a CloudFormation stack with confirmation
func (d *StackDeleter) DeleteStack(ctx context.Context, stack *model.Stack) error {
	// Get region-specific CloudFormation operations
	cfnOps, err := d.clientFactory.GetCloudFormationOperations(ctx, stack.Context.Region)
	if err != nil {
		return fmt.Errorf("failed to get CloudFormation operations for region %s: %w", stack.Context.Region, err)
	}

	// Check if stack exists
	exists, err := cfnOps.StackExists(ctx, stack.Name)
	if err != nil {
		return fmt.Errorf("failed to check if stack exists: %w", err)
	}

	if !exists {
		fmt.Printf("Stack %s does not exist, skipping deletion\n", stack.Name)
		return nil
	}

	// Get stack information to show what will be deleted
	stackInfo, err := cfnOps.DescribeStack(ctx, stack.Name)
	if err != nil {
		return fmt.Errorf("failed to describe stack %s: %w", stack.Name, err)
	}

	// Show what will be deleted
	fmt.Printf("\n=== Stack Deletion Preview ===\n")
	fmt.Printf("Stack Name: %s\n", stack.Name)
	fmt.Printf("Context: %s\n", stack.Context.Name)
	fmt.Printf("Status: %s\n", stackInfo.Status)
	if stackInfo.Description != "" {
		fmt.Printf("Description: %s\n", stackInfo.Description)
	}

	fmt.Printf("\nThis will permanently delete the CloudFormation stack and all its resources.\n")
	fmt.Printf("WARNING: This operation cannot be undone!\n")

	// Prompt for confirmation
	message := fmt.Sprintf("Do you want to delete stack %s? This cannot be undone.", stack.Name)
	confirmed, err := prompt.Confirm(message)
	if err != nil {
		return fmt.Errorf("failed to get user confirmation: %w", err)
	}

	if !confirmed {
		fmt.Printf("Deletion of stack %s cancelled by user\n", stack.Name)
		return nil
	}

	// Perform the deletion
	fmt.Printf("Deleting stack %s...\n", stack.Name)

	// Capture start time to filter events to only this deletion
	startTime := time.Now()

	deleteInput := aws.DeleteStackInput{
		StackName: stack.Name,
	}

	err = cfnOps.DeleteStack(ctx, deleteInput)
	if err != nil {
		return fmt.Errorf("failed to delete stack %s: %w", stack.Name, err)
	}

	// Wait for deletion to complete
	fmt.Printf("Waiting for stack deletion to complete...\n")
	err = cfnOps.WaitForStackOperation(ctx, stack.Name, startTime, func(event aws.StackEvent) {
		fmt.Printf("  %s: %s - %s\n", event.Timestamp.Format("15:04:05"), event.ResourceType, event.ResourceStatus)
		if event.ResourceStatusReason != "" {
			fmt.Printf("    Reason: %s\n", event.ResourceStatusReason)
		}
	})
	if err != nil {
		return fmt.Errorf("failed to wait for stack deletion: %w", err)
	}

	return nil
}

// DeleteSingleStack handles deletion of a single stack
func (d *StackDeleter) DeleteSingleStack(ctx context.Context, stackName, contextName string) error {
	// Resolve single stack
	stack, err := d.resolver.ResolveStack(ctx, contextName, stackName)
	if err != nil {
		return err
	}

	return d.deleteStackWithFeedback(ctx, stack, contextName)
}

// DeleteAllStacks handles deletion of all stacks in a context
func (d *StackDeleter) DeleteAllStacks(ctx context.Context, contextName string) error {
	// Get list of stacks to delete
	stackNames, err := d.configProvider.ListStacks(contextName)
	if err != nil {
		return err
	}
	if len(stackNames) == 0 {
		fmt.Printf("No stacks found in context %s\n", contextName)
		return nil
	}

	// Get dependency order without resolving stacks
	deploymentOrder, err := d.resolver.GetDependencyOrder(contextName, stackNames)
	if err != nil {
		return err
	}

	// Reverse the deployment order for safe deletion
	// Dependencies should be deleted before the stacks that depend on them
	deletionOrder := make([]string, len(deploymentOrder))
	for i, stackName := range deploymentOrder {
		deletionOrder[len(deploymentOrder)-1-i] = stackName
	}

	// Delete each stack in reverse dependency order, resolving individually
	for _, stackName := range deletionOrder {
		// Resolve this specific stack
		stack, err := d.resolver.ResolveStack(ctx, contextName, stackName)
		if err != nil {
			return err
		}

		err = d.deleteStackWithFeedback(ctx, stack, contextName)
		if err != nil {
			return err
		}
	}

	return nil
}

// deleteStackWithFeedback deletes a stack and provides feedback
func (d *StackDeleter) deleteStackWithFeedback(ctx context.Context, stack *model.Stack, contextName string) error {
	err := d.DeleteStack(ctx, stack)
	if err != nil {
		return fmt.Errorf("error deleting stack %s: %w", stack.Name, err)
	}

	fmt.Printf("Successfully deleted stack %s in context %s\n", stack.Name, contextName)
	return nil
}
