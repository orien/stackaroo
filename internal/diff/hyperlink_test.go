/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package diff

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHyperlink(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		text     string
		expected string
	}{
		{
			name:     "valid URL and text",
			url:      "https://example.com",
			text:     "Example",
			expected: "\033]8;;https://example.com\033\\Example\033]8;;\033\\",
		},
		{
			name:     "CloudFormation docs URL",
			url:      "https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-s3-bucket.html",
			text:     "AWS::S3::Bucket",
			expected: "\033]8;;https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-s3-bucket.html\033\\AWS::S3::Bucket\033]8;;\033\\",
		},
		{
			name:     "empty URL returns text only",
			url:      "",
			text:     "SomeText",
			expected: "SomeText",
		},
		{
			name:     "empty text returns empty",
			url:      "https://example.com",
			text:     "",
			expected: "",
		},
		{
			name:     "both empty returns empty",
			url:      "",
			text:     "",
			expected: "",
		},
		{
			name:     "URL with query parameters",
			url:      "https://example.com/page?foo=bar&baz=qux",
			text:     "Link with params",
			expected: "\033]8;;https://example.com/page?foo=bar&baz=qux\033\\Link with params\033]8;;\033\\",
		},
		{
			name:     "URL with fragment",
			url:      "https://example.com/page#section",
			text:     "Link with fragment",
			expected: "\033]8;;https://example.com/page#section\033\\Link with fragment\033]8;;\033\\",
		},
		{
			name:     "text with special characters",
			url:      "https://example.com",
			text:     "Hello, World! 🎉",
			expected: "\033]8;;https://example.com\033\\Hello, World! 🎉\033]8;;\033\\",
		},
		{
			name:     "text with spaces",
			url:      "https://example.com",
			text:     "Multiple Words Here",
			expected: "\033]8;;https://example.com\033\\Multiple Words Here\033]8;;\033\\",
		},
		{
			name:     "long URL",
			url:      "https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-elasticloadbalancingv2-targetgroup-targetgroupattribute.html",
			text:     "Very Long Resource Type",
			expected: "\033]8;;https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-elasticloadbalancingv2-targetgroup-targetgroupattribute.html\033\\Very Long Resource Type\033]8;;\033\\",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Hyperlink(tt.url, tt.text)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHyperlink_EscapeSequenceFormat(t *testing.T) {
	// Test that the escape sequence format is exactly correct
	url := "https://example.com"
	text := "Example"
	result := Hyperlink(url, text)

	// Should start with OSC 8 sequence opening
	assert.Contains(t, result, "\033]8;;https://example.com\033\\")

	// Should contain the text
	assert.Contains(t, result, "Example")

	// Should end with OSC 8 sequence closing
	assert.Contains(t, result, "\033]8;;\033\\")

	// Should have the exact structure: opening + text + closing
	expected := "\033]8;;https://example.com\033\\Example\033]8;;\033\\"
	assert.Equal(t, expected, result)
}

func TestHyperlink_NoLeakage(t *testing.T) {
	// Test that empty inputs don't leak escape sequences
	tests := []struct {
		name     string
		url      string
		text     string
		wantText string
	}{
		{
			name:     "empty URL should not add escape codes",
			url:      "",
			text:     "Plain text",
			wantText: "Plain text",
		},
		{
			name:     "empty text should not add escape codes",
			url:      "https://example.com",
			text:     "",
			wantText: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Hyperlink(tt.url, tt.text)
			assert.Equal(t, tt.wantText, result)
			assert.NotContains(t, result, "\033]8;;")
		})
	}
}

func TestHyperlink_MultipleLinks(t *testing.T) {
	// Test creating multiple different links
	link1 := Hyperlink("https://example.com/1", "Link 1")
	link2 := Hyperlink("https://example.com/2", "Link 2")
	link3 := Hyperlink("https://example.com/3", "Link 3")

	assert.Contains(t, link1, "Link 1")
	assert.Contains(t, link1, "https://example.com/1")

	assert.Contains(t, link2, "Link 2")
	assert.Contains(t, link2, "https://example.com/2")

	assert.Contains(t, link3, "Link 3")
	assert.Contains(t, link3, "https://example.com/3")

	// Each should be independent
	assert.NotEqual(t, link1, link2)
	assert.NotEqual(t, link2, link3)
	assert.NotEqual(t, link1, link3)
}
