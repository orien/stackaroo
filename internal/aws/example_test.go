/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package aws_test

import (
	"fmt"

	"github.com/orien/stackaroo/internal/aws"
)

// ExampleClientFactory demonstrates how to create and use the AWS client factory
func ExampleClientFactory() {
	// Example of client factory usage
	// ctx := context.Background()

	// Create a client factory (would use actual AWS credentials in real usage)
	// factory, _ := aws.NewClientFactory(ctx)

	// Example of getting CloudFormation operations for a specific region
	region := "us-east-1"
	// cfOps, _ := factory.GetCloudFormationOperations(ctx, region)

	fmt.Printf("ClientFactory would create operations for region: %s\n", region)
	// Output: ClientFactory would create operations for region: us-east-1
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

// ExampleClientFactory_extensibleDesign demonstrates how to extend for other AWS services
func ExampleClientFactory_extensibleDesign() {
	// Example of how the client factory could be extended
	fmt.Println("CloudFormation operations available")
	fmt.Println("Future ClientFactory methods could include:")
	fmt.Println("- GetS3Operations(ctx, region)")
	fmt.Println("- GetEC2Operations(ctx, region)")
	fmt.Println("- GetIAMOperations(ctx, region)")
	// Output: CloudFormation operations available
	// Future ClientFactory methods could include:
	// - GetS3Operations(ctx, region)
	// - GetEC2Operations(ctx, region)
	// - GetIAMOperations(ctx, region)
}
