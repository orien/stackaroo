/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"

	"time"

	"github.com/stackaroo/stackaroo/internal/aws"
)

func main() {
	var (
		region    = flag.String("region", "us-east-1", "AWS region")
		profile   = flag.String("profile", "", "AWS profile")
		stackName = flag.String("stack", "stackaroo-test-stack", "Stack name for testing")
		dryRun    = flag.Bool("dry-run", true, "Dry run mode (don't actually create/modify stacks)")
		verbose   = flag.Bool("verbose", false, "Verbose output")
	)
	flag.Parse()

	fmt.Println("🚀 Stackaroo AWS Module Test")
	fmt.Printf("Region: %s\n", *region)
	if *profile != "" {
		fmt.Printf("Profile: %s\n", *profile)
	}
	fmt.Printf("Dry Run: %t\n", *dryRun)
	fmt.Println()

	ctx := context.Background()

	// Test 1: Create AWS Client
	fmt.Println("1️⃣  Testing AWS Client Creation")
	client, err := aws.NewClient(ctx, aws.Config{
		Region:  *region,
		Profile: *profile,
	})
	if err != nil {
		log.Fatalf("❌ Failed to create AWS client: %v", err)
	}
	fmt.Printf("✅ AWS client created successfully for region: %s\n", client.Region())
	fmt.Println()

	// Test 2: Create CloudFormation Operations
	fmt.Println("2️⃣  Testing CloudFormation Operations Creation")
	cfnOps := client.NewCloudFormationOperations()
	if cfnOps == nil {
		log.Fatal("❌ Failed to create CloudFormation operations")
	}
	fmt.Println("✅ CloudFormation operations created successfully")
	fmt.Println()

	// Test 3: Template Validation
	fmt.Println("3️⃣  Testing Template Validation")
	testTemplate := createTestTemplate()
	if *verbose {
		fmt.Printf("Template:\n%s\n", testTemplate)
	}

	err = cfnOps.ValidateTemplate(ctx, testTemplate)
	if err != nil {
		fmt.Printf("❌ Template validation failed: %v\n", err)
	} else {
		fmt.Println("✅ Template validation successful")
	}
	fmt.Println()

	// Test 4: List Existing Stacks
	fmt.Println("4️⃣  Testing Stack Listing")
	stacks, err := cfnOps.ListStacks(ctx)
	if err != nil {
		fmt.Printf("⚠️  Failed to list stacks: %v\n", err)
	} else {
		fmt.Printf("✅ Found %d stacks\n", len(stacks))
		if *verbose && len(stacks) > 0 {
			fmt.Println("Stacks:")
			for i, stack := range stacks {
				if i >= 5 { // Limit to first 5 stacks
					fmt.Printf("  ... and %d more\n", len(stacks)-5)
					break
				}
				fmt.Printf("  - %s (%s)\n", stack.Name, stack.Status)
			}
		}
	}
	fmt.Println()

	// Test 5: Check if Test Stack Exists
	fmt.Println("5️⃣  Testing Stack Existence Check")
	exists, err := cfnOps.StackExists(ctx, *stackName)
	if err != nil {
		fmt.Printf("⚠️  Failed to check stack existence: %v\n", err)
	} else {
		if exists {
			fmt.Printf("✅ Stack '%s' exists\n", *stackName)
		} else {
			fmt.Printf("ℹ️  Stack '%s' does not exist\n", *stackName)
		}
	}
	fmt.Println()

	// Test 6: Get Stack Details (if exists)
	if exists && err == nil {
		fmt.Println("6️⃣  Testing Get Stack Details")
		stack, err := cfnOps.GetStack(ctx, *stackName)
		if err != nil {
			fmt.Printf("❌ Failed to get stack details: %v\n", err)
		} else {
			fmt.Printf("✅ Retrieved stack details for '%s'\n", stack.Name)
			if *verbose {
				printStackDetails(stack)
			}
		}
		fmt.Println()
	}

	// Test 7: Deploy Stack (if dry-run is false)
	if !*dryRun {
		fmt.Println("7️⃣  Testing Stack Deployment")
		if exists {
			fmt.Printf("⚠️  Stack '%s' already exists, skipping deployment\n", *stackName)
		} else {
			fmt.Printf("🚀 Deploying stack '%s'...\n", *stackName)
			err := cfnOps.DeployStack(ctx, aws.DeployStackInput{
				StackName:    *stackName,
				TemplateBody: testTemplate,
				Parameters: []aws.Parameter{
					{Key: "BucketPrefix", Value: "stackaroo-test"},
					{Key: "Environment", Value: "test"},
				},
				Tags: map[string]string{
					"Project":     "stackaroo",
					"Purpose":     "testing",
					"CreatedBy":   "stackaroo-test-program",
					"Environment": "test",
				},
				Capabilities: []string{"CAPABILITY_IAM"},
			})
			if err != nil {
				fmt.Printf("❌ Stack deployment failed: %v\n", err)
			} else {
				fmt.Printf("✅ Stack deployment initiated successfully\n")
				fmt.Printf("⏳ Note: Stack creation is asynchronous. Check AWS console for progress.\n")
			}
		}
		fmt.Println()
	} else {
		fmt.Println("7️⃣  Skipping Stack Deployment (dry-run mode)")
		fmt.Printf("ℹ️  Would deploy stack '%s' with test template\n", *stackName)
		fmt.Println()
	}

	// Test 8: Error Handling Test
	fmt.Println("8️⃣  Testing Error Handling")
	_, err = cfnOps.GetStack(ctx, "non-existent-stack-12345")
	if err != nil {
		fmt.Printf("✅ Error handling works correctly: %s\n", err.Error())
	} else {
		fmt.Printf("⚠️  Expected error for non-existent stack, but got none\n")
	}
	fmt.Println()

	// Summary
	fmt.Println("🎉 AWS Module Test Complete!")
	fmt.Println()
	if *dryRun {
		fmt.Println("💡 To test actual deployment, run with -dry-run=false")
		fmt.Printf("💡 Example: go run cmd/test-aws/main.go -region=%s -dry-run=false\n", *region)
	} else {
		fmt.Println("⚠️  Remember to clean up any test stacks created!")
		fmt.Printf("💡 To delete test stack: aws cloudformation delete-stack --stack-name %s --region %s\n", *stackName, *region)
	}
}

func createTestTemplate() string {
	template := map[string]interface{}{
		"AWSTemplateFormatVersion": "2010-09-09",
		"Description":              "Stackaroo Test Template - Simple S3 bucket for testing",
		"Parameters": map[string]interface{}{
			"BucketPrefix": map[string]interface{}{
				"Type":        "String",
				"Description": "Prefix for the S3 bucket name",
				"Default":     "stackaroo-test",
			},
			"Environment": map[string]interface{}{
				"Type":        "String",
				"Description": "Environment name",
				"Default":     "test",
				"AllowedValues": []string{"dev", "test", "staging", "prod"},
			},
		},
		"Resources": map[string]interface{}{
			"TestBucket": map[string]interface{}{
				"Type": "AWS::S3::Bucket",
				"Properties": map[string]interface{}{
					"BucketName": map[string]interface{}{
						"Fn::Sub": "${BucketPrefix}-${Environment}-${AWS::AccountId}-${AWS::Region}",
					},
					"PublicAccessBlockConfiguration": map[string]interface{}{
						"BlockPublicAcls":       true,
						"BlockPublicPolicy":     true,
						"IgnorePublicAcls":      true,
						"RestrictPublicBuckets": true,
					},
					"BucketEncryption": map[string]interface{}{
						"ServerSideEncryptionConfiguration": []map[string]interface{}{
							{
								"ServerSideEncryptionByDefault": map[string]interface{}{
									"SSEAlgorithm": "AES256",
								},
							},
						},
					},
					"Tags": []map[string]interface{}{
						{
							"Key":   "Project",
							"Value": "stackaroo",
						},
						{
							"Key":   "Purpose",
							"Value": "testing",
						},
						{
							"Key": "Environment",
							"Value": map[string]interface{}{
								"Ref": "Environment",
							},
						},
					},
				},
			},
		},
		"Outputs": map[string]interface{}{
			"BucketName": map[string]interface{}{
				"Description": "Name of the created S3 bucket",
				"Value": map[string]interface{}{
					"Ref": "TestBucket",
				},
				"Export": map[string]interface{}{
					"Name": map[string]interface{}{
						"Fn::Sub": "${AWS::StackName}-BucketName",
					},
				},
			},
			"BucketArn": map[string]interface{}{
				"Description": "ARN of the created S3 bucket",
				"Value": map[string]interface{}{
					"Fn::GetAtt": []string{"TestBucket", "Arn"},
				},
			},
		},
	}

	jsonBytes, err := json.MarshalIndent(template, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal template: %v", err)
	}
	return string(jsonBytes)
}

func printStackDetails(stack *aws.Stack) {
	fmt.Printf("  Name: %s\n", stack.Name)
	fmt.Printf("  Status: %s\n", stack.Status)
	if stack.CreatedTime != nil {
		fmt.Printf("  Created: %s\n", stack.CreatedTime.Format(time.RFC3339))
	}
	if stack.UpdatedTime != nil {
		fmt.Printf("  Updated: %s\n", stack.UpdatedTime.Format(time.RFC3339))
	}
	if stack.Description != "" {
		fmt.Printf("  Description: %s\n", stack.Description)
	}

	if len(stack.Parameters) > 0 {
		fmt.Println("  Parameters:")
		for k, v := range stack.Parameters {
			fmt.Printf("    %s: %s\n", k, v)
		}
	}

	if len(stack.Outputs) > 0 {
		fmt.Println("  Outputs:")
		for k, v := range stack.Outputs {
			fmt.Printf("    %s: %s\n", k, v)
		}
	}

	if len(stack.Tags) > 0 {
		fmt.Println("  Tags:")
		for k, v := range stack.Tags {
			fmt.Printf("    %s: %s\n", k, v)
		}
	}
}
