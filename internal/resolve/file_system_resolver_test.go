/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package resolve

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultFileSystemResolver_ReadTemplate_Success(t *testing.T) {
	tests := []struct {
		name        string
		templateURI string
		content     string
	}{
		{
			name:        "file URI with simple template",
			templateURI: "file://templates/vpc.yaml",
			content: `AWSTemplateFormatVersion: '2010-09-09'
Resources:
  VPC:
    Type: AWS::EC2::VPC
    Properties:
      CidrBlock: 10.0.0.0/16`,
		},

		{
			name:        "absolute file URI",
			templateURI: "file:///tmp/absolute-template.yaml",
			content: `AWSTemplateFormatVersion: '2010-09-09'
Description: Absolute path template
Resources: {}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory and file
			tmpDir := t.TempDir()

			var filePath string
			if tt.templateURI == "file:///tmp/absolute-template.yaml" {
				// Special case for absolute path test
				filePath = filepath.Join(tmpDir, "absolute-template.yaml")
				// Update the URI to use the actual temp directory
				tt.templateURI = "file://" + filePath
			} else if strings.HasPrefix(tt.templateURI, "file://") {
				// Extract relative path from file:// URI
				relPath := tt.templateURI[7:] // Remove "file://"
				filePath = filepath.Join(tmpDir, relPath)
			} else {
				// Relative path
				filePath = filepath.Join(tmpDir, tt.templateURI)
			}

			// Create directory if needed
			err := os.MkdirAll(filepath.Dir(filePath), 0755)
			require.NoError(t, err)

			// Write test content to file
			err = os.WriteFile(filePath, []byte(tt.content), 0644)
			require.NoError(t, err)

			// Change to temp directory for relative path resolution
			originalWd, err := os.Getwd()
			require.NoError(t, err)
			defer func() {
				err := os.Chdir(originalWd)
				require.NoError(t, err)
			}()
			err = os.Chdir(tmpDir)
			require.NoError(t, err)

			// Test the resolver
			resolver := &DefaultFileSystemResolver{}
			result, err := resolver.Resolve(tt.templateURI)

			// Assertions
			assert.NoError(t, err)
			assert.Equal(t, tt.content, result)
		})
	}
}

func TestDefaultFileSystemResolver_ReadTemplate_Errors(t *testing.T) {
	tests := []struct {
		name        string
		templateURI string
		setupFunc   func(t *testing.T, tmpDir string) string // Returns the URI to use
		expectedErr string
	}{
		{
			name:        "file not found with file:// URI",
			templateURI: "file://nonexistent/template.yaml",
			setupFunc: func(t *testing.T, tmpDir string) string {
				return "file://" + filepath.Join(tmpDir, "nonexistent/template.yaml")
			},
			expectedErr: "failed to read file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip permission test on Windows as it behaves differently
			if tt.name == "permission denied" && os.Getenv("GOOS") == "windows" {
				t.Skip("Skipping permission test on Windows")
			}

			tmpDir := t.TempDir()

			// Setup test scenario
			templateURI := tt.setupFunc(t, tmpDir)

			// Change to temp directory for relative path resolution
			originalWd, err := os.Getwd()
			require.NoError(t, err)
			defer func() {
				err := os.Chdir(originalWd)
				require.NoError(t, err)
			}()
			err = os.Chdir(tmpDir)
			require.NoError(t, err)

			// Test the resolver
			resolver := &DefaultFileSystemResolver{}
			result, err := resolver.Resolve(templateURI)

			// Assertions
			assert.Error(t, err)
			assert.Empty(t, result)
			assert.Contains(t, err.Error(), tt.expectedErr)
		})
	}
}

func TestDefaultFileSystemResolver_RejectsNonFileURIs(t *testing.T) {
	tests := []struct {
		name        string
		uri         string
		expectedErr string
	}{
		{
			name:        "relative path",
			uri:         "templates/vpc.yaml",
			expectedErr: "URI must start with file://",
		},
		{
			name:        "absolute path",
			uri:         "/home/user/templates/stack.yaml",
			expectedErr: "URI must start with file://",
		},
		{
			name:        "current directory path",
			uri:         "./local-template.yaml",
			expectedErr: "URI must start with file://",
		},
		{
			name:        "parent directory path",
			uri:         "../shared/template.yaml",
			expectedErr: "URI must start with file://",
		},
		{
			name:        "http URI",
			uri:         "http://example.com/template.yaml",
			expectedErr: "URI must start with file://",
		},
		{
			name:        "https URI",
			uri:         "https://example.com/template.yaml",
			expectedErr: "URI must start with file://",
		},
		{
			name:        "s3 URI",
			uri:         "s3://bucket/template.yaml",
			expectedErr: "URI must start with file://",
		},
		{
			name:        "empty URI",
			uri:         "",
			expectedErr: "URI must start with file://",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolver := &DefaultFileSystemResolver{}
			result, err := resolver.Resolve(tt.uri)

			assert.Error(t, err)
			assert.Empty(t, result)
			assert.Contains(t, err.Error(), tt.expectedErr)
		})
	}
}

func TestParseFileURI(t *testing.T) {
	tests := []struct {
		name         string
		uri          string
		expectedPath string
		expectedErr  string
	}{
		{
			name:         "file:// URI with relative path",
			uri:          "file://templates/vpc.yaml",
			expectedPath: "templates/vpc.yaml",
		},
		{
			name:         "file:// URI with absolute path",
			uri:          "file:///usr/local/templates/app.yaml",
			expectedPath: "/usr/local/templates/app.yaml",
		},
		{
			name:         "file:// URI with Windows-style path",
			uri:          "file://C:/templates/database.yaml",
			expectedPath: "C:/templates/database.yaml",
		},
		{
			name:        "relative path without scheme",
			uri:         "templates/service.yaml",
			expectedErr: "URI must start with file://",
		},
		{
			name:        "absolute path without scheme",
			uri:         "/home/user/templates/stack.yaml",
			expectedErr: "URI must start with file://",
		},
		{
			name:        "http URI",
			uri:         "http://example.com/template.yaml",
			expectedErr: "URI must start with file://",
		},
		{
			name:        "s3 URI",
			uri:         "s3://bucket/template.yaml",
			expectedErr: "URI must start with file://",
		},
		{
			name:        "empty URI",
			uri:         "",
			expectedErr: "URI must start with file://",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseFileURI(tt.uri)
			if tt.expectedErr != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErr)
				assert.Empty(t, result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedPath, result)
			}
		})
	}
}

func TestFileSystemResolver_Interface(t *testing.T) {
	// Test that DefaultFileSystemResolver implements FileSystemResolver interface
	var resolver FileSystemResolver = &DefaultFileSystemResolver{}
	assert.NotNil(t, resolver)

	// Test that the interface has the expected method
	assert.NotNil(t, resolver.Resolve)
}

func TestDefaultFileSystemResolver_Integration(t *testing.T) {
	// Integration test with actual file system operations
	tmpDir := t.TempDir()

	// Create a complex directory structure
	templatesDir := filepath.Join(tmpDir, "templates", "nested")
	err := os.MkdirAll(templatesDir, 0755)
	require.NoError(t, err)

	// Create test template with complex content
	templateContent := `AWSTemplateFormatVersion: '2010-09-09'
Description: Integration test template
Parameters:
  Environment:
    Type: String
    Default: test
    AllowedValues: [dev, test, prod]
Resources:
  TestResource:
    Type: AWS::CloudFormation::WaitConditionHandle
    Properties: {}
Outputs:
  ResourceId:
    Value: !Ref TestResource
    Export:
      Name: !Sub "${AWS::StackName}-ResourceId"`

	templatePath := filepath.Join(templatesDir, "integration-template.yaml")
	err = os.WriteFile(templatePath, []byte(templateContent), 0644)
	require.NoError(t, err)

	// Change to temp directory
	originalWd, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		err := os.Chdir(originalWd)
		require.NoError(t, err)
	}()
	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	// Test with file:// URI
	resolver := &DefaultFileSystemResolver{}

	// Test file:// URI
	result, err := resolver.Resolve("file://templates/nested/integration-template.yaml")
	assert.NoError(t, err)
	assert.Equal(t, templateContent, result)
}
