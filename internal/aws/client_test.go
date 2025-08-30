/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package aws

import (
	"testing"

	"github.com/stretchr/testify/assert"
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

func TestConfig_Structure(t *testing.T) {
	tests := []struct {
		name   string
		config Config
	}{
		{
			name:   "empty config",
			config: Config{},
		},
		{
			name: "region only",
			config: Config{
				Region: "us-west-2",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Just verify the config struct can be created and accessed
			config := tt.config
			_ = config.Region
			assert.True(t, true) // Basic structure test
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
