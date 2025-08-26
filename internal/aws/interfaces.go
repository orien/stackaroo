/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
)

// CloudFormationClient defines the interface for CloudFormation client operations
// This allows for easier testing with mock implementations
type CloudFormationClient interface {
	CreateStack(ctx context.Context, params *cloudformation.CreateStackInput, optFns ...func(*cloudformation.Options)) (*cloudformation.CreateStackOutput, error)
	UpdateStack(ctx context.Context, params *cloudformation.UpdateStackInput, optFns ...func(*cloudformation.Options)) (*cloudformation.UpdateStackOutput, error)
	DeleteStack(ctx context.Context, params *cloudformation.DeleteStackInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DeleteStackOutput, error)
	DescribeStacks(ctx context.Context, params *cloudformation.DescribeStacksInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DescribeStacksOutput, error)
	ListStacks(ctx context.Context, params *cloudformation.ListStacksInput, optFns ...func(*cloudformation.Options)) (*cloudformation.ListStacksOutput, error)
	ValidateTemplate(ctx context.Context, params *cloudformation.ValidateTemplateInput, optFns ...func(*cloudformation.Options)) (*cloudformation.ValidateTemplateOutput, error)
	GetTemplate(ctx context.Context, params *cloudformation.GetTemplateInput, optFns ...func(*cloudformation.Options)) (*cloudformation.GetTemplateOutput, error)
	CreateChangeSet(ctx context.Context, params *cloudformation.CreateChangeSetInput, optFns ...func(*cloudformation.Options)) (*cloudformation.CreateChangeSetOutput, error)
	DeleteChangeSet(ctx context.Context, params *cloudformation.DeleteChangeSetInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DeleteChangeSetOutput, error)
	DescribeChangeSet(ctx context.Context, params *cloudformation.DescribeChangeSetInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DescribeChangeSetOutput, error)
	DescribeStackEvents(ctx context.Context, params *cloudformation.DescribeStackEventsInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DescribeStackEventsOutput, error)
}

// Ensure that the actual CloudFormation client implements our interface
var _ CloudFormationClient = (*cloudformation.Client)(nil)

// Ensure that our DefaultClient implements Client
var _ Client = (*DefaultClient)(nil)

// Ensure that DefaultCloudFormationOperations implements CloudFormationOperations
var _ CloudFormationOperations = (*DefaultCloudFormationOperations)(nil)

// Client defines the interface for AWS client operations
type Client interface {
	NewCloudFormationOperations() CloudFormationOperations
}

// CloudFormationOperations defines the interface for CloudFormation operations
type CloudFormationOperations interface {
	DeployStack(ctx context.Context, input DeployStackInput) error
	DeployStackWithCallback(ctx context.Context, input DeployStackInput, eventCallback func(StackEvent)) error
	UpdateStack(ctx context.Context, input UpdateStackInput) error
	DeleteStack(ctx context.Context, input DeleteStackInput) error
	GetStack(ctx context.Context, stackName string) (*Stack, error)
	ListStacks(ctx context.Context) ([]*Stack, error)
	ValidateTemplate(ctx context.Context, templateBody string) error
	StackExists(ctx context.Context, stackName string) (bool, error)
	GetTemplate(ctx context.Context, stackName string) (string, error)
	DescribeStack(ctx context.Context, stackName string) (*StackInfo, error)
	CreateChangeSet(ctx context.Context, params *cloudformation.CreateChangeSetInput, optFns ...func(*cloudformation.Options)) (*cloudformation.CreateChangeSetOutput, error)
	DeleteChangeSet(ctx context.Context, params *cloudformation.DeleteChangeSetInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DeleteChangeSetOutput, error)
	DescribeChangeSet(ctx context.Context, params *cloudformation.DescribeChangeSetInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DescribeChangeSetOutput, error)
	DescribeStackEvents(ctx context.Context, stackName string) ([]StackEvent, error)
	WaitForStackOperation(ctx context.Context, stackName string, eventCallback func(StackEvent)) error
}
