/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package prompt

import (
	"github.com/stretchr/testify/mock"
)

// MockPrompter implements Prompter for testing
type MockPrompter struct {
	mock.Mock
}

// Confirm mock implementation
func (m *MockPrompter) Confirm(message string) (bool, error) {
	args := m.Called(message)
	return args.Bool(0), args.Error(1)
}
