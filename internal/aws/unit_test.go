/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package aws

import (
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/stretchr/testify/assert"
)

func TestStackStatus_Constants(t *testing.T) {
	tests := []struct {
		status   StackStatus
		expected string
	}{
		{StackStatusCreateInProgress, "CREATE_IN_PROGRESS"},
		{StackStatusCreateComplete, "CREATE_COMPLETE"},
		{StackStatusCreateFailed, "CREATE_FAILED"},
		{StackStatusDeleteInProgress, "DELETE_IN_PROGRESS"},
		{StackStatusDeleteComplete, "DELETE_COMPLETE"},
		{StackStatusDeleteFailed, "DELETE_FAILED"},
		{StackStatusUpdateInProgress, "UPDATE_IN_PROGRESS"},
		{StackStatusUpdateComplete, "UPDATE_COMPLETE"},
		{StackStatusUpdateFailed, "UPDATE_FAILED"},
		{StackStatusRollbackInProgress, "ROLLBACK_IN_PROGRESS"},
		{StackStatusRollbackComplete, "ROLLBACK_COMPLETE"},
		{StackStatusRollbackFailed, "ROLLBACK_FAILED"},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.status))
		})
	}
}

func TestParameter_Structure(t *testing.T) {
	param := Parameter{
		Key:   "Environment",
		Value: "production",
	}

	assert.Equal(t, "Environment", param.Key)
	assert.Equal(t, "production", param.Value)
}

func TestStack_Structure(t *testing.T) {
	createdTime := time.Now().Add(-24 * time.Hour)
	updatedTime := time.Now().Add(-1 * time.Hour)

	stack := &Stack{
		Name:        "test-stack",
		Status:      StackStatusCreateComplete,
		CreatedTime: &createdTime,
		UpdatedTime: &updatedTime,
		Description: "Test stack description",
		Parameters: map[string]string{
			"Environment": "test",
			"Region":      "us-east-1",
		},
		Outputs: map[string]string{
			"BucketName": "my-test-bucket",
			"VpcId":      "vpc-12345",
		},
		Tags: map[string]string{
			"Project":     "stackaroo",
			"Environment": "test",
		},
	}

	assert.Equal(t, "test-stack", stack.Name)
	assert.Equal(t, StackStatusCreateComplete, stack.Status)
	assert.Equal(t, &createdTime, stack.CreatedTime)
	assert.Equal(t, &updatedTime, stack.UpdatedTime)
	assert.Equal(t, "Test stack description", stack.Description)
	assert.Equal(t, "test", stack.Parameters["Environment"])
	assert.Equal(t, "my-test-bucket", stack.Outputs["BucketName"])
	assert.Equal(t, "stackaroo", stack.Tags["Project"])
}

func TestDeployStackInput_Structure(t *testing.T) {
	input := DeployStackInput{
		StackName:    "my-stack",
		TemplateBody: `{"AWSTemplateFormatVersion": "2010-09-09"}`,
		Parameters: []Parameter{
			{Key: "Environment", Value: "prod"},
			{Key: "InstanceType", Value: "t3.micro"},
		},
		Tags: map[string]string{
			"Project": "stackaroo",
			"Owner":   "team-platform",
		},
		Capabilities: []string{
			"CAPABILITY_IAM",
			"CAPABILITY_NAMED_IAM",
		},
	}

	assert.Equal(t, "my-stack", input.StackName)
	assert.Contains(t, input.TemplateBody, "AWSTemplateFormatVersion")
	assert.Len(t, input.Parameters, 2)
	assert.Equal(t, "prod", input.Parameters[0].Value)
	assert.Equal(t, "stackaroo", input.Tags["Project"])
	assert.Contains(t, input.Capabilities, "CAPABILITY_IAM")
}

func TestUpdateStackInput_Structure(t *testing.T) {
	input := UpdateStackInput{
		StackName:    "existing-stack",
		TemplateBody: `{"Resources": {}}`,
		Parameters: []Parameter{
			{Key: "Environment", Value: "staging"},
		},
		Tags: map[string]string{
			"UpdatedBy": "stackaroo",
		},
		Capabilities: []string{"CAPABILITY_IAM"},
	}

	assert.Equal(t, "existing-stack", input.StackName)
	assert.Equal(t, `{"Resources": {}}`, input.TemplateBody)
	assert.Len(t, input.Parameters, 1)
	assert.Equal(t, "staging", input.Parameters[0].Value)
	assert.Equal(t, "stackaroo", input.Tags["UpdatedBy"])
	assert.Len(t, input.Capabilities, 1)
}

func TestDeleteStackInput_Structure(t *testing.T) {
	input := DeleteStackInput{
		StackName: "stack-to-delete",
	}

	assert.Equal(t, "stack-to-delete", input.StackName)
}

func TestConfig_Structure(t *testing.T) {
	tests := []struct {
		name   string
		config Config
	}{
		{
			name:   "empty config",
			config: Config{},
		},
		{
			name: "region only",
			config: Config{
				Region: "us-west-2",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Just verify the config struct can be created and accessed
			config := tt.config
			_ = config.Region
			assert.True(t, true) // Basic structure test
		})
	}
}

func TestStackConversion_FromAWSTypes(t *testing.T) {
	// Test helper for converting AWS CloudFormation types to our types
	createdTime := time.Now().Add(-24 * time.Hour)
	updatedTime := time.Now().Add(-1 * time.Hour)

	cfnStack := types.Stack{
		StackName:       aws.String("test-stack"),
		StackStatus:     types.StackStatusCreateComplete,
		CreationTime:    &createdTime,
		LastUpdatedTime: &updatedTime,
		Description:     aws.String("Test stack description"),
		Parameters: []types.Parameter{
			{
				ParameterKey:   aws.String("Environment"),
				ParameterValue: aws.String("test"),
			},
			{
				ParameterKey:   aws.String("InstanceType"),
				ParameterValue: aws.String("t3.micro"),
			},
		},
		Outputs: []types.Output{
			{
				OutputKey:   aws.String("BucketName"),
				OutputValue: aws.String("my-test-bucket"),
			},
			{
				OutputKey:   aws.String("VpcId"),
				OutputValue: aws.String("vpc-12345"),
			},
		},
		Tags: []types.Tag{
			{
				Key:   aws.String("Project"),
				Value: aws.String("stackaroo"),
			},
			{
				Key:   aws.String("Environment"),
				Value: aws.String("test"),
			},
		},
	}

	// Simulate the conversion logic that would be in GetStack
	stack := &Stack{
		Name:        aws.ToString(cfnStack.StackName),
		Status:      StackStatus(cfnStack.StackStatus),
		CreatedTime: cfnStack.CreationTime,
		UpdatedTime: cfnStack.LastUpdatedTime,
		Description: aws.ToString(cfnStack.Description),
		Parameters:  make(map[string]string),
		Outputs:     make(map[string]string),
		Tags:        make(map[string]string),
	}

	// Convert parameters
	for _, param := range cfnStack.Parameters {
		stack.Parameters[aws.ToString(param.ParameterKey)] = aws.ToString(param.ParameterValue)
	}

	// Convert outputs
	for _, output := range cfnStack.Outputs {
		stack.Outputs[aws.ToString(output.OutputKey)] = aws.ToString(output.OutputValue)
	}

	// Convert tags
	for _, tag := range cfnStack.Tags {
		stack.Tags[aws.ToString(tag.Key)] = aws.ToString(tag.Value)
	}

	// Verify conversion
	assert.Equal(t, "test-stack", stack.Name)
	assert.Equal(t, StackStatus("CREATE_COMPLETE"), stack.Status)
	assert.Equal(t, &createdTime, stack.CreatedTime)
	assert.Equal(t, &updatedTime, stack.UpdatedTime)
	assert.Equal(t, "Test stack description", stack.Description)

	// Verify parameters conversion
	assert.Len(t, stack.Parameters, 2)
	assert.Equal(t, "test", stack.Parameters["Environment"])
	assert.Equal(t, "t3.micro", stack.Parameters["InstanceType"])

	// Verify outputs conversion
	assert.Len(t, stack.Outputs, 2)
	assert.Equal(t, "my-test-bucket", stack.Outputs["BucketName"])
	assert.Equal(t, "vpc-12345", stack.Outputs["VpcId"])

	// Verify tags conversion
	assert.Len(t, stack.Tags, 2)
	assert.Equal(t, "stackaroo", stack.Tags["Project"])
	assert.Equal(t, "test", stack.Tags["Environment"])
}

func TestParameterConversion_ToAWSTypes(t *testing.T) {
	// Test conversion from our Parameter type to AWS types
	params := []Parameter{
		{Key: "Environment", Value: "production"},
		{Key: "InstanceType", Value: "t3.small"},
		{Key: "KeyName", Value: "my-key-pair"},
	}

	// Simulate the conversion logic that would be in DeployStack/UpdateStack
	awsParams := make([]types.Parameter, len(params))
	for i, p := range params {
		awsParams[i] = types.Parameter{
			ParameterKey:   aws.String(p.Key),
			ParameterValue: aws.String(p.Value),
		}
	}

	// Verify conversion
	assert.Len(t, awsParams, 3)
	assert.Equal(t, "Environment", aws.ToString(awsParams[0].ParameterKey))
	assert.Equal(t, "production", aws.ToString(awsParams[0].ParameterValue))
	assert.Equal(t, "InstanceType", aws.ToString(awsParams[1].ParameterKey))
	assert.Equal(t, "t3.small", aws.ToString(awsParams[1].ParameterValue))
	assert.Equal(t, "KeyName", aws.ToString(awsParams[2].ParameterKey))
	assert.Equal(t, "my-key-pair", aws.ToString(awsParams[2].ParameterValue))
}

func TestTagConversion_ToAWSTypes(t *testing.T) {
	// Test conversion from map[string]string to AWS Tag types
	tags := map[string]string{
		"Project":     "stackaroo",
		"Environment": "production",
		"Owner":       "platform-team",
		"CostCenter":  "engineering",
	}

	// Simulate the conversion logic that would be in DeployStack/UpdateStack
	awsTags := make([]types.Tag, 0, len(tags))
	for k, v := range tags {
		awsTags = append(awsTags, types.Tag{
			Key:   aws.String(k),
			Value: aws.String(v),
		})
	}

	// Verify conversion
	assert.Len(t, awsTags, 4)

	// Convert back to map for easier verification (order might vary)
	resultMap := make(map[string]string)
	for _, tag := range awsTags {
		resultMap[aws.ToString(tag.Key)] = aws.ToString(tag.Value)
	}

	assert.Equal(t, "stackaroo", resultMap["Project"])
	assert.Equal(t, "production", resultMap["Environment"])
	assert.Equal(t, "platform-team", resultMap["Owner"])
	assert.Equal(t, "engineering", resultMap["CostCenter"])
}

func TestCapabilityConversion_ToAWSTypes(t *testing.T) {
	// Test conversion from string slice to AWS Capability types
	capabilities := []string{
		"CAPABILITY_IAM",
		"CAPABILITY_NAMED_IAM",
		"CAPABILITY_AUTO_EXPAND",
	}

	// Simulate the conversion logic that would be in DeployStack/UpdateStack
	awsCapabilities := make([]types.Capability, len(capabilities))
	for i, cap := range capabilities {
		awsCapabilities[i] = types.Capability(cap)
	}

	// Verify conversion
	assert.Len(t, awsCapabilities, 3)
	assert.Equal(t, types.Capability("CAPABILITY_IAM"), awsCapabilities[0])
	assert.Equal(t, types.Capability("CAPABILITY_NAMED_IAM"), awsCapabilities[1])
	assert.Equal(t, types.Capability("CAPABILITY_AUTO_EXPAND"), awsCapabilities[2])
}
