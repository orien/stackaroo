/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package aws_test

import (
	"fmt"

	"github.com/orien/stackaroo/internal/aws"
)

// ExampleClient demonstrates how to create and use the AWS client
func ExampleClient() {
	// Example of client configuration structure
	config := aws.Config{
		Region: "us-east-1",
	}

	fmt.Printf("Client would be configured for region: %s\n", config.Region)
	// Output: Client would be configured for region: us-east-1
}

// ExampleCloudFormationOperations demonstrates CloudFormation operations
func ExampleCloudFormationOperations() {
	// Example template structure
	template := `{
		"AWSTemplateFormatVersion": "2010-09-09",
		"Resources": {
			"MyBucket": {
				"Type": "AWS::S3::Bucket"
			}
		}
	}`

	// Example of deploy stack input structure
	deployInput := aws.DeployStackInput{
		StackName:    "my-example-stack",
		TemplateBody: template,
		Parameters: []aws.Parameter{
			{Key: "Environment", Value: "dev"},
		},
		Tags: map[string]string{
			"Project": "stackaroo-example",
		},
		Capabilities: []string{"CAPABILITY_IAM"},
	}

	fmt.Printf("Stack deployment would use template with %d resources\n", 1)
	fmt.Printf("Stack name: %s\n", deployInput.StackName)
	// Output: Stack deployment would use template with 1 resources
	// Stack name: my-example-stack
}

// ExampleClient_extensibleDesign demonstrates how to extend for other AWS services
func ExampleClient_extensibleDesign() {
	// Example of how the client would be extensible
	fmt.Println("CloudFormation operations available")
	fmt.Println("Future services could include:")
	fmt.Println("- S3Operations")
	fmt.Println("- EC2Operations")
	fmt.Println("- IAMOperations")
	// Output: CloudFormation operations available
	// Future services could include:
	// - S3Operations
	// - EC2Operations
	// - IAMOperations
}
