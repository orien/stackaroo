/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package validate

import (
	"context"

	"github.com/stretchr/testify/mock"
)

// MockValidator is a mock implementation of Validator for testing
type MockValidator struct {
	mock.Mock
}

// ValidateSingleStack mocks the ValidateSingleStack method
func (m *MockValidator) ValidateSingleStack(ctx context.Context, stackName, contextName string) error {
	args := m.Called(ctx, stackName, contextName)
	return args.Error(0)
}

// ValidateAllStacks mocks the ValidateAllStacks method
func (m *MockValidator) ValidateAllStacks(ctx context.Context, contextName string) error {
	args := m.Called(ctx, contextName)
	return args.Error(0)
}
