/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package aws

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/stretchr/testify/mock"
)

// NewMockClientFactoryWithOperations creates a MockClientFactory and sets up operations for a region
func NewMockClientFactoryWithOperations(region string, ops CloudFormationOperations) *MockClientFactory {
	factory := NewMockClientFactory()
	factory.SetOperations(region, ops)
	return factory
}

// NewMockClientFactoryForRegion creates a MockClientFactory with MockCloudFormationOperations for a region
func NewMockClientFactoryForRegion(region string) (*MockClientFactory, *MockCloudFormationOperations) {
	mockOps := &MockCloudFormationOperations{}
	factory := NewMockClientFactoryWithOperations(region, mockOps)
	// Set baseConfig with the region for ValidateTemplate and other methods that need it
	factory.baseConfig.Region = region
	return factory, mockOps
}

// SetupMockFactoryForMultiRegion creates a factory with operations for multiple regions
func SetupMockFactoryForMultiRegion(regions map[string]CloudFormationOperations) *MockClientFactory {
	factory := NewMockClientFactory()
	for region, ops := range regions {
		factory.SetOperations(region, ops)
	}
	return factory
}

// MockClientFactory provides a test implementation of ClientFactory
type MockClientFactory struct {
	operations map[string]CloudFormationOperations
	baseConfig aws.Config
	mutex      sync.RWMutex
}

// NewMockClientFactory creates a mock factory for testing
func NewMockClientFactory() *MockClientFactory {
	return &MockClientFactory{
		operations: make(map[string]CloudFormationOperations),
		baseConfig: aws.Config{}, // Empty config for testing
	}
}

// SetOperations sets mock operations for a specific region
func (m *MockClientFactory) SetOperations(region string, ops CloudFormationOperations) {
	m.mutex.Lock()
	m.operations[region] = ops
	m.mutex.Unlock()
}

// GetCloudFormationOperations returns mock operations for the specified region
func (m *MockClientFactory) GetCloudFormationOperations(ctx context.Context, region string) (CloudFormationOperations, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	ops, exists := m.operations[region]
	if !exists {
		return nil, fmt.Errorf("no mock operations configured for region %s", region)
	}

	return ops, nil
}

// GetBaseConfig returns the mock base configuration
func (m *MockClientFactory) GetBaseConfig() aws.Config {
	return m.baseConfig
}

// ValidateRegion always returns nil for mock
func (m *MockClientFactory) ValidateRegion(region string) error {
	if region == "" {
		return fmt.Errorf("region cannot be empty")
	}
	return nil
}

// MockCloudFormationOperations implements CloudFormationOperations for testing
type MockCloudFormationOperations struct {
	mock.Mock
}

func (m *MockCloudFormationOperations) DeployStack(ctx context.Context, input DeployStackInput) error {
	args := m.Called(ctx, input)
	return args.Error(0)
}

func (m *MockCloudFormationOperations) DeployStackWithCallback(ctx context.Context, input DeployStackInput, eventCallback func(StackEvent)) error {
	args := m.Called(ctx, input, eventCallback)
	return args.Error(0)
}

func (m *MockCloudFormationOperations) UpdateStack(ctx context.Context, input UpdateStackInput) error {
	args := m.Called(ctx, input)
	return args.Error(0)
}

func (m *MockCloudFormationOperations) DeleteStack(ctx context.Context, input DeleteStackInput) error {
	args := m.Called(ctx, input)
	return args.Error(0)
}

func (m *MockCloudFormationOperations) GetStack(ctx context.Context, stackName string) (*Stack, error) {
	args := m.Called(ctx, stackName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Stack), args.Error(1)
}

func (m *MockCloudFormationOperations) ListStacks(ctx context.Context) ([]*Stack, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*Stack), args.Error(1)
}

func (m *MockCloudFormationOperations) ValidateTemplate(ctx context.Context, templateBody string) error {
	args := m.Called(ctx, templateBody)
	return args.Error(0)
}

func (m *MockCloudFormationOperations) StackExists(ctx context.Context, stackName string) (bool, error) {
	args := m.Called(ctx, stackName)
	return args.Bool(0), args.Error(1)
}

func (m *MockCloudFormationOperations) GetTemplate(ctx context.Context, stackName string) (string, error) {
	args := m.Called(ctx, stackName)
	return args.String(0), args.Error(1)
}

func (m *MockCloudFormationOperations) DescribeStack(ctx context.Context, stackName string) (*StackInfo, error) {
	args := m.Called(ctx, stackName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*StackInfo), args.Error(1)
}

func (m *MockCloudFormationOperations) ExecuteChangeSet(ctx context.Context, changeSetID string) error {
	args := m.Called(ctx, changeSetID)
	return args.Error(0)
}

func (m *MockCloudFormationOperations) DeleteChangeSet(ctx context.Context, changeSetID string) error {
	args := m.Called(ctx, changeSetID)
	return args.Error(0)
}

func (m *MockCloudFormationOperations) DescribeStackEvents(ctx context.Context, stackName string) ([]StackEvent, error) {
	args := m.Called(ctx, stackName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]StackEvent), args.Error(1)
}

func (m *MockCloudFormationOperations) WaitForStackOperation(ctx context.Context, stackName string, startTime time.Time, eventCallback func(StackEvent)) error {
	args := m.Called(ctx, stackName, startTime, eventCallback)
	// Call the callback with a sample event for testing
	if eventCallback != nil {
		eventCallback(StackEvent{
			EventId:              "event-1",
			StackName:            stackName,
			LogicalResourceId:    stackName,
			ResourceType:         "AWS::CloudFormation::Stack",
			Timestamp:            time.Now(),
			ResourceStatus:       "OPERATION_IN_PROGRESS",
			ResourceStatusReason: "",
		})
	}
	return args.Error(0)
}

func (m *MockCloudFormationOperations) CreateChangeSetPreview(ctx context.Context, stackName string, template string, parameters map[string]string, capabilities []string, tags map[string]string) (*ChangeSetInfo, error) {
	args := m.Called(ctx, stackName, template, parameters, capabilities, tags)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ChangeSetInfo), args.Error(1)
}

func (m *MockCloudFormationOperations) CreateChangeSetForDeployment(ctx context.Context, stackName string, template string, parameters map[string]string, capabilities []string, tags map[string]string) (*ChangeSetInfo, error) {
	args := m.Called(ctx, stackName, template, parameters, capabilities, tags)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ChangeSetInfo), args.Error(1)
}

// MockCloudFormationClient implements the AWS CloudFormation service client interface for testing
type MockCloudFormationClient struct {
	mock.Mock
}

func (m *MockCloudFormationClient) CreateStack(ctx context.Context, params *cloudformation.CreateStackInput, optFns ...func(*cloudformation.Options)) (*cloudformation.CreateStackOutput, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*cloudformation.CreateStackOutput), args.Error(1)
}

func (m *MockCloudFormationClient) UpdateStack(ctx context.Context, params *cloudformation.UpdateStackInput, optFns ...func(*cloudformation.Options)) (*cloudformation.UpdateStackOutput, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*cloudformation.UpdateStackOutput), args.Error(1)
}

func (m *MockCloudFormationClient) DeleteStack(ctx context.Context, params *cloudformation.DeleteStackInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DeleteStackOutput, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(*cloudformation.DeleteStackOutput), args.Error(1)
}

func (m *MockCloudFormationClient) DescribeStacks(ctx context.Context, params *cloudformation.DescribeStacksInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DescribeStacksOutput, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*cloudformation.DescribeStacksOutput), args.Error(1)
}

func (m *MockCloudFormationClient) ListStacks(ctx context.Context, params *cloudformation.ListStacksInput, optFns ...func(*cloudformation.Options)) (*cloudformation.ListStacksOutput, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(*cloudformation.ListStacksOutput), args.Error(1)
}

func (m *MockCloudFormationClient) ValidateTemplate(ctx context.Context, params *cloudformation.ValidateTemplateInput, optFns ...func(*cloudformation.Options)) (*cloudformation.ValidateTemplateOutput, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(*cloudformation.ValidateTemplateOutput), args.Error(1)
}

func (m *MockCloudFormationClient) GetTemplate(ctx context.Context, params *cloudformation.GetTemplateInput, optFns ...func(*cloudformation.Options)) (*cloudformation.GetTemplateOutput, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(*cloudformation.GetTemplateOutput), args.Error(1)
}

func (m *MockCloudFormationClient) CreateChangeSet(ctx context.Context, params *cloudformation.CreateChangeSetInput, optFns ...func(*cloudformation.Options)) (*cloudformation.CreateChangeSetOutput, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(*cloudformation.CreateChangeSetOutput), args.Error(1)
}

func (m *MockCloudFormationClient) ExecuteChangeSet(ctx context.Context, params *cloudformation.ExecuteChangeSetInput, optFns ...func(*cloudformation.Options)) (*cloudformation.ExecuteChangeSetOutput, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(*cloudformation.ExecuteChangeSetOutput), args.Error(1)
}

func (m *MockCloudFormationClient) DeleteChangeSet(ctx context.Context, params *cloudformation.DeleteChangeSetInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DeleteChangeSetOutput, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(*cloudformation.DeleteChangeSetOutput), args.Error(1)
}

func (m *MockCloudFormationClient) DescribeChangeSet(ctx context.Context, params *cloudformation.DescribeChangeSetInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DescribeChangeSetOutput, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(*cloudformation.DescribeChangeSetOutput), args.Error(1)
}

func (m *MockCloudFormationClient) DescribeStackEvents(ctx context.Context, params *cloudformation.DescribeStackEventsInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DescribeStackEventsOutput, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*cloudformation.DescribeStackEventsOutput), args.Error(1)
}
