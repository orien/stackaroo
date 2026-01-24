/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package describe

import (
	"context"
	"errors"
	"testing"
	"time"

	"codeberg.org/orien/stackaroo/internal/aws"
	"codeberg.org/orien/stackaroo/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewStackDescriber(t *testing.T) {
	// Test that NewStackDescriber creates a proper describer instance
	mockFactory, _ := aws.NewMockClientFactoryForRegion("us-east-1")

	describer := NewStackDescriber(mockFactory)

	assert.NotNil(t, describer, "NewStackDescriber should return a non-nil describer")

	// Verify that it's the correct type
	stackDescriber, ok := describer.(*StackDescriber)
	assert.True(t, ok, "NewStackDescriber should return a StackDescriber")
	assert.Equal(t, mockFactory, stackDescriber.clientFactory, "StackDescriber should use the provided client factory")
}

func TestStackDescriber_DescribeStack_Success(t *testing.T) {
	// Test successful stack description retrieval
	mockFactory, mockCFOps := aws.NewMockClientFactoryForRegion("us-east-1")
	describer := NewStackDescriber(mockFactory)

	ctx := context.Background()
	stack := &model.Stack{
		Name:    "test-stack",
		Context: model.NewTestContext("production", "us-east-1", "123456789012"),
	}

	createdTime := time.Date(2025, 1, 15, 10, 30, 45, 0, time.UTC)
	updatedTime := time.Date(2025, 1, 15, 14, 22, 10, 0, time.UTC)

	expectedStackInfo := &aws.StackInfo{
		Name:        "test-stack",
		Status:      aws.StackStatusCreateComplete,
		CreatedTime: &createdTime,
		UpdatedTime: &updatedTime,
		Description: "Test stack description",
		Parameters: map[string]string{
			"Environment":  "production",
			"InstanceType": "t3.medium",
		},
		Outputs: map[string]string{
			"LoadBalancerDNS": "my-app-lb-123.eu-west-1.elb.amazonaws.com",
			"ApplicationURL":  "https://my-app.example.com",
		},
		Tags: map[string]string{
			"Environment": "production",
			"Project":     "my-application",
		},
	}

	// Set up expectations
	mockCFOps.On("DescribeStack", ctx, "test-stack").Return(expectedStackInfo, nil)

	// Execute
	result, err := describer.DescribeStack(ctx, stack)

	// Verify
	require.NoError(t, err, "DescribeStack should succeed")
	assert.NotNil(t, result, "Result should not be nil")

	assert.Equal(t, "test-stack", result.Name)
	assert.Equal(t, "CREATE_COMPLETE", result.Status)
	assert.Equal(t, createdTime, result.CreatedTime)
	assert.Equal(t, &updatedTime, result.UpdatedTime)
	assert.Equal(t, "Test stack description", result.Description)
	assert.Equal(t, "us-east-1", result.Region)

	// Verify parameters conversion
	expectedParams := map[string]string{
		"Environment":  "production",
		"InstanceType": "t3.medium",
	}
	assert.Equal(t, expectedParams, result.Parameters)

	// Verify outputs
	expectedOutputs := map[string]string{
		"LoadBalancerDNS": "my-app-lb-123.eu-west-1.elb.amazonaws.com",
		"ApplicationURL":  "https://my-app.example.com",
	}
	assert.Equal(t, expectedOutputs, result.Outputs)

	// Verify tags
	expectedTags := map[string]string{
		"Environment": "production",
		"Project":     "my-application",
	}
	assert.Equal(t, expectedTags, result.Tags)

	mockCFOps.AssertExpectations(t)
}

func TestStackDescriber_DescribeStack_AWSError(t *testing.T) {
	// Test error handling when AWS operations fail
	mockFactory, mockCFOps := aws.NewMockClientFactoryForRegion("us-east-1")
	describer := NewStackDescriber(mockFactory)

	ctx := context.Background()
	stack := &model.Stack{
		Name:    "failing-stack",
		Context: model.NewTestContext("production", "us-east-1", "123456789012"),
	}

	expectedError := errors.New("AWS CloudFormation error: stack not found")

	// Set up expectations
	mockCFOps.On("DescribeStack", ctx, "failing-stack").Return(nil, expectedError)

	// Execute
	result, err := describer.DescribeStack(ctx, stack)

	// Verify
	assert.Error(t, err, "DescribeStack should return error when AWS operations fail")
	assert.Nil(t, result, "Result should be nil on error")
	assert.Contains(t, err.Error(), "AWS CloudFormation error: stack not found")

	mockCFOps.AssertExpectations(t)
}

func TestStackDescriber_DescribeStack_MinimalData(t *testing.T) {
	// Test with minimal stack information
	mockFactory, mockCFOps := aws.NewMockClientFactoryForRegion("us-east-1")
	describer := NewStackDescriber(mockFactory)

	ctx := context.Background()
	stack := &model.Stack{
		Name:    "minimal-stack",
		Context: model.NewTestContext("dev", "us-east-1", "123456789012"),
	}

	createdTime := time.Date(2025, 1, 15, 10, 30, 45, 0, time.UTC)

	minimalStackInfo := &aws.StackInfo{
		Name:        "minimal-stack",
		Status:      aws.StackStatusUpdateComplete,
		CreatedTime: &createdTime,
		// No UpdatedTime, Description, Parameters, Outputs, or Tags
	}

	// Set up expectations
	mockCFOps.On("DescribeStack", ctx, "minimal-stack").Return(minimalStackInfo, nil)

	// Execute
	result, err := describer.DescribeStack(ctx, stack)

	// Verify
	require.NoError(t, err, "DescribeStack should succeed with minimal data")
	assert.NotNil(t, result, "Result should not be nil")

	assert.Equal(t, "minimal-stack", result.Name)
	assert.Equal(t, "UPDATE_COMPLETE", result.Status)
	assert.Equal(t, createdTime, result.CreatedTime)
	assert.Nil(t, result.UpdatedTime, "UpdatedTime should be nil when not provided")
	assert.Equal(t, "", result.Description, "Description should be empty when not provided")
	assert.Equal(t, "us-east-1", result.Region)

	// Verify empty maps
	assert.Empty(t, result.Parameters, "Parameters should be empty")
	assert.Empty(t, result.Outputs, "Outputs should be empty")
	assert.Empty(t, result.Tags, "Tags should be empty")

	mockCFOps.AssertExpectations(t)
}

func TestDereferenceTime_Success(t *testing.T) {
	// Test time dereferencing
	testTime := time.Date(2025, 1, 15, 10, 30, 45, 0, time.UTC)
	timePtr := &testTime

	result := dereferenceTime(timePtr)

	assert.Equal(t, testTime, result, "dereferenceTime should properly dereference time pointer")
}

func TestDereferenceTime_NilPointer(t *testing.T) {
	// Test time dereferencing with nil pointer
	result := dereferenceTime(nil)

	assert.True(t, result.IsZero(), "dereferenceTime should return zero time for nil pointer")
}

func TestConvertOutputs_Success(t *testing.T) {
	// Test output conversion
	outputs := map[string]string{
		"LoadBalancerDNS": "my-app-lb-123.eu-west-1.elb.amazonaws.com",
		"ApplicationURL":  "https://my-app.example.com",
		"VpcId":           "vpc-87654321",
	}

	result := convertOutputs(outputs)

	assert.Equal(t, outputs, result, "convertOutputs should return the same map")
}

func TestConvertOutputs_NilMap(t *testing.T) {
	// Test output conversion with nil map
	result := convertOutputs(nil)

	assert.NotNil(t, result, "convertOutputs should return non-nil map")
	assert.Empty(t, result, "convertOutputs should return empty map for nil input")
}

func TestConvertOutputs_EmptyMap(t *testing.T) {
	// Test output conversion with empty map
	outputs := make(map[string]string)

	result := convertOutputs(outputs)

	assert.Equal(t, outputs, result, "convertOutputs should return the same empty map")
}

func TestConvertTags_Success(t *testing.T) {
	// Test tag conversion
	tags := map[string]string{
		"Environment": "production",
		"Project":     "my-application",
		"Owner":       "platform-team",
	}

	result := convertTags(tags)

	assert.Equal(t, tags, result, "convertTags should return the same map")
}

func TestConvertTags_NilMap(t *testing.T) {
	// Test tag conversion with nil map
	result := convertTags(nil)

	assert.NotNil(t, result, "convertTags should return non-nil map")
	assert.Empty(t, result, "convertTags should return empty map for nil input")
}

func TestConvertTags_EmptyMap(t *testing.T) {
	// Test tag conversion with empty map
	tags := make(map[string]string)

	result := convertTags(tags)

	assert.Equal(t, tags, result, "convertTags should return the same empty map")
}
