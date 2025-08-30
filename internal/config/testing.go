/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package config

import (
	"context"

	"github.com/stretchr/testify/mock"
)

// MockConfigProvider implements ConfigProvider for testing
type MockConfigProvider struct {
	mock.Mock
}

func (m *MockConfigProvider) LoadConfig(ctx context.Context, context string) (*Config, error) {
	args := m.Called(ctx, context)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Config), args.Error(1)
}

func (m *MockConfigProvider) ListContexts() ([]string, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockConfigProvider) GetStack(stackName, context string) (*StackConfig, error) {
	args := m.Called(stackName, context)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*StackConfig), args.Error(1)
}

func (m *MockConfigProvider) ListStacks(context string) ([]string, error) {
	args := m.Called(context)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockConfigProvider) Validate() error {
	args := m.Called()
	return args.Error(0)
}
