/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package diff

import (
	"fmt"
	"strings"
)

// CloudFormationDocsBaseURL is the base URL for CloudFormation resource documentation
const CloudFormationDocsBaseURL = "https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/"

// GetResourceTypeURL returns the CloudFormation documentation URL for a given resource type.
//
// CloudFormation resource types follow the pattern: AWS::Service::ResourceType
// Documentation URLs follow the pattern: aws-resource-service-resourcetype.html
//
// Parameters:
//   - resourceType: A CloudFormation resource type (e.g., "AWS::S3::Bucket")
//
// Returns the full documentation URL, or an empty string if the resource type
// is invalid or doesn't match the expected format.
//
// Examples:
//
//	GetResourceTypeURL("AWS::S3::Bucket")
//	// Returns: "https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-s3-bucket.html"
//
//	GetResourceTypeURL("AWS::EC2::Instance")
//	// Returns: "https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-ec2-instance.html"
//
//	GetResourceTypeURL("InvalidFormat")
//	// Returns: ""
func GetResourceTypeURL(resourceType string) string {
	if resourceType == "" {
		return ""
	}

	parts := strings.Split(resourceType, "::")
	if len(parts) != 3 {
		return ""
	}

	// parts[0] is "AWS", parts[1] is the service, parts[2] is the resource type
	service := strings.ToLower(parts[1])
	resource := strings.ToLower(parts[2])

	// Most resources use "aws-resource-" prefix
	// Convert to kebab-case URL slug
	urlSlug := fmt.Sprintf("aws-resource-%s-%s.html", service, resource)

	return CloudFormationDocsBaseURL + urlSlug
}

// HyperlinkResourceType creates a clickable hyperlink for a CloudFormation resource type.
//
// This is a convenience function that combines GetResourceTypeURL with Hyperlink
// to create a terminal hyperlink pointing to the CloudFormation documentation
// for the given resource type.
//
// Parameters:
//   - resourceType: A CloudFormation resource type (e.g., "AWS::S3::Bucket")
//
// Returns the resource type text wrapped with hyperlink escape codes, or just
// the original text if the resource type is invalid or empty.
//
// Example:
//
//	link := HyperlinkResourceType("AWS::S3::Bucket")
//	// In supported terminals, displays "AWS::S3::Bucket" as a clickable link
//	// to the S3 Bucket CloudFormation documentation
func HyperlinkResourceType(resourceType string) string {
	url := GetResourceTypeURL(resourceType)
	if url == "" {
		return resourceType
	}
	return Hyperlink(url, resourceType)
}
