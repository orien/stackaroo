/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package resolve

import (
	"fmt"
	"os"
)

// FileSystemResolver defines the interface for resolving and reading files from `file://` URIs
type FileSystemResolver interface {
	ReadTemplate(fileURI string) (string, error)
}

// DefaultFileSystemResolver implements FileSystemResolver for reading files from `file://` URIs
type DefaultFileSystemResolver struct{}

// ReadTemplate reads template content from a file:// URI
func (fsr *DefaultFileSystemResolver) ReadTemplate(fileURI string) (string, error) {
	filePath, err := parseFileURI(fileURI)
	if err != nil {
		return "", fmt.Errorf("invalid template URI %s: %w", fileURI, err)
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read template file %s: %w", filePath, err)
	}
	return string(content), nil
}

// parseFileURI extracts the file path from a file:// URI or treats as relative path
func parseFileURI(uri string) (string, error) {
	// Handle file:// scheme
	if len(uri) > 7 && uri[:7] == "file://" {
		return uri[7:], nil
	}

	// Handle relative paths as-is for backward compatibility
	return uri, nil
}
