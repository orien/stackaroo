/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package describe

import (
	"context"
	"fmt"
	"time"

	"github.com/orien/stackaroo/internal/aws"
	"github.com/orien/stackaroo/internal/model"
)

// StackDescriber implements the Describer interface using AWS CloudFormation operations
type StackDescriber struct {
	cfOps aws.CloudFormationOperations
}

// NewStackDescriber creates a new describer with the provided CloudFormation operations
func NewStackDescriber(cfOps aws.CloudFormationOperations) Describer {
	return &StackDescriber{
		cfOps: cfOps,
	}
}

// DescribeStack retrieves comprehensive information about a CloudFormation stack
func (d *StackDescriber) DescribeStack(ctx context.Context, stack *model.Stack) (*StackDescription, error) {
	// Use existing AWS operations to get stack information
	stackInfo, err := d.cfOps.DescribeStack(ctx, stack.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to describe stack %s: %w", stack.Name, err)
	}

	// Convert AWS StackInfo to our StackDescription format
	description := &StackDescription{
		Name:        stackInfo.Name,
		Status:      string(stackInfo.Status),
		CreatedTime: dereferenceTime(stackInfo.CreatedTime),
		UpdatedTime: stackInfo.UpdatedTime,
		Description: stackInfo.Description,
		Parameters:  convertOutputs(stackInfo.Parameters),
		Outputs:     convertOutputs(stackInfo.Outputs),
		Tags:        convertTags(stackInfo.Tags),
		Region:      stack.Context, // Use context as region indicator for now
	}

	// Extract stack ID from the stack information if available
	// For now, we'll use the stack name as ID since StackInfo doesn't expose the full ARN
	description.StackID = stackInfo.Name

	return description, nil
}

// dereferenceTime safely dereferences a time pointer
func dereferenceTime(t *time.Time) time.Time {
	if t == nil {
		return time.Time{}
	}
	return *t
}

// convertOutputs converts AWS outputs to our string map
func convertOutputs(outputs map[string]string) map[string]string {
	if outputs == nil {
		return make(map[string]string)
	}
	return outputs
}

// convertTags converts AWS tags to our string map
func convertTags(tags map[string]string) map[string]string {
	if tags == nil {
		return make(map[string]string)
	}
	return tags
}
