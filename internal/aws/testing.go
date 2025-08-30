/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package aws

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/stretchr/testify/mock"
)

// MockClient implements Client for testing
type MockClient struct {
	mock.Mock
}

func (m *MockClient) NewCloudFormationOperations() CloudFormationOperations {
	args := m.Called()
	return args.Get(0).(CloudFormationOperations)
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

func (m *MockCloudFormationOperations) WaitForStackOperation(ctx context.Context, stackName string, eventCallback func(StackEvent)) error {
	args := m.Called(ctx, stackName, eventCallback)
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

func (m *MockCloudFormationOperations) CreateChangeSetPreview(ctx context.Context, stackName string, template string, parameters map[string]string) (*ChangeSetInfo, error) {
	args := m.Called(ctx, stackName, template, parameters)
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
