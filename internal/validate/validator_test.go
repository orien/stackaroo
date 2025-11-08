/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package validate

import (
	"context"
	"errors"
	"testing"

	"github.com/orien/stackaroo/internal/aws"
	"github.com/orien/stackaroo/internal/config"
	"github.com/orien/stackaroo/internal/model"
	"github.com/orien/stackaroo/internal/resolve"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTemplateValidator_ValidateSingleStack_Success(t *testing.T) {
	// Test successful validation of a single stack
	ctx := context.Background()
	stackName := "vpc"
	contextName := "development"

	// Create test stack
	testStack := &model.Stack{
		Name: stackName,
		Context: &model.Context{
			Name:   contextName,
			Region: "us-east-1",
		},
		TemplateBody: `{"AWSTemplateFormatVersion": "2010-09-09"}`,
	}

	// Setup mocks
	mockFactory, mockCfnOps := aws.NewMockClientFactoryForRegion("us-east-1")
	mockResolver := &resolve.MockResolver{}
	mockConfigProvider := &config.MockConfigProvider{}

	mockResolver.On("ResolveStack", ctx, contextName, stackName).Return(testStack, nil)
	mockCfnOps.On("ValidateTemplate", ctx, testStack.TemplateBody).Return(nil)

	// Create validator
	validator := NewTemplateValidator(mockFactory, mockConfigProvider, mockResolver)

	// Execute
	err := validator.ValidateSingleStack(ctx, stackName, contextName)

	// Verify
	assert.NoError(t, err)
	mockResolver.AssertExpectations(t)
	mockCfnOps.AssertExpectations(t)
}

func TestTemplateValidator_ValidateSingleStack_InvalidTemplate(t *testing.T) {
	// Test validation failure when template is invalid
	ctx := context.Background()
	stackName := "vpc"
	contextName := "development"

	// Create test stack
	testStack := &model.Stack{
		Name: stackName,
		Context: &model.Context{
			Name:   contextName,
			Region: "us-east-1",
		},
		TemplateBody: `{"invalid": "template"}`,
	}

	// Setup mocks
	mockFactory, mockCfnOps := aws.NewMockClientFactoryForRegion("us-east-1")
	mockResolver := &resolve.MockResolver{}
	mockConfigProvider := &config.MockConfigProvider{}

	mockResolver.On("ResolveStack", ctx, contextName, stackName).Return(testStack, nil)
	validationError := errors.New("template validation failed: Invalid template format")
	mockCfnOps.On("ValidateTemplate", ctx, testStack.TemplateBody).Return(validationError)

	// Create validator
	validator := NewTemplateValidator(mockFactory, mockConfigProvider, mockResolver)

	// Execute
	err := validator.ValidateSingleStack(ctx, stackName, contextName)

	// Verify
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "template validation failed")
	mockResolver.AssertExpectations(t)
	mockCfnOps.AssertExpectations(t)
}

func TestTemplateValidator_ValidateSingleStack_ResolveFailure(t *testing.T) {
	// Test when stack resolution fails
	ctx := context.Background()
	stackName := "nonexistent"
	contextName := "development"

	// Setup mocks
	mockFactory, _ := aws.NewMockClientFactoryForRegion("us-east-1")
	mockResolver := &resolve.MockResolver{}
	mockConfigProvider := &config.MockConfigProvider{}

	resolveError := errors.New("stack not found in configuration")
	mockResolver.On("ResolveStack", ctx, contextName, stackName).Return(nil, resolveError)

	// Create validator
	validator := NewTemplateValidator(mockFactory, mockConfigProvider, mockResolver)

	// Execute
	err := validator.ValidateSingleStack(ctx, stackName, contextName)

	// Verify
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to resolve stack")
	mockResolver.AssertExpectations(t)
}

func TestTemplateValidator_ValidateAllStacks_Success(t *testing.T) {
	// Test successful validation of all stacks
	ctx := context.Background()
	contextName := "development"
	stackNames := []string{"vpc", "database", "app"}

	// Setup mocks
	mockFactory, mockCfnOps := aws.NewMockClientFactoryForRegion("us-east-1")
	mockResolver := &resolve.MockResolver{}
	mockConfigProvider := &config.MockConfigProvider{}

	mockConfigProvider.On("ListStacks", contextName).Return(stackNames, nil)

	// Mock each stack resolution and validation
	for _, stackName := range stackNames {
		testStack := &model.Stack{
			Name: stackName,
			Context: &model.Context{
				Name:   contextName,
				Region: "us-east-1",
			},
			TemplateBody: `{"AWSTemplateFormatVersion": "2010-09-09"}`,
		}
		mockResolver.On("ResolveStack", ctx, contextName, stackName).Return(testStack, nil)
		mockCfnOps.On("ValidateTemplate", ctx, testStack.TemplateBody).Return(nil)
	}

	// Create validator
	validator := NewTemplateValidator(mockFactory, mockConfigProvider, mockResolver)

	// Execute
	err := validator.ValidateAllStacks(ctx, contextName)

	// Verify
	assert.NoError(t, err)
	mockConfigProvider.AssertExpectations(t)
	mockResolver.AssertExpectations(t)
	mockCfnOps.AssertExpectations(t)
}

func TestTemplateValidator_ValidateAllStacks_MixedResults(t *testing.T) {
	// Test validation with some valid and some invalid templates
	ctx := context.Background()
	contextName := "development"
	stackNames := []string{"vpc", "database", "app"}

	// Setup mocks
	mockFactory, mockCfnOps := aws.NewMockClientFactoryForRegion("us-east-1")
	mockResolver := &resolve.MockResolver{}
	mockConfigProvider := &config.MockConfigProvider{}

	mockConfigProvider.On("ListStacks", contextName).Return(stackNames, nil)

	// vpc - valid
	vpcStack := &model.Stack{
		Name: "vpc",
		Context: &model.Context{
			Name:   contextName,
			Region: "us-east-1",
		},
		TemplateBody: `{"AWSTemplateFormatVersion": "2010-09-09"}`,
	}
	mockResolver.On("ResolveStack", ctx, contextName, "vpc").Return(vpcStack, nil)
	mockCfnOps.On("ValidateTemplate", ctx, vpcStack.TemplateBody).Return(nil)

	// database - invalid
	dbStack := &model.Stack{
		Name: "database",
		Context: &model.Context{
			Name:   contextName,
			Region: "us-east-1",
		},
		TemplateBody: `{"invalid": "template"}`,
	}
	mockResolver.On("ResolveStack", ctx, contextName, "database").Return(dbStack, nil)
	mockCfnOps.On("ValidateTemplate", ctx, dbStack.TemplateBody).Return(errors.New("validation failed"))

	// app - valid
	appStack := &model.Stack{
		Name: "app",
		Context: &model.Context{
			Name:   contextName,
			Region: "us-east-1",
		},
		TemplateBody: `{"AWSTemplateFormatVersion": "2010-09-09"}`,
	}
	mockResolver.On("ResolveStack", ctx, contextName, "app").Return(appStack, nil)
	mockCfnOps.On("ValidateTemplate", ctx, appStack.TemplateBody).Return(nil)

	// Create validator
	validator := NewTemplateValidator(mockFactory, mockConfigProvider, mockResolver)

	// Execute
	err := validator.ValidateAllStacks(ctx, contextName)

	// Verify - should return error because one failed
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "validation failed for one or more stacks")
	mockConfigProvider.AssertExpectations(t)
	mockResolver.AssertExpectations(t)
	mockCfnOps.AssertExpectations(t)
}

func TestTemplateValidator_ValidateAllStacks_NoStacks(t *testing.T) {
	// Test validation when no stacks are defined
	ctx := context.Background()
	contextName := "development"

	// Setup mocks
	mockFactory, _ := aws.NewMockClientFactoryForRegion("us-east-1")
	mockResolver := &resolve.MockResolver{}
	mockConfigProvider := &config.MockConfigProvider{}

	mockConfigProvider.On("ListStacks", contextName).Return([]string{}, nil)

	// Create validator
	validator := NewTemplateValidator(mockFactory, mockConfigProvider, mockResolver)

	// Execute
	err := validator.ValidateAllStacks(ctx, contextName)

	// Verify - should succeed with no stacks
	assert.NoError(t, err)
	mockConfigProvider.AssertExpectations(t)
}

func TestTemplateValidator_ValidateAllStacks_ListStacksError(t *testing.T) {
	// Test when listing stacks fails
	ctx := context.Background()
	contextName := "nonexistent"

	// Setup mocks
	mockFactory, _ := aws.NewMockClientFactoryForRegion("us-east-1")
	mockResolver := &resolve.MockResolver{}
	mockConfigProvider := &config.MockConfigProvider{}

	listError := errors.New("context not found")
	mockConfigProvider.On("ListStacks", contextName).Return(nil, listError)

	// Create validator
	validator := NewTemplateValidator(mockFactory, mockConfigProvider, mockResolver)

	// Execute
	err := validator.ValidateAllStacks(ctx, contextName)

	// Verify
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to list stacks")
	mockConfigProvider.AssertExpectations(t)
}

func TestTemplateValidator_ValidateAllStacks_ResolveFailure(t *testing.T) {
	// Test when one stack fails to resolve but others succeed
	ctx := context.Background()
	contextName := "development"
	stackNames := []string{"vpc", "database"}

	// Setup mocks
	mockFactory, mockCfnOps := aws.NewMockClientFactoryForRegion("us-east-1")
	mockResolver := &resolve.MockResolver{}
	mockConfigProvider := &config.MockConfigProvider{}

	mockConfigProvider.On("ListStacks", contextName).Return(stackNames, nil)

	// vpc - resolve fails
	mockResolver.On("ResolveStack", ctx, contextName, "vpc").Return(nil, errors.New("template file not found"))

	// database - succeeds
	dbStack := &model.Stack{
		Name: "database",
		Context: &model.Context{
			Name:   contextName,
			Region: "us-east-1",
		},
		TemplateBody: `{"AWSTemplateFormatVersion": "2010-09-09"}`,
	}
	mockResolver.On("ResolveStack", ctx, contextName, "database").Return(dbStack, nil)
	mockCfnOps.On("ValidateTemplate", ctx, dbStack.TemplateBody).Return(nil)

	// Create validator
	validator := NewTemplateValidator(mockFactory, mockConfigProvider, mockResolver)

	// Execute
	err := validator.ValidateAllStacks(ctx, contextName)

	// Verify - should return error but continue validation
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "validation failed for one or more stacks")
	mockConfigProvider.AssertExpectations(t)
	mockResolver.AssertExpectations(t)
	mockCfnOps.AssertExpectations(t)
}

func TestTemplateValidator_ValidateStack_DifferentRegions(t *testing.T) {
	// Test validation with different AWS regions
	ctx := context.Background()
	stackName := "vpc"
	contextName := "production"

	testCases := []struct {
		name   string
		region string
	}{
		{"us-east-1", "us-east-1"},
		{"eu-west-1", "eu-west-1"},
		{"ap-southeast-2", "ap-southeast-2"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create test stack with specific region
			testStack := &model.Stack{
				Name: stackName,
				Context: &model.Context{
					Name:   contextName,
					Region: tc.region,
				},
				TemplateBody: `{"AWSTemplateFormatVersion": "2010-09-09"}`,
			}

			// Setup mocks
			mockFactory, mockCfnOps := aws.NewMockClientFactoryForRegion(tc.region)
			mockResolver := &resolve.MockResolver{}
			mockConfigProvider := &config.MockConfigProvider{}

			mockResolver.On("ResolveStack", ctx, contextName, stackName).Return(testStack, nil)
			mockCfnOps.On("ValidateTemplate", ctx, testStack.TemplateBody).Return(nil)

			// Create validator
			validator := NewTemplateValidator(mockFactory, mockConfigProvider, mockResolver)

			// Execute
			err := validator.ValidateSingleStack(ctx, stackName, contextName)

			// Verify
			assert.NoError(t, err)
			mockResolver.AssertExpectations(t)
			mockCfnOps.AssertExpectations(t)
		})
	}
}

func TestTemplateValidator_ValidateStack_CloudFormationClientError(t *testing.T) {
	// Test when getting CloudFormation client fails
	ctx := context.Background()
	stackName := "vpc"
	contextName := "development"

	// Create test stack
	testStack := &model.Stack{
		Name: stackName,
		Context: &model.Context{
			Name:   contextName,
			Region: "invalid-region",
		},
		TemplateBody: `{"AWSTemplateFormatVersion": "2010-09-09"}`,
	}

	// Setup mocks
	mockFactory := aws.NewMockClientFactory()
	// Don't set operations for "invalid-region" - this will cause GetCloudFormationOperations to fail
	mockResolver := &resolve.MockResolver{}
	mockConfigProvider := &config.MockConfigProvider{}

	mockResolver.On("ResolveStack", ctx, contextName, stackName).Return(testStack, nil)

	// Create validator
	validator := NewTemplateValidator(mockFactory, mockConfigProvider, mockResolver)

	// Execute
	err := validator.ValidateSingleStack(ctx, stackName, contextName)

	// Verify
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get CloudFormation operations")
	mockResolver.AssertExpectations(t)
}
