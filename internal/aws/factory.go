/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package aws

import (
	"context"
	"fmt"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
)

// ClientFactory creates AWS clients with proper region configuration
type ClientFactory interface {
	// GetCloudFormationOperations returns CloudFormation operations for specified region
	GetCloudFormationOperations(ctx context.Context, region string) (CloudFormationOperations, error)

	// GetBaseConfig returns the shared AWS configuration (for debugging)
	GetBaseConfig() aws.Config

	// ValidateRegion checks if a region is valid (optional validation)
	ValidateRegion(region string) error
}

// DefaultClientFactory implements ClientFactory with caching and shared authentication
type DefaultClientFactory struct {
	baseConfig  aws.Config
	clientCache map[string]CloudFormationOperations
	mutex       sync.RWMutex
}

// NewClientFactory creates a client factory with shared authentication
func NewClientFactory(ctx context.Context) (ClientFactory, error) {
	// Load base config with credentials but allow region override per-client
	baseConfig, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS configuration: %w", err)
	}

	return &DefaultClientFactory{
		baseConfig:  baseConfig,
		clientCache: make(map[string]CloudFormationOperations),
	}, nil
}

// GetCloudFormationOperations returns CloudFormation operations for the specified region
func (f *DefaultClientFactory) GetCloudFormationOperations(ctx context.Context, region string) (CloudFormationOperations, error) {
	if region == "" {
		return nil, fmt.Errorf("region cannot be empty")
	}

	// Check cache first (read lock)
	f.mutex.RLock()
	if ops, exists := f.clientCache[region]; exists {
		f.mutex.RUnlock()
		return ops, nil
	}
	f.mutex.RUnlock()

	// Create region-specific config from base config
	regionConfig := f.baseConfig.Copy()
	regionConfig.Region = region

	// Create service client with region-specific config
	cfnClient := cloudformation.NewFromConfig(regionConfig)
	ops := NewCloudFormationOperationsWithClient(cfnClient)

	// Cache for future use (write lock)
	f.mutex.Lock()
	f.clientCache[region] = ops
	f.mutex.Unlock()

	return ops, nil
}

// GetBaseConfig returns the shared AWS configuration
func (f *DefaultClientFactory) GetBaseConfig() aws.Config {
	return f.baseConfig
}

// ValidateRegion performs basic region validation
func (f *DefaultClientFactory) ValidateRegion(region string) error {
	if region == "" {
		return fmt.Errorf("region cannot be empty")
	}

	// Basic AWS region format validation
	// AWS regions follow pattern: <partition>-<service>-<number> (e.g., us-east-1, eu-west-2)
	// This is a simple check - AWS SDK will do full validation
	if len(region) < 9 { // Minimum length for valid region
		return fmt.Errorf("region '%s' appears to be invalid", region)
	}

	return nil
}
