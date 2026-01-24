/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package resolve

import (
	"context"

	"codeberg.org/orien/stackaroo/internal/model"
	"github.com/stretchr/testify/mock"
)

// MockFileSystemResolver implements FileSystemResolver for testing
type MockFileSystemResolver struct {
	mock.Mock
}

func (m *MockFileSystemResolver) Resolve(templateURI string) (string, error) {
	args := m.Called(templateURI)
	return args.String(0), args.Error(1)
}

// MockResolver implements Resolver for testing
type MockResolver struct {
	mock.Mock
}

func (m *MockResolver) ResolveStack(ctx context.Context, context string, stackName string) (*model.Stack, error) {
	args := m.Called(ctx, context, stackName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Stack), args.Error(1)
}

func (m *MockResolver) GetDependencyOrder(context string, stackNames []string) ([]string, error) {
	args := m.Called(context, stackNames)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

// MockTemplateProcessor implements TemplateProcessor for testing
type MockTemplateProcessor struct {
	mock.Mock
}

func (m *MockTemplateProcessor) Process(templateContent string, variables map[string]interface{}) (string, error) {
	args := m.Called(templateContent, variables)
	return args.String(0), args.Error(1)
}
