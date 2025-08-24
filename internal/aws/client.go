/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
)

// CloudFormationOperationsClient is an alias for the AWS CloudFormation operations
type CloudFormationOperationsClient = CloudFormationOperations

// DefaultClient provides a high-level interface for AWS operations
type DefaultClient struct {
	config aws.Config
	cfn    *cloudformation.Client
}

// Config holds configuration for creating an AWS client
type Config struct {
	Region  string
	Profile string
}

// NewDefaultClient creates a new AWS client with the specified configuration
func NewDefaultClient(ctx context.Context, cfg Config) (*DefaultClient, error) {
	var opts []func(*config.LoadOptions) error

	// Set region if specified
	if cfg.Region != "" {
		opts = append(opts, config.WithRegion(cfg.Region))
	}

	// Set profile if specified
	if cfg.Profile != "" {
		opts = append(opts, config.WithSharedConfigProfile(cfg.Profile))
	}

	// Load AWS configuration
	awsCfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS configuration: %w", err)
	}

	// Create service clients
	cfnClient := cloudformation.NewFromConfig(awsCfg)

	return &DefaultClient{
		config: awsCfg,
		cfn:    cfnClient,
	}, nil
}

// CloudFormation returns the CloudFormation client
func (c *DefaultClient) CloudFormation() *cloudformation.Client {
	return c.cfn
}

// Region returns the configured AWS region
func (c *DefaultClient) Region() string {
	return c.config.Region
}

// NewCloudFormationClient creates a new CloudFormation client with default AWS configuration
func NewCloudFormationClient(ctx context.Context) (CloudFormationOperationsClient, error) {
	// Load default AWS configuration
	awsCfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS configuration: %w", err)
	}

	// Create CloudFormation client
	cfnClient := cloudformation.NewFromConfig(awsCfg)

	// Create operations wrapper
	operations := NewCloudFormationOperationsWithClient(cfnClient)

	return operations, nil
}
