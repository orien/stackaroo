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
	validationError := errors.New("ValidationError: Template format error: Invalid template format")
	mockCfnOps.On("ValidateTemplate", ctx, testStack.TemplateBody).Return(validationError)

	// Create validator
	validator := NewTemplateValidator(mockFactory, mockConfigProvider, mockResolver)

	// Execute
	err := validator.ValidateSingleStack(ctx, stackName, contextName)

	// Verify
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ValidationError")
	assert.Contains(t, err.Error(), "Template format error")
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
	assert.Contains(t, err.Error(), "stack not found in configuration")
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
	mockCfnOps.On("ValidateTemplate", ctx, dbStack.TemplateBody).Return(errors.New("api error ValidationError: Template format error: Unrecognized resource types: [AWS::RDS::InvalidDB]"))

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
	assert.Contains(t, err.Error(), "context not found")
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

func TestTemplateValidator_UserReportedError(t *testing.T) {
	// Test with the exact error format from the user's issue report
	ctx := context.Background()
	stackName := "iam-users"
	contextName := "production"

	// Create test stack
	testStack := &model.Stack{
		Name: stackName,
		Context: &model.Context{
			Name:   contextName,
			Region: "us-east-1",
		},
		TemplateBody: `{"AWSTemplateFormatVersion": "2010-09-09", "Resources": {"Group": {"Type": "AWS::IAM::Groupx"}}}`,
	}

	// Setup mocks
	mockFactory, mockCfnOps := aws.NewMockClientFactoryForRegion("us-east-1")
	mockResolver := &resolve.MockResolver{}
	mockConfigProvider := &config.MockConfigProvider{}

	mockResolver.On("ResolveStack", ctx, contextName, stackName).Return(testStack, nil)

	// This is the exact error format from the user's report
	awsError := errors.New("operation error CloudFormation: ValidateTemplate, https response error StatusCode: 400, RequestID: c3ecc4b3-1c1f-4f25-a338-a913c213b1d1, api error ValidationError: Template format error: Unrecognized resource types: [AWS::IAM::Groupx]")
	mockCfnOps.On("ValidateTemplate", ctx, testStack.TemplateBody).Return(awsError)

	// Create validator
	validator := NewTemplateValidator(mockFactory, mockConfigProvider, mockResolver)

	// Execute
	err := validator.ValidateSingleStack(ctx, stackName, contextName)

	// Verify error is returned
	require.Error(t, err)

	// The error should still be returned, but the user-friendly display
	// should have parsed and presented it nicely (we can't test the printed output easily,
	// but we verify the parsing works in the next test)
	assert.Contains(t, err.Error(), "AWS::IAM::Groupx")
	mockResolver.AssertExpectations(t)
	mockCfnOps.AssertExpectations(t)
}

func TestParseValidationError_UnrecognizedResourceType(t *testing.T) {
	// Test parsing the exact error from user's report
	awsError := errors.New("operation error CloudFormation: ValidateTemplate, https response error StatusCode: 400, RequestID: c3ecc4b3-1c1f-4f25-a338-a913c213b1d1, api error ValidationError: Template format error: Unrecognized resource types: [AWS::IAM::Groupx]")

	issues := parseValidationError(awsError)

	require.Len(t, issues, 1)
	assert.Equal(t, "Invalid Resource Type", issues[0].Title)
	assert.Contains(t, issues[0].Detail, "AWS::IAM::Groupx")
	assert.Contains(t, issues[0].Detail, "not recognized")
}

func TestTemplateValidator_RealisticAWSValidationErrors(t *testing.T) {
	// Test with realistic AWS CloudFormation validation error messages
	ctx := context.Background()
	stackName := "test-stack"
	contextName := "development"

	testCases := []struct {
		name          string
		templateBody  string
		awsError      string
		expectedParts []string
	}{
		{
			name: "invalid resource type",
			templateBody: `{
				"AWSTemplateFormatVersion": "2010-09-09",
				"Resources": {
					"MyResource": {
						"Type": "AWS::EC2::InvalidType"
					}
				}
			}`,
			awsError: "ValidationError: Template format error: Unrecognized resource types: [AWS::EC2::InvalidType]",
			expectedParts: []string{
				"ValidationError",
				"Unrecognized resource types",
				"AWS::EC2::InvalidType",
			},
		},
		{
			name: "invalid parameter type",
			templateBody: `{
				"AWSTemplateFormatVersion": "2010-09-09",
				"Parameters": {
					"MyParam": {
						"Type": "InvalidType"
					}
				}
			}`,
			awsError: "ValidationError: Template format error: Invalid value for parameter type: InvalidType",
			expectedParts: []string{
				"ValidationError",
				"Invalid value for parameter type",
			},
		},
		{
			name: "missing required property",
			templateBody: `{
				"AWSTemplateFormatVersion": "2010-09-09",
				"Resources": {
					"MyBucket": {
						"Type": "AWS::S3::Bucket"
					}
				}
			}`,
			awsError: "ValidationError: Template error: instance of Fn::GetAtt references undefined resource MyOtherBucket",
			expectedParts: []string{
				"ValidationError",
				"undefined resource",
			},
		},
		{
			name:         "syntax error in template",
			templateBody: `{invalid json}`,
			awsError:     "ValidationError: Template format error: JSON not well-formed",
			expectedParts: []string{
				"ValidationError",
				"JSON not well-formed",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create test stack
			testStack := &model.Stack{
				Name: stackName,
				Context: &model.Context{
					Name:   contextName,
					Region: "us-east-1",
				},
				TemplateBody: tc.templateBody,
			}

			// Setup mocks
			mockFactory, mockCfnOps := aws.NewMockClientFactoryForRegion("us-east-1")
			mockResolver := &resolve.MockResolver{}
			mockConfigProvider := &config.MockConfigProvider{}

			mockResolver.On("ResolveStack", ctx, contextName, stackName).Return(testStack, nil)
			mockCfnOps.On("ValidateTemplate", ctx, testStack.TemplateBody).Return(errors.New(tc.awsError))

			// Create validator
			validator := NewTemplateValidator(mockFactory, mockConfigProvider, mockResolver)

			// Execute
			err := validator.ValidateSingleStack(ctx, stackName, contextName)

			// Verify - error should contain AWS error message directly without extra wrapping
			require.Error(t, err)
			for _, expectedPart := range tc.expectedParts {
				assert.Contains(t, err.Error(), expectedPart,
					"Error should contain '%s'. Got: %s", expectedPart, err.Error())
			}

			// Verify the error message is clean - should NOT have double "template validation failed"
			errorMsg := err.Error()
			lastIdx := -1
			searchStr := "template validation failed"

			// Find first occurrence
			if idx := containsAt(errorMsg, searchStr, 0); idx >= 0 {
				// Find second occurrence after the first
				if idx2 := containsAt(errorMsg, searchStr, idx+len(searchStr)); idx2 >= 0 {
					lastIdx = idx2
				}
			}

			// Should not have duplicate wrapping
			assert.Equal(t, -1, lastIdx, "Error message should not contain duplicate 'template validation failed' wrapping. Got: %s", errorMsg)

			mockResolver.AssertExpectations(t)
			mockCfnOps.AssertExpectations(t)
		})
	}
}

// containsAt finds the substring in s starting from position start
func containsAt(s, substr string, start int) int {
	if start >= len(s) {
		return -1
	}
	remaining := s[start:]
	for i := 0; i <= len(remaining)-len(substr); i++ {
		if remaining[i:i+len(substr)] == substr {
			return start + i
		}
	}
	return -1
}
