/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package describe

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFormatStackDescription_CompleteStack(t *testing.T) {
	// Test formatting with complete stack information
	createdTime := time.Date(2025, 1, 15, 10, 30, 45, 0, time.UTC)
	updatedTime := time.Date(2025, 1, 15, 14, 22, 10, 0, time.UTC)

	desc := &StackDescription{
		Name:        "test-stack",
		Status:      "CREATE_COMPLETE",
		StackID:     "arn:aws:cloudformation:eu-west-1:123456789:stack/test-stack/abc123",
		CreatedTime: createdTime,
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
		Region: "eu-west-1",
	}

	result := FormatStackDescription(desc)

	// Check basic stack information
	assert.Contains(t, result, "Stack: test-stack")
	assert.Contains(t, result, "Status: CREATE_COMPLETE")
	assert.Contains(t, result, "Created: 2025-01-15 10:30:45 UTC")
	assert.Contains(t, result, "Updated: 2025-01-15 14:22:10 UTC")
	assert.Contains(t, result, "Stack ID: arn:aws:cloudformation:eu-west-1:123456789:stack/test-stack/abc123")
	assert.Contains(t, result, "Description: Test stack description")

	// Check parameters section
	assert.Contains(t, result, "Parameters:")
	assert.Contains(t, result, "  Environment: production")
	assert.Contains(t, result, "  InstanceType: t3.medium")

	// Check outputs section
	assert.Contains(t, result, "Outputs:")
	assert.Contains(t, result, "  LoadBalancerDNS: my-app-lb-123.eu-west-1.elb.amazonaws.com")
	assert.Contains(t, result, "  ApplicationURL: https://my-app.example.com")

	// Check tags section
	assert.Contains(t, result, "Tags:")
	assert.Contains(t, result, "  Environment: production")
	assert.Contains(t, result, "  Project: my-application")
}

func TestFormatStackDescription_MinimalStack(t *testing.T) {
	// Test formatting with minimal stack information
	createdTime := time.Date(2025, 1, 15, 10, 30, 45, 0, time.UTC)

	desc := &StackDescription{
		Name:        "minimal-stack",
		Status:      "UPDATE_COMPLETE",
		CreatedTime: createdTime,
		// No UpdatedTime, StackID, Description, Parameters, Outputs, or Tags
		Parameters: map[string]string{},
		Outputs:    map[string]string{},
		Tags:       map[string]string{},
	}

	result := FormatStackDescription(desc)

	// Check basic information is present
	assert.Contains(t, result, "Stack: minimal-stack")
	assert.Contains(t, result, "Status: UPDATE_COMPLETE")
	assert.Contains(t, result, "Created: 2025-01-15 10:30:45 UTC")

	// Check that optional fields are not shown when empty
	assert.NotContains(t, result, "Updated:")
	assert.NotContains(t, result, "Stack ID:")
	assert.NotContains(t, result, "Description:")
	assert.NotContains(t, result, "Parameters:")
	assert.NotContains(t, result, "Outputs:")
	assert.NotContains(t, result, "Tags:")
}

func TestFormatStackDescription_NoUpdatedTime(t *testing.T) {
	// Test formatting when UpdatedTime is nil
	createdTime := time.Date(2025, 1, 15, 10, 30, 45, 0, time.UTC)

	desc := &StackDescription{
		Name:        "new-stack",
		Status:      "CREATE_COMPLETE",
		CreatedTime: createdTime,
		UpdatedTime: nil, // Explicitly nil
		Parameters:  map[string]string{},
		Outputs:     map[string]string{},
		Tags:        map[string]string{},
	}

	result := FormatStackDescription(desc)

	assert.Contains(t, result, "Stack: new-stack")
	assert.Contains(t, result, "Status: CREATE_COMPLETE")
	assert.Contains(t, result, "Created: 2025-01-15 10:30:45 UTC")
	assert.NotContains(t, result, "Updated:")
}

func TestFormatStackDescription_StackIDSameAsName(t *testing.T) {
	// Test that Stack ID is not shown when it's the same as the name
	createdTime := time.Date(2025, 1, 15, 10, 30, 45, 0, time.UTC)

	desc := &StackDescription{
		Name:        "test-stack",
		Status:      "CREATE_COMPLETE",
		StackID:     "test-stack", // Same as name
		CreatedTime: createdTime,
		Parameters:  map[string]string{},
		Outputs:     map[string]string{},
		Tags:        map[string]string{},
	}

	result := FormatStackDescription(desc)

	assert.Contains(t, result, "Stack: test-stack")
	assert.NotContains(t, result, "Stack ID:")
}

func TestFormatStackDescription_EmptyDescription(t *testing.T) {
	// Test that empty description is not shown
	createdTime := time.Date(2025, 1, 15, 10, 30, 45, 0, time.UTC)

	desc := &StackDescription{
		Name:        "test-stack",
		Status:      "CREATE_COMPLETE",
		CreatedTime: createdTime,
		Description: "", // Empty string
		Parameters:  map[string]string{},
		Outputs:     map[string]string{},
		Tags:        map[string]string{},
	}

	result := FormatStackDescription(desc)

	assert.Contains(t, result, "Stack: test-stack")
	assert.NotContains(t, result, "Description:")
}

func TestFormatStackDescription_SortedOutput(t *testing.T) {
	// Test that parameters, outputs, and tags are sorted alphabetically
	createdTime := time.Date(2025, 1, 15, 10, 30, 45, 0, time.UTC)

	desc := &StackDescription{
		Name:        "sorted-stack",
		Status:      "CREATE_COMPLETE",
		CreatedTime: createdTime,
		Parameters: map[string]string{
			"ZetaParam":  "zeta-value",
			"AlphaParam": "alpha-value",
			"BetaParam":  "beta-value",
		},
		Outputs: map[string]string{
			"ZetaOutput":  "zeta-output",
			"AlphaOutput": "alpha-output",
			"BetaOutput":  "beta-output",
		},
		Tags: map[string]string{
			"ZetaTag":  "zeta-tag",
			"AlphaTag": "alpha-tag",
			"BetaTag":  "beta-tag",
		},
	}

	result := FormatStackDescription(desc)

	// Find the positions of each parameter to verify sorting
	alphaParamPos := strings.Index(result, "  AlphaParam: alpha-value")
	betaParamPos := strings.Index(result, "  BetaParam: beta-value")
	zetaParamPos := strings.Index(result, "  ZetaParam: zeta-value")

	assert.True(t, alphaParamPos < betaParamPos, "AlphaParam should come before BetaParam")
	assert.True(t, betaParamPos < zetaParamPos, "BetaParam should come before ZetaParam")

	// Find the positions of each output to verify sorting
	alphaOutputPos := strings.Index(result, "  AlphaOutput: alpha-output")
	betaOutputPos := strings.Index(result, "  BetaOutput: beta-output")
	zetaOutputPos := strings.Index(result, "  ZetaOutput: zeta-output")

	assert.True(t, alphaOutputPos < betaOutputPos, "AlphaOutput should come before BetaOutput")
	assert.True(t, betaOutputPos < zetaOutputPos, "BetaOutput should come before ZetaOutput")

	// Find the positions of each tag to verify sorting
	alphaTagPos := strings.Index(result, "  AlphaTag: alpha-tag")
	betaTagPos := strings.Index(result, "  BetaTag: beta-tag")
	zetaTagPos := strings.Index(result, "  ZetaTag: zeta-tag")

	assert.True(t, alphaTagPos < betaTagPos, "AlphaTag should come before BetaTag")
	assert.True(t, betaTagPos < zetaTagPos, "BetaTag should come before ZetaTag")
}

func TestFormatTime_UTCTime(t *testing.T) {
	// Test time formatting with UTC
	utcTime := time.Date(2025, 1, 15, 10, 30, 45, 0, time.UTC)

	result := formatTime(utcTime)

	assert.Equal(t, "2025-01-15 10:30:45 UTC", result)
}

func TestFormatTime_LocalTime(t *testing.T) {
	// Test time formatting with local timezone
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Skip("Could not load America/New_York timezone for test")
	}

	localTime := time.Date(2025, 1, 15, 10, 30, 45, 0, loc)

	result := formatTime(localTime)

	// Should include the timezone abbreviation
	assert.Contains(t, result, "2025-01-15 10:30:45")
	assert.Contains(t, result, "EST") // or EDT depending on date
}

func TestWriteKeyValueMap_Success(t *testing.T) {
	// Test key-value map writing
	var output strings.Builder
	m := map[string]string{
		"Beta":  "beta-value",
		"Alpha": "alpha-value",
		"Gamma": "gamma-value",
	}

	writeKeyValueMap(&output, m)

	result := output.String()

	// Check that all key-value pairs are present with proper indentation
	assert.Contains(t, result, "  Alpha: alpha-value\n")
	assert.Contains(t, result, "  Beta: beta-value\n")
	assert.Contains(t, result, "  Gamma: gamma-value\n")

	// Check sorting by finding positions
	alphaPos := strings.Index(result, "  Alpha: alpha-value")
	betaPos := strings.Index(result, "  Beta: beta-value")
	gammaPos := strings.Index(result, "  Gamma: gamma-value")

	assert.True(t, alphaPos < betaPos, "Alpha should come before Beta")
	assert.True(t, betaPos < gammaPos, "Beta should come before Gamma")
}

func TestWriteKeyValueMap_EmptyMap(t *testing.T) {
	// Test with empty map
	var output strings.Builder
	m := map[string]string{}

	writeKeyValueMap(&output, m)

	result := output.String()
	assert.Empty(t, result, "Empty map should produce no output")
}

func TestWriteKeyValueMap_NilMap(t *testing.T) {
	// Test with nil map
	var output strings.Builder
	var m map[string]string

	writeKeyValueMap(&output, m)

	result := output.String()
	assert.Empty(t, result, "Nil map should produce no output")
}

func TestWriteKeyValueMap_SingleItem(t *testing.T) {
	// Test with single item
	var output strings.Builder
	m := map[string]string{
		"OnlyKey": "only-value",
	}

	writeKeyValueMap(&output, m)

	result := output.String()
	assert.Equal(t, "  OnlyKey: only-value\n", result)
}

func TestWriteKeyValueMap_SpecialCharacters(t *testing.T) {
	// Test with keys and values containing special characters
	var output strings.Builder
	m := map[string]string{
		"Key With Spaces":    "value with spaces",
		"Key-With-Dashes":    "value-with-dashes",
		"Key.With.Dots":      "value.with.dots",
		"Key/With/Slashes":   "value/with/slashes",
		"Key:With:Colons":    "value:with:colons",
		"KeyWithUnicode-£€¥": "valueWithUnicode-£€¥",
	}

	writeKeyValueMap(&output, m)

	result := output.String()

	// Check that special characters are preserved
	assert.Contains(t, result, "  Key With Spaces: value with spaces")
	assert.Contains(t, result, "  Key-With-Dashes: value-with-dashes")
	assert.Contains(t, result, "  Key.With.Dots: value.with.dots")
	assert.Contains(t, result, "  Key/With/Slashes: value/with/slashes")
	assert.Contains(t, result, "  Key:With:Colons: value:with:colons")
	assert.Contains(t, result, "  KeyWithUnicode-£€¥: valueWithUnicode-£€¥")
}
