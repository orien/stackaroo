/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
)

// CloudFormationClientInterface defines the interface for CloudFormation client operations
// This allows for easier testing with mock implementations
type CloudFormationClientInterface interface {
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
}

// Ensure that the actual CloudFormation client implements our interface
var _ CloudFormationClientInterface = (*cloudformation.Client)(nil)

// Ensure that our Client implements ClientInterface
var _ ClientInterface = (*Client)(nil)

// Ensure that CloudFormationOperations implements CloudFormationOperationsInterface
var _ CloudFormationOperationsInterface = (*CloudFormationOperations)(nil)

// ClientInterface defines the interface for AWS client operations
type ClientInterface interface {
	NewCloudFormationOperations() CloudFormationOperationsInterface
}

// CloudFormationOperationsInterface defines the interface for CloudFormation operations
type CloudFormationOperationsInterface interface {
	DeployStack(ctx context.Context, input DeployStackInput) error
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
}
