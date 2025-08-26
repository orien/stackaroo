/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package deploy

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	awsinternal "github.com/orien/stackaroo/internal/aws"
	"github.com/orien/stackaroo/internal/diff"
	"github.com/orien/stackaroo/internal/model"
)

// Deployer defines the interface for stack deployment operations
type Deployer interface {
	DeployStack(ctx context.Context, resolvedStack *model.Stack) error
	ValidateTemplate(ctx context.Context, templateFile string) error
}

// AWSDeployer implements Deployer using AWS CloudFormation
type AWSDeployer struct {
	awsClient awsinternal.Client
}

// NewAWSDeployer creates a new AWSDeployer
func NewAWSDeployer(awsClient awsinternal.Client) *AWSDeployer {
	return &AWSDeployer{
		awsClient: awsClient,
	}
}

// NewDefaultDeployer creates a deployer with default AWS configuration
func NewDefaultDeployer(ctx context.Context) (*AWSDeployer, error) {
	client, err := awsinternal.NewDefaultClient(ctx, awsinternal.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to create AWS client: %w", err)
	}

	return NewAWSDeployer(client), nil
}

// DeployStack deploys a CloudFormation stack using changesets for preview and deployment
func (d *AWSDeployer) DeployStack(ctx context.Context, resolvedStack *model.Stack) error {
	// Get CloudFormation operations
	cfnOps := d.awsClient.NewCloudFormationOperations()

	// Check if stack exists to determine deployment approach
	exists, err := cfnOps.StackExists(ctx, resolvedStack.Name)
	if err != nil {
		return fmt.Errorf("failed to check if stack exists: %w", err)
	}

	if !exists {
		// For new stacks, use direct creation (changesets are less useful)
		return d.deployNewStack(ctx, resolvedStack)
	}

	// For existing stacks, use changeset approach for preview + deployment
	return d.deployWithChangeSet(ctx, resolvedStack)
}

// deployNewStack handles deployment of new stacks using direct creation
func (d *AWSDeployer) deployNewStack(ctx context.Context, resolvedStack *model.Stack) error {
	fmt.Printf("=== Creating new stack %s ===\n", resolvedStack.Name)

	// Convert parameters to AWS format
	awsParams := make([]awsinternal.Parameter, 0, len(resolvedStack.Parameters))
	for key, value := range resolvedStack.Parameters {
		awsParams = append(awsParams, awsinternal.Parameter{
			Key:   key,
			Value: value,
		})
	}

	// Use capabilities from resolved stack, with default fallback
	capabilities := resolvedStack.Capabilities
	if len(capabilities) == 0 {
		capabilities = []string{"CAPABILITY_IAM"} // Default capability
	}

	// Set up event callback for user feedback
	eventCallback := func(event awsinternal.StackEvent) {
		timestamp := event.Timestamp.Format("2006-01-02 15:04:05")
		fmt.Printf("[%s] %-20s %-40s %s %s\n",
			timestamp,
			event.ResourceStatus,
			event.ResourceType,
			event.LogicalResourceId,
			event.ResourceStatusReason,
		)
	}

	deployInput := awsinternal.DeployStackInput{
		StackName:    resolvedStack.Name,
		TemplateBody: resolvedStack.TemplateBody,
		Parameters:   awsParams,
		Tags:         resolvedStack.Tags,
		Capabilities: capabilities,
	}

	// Get CloudFormation operations
	cfnOps := d.awsClient.NewCloudFormationOperations()

	// Deploy the stack with event streaming
	err := cfnOps.DeployStackWithCallback(ctx, deployInput, eventCallback)
	if err != nil {
		return fmt.Errorf("failed to create stack: %w", err)
	}

	fmt.Printf("Stack %s create completed successfully\n", resolvedStack.Name)
	return nil
}

// deployWithChangeSet handles deployment using changeset preview + execution
func (d *AWSDeployer) deployWithChangeSet(ctx context.Context, resolvedStack *model.Stack) error {
	// Create differ for consistent change display
	fmt.Printf("=== Calculating changes for stack %s ===\n", resolvedStack.Name)

	cfnOps := d.awsClient.NewCloudFormationOperations()
	differ := diff.NewDiffer(cfnOps)

	// Generate diff result using the same system as 'stackaroo diff'
	diffOptions := diff.Options{Format: "text"}
	diffResult, err := differ.DiffStack(ctx, resolvedStack, diffOptions)
	if err != nil {
		return fmt.Errorf("failed to calculate changes: %w", err)
	}

	// Show preview using consistent formatting
	if diffResult.HasChanges() {
		fmt.Printf("Changes to be applied to stack %s:\n\n", resolvedStack.Name)
		fmt.Print(diffResult.String())
		fmt.Println()
	} else {
		fmt.Printf("No changes detected for stack %s\n", resolvedStack.Name)
		return nil
	}

	// Create separate changeset for execution (since differ deletes its changesets)
	changeSetMgr := diff.NewChangeSetManager(cfnOps)

	// Use capabilities from resolved stack, with default fallback
	capabilities := resolvedStack.Capabilities
	if len(capabilities) == 0 {
		capabilities = []string{"CAPABILITY_IAM"} // Default capability
	}

	changeSetInfo, err := changeSetMgr.CreateChangeSetForDeployment(
		ctx,
		resolvedStack.Name,
		resolvedStack.TemplateBody,
		resolvedStack.Parameters,
		capabilities,
		resolvedStack.Tags,
	)
	if err != nil {
		return fmt.Errorf("failed to create changeset for execution: %w", err)
	}

	// Execute the changeset
	fmt.Printf("=== Deploying stack %s ===\n", resolvedStack.Name)

	executeInput := &cloudformation.ExecuteChangeSetInput{
		ChangeSetName: aws.String(changeSetInfo.ChangeSetID),
	}
	_, err = cfnOps.ExecuteChangeSet(ctx, executeInput)
	if err != nil {
		// Clean up changeset on failure
		_ = changeSetMgr.DeleteChangeSet(ctx, changeSetInfo.ChangeSetID)
		return fmt.Errorf("failed to execute changeset: %w", err)
	}

	// Wait for deployment to complete with progress updates
	eventCallback := func(event awsinternal.StackEvent) {
		timestamp := event.Timestamp.Format("2006-01-02 15:04:05")
		fmt.Printf("[%s] %-20s %-40s %s %s\n",
			timestamp,
			event.ResourceStatus,
			event.ResourceType,
			event.LogicalResourceId,
			event.ResourceStatusReason,
		)
	}

	err = cfnOps.WaitForStackOperation(ctx, resolvedStack.Name, eventCallback)
	if err != nil {
		return fmt.Errorf("stack deployment failed: %w", err)
	}

	// Clean up changeset after successful deployment
	_ = changeSetMgr.DeleteChangeSet(ctx, changeSetInfo.ChangeSetID)

	fmt.Printf("Stack %s update completed successfully\n", resolvedStack.Name)
	return nil
}

// ValidateTemplate validates a CloudFormation template
func (d *AWSDeployer) ValidateTemplate(ctx context.Context, templateFile string) error {
	// Read the template file
	templateContent, err := d.readTemplateFile(templateFile)
	if err != nil {
		return fmt.Errorf("failed to read template: %w", err)
	}

	// Get CloudFormation operations
	cfnOps := d.awsClient.NewCloudFormationOperations()

	// Validate the template
	err = cfnOps.ValidateTemplate(ctx, templateContent)
	if err != nil {
		return fmt.Errorf("template validation failed: %w", err)
	}

	return nil
}

// readTemplateFile reads the content of a template file
func (d *AWSDeployer) readTemplateFile(filename string) (string, error) {
	content, err := os.ReadFile(filename)
	if err != nil {
		return "", fmt.Errorf("failed to read template file %s: %w", filename, err)
	}
	return string(content), nil
}
