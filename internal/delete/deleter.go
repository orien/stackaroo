/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package delete

import (
	"context"
	"fmt"

	"github.com/orien/stackaroo/internal/aws"
	"github.com/orien/stackaroo/internal/model"
	"github.com/orien/stackaroo/internal/prompt"
)

// Deleter defines the interface for stack deletion operations
type Deleter interface {
	DeleteStack(ctx context.Context, stack *model.Stack) error
}

// AWSDeleter implements Deleter using AWS CloudFormation
type AWSDeleter struct {
	awsClient aws.Client
}

// NewAWSDeleter creates a new AWSDeleter
func NewAWSDeleter(awsClient aws.Client) *AWSDeleter {
	return &AWSDeleter{
		awsClient: awsClient,
	}
}

// DeleteStack deletes a CloudFormation stack with confirmation
func (d *AWSDeleter) DeleteStack(ctx context.Context, stack *model.Stack) error {
	// Get CloudFormation operations
	cfnOps := d.awsClient.NewCloudFormationOperations()

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
	fmt.Printf("Context: %s\n", stack.Context)
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

	deleteInput := aws.DeleteStackInput{
		StackName: stack.Name,
	}

	err = cfnOps.DeleteStack(ctx, deleteInput)
	if err != nil {
		return fmt.Errorf("failed to delete stack %s: %w", stack.Name, err)
	}

	// Wait for deletion to complete
	fmt.Printf("Waiting for stack deletion to complete...\n")
	err = cfnOps.WaitForStackOperation(ctx, stack.Name, func(event aws.StackEvent) {
		fmt.Printf("  %s: %s - %s\n", event.Timestamp.Format("15:04:05"), event.ResourceType, event.ResourceStatus)
		if event.ResourceStatusReason != "" {
			fmt.Printf("    Reason: %s\n", event.ResourceStatusReason)
		}
	})

	if err != nil {
		return fmt.Errorf("stack deletion failed or timed out: %w", err)
	}

	return nil
}
