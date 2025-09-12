/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStack_GetTemplateContent(t *testing.T) {
	tests := []struct {
		name         string
		templateBody string
		wantContent  string
		wantError    bool
	}{
		{
			name:         "valid template content",
			templateBody: `{"AWSTemplateFormatVersion": "2010-09-09"}`,
			wantContent:  `{"AWSTemplateFormatVersion": "2010-09-09"}`,
			wantError:    false,
		},
		{
			name:         "empty template content",
			templateBody: "",
			wantContent:  "",
			wantError:    false,
		},
		{
			name:         "yaml template content",
			templateBody: "AWSTemplateFormatVersion: '2010-09-09'",
			wantContent:  "AWSTemplateFormatVersion: '2010-09-09'",
			wantError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rs := &Stack{
				Name:         "test-stack",
				Context:      "dev",
				TemplateBody: tt.templateBody,
				Parameters:   map[string]string{},
				Tags:         map[string]string{},
				Capabilities: []string{},
				Dependencies: []string{},
			}

			content, err := rs.GetTemplateContent()

			if tt.wantError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantContent, content)
			}
		})
	}
}

func TestStack_Creation(t *testing.T) {
	t.Run("create resolved stack with all fields", func(t *testing.T) {
		rs := &Stack{
			Name:         "test-stack",
			Context:      "production",
			TemplateBody: `{"AWSTemplateFormatVersion": "2010-09-09"}`,
			Parameters: map[string]string{
				"Environment":  "prod",
				"InstanceType": "t3.large",
			},
			Tags: map[string]string{
				"Environment": "production",
				"Project":     "stackaroo",
			},
			Capabilities: []string{"CAPABILITY_IAM", "CAPABILITY_NAMED_IAM"},
			Dependencies: []string{"vpc-stack", "security-stack"},
		}

		assert.Equal(t, "test-stack", rs.Name)
		assert.Equal(t, "production", rs.Context)
		assert.Equal(t, `{"AWSTemplateFormatVersion": "2010-09-09"}`, rs.TemplateBody)
		assert.Equal(t, 2, len(rs.Parameters))
		assert.Equal(t, "prod", rs.Parameters["Environment"])
		assert.Equal(t, "t3.large", rs.Parameters["InstanceType"])
		assert.Equal(t, 2, len(rs.Tags))
		assert.Equal(t, "production", rs.Tags["Environment"])
		assert.Equal(t, "stackaroo", rs.Tags["Project"])
		assert.Equal(t, 2, len(rs.Capabilities))
		assert.Contains(t, rs.Capabilities, "CAPABILITY_IAM")
		assert.Contains(t, rs.Capabilities, "CAPABILITY_NAMED_IAM")
		assert.Equal(t, 2, len(rs.Dependencies))
		assert.Contains(t, rs.Dependencies, "vpc-stack")
		assert.Contains(t, rs.Dependencies, "security-stack")
	})

	t.Run("create resolved stack with minimal fields", func(t *testing.T) {
		rs := &Stack{
			Name:         "minimal-stack",
			Context:      "dev",
			TemplateBody: "",
			Parameters:   map[string]string{},
			Tags:         map[string]string{},
			Capabilities: []string{},
			Dependencies: []string{},
		}

		assert.Equal(t, "minimal-stack", rs.Name)
		assert.Equal(t, "dev", rs.Context)
		assert.Equal(t, "", rs.TemplateBody)
		assert.Empty(t, rs.Parameters)
		assert.Empty(t, rs.Tags)
		assert.Empty(t, rs.Capabilities)
		assert.Empty(t, rs.Dependencies)
	})
}

func TestStack_NilMaps(t *testing.T) {
	t.Run("resolved stack with nil maps should work", func(t *testing.T) {
		rs := &Stack{
			Name:         "test-stack",
			Context:      "dev",
			TemplateBody: "test template",
			Parameters:   nil,
			Tags:         nil,
			Capabilities: nil,
			Dependencies: nil,
		}

		// These should all work without panicking
		assert.Equal(t, "test-stack", rs.Name)
		assert.Equal(t, "dev", rs.Context)
		assert.Equal(t, "test template", rs.TemplateBody)
		assert.Nil(t, rs.Parameters)
		assert.Nil(t, rs.Tags)
		assert.Nil(t, rs.Capabilities)
		assert.Nil(t, rs.Dependencies)

		// GetTemplateContent should still work
		content, err := rs.GetTemplateContent()
		require.NoError(t, err)
		assert.Equal(t, "test template", content)
	})
}
