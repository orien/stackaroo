# Stackaroo

A command-line tool for managing AWS CloudFormation stacks as code.

📚 **[Read the user documentation](https://orien.io/stackaroo/)** for tutorials, how-to guides, explanations, and detailed reference material.

## Overview

Stackaroo simplifies CloudFormation stack management through declarative YAML configuration, allowing you to define your infrastructure once and deploy it consistently across multiple environments. It provides comprehensive parameter management with support for literal values, dynamic stack output resolution, and cross-region references. Features include dependency-aware deployment ordering, integrated change previews, template validation, and real-time event streaming during stack operations.

## Features

### Environment Management
- Deploy the same templates across multiple contexts
- Different AWS regions and parameters per context

### Dependency Management

- Define stack dependencies with `depends_on`
- Automatic deployment ordering

### Change Preview

- Shows template, parameter, tag, and resource changes in a unified diff format (similar to `git diff`).
- Uses the AWS ChangeSet API to identify which resources will be created, modified, deleted, or replaced.
- Highlights resources that require replacement and uses the same format as the dedicated `diff` command.

### Stack Information

- Displays status, creation time, last update, description, parameters, outputs, and tags for a deployed stack.
- Retrieves current data directly from AWS CloudFormation in a clean, consistently formatted layout.

### Template Validation

- Validates templates against the CloudFormation API in the target region without deploying, catching syntax errors and invalid resource types.
- Supports single-stack and whole-context validation, making it suitable for pre-deployment checks in CI/CD pipelines.

Validate templates early in your development workflow:

```bash
# Validate a single stack's template
stackaroo validate dev vpc

# Validate all stacks in a context
stackaroo validate production
```

The validation command provides immediate feedback on template errors without requiring actual deployment, making it ideal for development workflows and continuous integration pipelines. It processes templates through the same resolution pipeline as deployment, including Go template processing, ensuring validation matches what will actually be deployed.

### Parameter System

Stackaroo provides a comprehensive parameter system supporting multiple resolution types:

#### Literal Parameters
Direct string values defined in configuration:
```yaml
parameters:
  Environment: production
  InstanceType: t3.medium
  Port: "8080"
```

#### Stack Output Parameters
Pull values dynamically from existing CloudFormation stack outputs:
```yaml
parameters:
  VpcId:
    type: stack-output
    stack: networking
    output: VpcId

  DatabaseEndpoint:
    type: stack-output
    stack: database
    output: DatabaseEndpoint
```

#### Cross-Region Stack Outputs
Reference outputs from stacks in different AWS regions:
```yaml
parameters:
  SharedBucketArn:
    type: stack-output
    stack: shared-resources
    output: BucketArn
    region: us-east-1
```

#### List Parameters
Support for CloudFormation `List<Type>` and `CommaDelimitedList` parameters with mixed resolution types:
```yaml
parameters:
  # Mix literals and stack outputs in a single list parameter
  SecurityGroupIds:
    - sg-baseline123         # Literal value
    - type: stack-output     # Dynamic from stack output
      stack: security-stack
      output: WebSGId
    - sg-additional456       # Another literal

  # Simple literal list
  AllowedPorts:
    - "80"
    - "443"
    - "8080"
```

#### Context Overrides
Different parameter values per deployment context:
```yaml
parameters:
  InstanceType: t3.micro    # Default value
contexts:
  production:
    parameters:
      InstanceType: t3.large  # Production override
```

Stack outputs are resolved at deployment time, so cross-stack dependencies always reflect the current live state. Different values per context are supported without modifying templates, and existing literal parameter configurations continue to work unchanged.

### CloudFormation Template Templating

Stackaroo supports dynamic CloudFormation template generation using Go templates with Sprig functions. This allows you to use the same template file across different contexts with context-specific variations:

```yaml
# Template: templates/storage.yaml
AWSTemplateFormatVersion: '2010-09-09'
Description: {{ .Context | title }} storage for {{ .StackName }}

Resources:
  AppBucket:
    Type: AWS::S3::Bucket
    Properties:
      BucketName: {{ .StackName }}-bucket-{{ .Context | lower }}
      Tags:
        - Key: Environment
          Value: {{ .Context }}
{{- if eq .Context "production" }}
        - Key: BackupEnabled
          Value: "true"
{{- end }}
```

When deployed to the `development` context, this generates a bucket named `my-app-bucket-development`. When deployed to `production`, it generates `my-app-bucket-production` with an additional backup tag.

Templates have access to `{{ .Context }}` and `{{ .StackName }}` variables, the full [Sprig](https://masterminds.github.io/sprig/) function library, and standard Go template conditionals for environment-specific resources. Processing is automatic and backwards compatible with static templates.

### Real-time Event Streaming

- Streams live CloudFormation events during deployments, showing resource creation, updates, and completion status as they happen.
- Automatically detects create vs update operations and handles "no changes" scenarios gracefully.

## Installation

### Using Go Install

```bash
go install codeberg.org/orien/stackaroo@latest
```

### Download Binary

Download the latest release from the [releases page](https://codeberg.org/orien/stackaroo/releases).

#### Linux/macOS

```bash
# Download and install (replace VERSION and ARCH as needed)
VERSION=1.0.0
ARCH=linux-x86_64
URL="https://codeberg.org/orien/stackaroo/releases/download/v${VERSION}/stackaroo-${VERSION}-${ARCH}.tar.gz"
DIR="stackaroo-${VERSION}-${ARCH}"

curl -sL "$URL" | tar -xz
sudo mv "${DIR}/stackaroo" /usr/local/bin/
rm -rf "${DIR}"

# Verify installation
stackaroo --version
```

#### Windows

Download the `.zip` file from the releases page, extract it, and add the binary to your PATH.

### Verify Installation

```bash
stackaroo --version
```

## Quick Start

### Configuration

Create a `stackaroo.yaml` file defining your stacks and contexts:

```yaml
project: my-infrastructure
region: us-east-1

contexts:
  development:
    account: "123456789012"
    region: ap-southeast-4
    tags:
      Environment: development
  production:
    account: "987654321098"
    region: us-east-1
    tags:
      Environment: production

stacks:
  vpc:
    template: templates/vpc.yaml
    parameters:
      # Literal parameters
      Environment: development
      VpcCidr: "10.0.0.0/16"
      EnableDnsSupport: "true"
    contexts:
      production:
        parameters:
          Environment: production
          VpcCidr: "172.16.0.0/16"

  app:
    template: templates/app.yaml
    parameters:
      # Literal parameters
      InstanceType: t3.micro
      MinCapacity: "1"
      MaxCapacity: "3"

      # Stack output parameters (pull from existing stacks)
      VpcId:
        type: stack-output
        stack: vpc
        output: VpcId

      PrivateSubnetId:
        type: stack-output
        stack: vpc
        output: PrivateSubnet1Id

      # Cross-region stack output (optional region parameter)
      SharedBucketArn:
        type: stack-output
        stack: shared-resources
        output: BucketArn
        region: us-east-1
    contexts:
      production:
        parameters:
          InstanceType: t3.small
          MinCapacity: "2"
          MaxCapacity: "10"
    depends_on:
      - vpc
```

### Deployment

Deploy stacks using either pattern:

```bash
# Deploy all stacks in a context (with dependency ordering)
stackaroo deploy development

# Deploy a specific stack
stackaroo deploy development vpc

# Preview changes before deployment
stackaroo diff development app

# View detailed stack information
stackaroo describe production vpc
```

### Key Commands

#### Core Commands
- `deploy <context> [stack-name]` - Deploy all stacks or a specific stack with dependency-aware ordering and integrated change preview
- `diff <context> <stack-name>` - Preview changes between deployed stack and local configuration
- `describe <context> <stack-name>` - Display detailed information about a deployed CloudFormation stack
- `validate <context> [stack-name]` - Validate CloudFormation templates for syntax and AWS-specific requirements
- `delete <context> [stack-name]` - Delete stacks with dependency-aware ordering and confirmation prompts

#### Global Flags
- `--config, -c` - Specify config file (default: stackaroo.yaml)
- `--verbose, -v` - Enable verbose output for detailed logging
- `--version` - Show version information
- `--help` - Show help for any command

#### Usage Examples
```bash
# Deploy all stacks in development context
stackaroo deploy development

# Deploy specific stack with verbose output
stackaroo deploy production app --verbose

# Preview changes before deployment
stackaroo diff staging vpc

# View detailed stack information
stackaroo describe production app

# Validate templates before deployment
stackaroo validate development vpc

# Validate all templates in a context
stackaroo validate production

# Delete specific stack with confirmation
stackaroo delete development app

# Delete all stacks in context (reverse dependency order)
stackaroo delete development

# Use custom config file
stackaroo deploy production --config custom-config.yaml
```
