/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package resolve

import (
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
