# AWS Module Test Program

This is a standalone test program for the Stackaroo AWS client module. It provides comprehensive testing of all AWS operations without requiring the full Stackaroo CLI to be implemented.

## Purpose

This test program validates:
- AWS client creation and configuration
- CloudFormation operations wrapper functionality
- Template validation
- Stack listing, checking, and details retrieval
- Error handling
- Real AWS integration (when not in dry-run mode)

## Usage

### Basic Usage

```bash
# Build and run in dry-run mode (safe, no AWS resources created)
make test-aws

# Or build and run manually
go build -o bin/test-aws cmd/test-aws
./bin/test-aws
```

### Command Line Options

```bash
./bin/test-aws [options]

Options:
  -region string     AWS region (default "us-east-1")
  -profile string    AWS profile name (optional)
  -stack string      Stack name for testing (default "stackaroo-test-stack")
  -dry-run          Dry run mode - don't create real resources (default true)
  -verbose          Verbose output showing more details (default false)
```

### Examples

```bash
# Test in us-west-2 region
./bin/test-aws -region=us-west-2

# Test with specific AWS profile
./bin/test-aws -profile=production -region=eu-west-1

# Test with verbose output
./bin/test-aws -verbose=true

# CAREFUL: Test against real AWS (creates actual resources)
./bin/test-aws -dry-run=false -stack=my-test-stack
```

## What It Tests

### 1. AWS Client Creation ✅
- Creates AWS client with custom region/profile
- Validates configuration loading
- Tests credential chain resolution

### 2. CloudFormation Operations ✅
- Creates CloudFormation operations wrapper
- Validates service client initialization

### 3. Template Validation ✅
- Tests CloudFormation template validation
- Uses a real S3 bucket template with parameters

### 4. Stack Listing ✅
- Lists all existing CloudFormation stacks
- Handles pagination and filtering
- Shows stack names and statuses

### 5. Stack Existence Check ✅
- Tests whether a specific stack exists
- Validates error handling for non-existent stacks

### 6. Stack Details Retrieval ✅
- Gets complete stack information
- Displays parameters, outputs, and tags
- Shows creation and update timestamps

### 7. Stack Deployment 🚨
- **Only in non-dry-run mode**
- Creates a real CloudFormation stack
- Uses test template with S3 bucket
- Applies tags and parameters

### 8. Error Handling ✅
- Tests error responses for invalid operations
- Validates error message formatting
- Ensures proper error propagation

## Test Template

The program uses a CloudFormation template that creates:
- **S3 Bucket** with secure defaults
- **Encryption** enabled (AES256)
- **Public access blocking** enabled
- **Parameterized naming** with account ID and region
- **Tags** for identification
- **Outputs** for bucket name and ARN

This template is safe to deploy and easy to clean up.

## Prerequisites

### AWS Configuration

Ensure you have AWS credentials configured via one of:

1. **AWS Profile** (recommended for testing):
   ```bash
   aws configure --profile test-profile
   ```

2. **Environment Variables**:
   ```bash
   export AWS_ACCESS_KEY_ID=your-access-key
   export AWS_SECRET_ACCESS_KEY=your-secret-key
   export AWS_REGION=us-east-1
   ```

3. **IAM Role** (for EC2/ECS/Lambda execution)

### Required IAM Permissions

For full testing, your AWS credentials need:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "cloudformation:ListStacks",
        "cloudformation:DescribeStacks",
        "cloudformation:ValidateTemplate",
        "cloudformation:CreateStack",
        "cloudformation:UpdateStack",
        "cloudformation:DeleteStack",
        "s3:CreateBucket",
        "s3:DeleteBucket",
        "s3:GetBucketLocation"
      ],
      "Resource": "*"
    }
  ]
}
```

**Note**: The `s3:*` permissions are only needed if you run with `-dry-run=false`.

## Safety Features

### Dry Run Mode (Default)
- **No AWS resources created**
- Tests all read operations
- Skips destructive operations
- Safe to run anywhere

### Resource Naming
- Uses unique stack names with timestamps
- Includes account ID and region in bucket names
- Prevents naming conflicts

### Clean Up Instructions
The program provides cleanup instructions when creating real resources:

```bash
# Delete the test stack
aws cloudformation delete-stack --stack-name stackaroo-test-stack --region us-east-1

# Wait for deletion to complete
aws cloudformation wait stack-delete-complete --stack-name stackaroo-test-stack --region us-east-1
```

## Integration with Makefile

The project Makefile includes several shortcuts:

```bash
# Test AWS module (dry-run)
make test-aws

# Test with specific region
make aws-test-us-east-1
make aws-test-us-west-2

# Test with specific profile
PROFILE=myprofile make aws-test-profile

# DANGEROUS: Test against real AWS
make test-aws-live
```

## Troubleshooting

### Common Issues

1. **"NoCredentialsError"**
   - Set up AWS credentials (see Prerequisites)
   - Check AWS profile configuration

2. **"UnauthorizedOperation"**
   - Ensure IAM permissions are configured
   - Check the Required IAM Permissions section

3. **"InvalidRegion"**
   - Use a valid AWS region name
   - Check `aws ec2 describe-regions` for available regions

4. **"StackAlreadyExists"**
   - The test stack already exists
   - Delete it manually or use a different stack name

### Debug Output

Run with `-verbose=true` for detailed output:
- Full stack details
- Template content
- All API responses
- Timing information

### AWS CLI Verification

Verify your AWS setup:
```bash
# Check AWS identity
aws sts get-caller-identity

# List CloudFormation stacks
aws cloudformation list-stacks

# Validate AWS configuration
aws configure list
```

## Development Notes

This test program is designed to:
- **Exercise all code paths** in the AWS module
- **Provide immediate feedback** on AWS integration
- **Serve as example usage** for the AWS client
- **Enable safe testing** without infrastructure impact
- **Support CI/CD integration** for automated testing

The program can be extended to test additional AWS services as they're added to the Stackaroo AWS client module.
