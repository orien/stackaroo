/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package deploy

import (
	"context"
	"fmt"
	"os"

	"github.com/orien/stackaroo/internal/aws"
	"github.com/orien/stackaroo/internal/resolve"
)

// Deployer defines the interface for stack deployment operations
type Deployer interface {
	DeployStack(ctx context.Context, resolvedStack *resolve.ResolvedStack) error
	ValidateTemplate(ctx context.Context, templateFile string) error
}

// AWSDeployer implements Deployer using AWS CloudFormation
type AWSDeployer struct {
	awsClient aws.Client
}

// NewAWSDeployer creates a new AWSDeployer
func NewAWSDeployer(awsClient aws.Client) *AWSDeployer {
	return &AWSDeployer{
		awsClient: awsClient,
	}
}

// NewDefaultDeployer creates a deployer with default AWS configuration
func NewDefaultDeployer(ctx context.Context) (*AWSDeployer, error) {
	client, err := aws.NewDefaultClient(ctx, aws.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to create AWS client: %w", err)
	}

	return NewAWSDeployer(client), nil
}

// DeployStack deploys a CloudFormation stack
func (d *AWSDeployer) DeployStack(ctx context.Context, resolvedStack *resolve.ResolvedStack) error {
	// Convert parameters to AWS format
	awsParams := make([]aws.Parameter, 0, len(resolvedStack.Parameters))
	for key, value := range resolvedStack.Parameters {
		awsParams = append(awsParams, aws.Parameter{
			Key:   key,
			Value: value,
		})
	}

	// Use capabilities from resolved stack, with default fallback
	capabilities := resolvedStack.Capabilities
	if len(capabilities) == 0 {
		capabilities = []string{"CAPABILITY_IAM"} // Default capability
	}

	// Get CloudFormation operations
	cfnOps := d.awsClient.NewCloudFormationOperations()

	// Deploy the stack
	err := cfnOps.DeployStack(ctx, aws.DeployStackInput{
		StackName:    resolvedStack.Name,
		TemplateBody: resolvedStack.TemplateBody,
		Parameters:   awsParams,
		Tags:         resolvedStack.Tags,
		Capabilities: capabilities,
	})

	if err != nil {
		return fmt.Errorf("failed to deploy stack: %w", err)
	}

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
