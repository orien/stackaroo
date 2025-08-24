/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package resolve

import (
	"fmt"
	"os"
)

// FileSystemResolver defines the interface for resolving and reading templates from URIs
type FileSystemResolver interface {
	Resolve(fileURI string) (string, error)
}

// DefaultFileSystemResolver implements FileSystemResolver for reading files from `file://` URIs
type DefaultFileSystemResolver struct{}

// Resolve reads template content from a file:// URI
func (fsr *DefaultFileSystemResolver) Resolve(fileURI string) (string, error) {
	filePath, err := parseFileURI(fileURI)
	if err != nil {
		return "", fmt.Errorf("invalid file URI %s: %w", fileURI, err)
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read file %s: %w", filePath, err)
	}
	return string(content), nil
}

// parseFileURI extracts the file path from a file:// URI
func parseFileURI(uri string) (string, error) {
	// Handle file:// scheme
	if len(uri) > 7 && uri[:7] == "file://" {
		return uri[7:], nil
	}

	// Return error for non-file:// URIs
	return "", fmt.Errorf("URI must start with file://, got: %s", uri)
}
