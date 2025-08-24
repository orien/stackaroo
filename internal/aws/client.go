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

// CloudFormationClient is an alias for the AWS CloudFormation client
type CloudFormationClient = CloudFormationOperationsInterface

// Client provides a high-level interface for AWS operations
type Client struct {
	config aws.Config
	cfn    *cloudformation.Client
}

// Config holds configuration for creating an AWS client
type Config struct {
	Region  string
	Profile string
}

// NewClient creates a new AWS client with the specified configuration
func NewClient(ctx context.Context, cfg Config) (*Client, error) {
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

	return &Client{
		config: awsCfg,
		cfn:    cfnClient,
	}, nil
}

// CloudFormation returns the CloudFormation client
func (c *Client) CloudFormation() *cloudformation.Client {
	return c.cfn
}

// Region returns the configured AWS region
func (c *Client) Region() string {
	return c.config.Region
}

// NewCloudFormationClient creates a new CloudFormation client with default AWS configuration
func NewCloudFormationClient(ctx context.Context) (CloudFormationClient, error) {
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
