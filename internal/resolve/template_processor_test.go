/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package resolve

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCfnTemplateProcessor_Process_BasicSubstitution(t *testing.T) {
	processor := NewCfnTemplateProcessor()

	template := `AWSTemplateFormatVersion: '2010-09-09'
Resources:
  MyInstance:
    Type: AWS::EC2::Instance
    Properties:
      InstanceType: {{ .InstanceType }}
      Environment: {{ .Environment }}`

	variables := map[string]interface{}{
		"InstanceType": "t3.micro",
		"Environment":  "production",
	}

	result, err := processor.Process(template, variables)

	require.NoError(t, err)
	assert.Contains(t, result, "InstanceType: t3.micro")
	assert.Contains(t, result, "Environment: production")
}

func TestCfnTemplateProcessor_Process_MultilineStringInjection(t *testing.T) {
	processor := NewCfnTemplateProcessor()

	template := `Resources:
  WebServer:
    Type: AWS::EC2::Instance
    Properties:
      UserData:
        Fn::Base64: |
{{- .UserDataScript | nindent 10 }}`

	userDataScript := `#!/bin/bash
yum update -y
yum install -y docker
systemctl start docker
systemctl enable docker

# Deploy application
docker pull nginx:latest
docker run -d -p 80:80 nginx`

	variables := map[string]interface{}{
		"UserDataScript": userDataScript,
	}

	result, err := processor.Process(template, variables)

	require.NoError(t, err)
	assert.Contains(t, result, "#!/bin/bash")
	assert.Contains(t, result, "          yum update -y") // Check indentation
	assert.Contains(t, result, "          systemctl start docker")
}

func TestCfnTemplateProcessor_Process_SprigFunctions(t *testing.T) {
	tests := []struct {
		name       string
		template   string
		variables  map[string]interface{}
		assertions func(t *testing.T, result string)
	}{
		{
			name: "string transformation functions",
			template: `Environment: {{ .env | upper }}
Application: {{ .app | title }}
Version: {{ .version | quote }}`,
			variables: map[string]interface{}{
				"env":     "production",
				"app":     "web-server",
				"version": "1.2.3",
			},
			assertions: func(t *testing.T, result string) {
				assert.Contains(t, result, "Environment: PRODUCTION")
				assert.Contains(t, result, "Application: Web-Server")
				assert.Contains(t, result, `Version: "1.2.3"`)
			},
		},
		{
			name: "conditional logic",
			template: `Resources:
{{- if .EnableMonitoring }}
  MonitoringRole:
    Type: AWS::IAM::Role
{{- end }}
  MainResource:
    Type: AWS::EC2::Instance`,
			variables: map[string]interface{}{
				"EnableMonitoring": true,
			},
			assertions: func(t *testing.T, result string) {
				assert.Contains(t, result, "MonitoringRole:")
				assert.Contains(t, result, "MainResource:")
			},
		},
		{
			name: "loops and iteration",
			template: `SecurityGroupIds:
{{- range .SecurityGroups }}
  - {{ . }}
{{- end }}`,
			variables: map[string]interface{}{
				"SecurityGroups": []string{"sg-123", "sg-456", "sg-789"},
			},
			assertions: func(t *testing.T, result string) {
				assert.Contains(t, result, "- sg-123")
				assert.Contains(t, result, "- sg-456")
				assert.Contains(t, result, "- sg-789")
			},
		},
	}

	processor := NewCfnTemplateProcessor()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := processor.Process(tt.template, tt.variables)
			require.NoError(t, err)
			tt.assertions(t, result)
		})
	}
}

func TestCfnTemplateProcessor_Process_ErrorCases(t *testing.T) {
	tests := []struct {
		name        string
		template    string
		variables   map[string]interface{}
		expectedErr string
	}{
		{
			name:        "invalid template syntax",
			template:    `Invalid template: {{ .MissingClosingBrace`,
			variables:   map[string]interface{}{},
			expectedErr: "failed to parse template",
		},
		{
			name:     "template execution error",
			template: `Value: {{ div .numerator .denominator }}`,
			variables: map[string]interface{}{
				"numerator":   10,
				"denominator": 0,
			},
			expectedErr: "failed to execute template",
		},
		{
			name:        "invalid function call",
			template:    `Value: {{ .value | nonExistentFunction }}`,
			variables:   map[string]interface{}{"value": "test"},
			expectedErr: "failed to parse template",
		},
	}

	processor := NewCfnTemplateProcessor()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := processor.Process(tt.template, tt.variables)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedErr)
			assert.Empty(t, result)
		})
	}
}

func TestCfnTemplateProcessor_Process_EdgeCases(t *testing.T) {
	processor := NewCfnTemplateProcessor()

	t.Run("empty template", func(t *testing.T) {
		result, err := processor.Process("", map[string]interface{}{})
		assert.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("template without variables", func(t *testing.T) {
		template := `AWSTemplateFormatVersion: '2010-09-09'
Resources:
  StaticResource:
    Type: AWS::S3::Bucket`

		result, err := processor.Process(template, map[string]interface{}{})
		assert.NoError(t, err)
		assert.Equal(t, template, result)
	})

	t.Run("nil variables", func(t *testing.T) {
		template := `Static content without variables`

		result, err := processor.Process(template, nil)
		assert.NoError(t, err)
		assert.Equal(t, template, result)
	})

	t.Run("empty variables", func(t *testing.T) {
		template := `Environment: {{ .Environment | default "development" }}`

		result, err := processor.Process(template, map[string]interface{}{})
		assert.NoError(t, err)
		assert.Contains(t, result, "Environment: development")
	})
}
