/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package diff

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestYAMLTemplateComparator_Compare_IdenticalTemplates(t *testing.T) {
	comparator := NewYAMLTemplateComparator()
	ctx := context.Background()

	template := `AWSTemplateFormatVersion: '2010-09-09'
Resources:
  MyBucket:
    Type: AWS::S3::Bucket
    Properties:
      BucketName: test-bucket`

	result, err := comparator.Compare(ctx, template, template)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.HasChanges)
	assert.Equal(t, result.CurrentHash, result.ProposedHash)
	assert.NotEmpty(t, result.CurrentHash)
	assert.NotEmpty(t, result.ProposedHash)
}

func TestYAMLTemplateComparator_Compare_DifferentTemplates(t *testing.T) {
	comparator := NewYAMLTemplateComparator()
	ctx := context.Background()

	currentTemplate := `AWSTemplateFormatVersion: '2010-09-09'
Resources:
  MyBucket:
    Type: AWS::S3::Bucket`

	proposedTemplate := `AWSTemplateFormatVersion: '2010-09-09'
Resources:
  MyBucket:
    Type: AWS::S3::Bucket
    Properties:
      BucketName: test-bucket`

	result, err := comparator.Compare(ctx, currentTemplate, proposedTemplate)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.HasChanges)
	assert.NotEqual(t, result.CurrentHash, result.ProposedHash)
	assert.NotEmpty(t, result.Diff)
}

func TestYAMLTemplateComparator_Compare_ResourceCounting_Added(t *testing.T) {
	comparator := NewYAMLTemplateComparator()
	ctx := context.Background()

	currentTemplate := `AWSTemplateFormatVersion: '2010-09-09'
Resources:
  MyBucket:
    Type: AWS::S3::Bucket`

	proposedTemplate := `AWSTemplateFormatVersion: '2010-09-09'
Resources:
  MyBucket:
    Type: AWS::S3::Bucket
  MyQueue:
    Type: AWS::SQS::Queue
  MyTopic:
    Type: AWS::SNS::Topic`

	result, err := comparator.Compare(ctx, currentTemplate, proposedTemplate)

	require.NoError(t, err)
	assert.True(t, result.HasChanges)
	assert.Equal(t, 2, result.ResourceCount.Added)
	assert.Equal(t, 0, result.ResourceCount.Modified)
	assert.Equal(t, 0, result.ResourceCount.Removed)
}

func TestYAMLTemplateComparator_Compare_ResourceCounting_Removed(t *testing.T) {
	comparator := NewYAMLTemplateComparator()
	ctx := context.Background()

	currentTemplate := `AWSTemplateFormatVersion: '2010-09-09'
Resources:
  MyBucket:
    Type: AWS::S3::Bucket
  MyQueue:
    Type: AWS::SQS::Queue
  MyTopic:
    Type: AWS::SNS::Topic`

	proposedTemplate := `AWSTemplateFormatVersion: '2010-09-09'
Resources:
  MyBucket:
    Type: AWS::S3::Bucket`

	result, err := comparator.Compare(ctx, currentTemplate, proposedTemplate)

	require.NoError(t, err)
	assert.True(t, result.HasChanges)
	assert.Equal(t, 0, result.ResourceCount.Added)
	assert.Equal(t, 0, result.ResourceCount.Modified)
	assert.Equal(t, 2, result.ResourceCount.Removed)
}

func TestYAMLTemplateComparator_Compare_ResourceCounting_Modified(t *testing.T) {
	comparator := NewYAMLTemplateComparator()
	ctx := context.Background()

	currentTemplate := `AWSTemplateFormatVersion: '2010-09-09'
Resources:
  MyBucket:
    Type: AWS::S3::Bucket
  MyQueue:
    Type: AWS::SQS::Queue`

	proposedTemplate := `AWSTemplateFormatVersion: '2010-09-09'
Resources:
  MyBucket:
    Type: AWS::S3::Bucket
    Properties:
      BucketName: test-bucket
  MyQueue:
    Type: AWS::SQS::Queue
    Properties:
      QueueName: test-queue`

	result, err := comparator.Compare(ctx, currentTemplate, proposedTemplate)

	require.NoError(t, err)
	assert.True(t, result.HasChanges)
	assert.Equal(t, 0, result.ResourceCount.Added)
	assert.Equal(t, 2, result.ResourceCount.Modified)
	assert.Equal(t, 0, result.ResourceCount.Removed)
}

func TestYAMLTemplateComparator_Compare_ResourceCounting_Mixed(t *testing.T) {
	comparator := NewYAMLTemplateComparator()
	ctx := context.Background()

	currentTemplate := `AWSTemplateFormatVersion: '2010-09-09'
Resources:
  MyBucket:
    Type: AWS::S3::Bucket
  OldQueue:
    Type: AWS::SQS::Queue
  MyTopic:
    Type: AWS::SNS::Topic`

	proposedTemplate := `AWSTemplateFormatVersion: '2010-09-09'
Resources:
  MyBucket:
    Type: AWS::S3::Bucket
    Properties:
      BucketName: test-bucket
  NewQueue:
    Type: AWS::SQS::Queue
  MyTopic:
    Type: AWS::SNS::Topic`

	result, err := comparator.Compare(ctx, currentTemplate, proposedTemplate)

	require.NoError(t, err)
	assert.True(t, result.HasChanges)
	assert.Equal(t, 1, result.ResourceCount.Added)    // NewQueue
	assert.Equal(t, 1, result.ResourceCount.Modified) // MyBucket
	assert.Equal(t, 1, result.ResourceCount.Removed)  // OldQueue
}

func TestYAMLTemplateComparator_Compare_InvalidCurrentTemplate(t *testing.T) {
	comparator := NewYAMLTemplateComparator()
	ctx := context.Background()

	invalidTemplate := `{invalid yaml: structure`
	validTemplate := `AWSTemplateFormatVersion: '2010-09-09'`

	result, err := comparator.Compare(ctx, invalidTemplate, validTemplate)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to parse current template")
}

func TestYAMLTemplateComparator_Compare_InvalidProposedTemplate(t *testing.T) {
	comparator := NewYAMLTemplateComparator()
	ctx := context.Background()

	validTemplate := `AWSTemplateFormatVersion: '2010-09-09'`
	invalidTemplate := `{invalid yaml: structure`

	result, err := comparator.Compare(ctx, validTemplate, invalidTemplate)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to parse proposed template")
}

func TestYAMLTemplateComparator_Compare_EmptyTemplates(t *testing.T) {
	comparator := NewYAMLTemplateComparator()
	ctx := context.Background()

	result, err := comparator.Compare(ctx, "", "")

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.HasChanges)
	assert.Equal(t, result.CurrentHash, result.ProposedHash)
}

func TestYAMLTemplateComparator_Compare_TemplateWithoutResources(t *testing.T) {
	comparator := NewYAMLTemplateComparator()
	ctx := context.Background()

	currentTemplate := `AWSTemplateFormatVersion: '2010-09-09'
Description: Test template`

	proposedTemplate := `AWSTemplateFormatVersion: '2010-09-09'
Description: Updated test template`

	result, err := comparator.Compare(ctx, currentTemplate, proposedTemplate)

	require.NoError(t, err)
	assert.True(t, result.HasChanges)
	assert.Equal(t, 0, result.ResourceCount.Added)
	assert.Equal(t, 0, result.ResourceCount.Modified)
	assert.Equal(t, 0, result.ResourceCount.Removed)
}

func TestYAMLTemplateComparator_CalculateHash(t *testing.T) {
	comparator := &YAMLTemplateComparator{}

	tests := []struct {
		name     string
		template string
	}{
		{
			name:     "simple template",
			template: "AWSTemplateFormatVersion: '2010-09-09'",
		},
		{
			name:     "empty template",
			template: "",
		},
		{
			name:     "template with whitespace",
			template: "  AWSTemplateFormatVersion: '2010-09-09'  \n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := comparator.calculateHash(tt.template)
			assert.Len(t, result, 12, "Hash should be 12 characters")
			assert.Regexp(t, "^[0-9a-f]+$", result, "Hash should be hexadecimal")
		})
	}

	// Test consistency - same input should produce same output
	template := "AWSTemplateFormatVersion: '2010-09-09'"
	hash1 := comparator.calculateHash(template)
	hash2 := comparator.calculateHash(template)
	assert.Equal(t, hash1, hash2, "Hash should be consistent")

	// Test normalization - whitespace should be normalized
	template1 := "AWSTemplateFormatVersion: '2010-09-09'"
	template2 := "  AWSTemplateFormatVersion: '2010-09-09'  \n"
	hash1 = comparator.calculateHash(template1)
	hash2 = comparator.calculateHash(template2)
	assert.Equal(t, hash1, hash2, "Whitespace should be normalized")
}

func TestYAMLTemplateComparator_CalculateHash_Consistency(t *testing.T) {
	comparator := &YAMLTemplateComparator{}
	template := `AWSTemplateFormatVersion: '2010-09-09'
Resources:
  MyBucket:
    Type: AWS::S3::Bucket`

	hash1 := comparator.calculateHash(template)
	hash2 := comparator.calculateHash(template)

	assert.Equal(t, hash1, hash2, "Hash should be consistent")
	assert.Len(t, hash1, 12, "Hash should be 12 characters")
}

func TestYAMLTemplateComparator_GetResourcesSection(t *testing.T) {
	comparator := &YAMLTemplateComparator{}

	tests := []struct {
		name         string
		templateData map[string]interface{}
		expected     map[string]interface{}
	}{
		{
			name: "template with resources",
			templateData: map[string]interface{}{
				"AWSTemplateFormatVersion": "2010-09-09",
				"Resources": map[string]interface{}{
					"MyBucket": map[string]interface{}{
						"Type": "AWS::S3::Bucket",
					},
				},
			},
			expected: map[string]interface{}{
				"MyBucket": map[string]interface{}{
					"Type": "AWS::S3::Bucket",
				},
			},
		},
		{
			name: "template without resources",
			templateData: map[string]interface{}{
				"AWSTemplateFormatVersion": "2010-09-09",
			},
			expected: map[string]interface{}{},
		},
		{
			name:         "empty template",
			templateData: map[string]interface{}{},
			expected:     map[string]interface{}{},
		},
		{
			name: "resources with wrong type",
			templateData: map[string]interface{}{
				"Resources": "not a map",
			},
			expected: map[string]interface{}{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := comparator.getResourcesSection(tt.templateData)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestYAMLTemplateComparator_GetResourceType(t *testing.T) {
	comparator := &YAMLTemplateComparator{}

	tests := []struct {
		name     string
		resource interface{}
		expected string
	}{
		{
			name: "valid resource",
			resource: map[string]interface{}{
				"Type": "AWS::S3::Bucket",
				"Properties": map[string]interface{}{
					"BucketName": "test",
				},
			},
			expected: "AWS::S3::Bucket",
		},
		{
			name: "resource without type",
			resource: map[string]interface{}{
				"Properties": map[string]interface{}{
					"BucketName": "test",
				},
			},
			expected: "Unknown",
		},
		{
			name: "resource with non-string type",
			resource: map[string]interface{}{
				"Type": 123,
			},
			expected: "Unknown",
		},
		{
			name:     "non-map resource",
			resource: "not a map",
			expected: "Unknown",
		},
		{
			name:     "nil resource",
			resource: nil,
			expected: "Unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := comparator.getResourceType(tt.resource)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestYAMLTemplateComparator_HasSectionChanged(t *testing.T) {
	comparator := &YAMLTemplateComparator{}

	tests := []struct {
		name         string
		currentData  map[string]interface{}
		proposedData map[string]interface{}
		sectionName  string
		expected     bool
	}{
		{
			name: "section added",
			currentData: map[string]interface{}{
				"AWSTemplateFormatVersion": "2010-09-09",
			},
			proposedData: map[string]interface{}{
				"AWSTemplateFormatVersion": "2010-09-09",
				"Description":              "Test template",
			},
			sectionName: "Description",
			expected:    true,
		},
		{
			name: "section removed",
			currentData: map[string]interface{}{
				"AWSTemplateFormatVersion": "2010-09-09",
				"Description":              "Test template",
			},
			proposedData: map[string]interface{}{
				"AWSTemplateFormatVersion": "2010-09-09",
			},
			sectionName: "Description",
			expected:    true,
		},
		{
			name: "section modified",
			currentData: map[string]interface{}{
				"Description": "Old description",
			},
			proposedData: map[string]interface{}{
				"Description": "New description",
			},
			sectionName: "Description",
			expected:    true,
		},
		{
			name: "section unchanged",
			currentData: map[string]interface{}{
				"Description": "Same description",
			},
			proposedData: map[string]interface{}{
				"Description": "Same description",
			},
			sectionName: "Description",
			expected:    false,
		},
		{
			name: "section not in either template",
			currentData: map[string]interface{}{
				"AWSTemplateFormatVersion": "2010-09-09",
			},
			proposedData: map[string]interface{}{
				"AWSTemplateFormatVersion": "2010-09-09",
			},
			sectionName: "NonExistent",
			expected:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := comparator.hasSectionChanged(tt.currentData, tt.proposedData, tt.sectionName)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestYAMLTemplateComparator_GenerateDiff_TemplateStructure(t *testing.T) {
	comparator := &YAMLTemplateComparator{}

	currentTemplate := `AWSTemplateFormatVersion: "2010-09-09"
Resources:
  MyBucket:
    Type: AWS::S3::Bucket`

	proposedTemplate := `AWSTemplateFormatVersion: "2010-09-09"
Description: Test template
Resources:
  MyBucket:
    Type: AWS::S3::Bucket
    Properties:
      BucketName: test-bucket
  MyQueue:
    Type: AWS::SQS::Queue`

	result, err := comparator.generateDiff(currentTemplate, proposedTemplate)

	require.NoError(t, err)
	// Check for unified diff format
	assert.Contains(t, result, "@@") // Unified diff hunk header
	assert.Contains(t, result, "+Description: Test template")
	assert.Contains(t, result, "+    Properties:")
	assert.Contains(t, result, "+      BucketName: test-bucket")
	assert.Contains(t, result, "+  MyQueue:")
	assert.Contains(t, result, "+    Type: AWS::SQS::Queue")
	assert.Contains(t, result, " AWSTemplateFormatVersion:") // Context line
}

func TestYAMLTemplateComparator_GenerateResourceDiff(t *testing.T) {
	comparator := &YAMLTemplateComparator{}

	currentData := map[string]interface{}{
		"Resources": map[string]interface{}{
			"MyBucket": map[string]interface{}{
				"Type": "AWS::S3::Bucket",
			},
			"OldQueue": map[string]interface{}{
				"Type": "AWS::SQS::Queue",
			},
		},
	}

	proposedData := map[string]interface{}{
		"Resources": map[string]interface{}{
			"MyBucket": map[string]interface{}{
				"Type": "AWS::S3::Bucket",
				"Properties": map[string]interface{}{
					"BucketName": "test-bucket",
				},
			},
			"NewTopic": map[string]interface{}{
				"Type": "AWS::SNS::Topic",
			},
		},
	}

	result := comparator.generateResourceDiff(currentData, proposedData)

	assert.Contains(t, result, "+ NewTopic (AWS::SNS::Topic)")
	assert.Contains(t, result, "- OldQueue (AWS::SQS::Queue)")
	assert.Contains(t, result, "~ MyBucket (AWS::S3::Bucket)")
}

func TestYAMLTemplateComparator_JSONTemplate(t *testing.T) {
	comparator := NewYAMLTemplateComparator()
	ctx := context.Background()

	currentTemplate := `{
		"AWSTemplateFormatVersion": "2010-09-09",
		"Resources": {
			"MyBucket": {
				"Type": "AWS::S3::Bucket"
			}
		}
	}`

	proposedTemplate := `{
		"AWSTemplateFormatVersion": "2010-09-09",
		"Resources": {
			"MyBucket": {
				"Type": "AWS::S3::Bucket",
				"Properties": {
					"BucketName": "test-bucket"
				}
			}
		}
	}`

	result, err := comparator.Compare(ctx, currentTemplate, proposedTemplate)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.HasChanges)
	assert.Equal(t, 0, result.ResourceCount.Added)
	assert.Equal(t, 1, result.ResourceCount.Modified)
	assert.Equal(t, 0, result.ResourceCount.Removed)
}

func TestYAMLTemplateComparator_ComplexResourceChanges(t *testing.T) {
	comparator := NewYAMLTemplateComparator()
	ctx := context.Background()

	currentTemplate := `AWSTemplateFormatVersion: '2010-09-09'
Resources:
  VPC:
    Type: AWS::EC2::VPC
    Properties:
      CidrBlock: 10.0.0.0/16
  Subnet:
    Type: AWS::EC2::Subnet
    Properties:
      VpcId: !Ref VPC
      CidrBlock: 10.0.1.0/24
  OldResource:
    Type: AWS::S3::Bucket`

	proposedTemplate := `AWSTemplateFormatVersion: '2010-09-09'
Resources:
  VPC:
    Type: AWS::EC2::VPC
    Properties:
      CidrBlock: 10.0.0.0/16
      EnableDnsHostnames: true
  Subnet:
    Type: AWS::EC2::Subnet
    Properties:
      VpcId: !Ref VPC
      CidrBlock: 10.0.1.0/24
  NewResource:
    Type: AWS::SQS::Queue
  AnotherNewResource:
    Type: AWS::SNS::Topic`

	result, err := comparator.Compare(ctx, currentTemplate, proposedTemplate)

	require.NoError(t, err)
	assert.True(t, result.HasChanges)
	assert.Equal(t, 2, result.ResourceCount.Added)    // NewResource, AnotherNewResource
	assert.Equal(t, 1, result.ResourceCount.Modified) // VPC (added EnableDnsHostnames)
	assert.Equal(t, 1, result.ResourceCount.Removed)  // OldResource
}

func TestYAMLTemplateComparator_UnifiedDiff_RealisticExample(t *testing.T) {
	comparator := NewYAMLTemplateComparator()
	ctx := context.Background()

	// Realistic "before" template
	currentTemplate := `AWSTemplateFormatVersion: '2010-09-09'
Description: Web application infrastructure
Parameters:
  Environment:
    Type: String
    Default: dev
Resources:
  WebServerSecurityGroup:
    Type: AWS::EC2::SecurityGroup
    Properties:
      GroupDescription: Security group for web servers
      SecurityGroupIngress:
        - IpProtocol: tcp
          FromPort: 80
          ToPort: 80
          CidrIp: 0.0.0.0/0
  WebServer:
    Type: AWS::EC2::Instance
    Properties:
      InstanceType: t2.micro
      ImageId: ami-12345678`

	// Realistic "after" template with multiple changes
	proposedTemplate := `AWSTemplateFormatVersion: '2010-09-09'
Description: Web application infrastructure
Parameters:
  Environment:
    Type: String
    Default: prod
  InstanceType:
    Type: String
    Default: t3.small
Resources:
  WebServerSecurityGroup:
    Type: AWS::EC2::SecurityGroup
    Properties:
      GroupDescription: Security group for web servers
      SecurityGroupIngress:
        - IpProtocol: tcp
          FromPort: 80
          ToPort: 80
          CidrIp: 0.0.0.0/0
        - IpProtocol: tcp
          FromPort: 443
          ToPort: 443
          CidrIp: 0.0.0.0/0
  WebServer:
    Type: AWS::EC2::Instance
    Properties:
      InstanceType: t3.small
      ImageId: ami-87654321
  Database:
    Type: AWS::RDS::DBInstance
    Properties:
      Engine: postgres
      DBInstanceClass: db.t3.micro`

	result, err := comparator.Compare(ctx, currentTemplate, proposedTemplate)

	require.NoError(t, err)
	assert.True(t, result.HasChanges)

	// Verify unified diff format is present
	assert.NotEmpty(t, result.Diff)
	assert.Contains(t, result.Diff, "@@", "Should contain unified diff hunk header")

	// Verify additions are marked with +
	assert.Contains(t, result.Diff, "+    Default: prod", "Should show parameter change")
	assert.Contains(t, result.Diff, "+  InstanceType:", "Should show new parameter")
	assert.Contains(t, result.Diff, "+        - IpProtocol: tcp", "Should show added security group rule")
	assert.Contains(t, result.Diff, "+          FromPort: 443", "Should show HTTPS port addition")
	assert.Contains(t, result.Diff, "+  Database:", "Should show new resource")

	// Verify deletions are marked with -
	assert.Contains(t, result.Diff, "-    Default: dev", "Should show old parameter value")
	assert.Contains(t, result.Diff, "-      InstanceType: t2.micro", "Should show old instance type")
	assert.Contains(t, result.Diff, "-      ImageId: ami-12345678", "Should show old AMI")

	// Verify context lines (unchanged) are marked with space
	// Note: Context lines only appear within the hunk context window
	assert.Contains(t, result.Diff, " Parameters:", "Should show context line within hunk")

	// Verify resource counts
	assert.Equal(t, 1, result.ResourceCount.Added)    // Database
	assert.Equal(t, 2, result.ResourceCount.Modified) // WebServerSecurityGroup, WebServer
	assert.Equal(t, 0, result.ResourceCount.Removed)
}

func TestYAMLTemplateComparator_MultipleHunks_WhenChangesFarApart(t *testing.T) {
	comparator := NewYAMLTemplateComparator()
	ctx := context.Background()

	// Template with changes far apart (more than 6 lines = contextLines*2)
	currentTemplate := `AWSTemplateFormatVersion: '2010-09-09'
Resources:
  Resource1:
    Type: AWS::S3::Bucket
  Resource2:
    Type: AWS::S3::Bucket
  Resource3:
    Type: AWS::S3::Bucket
  Resource4:
    Type: AWS::S3::Bucket
  Resource5:
    Type: AWS::S3::Bucket
  Resource6:
    Type: AWS::S3::Bucket
  Resource7:
    Type: AWS::S3::Bucket
  Resource8:
    Type: AWS::S3::Bucket
  Resource9:
    Type: AWS::S3::Bucket
  Resource10:
    Type: AWS::S3::Bucket`

	proposedTemplate := `AWSTemplateFormatVersion: '2010-09-09'
Resources:
  Resource1:
    Type: AWS::S3::Bucket
    Properties:
      BucketName: bucket1
  Resource2:
    Type: AWS::S3::Bucket
  Resource3:
    Type: AWS::S3::Bucket
  Resource4:
    Type: AWS::S3::Bucket
  Resource5:
    Type: AWS::S3::Bucket
  Resource6:
    Type: AWS::S3::Bucket
  Resource7:
    Type: AWS::S3::Bucket
  Resource8:
    Type: AWS::S3::Bucket
  Resource9:
    Type: AWS::S3::Bucket
  Resource10:
    Type: AWS::S3::Bucket
    Properties:
      BucketName: bucket10`

	result, err := comparator.Compare(ctx, currentTemplate, proposedTemplate)

	require.NoError(t, err)
	assert.True(t, result.HasChanges)

	// Count hunk headers - should have 2 hunks because changes are far apart
	hunkCount := 0
	for _, line := range strings.Split(result.Diff, "\n") {
		if strings.HasPrefix(line, "@@") {
			hunkCount++
		}
	}
	assert.Equal(t, 2, hunkCount, "Should have 2 hunks when changes are far apart")

	// Verify first hunk contains Resource1 change
	assert.Contains(t, result.Diff, "+    Properties:")
	assert.Contains(t, result.Diff, "+      BucketName: bucket1")

	// Verify second hunk contains Resource10 change
	assert.Contains(t, result.Diff, "+      BucketName: bucket10")
}

func TestYAMLTemplateComparator_NewStack_AllAdditions(t *testing.T) {
	comparator := NewYAMLTemplateComparator()
	ctx := context.Background()

	// Simulate new stack: empty current template
	currentTemplate := `{}`

	// Proposed template for new stack
	proposedTemplate := `AWSTemplateFormatVersion: '2010-09-09'
Description: New stack template
Parameters:
  Environment:
    Type: String
    Default: dev
Resources:
  VPC:
    Type: AWS::EC2::VPC
    Properties:
      CidrBlock: 10.0.0.0/16
  Subnet:
    Type: AWS::EC2::Subnet
    Properties:
      VpcId: !Ref VPC
      CidrBlock: 10.0.1.0/24`

	result, err := comparator.Compare(ctx, currentTemplate, proposedTemplate)

	require.NoError(t, err)
	assert.True(t, result.HasChanges)
	assert.Equal(t, 2, result.ResourceCount.Added) // VPC and Subnet

	// Verify diff is generated
	assert.NotEmpty(t, result.Diff)

	// Count lines starting with + (additions)
	additionCount := 0
	deletionCount := 0
	for _, line := range strings.Split(result.Diff, "\n") {
		if len(line) > 0 {
			switch line[0] {
			case '+':
				additionCount++
			case '-':
				deletionCount++
			}
		}
	}

	// Should have many additions and one deletion (the empty {} object)
	assert.Greater(t, additionCount, 5, "Should have many addition lines for new stack")
	assert.Equal(t, 1, deletionCount, "Should have one deletion line for empty current template")

	// Verify key content is marked as additions
	assert.Contains(t, result.Diff, "+AWSTemplateFormatVersion:")
	assert.Contains(t, result.Diff, "+Description: New stack template")
	assert.Contains(t, result.Diff, "+Resources:")
	assert.Contains(t, result.Diff, "+  VPC:")
	assert.Contains(t, result.Diff, "+  Subnet:")
}
