/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package deploy

import (
	"context"

	"github.com/orien/stackaroo/internal/model"
	"github.com/stretchr/testify/mock"
)

// MockDeployer implements Deployer for testing
type MockDeployer struct {
	mock.Mock
}

func (m *MockDeployer) DeployStack(ctx context.Context, stack *model.Stack) error {
	args := m.Called(ctx, stack)
	return args.Error(0)
}

func (m *MockDeployer) ValidateTemplate(ctx context.Context, templateFile string) error {
	args := m.Called(ctx, templateFile)
	return args.Error(0)
}
