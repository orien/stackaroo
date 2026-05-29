/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package resolve

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
)

// FileSystemResolver defines the interface for resolving and reading files from URIs
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

	// Use Lstat (does not follow symlinks) so that symlinks swapped in after
	// path-confinement checks are caught here — covering both dangling symlinks
	// and the TOCTOU window between resolution and read.
	// When resolveTemplatePath successfully resolved a symlink it returns the real
	// target path, so Lstat on that path sees a regular file and proceeds normally.
	fi, err := os.Lstat(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read file %s: %w", filePath, err)
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("template file must not be a symlink: %s", filePath)
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read file %s: %w", filePath, err)
	}
	return string(content), nil
}

// parseFileURI extracts the file path from a file:// URI.
// Requires scheme "file" and an empty host — file://host/path is rejected
// because it silently treats the authority as a path segment.
func parseFileURI(uri string) (string, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return "", fmt.Errorf("invalid file URI %q: %w", uri, err)
	}
	if u.Scheme != "file" {
		return "", fmt.Errorf("URI must start with file://, got: %s", uri)
	}
	if u.Host != "" {
		return "", fmt.Errorf("file URI must not contain a host (use file:///path for absolute paths): %s", uri)
	}
	if !filepath.IsAbs(u.Path) {
		return "", fmt.Errorf("file URI must use an absolute path (use file:///path): %s", uri)
	}
	return filepath.Clean(u.Path), nil
}
