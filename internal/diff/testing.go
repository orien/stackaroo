/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package diff

import (
	"context"

	"github.com/orien/stackaroo/internal/model"
	"github.com/stretchr/testify/mock"
)

// MockDiffer implements Differ for testing
type MockDiffer struct {
	mock.Mock
}

func (m *MockDiffer) DiffStack(ctx context.Context, stack *model.Stack, options Options) (*Result, error) {
	args := m.Called(ctx, stack, options)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Result), args.Error(1)
}

// MockParameterComparator implements ParameterComparator for testing
type MockParameterComparator struct {
	mock.Mock
}

func (m *MockParameterComparator) Compare(currentParams, proposedParams map[string]string) ([]ParameterDiff, error) {
	args := m.Called(currentParams, proposedParams)
	return args.Get(0).([]ParameterDiff), args.Error(1)
}

// MockTagComparator implements TagComparator for testing
type MockTagComparator struct {
	mock.Mock
}

func (m *MockTagComparator) Compare(currentTags, proposedTags map[string]string) ([]TagDiff, error) {
	args := m.Called(currentTags, proposedTags)
	return args.Get(0).([]TagDiff), args.Error(1)
}

// MockTemplateComparator implements TemplateComparator for testing
type MockTemplateComparator struct {
	mock.Mock
}

func (m *MockTemplateComparator) Compare(ctx context.Context, currentTemplate, proposedTemplate string) (*TemplateChange, error) {
	args := m.Called(ctx, currentTemplate, proposedTemplate)
	return args.Get(0).(*TemplateChange), args.Error(1)
}
