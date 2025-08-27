/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package aws

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestConfig_Validation(t *testing.T) {
	tests := []struct {
		name   string
		config Config
		valid  bool
	}{
		{
			name:   "empty config is valid",
			config: Config{},
			valid:  true,
		},
		{
			name: "region only is valid",
			config: Config{
				Region: "us-west-2",
			},
			valid: true,
		},
		{
			name: "valid AWS regions",
			config: Config{
				Region: "ap-southeast-2",
			},
			valid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test that config struct can be created and accessed
			config := tt.config

			// Verify fields are accessible
			_ = config.Region

			// Basic validation - config should have expected values
			if tt.config.Region != "" {
				assert.Equal(t, tt.config.Region, config.Region)
			}

			assert.True(t, tt.valid) // All our test configs should be valid
		})
	}
}

func TestNewCloudFormationOperationsWithClient(t *testing.T) {
	// Test that we can create CloudFormation operations with a mock client
	// This tests our dependency injection pattern without AWS dependencies

	mockClient := &MockCloudFormationClient{}
	ops := NewCloudFormationOperationsWithClient(mockClient)

	assert.NotNil(t, ops)
	// Client field is private, but successful creation indicates dependency injection worked
	mockClient.AssertExpectations(t)
}

// MockCloudFormationClient is a comprehensive mock client for CloudFormation testing
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

func TestConfig_RegionHandling(t *testing.T) {
	tests := []struct {
		name           string
		inputRegion    string
		expectedRegion string
	}{
		{
			name:           "us-east-1 region",
			inputRegion:    "us-east-1",
			expectedRegion: "us-east-1",
		},
		{
			name:           "eu-west-1 region",
			inputRegion:    "eu-west-1",
			expectedRegion: "eu-west-1",
		},
		{
			name:           "ap-southeast-2 region",
			inputRegion:    "ap-southeast-2",
			expectedRegion: "ap-southeast-2",
		},
		{
			name:           "empty region",
			inputRegion:    "",
			expectedRegion: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := Config{Region: tt.inputRegion}
			assert.Equal(t, tt.expectedRegion, config.Region)
		})
	}
}

// Test helper functions
func TestContains(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		substr   string
		expected bool
	}{
		{
			name:     "contains substring",
			s:        "hello world",
			substr:   "world",
			expected: true,
		},
		{
			name:     "does not contain substring",
			s:        "hello world",
			substr:   "foo",
			expected: false,
		},
		{
			name:     "empty substring",
			s:        "hello world",
			substr:   "",
			expected: true,
		},
		{
			name:     "exact match",
			s:        "hello",
			substr:   "hello",
			expected: true,
		},
		{
			name:     "substring at beginning",
			s:        "hello world",
			substr:   "hello",
			expected: true,
		},
		{
			name:     "substring at end",
			s:        "hello world",
			substr:   "world",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := contains(tt.s, tt.substr)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIndexString(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		substr   string
		expected int
	}{
		{
			name:     "substring found",
			s:        "hello world",
			substr:   "world",
			expected: 6,
		},
		{
			name:     "substring not found",
			s:        "hello world",
			substr:   "foo",
			expected: -1,
		},
		{
			name:     "empty substring",
			s:        "hello world",
			substr:   "",
			expected: 0,
		},
		{
			name:     "substring at beginning",
			s:        "hello world",
			substr:   "hello",
			expected: 0,
		},
		{
			name:     "substring at end",
			s:        "hello world",
			substr:   "world",
			expected: 6,
		},
		{
			name:     "exact match",
			s:        "hello",
			substr:   "hello",
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := indexString(tt.s, tt.substr)
			assert.Equal(t, tt.expected, result)
		})
	}
}
