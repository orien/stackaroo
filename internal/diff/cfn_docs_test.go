/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package diff

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetResourceTypeURL(t *testing.T) {
	tests := []struct {
		name         string
		resourceType string
		expected     string
	}{
		{
			name:         "S3 Bucket",
			resourceType: "AWS::S3::Bucket",
			expected:     "https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-s3-bucket.html",
		},
		{
			name:         "EC2 Instance",
			resourceType: "AWS::EC2::Instance",
			expected:     "https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-ec2-instance.html",
		},
		{
			name:         "Lambda Function",
			resourceType: "AWS::Lambda::Function",
			expected:     "https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-lambda-function.html",
		},
		{
			name:         "DynamoDB Table",
			resourceType: "AWS::DynamoDB::Table",
			expected:     "https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-dynamodb-table.html",
		},
		{
			name:         "SQS Queue",
			resourceType: "AWS::SQS::Queue",
			expected:     "https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-sqs-queue.html",
		},
		{
			name:         "IAM Role",
			resourceType: "AWS::IAM::Role",
			expected:     "https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-iam-role.html",
		},
		{
			name:         "CloudWatch Alarm",
			resourceType: "AWS::CloudWatch::Alarm",
			expected:     "https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-cloudwatch-alarm.html",
		},
		{
			name:         "empty string",
			resourceType: "",
			expected:     "",
		},
		{
			name:         "invalid format - missing colons",
			resourceType: "AWS-S3-Bucket",
			expected:     "",
		},
		{
			name:         "invalid format - too few parts",
			resourceType: "AWS::S3",
			expected:     "",
		},
		{
			name:         "invalid format - too many parts",
			resourceType: "AWS::S3::Bucket::Extra",
			expected:     "",
		},
		{
			name:         "invalid format - no colons",
			resourceType: "NotAValidResourceType",
			expected:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetResourceTypeURL(tt.resourceType)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetResourceTypeURL_CaseHandling(t *testing.T) {
	// Verify that resource types are properly lowercased in URLs
	tests := []struct {
		name         string
		resourceType string
		expectedURL  string
	}{
		{
			name:         "mixed case service",
			resourceType: "AWS::DynamoDB::Table",
			expectedURL:  "https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-dynamodb-table.html",
		},
		{
			name:         "uppercase resource",
			resourceType: "AWS::IAM::Role",
			expectedURL:  "https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-iam-role.html",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetResourceTypeURL(tt.resourceType)
			assert.Equal(t, tt.expectedURL, result)
		})
	}
}

func TestHyperlinkResourceType(t *testing.T) {
	tests := []struct {
		name         string
		resourceType string
		shouldLink   bool
	}{
		{
			name:         "valid S3 Bucket",
			resourceType: "AWS::S3::Bucket",
			shouldLink:   true,
		},
		{
			name:         "valid EC2 Instance",
			resourceType: "AWS::EC2::Instance",
			shouldLink:   true,
		},
		{
			name:         "valid Lambda Function",
			resourceType: "AWS::Lambda::Function",
			shouldLink:   true,
		},
		{
			name:         "empty string",
			resourceType: "",
			shouldLink:   false,
		},
		{
			name:         "invalid format",
			resourceType: "NotValid",
			shouldLink:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HyperlinkResourceType(tt.resourceType)

			if tt.shouldLink {
				// Should contain OSC 8 escape sequences
				assert.Contains(t, result, "\033]8;;")
				assert.Contains(t, result, "https://docs.aws.amazon.com")
				assert.Contains(t, result, tt.resourceType)
			} else {
				// Should return original text without escape sequences
				assert.Equal(t, tt.resourceType, result)
				assert.NotContains(t, result, "\033]8;;")
			}
		})
	}
}

func TestHyperlinkResourceType_ExactOutput(t *testing.T) {
	// Test exact output for a known resource type
	resourceType := "AWS::S3::Bucket"
	expected := "\033]8;;https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-s3-bucket.html\033\\AWS::S3::Bucket\033]8;;\033\\"

	result := HyperlinkResourceType(resourceType)
	assert.Equal(t, expected, result)
}
