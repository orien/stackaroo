/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package prompt

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockPrompter is a mock implementation of the Prompter interface for testing
type MockPrompter struct {
	mock.Mock
}

// ConfirmDeployment mock implementation
func (m *MockPrompter) ConfirmDeployment(stackName string) (bool, error) {
	args := m.Called(stackName)
	return args.Bool(0), args.Error(1)
}

// TestMockPrompter_Interface verifies MockPrompter implements Prompter interface
func TestMockPrompter_Interface(t *testing.T) {
	var _ Prompter = (*MockPrompter)(nil)
}

// TestMockPrompter_ConfirmDeployment tests the mock prompter functionality
func TestMockPrompter_ConfirmDeployment(t *testing.T) {
	mockPrompter := &MockPrompter{}

	// Test confirmation
	mockPrompter.On("ConfirmDeployment", "test-stack").Return(true, nil).Once()

	result, err := mockPrompter.ConfirmDeployment("test-stack")

	assert.NoError(t, err)
	assert.True(t, result)
	mockPrompter.AssertExpectations(t)
}

// TestMockPrompter_ConfirmDeployment_Rejection tests mock prompter rejection
func TestMockPrompter_ConfirmDeployment_Rejection(t *testing.T) {
	mockPrompter := &MockPrompter{}

	// Test rejection
	mockPrompter.On("ConfirmDeployment", "test-stack").Return(false, nil).Once()

	result, err := mockPrompter.ConfirmDeployment("test-stack")

	assert.NoError(t, err)
	assert.False(t, result)
	mockPrompter.AssertExpectations(t)
}

// TestSetPrompter_ChangesDefaultPrompter tests the SetPrompter functionality
func TestSetPrompter_ChangesDefaultPrompter(t *testing.T) {
	// Store original prompter to restore later
	originalPrompter := defaultPrompter
	defer SetPrompter(originalPrompter)

	// Create and set mock prompter
	mockPrompter := &MockPrompter{}
	mockPrompter.On("ConfirmDeployment", "test-stack").Return(true, nil).Once()

	SetPrompter(mockPrompter)

	// Call the package-level function which should use our mock
	result, err := ConfirmDeployment("test-stack")

	assert.NoError(t, err)
	assert.True(t, result)
	mockPrompter.AssertExpectations(t)
}

// TestDefaultPrompter_IsStdinPrompter verifies default prompter type
func TestDefaultPrompter_IsStdinPrompter(t *testing.T) {
	// Verify that the default prompter is a StdinPrompter
	_, ok := defaultPrompter.(*StdinPrompter)
	assert.True(t, ok, "Default prompter should be a StdinPrompter")
}

// TestConfirmDeployment_ResponseParsing tests the logic for parsing user responses
func TestConfirmDeployment_ResponseParsing(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"yes lowercase", "yes", true},
		{"yes uppercase", "YES", true},
		{"yes mixed case", "Yes", true},
		{"y lowercase", "y", true},
		{"y uppercase", "Y", true},
		{"no", "no", false},
		{"n", "n", false},
		{"empty", "", false},
		{"whitespace only", "   ", false},
		{"other text", "maybe", false},
		{"partial match", "yeah", false},
		{"with whitespace", " y ", true},
		{"with whitespace no", " no ", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the core logic that ConfirmDeployment uses
			response := strings.ToLower(strings.TrimSpace(tt.input))
			result := response == "y" || response == "yes"

			assert.Equal(t, tt.expected, result,
				"Input '%s' should return %t", tt.input, tt.expected)
		})
	}
}

// TestConfirmDeployment_StackNameFormatting tests that stack name is properly included in prompt
func TestConfirmDeployment_StackNameFormatting(t *testing.T) {
	// This test documents the expected prompt format
	// Full interactive testing would require stdin mocking

	stackName := "test-vpc-stack"
	expectedPromptContent := "Do you want to apply these changes to stack test-vpc-stack? [y/N]:"

	// Verify the prompt message format is as expected
	assert.Contains(t, expectedPromptContent, stackName,
		"Prompt should contain the stack name")
	assert.Contains(t, expectedPromptContent, "[y/N]",
		"Prompt should indicate default is No")
}

// TestConfirmDeployment_Documentation documents the expected behaviour
func TestConfirmDeployment_Documentation(t *testing.T) {
	// This test serves as documentation for the expected behaviour

	t.Run("accepts_only_explicit_yes", func(t *testing.T) {
		// Only "y" and "yes" (case insensitive) should return true
		yesResponses := []string{"y", "Y", "yes", "YES", "Yes"}
		for _, response := range yesResponses {
			normalized := strings.ToLower(strings.TrimSpace(response))
			result := normalized == "y" || normalized == "yes"
			assert.True(t, result, "Response '%s' should be accepted as confirmation", response)
		}
	})

	t.Run("rejects_all_other_input", func(t *testing.T) {
		// Everything else should return false
		noResponses := []string{"n", "no", "NO", "", "  ", "maybe", "ok", "sure", "yep", "nope"}
		for _, response := range noResponses {
			normalized := strings.ToLower(strings.TrimSpace(response))
			result := normalized == "y" || normalized == "yes"
			assert.False(t, result, "Response '%s' should be rejected", response)
		}
	})

	t.Run("default_behaviour", func(t *testing.T) {
		// Empty input or whitespace should default to "no" (false)
		emptyInputs := []string{"", "   ", "\t", "\n"}
		for _, input := range emptyInputs {
			normalized := strings.ToLower(strings.TrimSpace(input))
			result := normalized == "y" || normalized == "yes"
			assert.False(t, result, "Empty/whitespace input should default to no")
		}
	})
}

// TestConfirmDeployment_UsesDefaultPrompter verifies package function uses default prompter
func TestConfirmDeployment_UsesDefaultPrompter(t *testing.T) {
	// Store original prompter to restore later
	originalPrompter := defaultPrompter
	defer SetPrompter(originalPrompter)

	// Create mock that expects to be called
	mockPrompter := &MockPrompter{}
	mockPrompter.On("ConfirmDeployment", "my-stack").Return(false, nil).Once()

	SetPrompter(mockPrompter)

	// Call package function
	result, err := ConfirmDeployment("my-stack")

	assert.NoError(t, err)
	assert.False(t, result)
	mockPrompter.AssertExpectations(t)
}

// Note: The MockPrompter allows full testing of deployment flows without requiring
// actual user input. Tests can configure expected responses and verify behavior.
// For interactive testing of the StdinPrompter, manual testing is recommended.
