/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package resolve

import (
	"os"
	"path/filepath"
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
			name:        "relative path (backward compatibility)",
			templateURI: "templates/database.json",
			content: `{
  "AWSTemplateFormatVersion": "2010-09-09",
  "Resources": {
    "Database": {
      "Type": "AWS::RDS::DBInstance",
      "Properties": {
        "DBInstanceClass": "db.t3.micro"
      }
    }
  }
}`,
		},
		{
			name:        "absolute file URI",
			templateURI: "file:///tmp/absolute-template.yaml",
			content: `AWSTemplateFormatVersion: '2010-09-09'
Description: Absolute path template
Resources: {}`,
		},
		{
			name:        "empty template file",
			templateURI: "empty-template.yaml",
			content:     "",
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
			} else if filepath.HasPrefix(tt.templateURI, "file://") {
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
		{
			name:        "file not found with relative path",
			templateURI: "missing-template.yaml",
			setupFunc: func(t *testing.T, tmpDir string) string {
				return "missing-template.yaml"
			},
			expectedErr: "failed to read file",
		},
		{
			name:        "permission denied",
			templateURI: "restricted-template.yaml",
			setupFunc: func(t *testing.T, tmpDir string) string {
				// Create file with no read permissions
				filePath := filepath.Join(tmpDir, "restricted-template.yaml")
				err := os.WriteFile(filePath, []byte("content"), 0644)
				require.NoError(t, err)
				
				// Remove read permissions
				err = os.Chmod(filePath, 0000)
				require.NoError(t, err)
				
				// Restore permissions after test for cleanup
				t.Cleanup(func() {
					os.Chmod(filePath, 0644)
				})
				
				return "restricted-template.yaml"
			},
			expectedErr: "failed to read file",
		},
		{
			name:        "directory instead of file",
			templateURI: "directory-not-file",
			setupFunc: func(t *testing.T, tmpDir string) string {
				// Create directory with the template name
				dirPath := filepath.Join(tmpDir, "directory-not-file")
				err := os.MkdirAll(dirPath, 0755)
				require.NoError(t, err)
				return "directory-not-file"
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
			name:         "relative path (no scheme)",
			uri:          "templates/service.yaml",
			expectedPath: "templates/service.yaml",
		},
		{
			name:         "absolute path (no scheme)",
			uri:          "/home/user/templates/stack.yaml",
			expectedPath: "/home/user/templates/stack.yaml",
		},
		{
			name:         "current directory path",
			uri:          "./local-template.yaml",
			expectedPath: "./local-template.yaml",
		},
		{
			name:         "parent directory path",
			uri:          "../shared/template.yaml",
			expectedPath: "../shared/template.yaml",
		},
		{
			name:         "empty URI",
			uri:          "",
			expectedPath: "",
		},
		{
			name:         "file:// with empty path",
			uri:          "file://",
			expectedPath: "file://",
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
	
	// Test with different URI formats
	resolver := &DefaultFileSystemResolver{}
	
	// Test relative path
	result1, err := resolver.Resolve("templates/nested/integration-template.yaml")
	assert.NoError(t, err)
	assert.Equal(t, templateContent, result1)
	
	// Test file:// URI
	result2, err := resolver.Resolve("file://templates/nested/integration-template.yaml")
	assert.NoError(t, err)
	assert.Equal(t, templateContent, result2)
	
	// Both should return the same content
	assert.Equal(t, result1, result2)
}