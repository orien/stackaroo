/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package delete

import (
	"context"

	"codeberg.org/orien/stackaroo/internal/model"
	"github.com/stretchr/testify/mock"
)

// MockDeleter implements Deleter for testing
type MockDeleter struct {
	mock.Mock
}

func (m *MockDeleter) DeleteStack(ctx context.Context, stack *model.Stack) error {
	args := m.Called(ctx, stack)
	return args.Error(0)
}

func (m *MockDeleter) DeleteSingleStack(ctx context.Context, stackName, contextName string) error {
	args := m.Called(ctx, stackName, contextName)
	return args.Error(0)
}

func (m *MockDeleter) DeleteAllStacks(ctx context.Context, contextName string) error {
	args := m.Called(ctx, contextName)
	return args.Error(0)
}
