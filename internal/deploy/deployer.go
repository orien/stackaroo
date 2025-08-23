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
	"github.com/orien/stackaroo/internal/config"
)

// Deployer defines the interface for stack deployment operations
type Deployer interface {
	DeployStack(ctx context.Context, stackConfig *config.StackConfig) error
	ValidateTemplate(ctx context.Context, templateFile string) error
}

// AWSDeployer implements Deployer using AWS CloudFormation
type AWSDeployer struct {
	awsClient aws.ClientInterface
}

// NewAWSDeployer creates a new AWSDeployer
func NewAWSDeployer(awsClient aws.ClientInterface) *AWSDeployer {
	return &AWSDeployer{
		awsClient: awsClient,
	}
}

// NewDefaultDeployer creates a deployer with default AWS configuration
func NewDefaultDeployer(ctx context.Context) (*AWSDeployer, error) {
	client, err := aws.NewClient(ctx, aws.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to create AWS client: %w", err)
	}

	return NewAWSDeployer(client), nil
}

// DeployStack deploys a CloudFormation stack
func (d *AWSDeployer) DeployStack(ctx context.Context, stackConfig *config.StackConfig) error {
	// Read the template file
	templateContent, err := d.readTemplateFile(stackConfig.Template)
	if err != nil {
		return fmt.Errorf("failed to read template: %w", err)
	}
	
	// Convert parameters to AWS format
	awsParams := make([]aws.Parameter, 0, len(stackConfig.Parameters))
	for key, value := range stackConfig.Parameters {
		awsParams = append(awsParams, aws.Parameter{
			Key:   key,
			Value: value,
		})
	}
	
	// Use capabilities from config, with default fallback
	capabilities := stackConfig.Capabilities
	if len(capabilities) == 0 {
		capabilities = []string{"CAPABILITY_IAM"} // Default capability
	}
	
	// Get CloudFormation operations
	cfnOps := d.awsClient.NewCloudFormationOperations()
	
	// Deploy the stack
	err = cfnOps.DeployStack(ctx, aws.DeployStackInput{
		StackName:    stackConfig.Name,
		TemplateBody: templateContent,
		Parameters:   awsParams,
		Tags:         stackConfig.Tags,
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
