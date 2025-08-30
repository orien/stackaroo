# Stackaroo

A command-line tool for managing AWS CloudFormation stacks as code.

## Overview

Stackaroo simplifies CloudFormation stack management by providing:

- **Declarative Configuration**: Define your stacks and parameters in YAML files
- **Environment Management**: Deploy the same templates across multiple contexts
- **Change Preview**: See exactly what changes will be made before deployment
- **Stack Information**: View comprehensive details about deployed CloudFormation stacks
- **Template Validation**: Validate CloudFormation templates before deployment
- **Stack Lifecycle**: Deploy, update, delete, and monitor stack status
- **Parameter Management**: Organize parameters by context and stack

## Features

### Environment Management
- Deploy the same templates across multiple contexts
- Different AWS regions and parameters per context

### Dependency Management

- Define stack dependencies with `depends_on`
- Automatic deployment ordering
- Parallel deployment where possible

### Change Preview

- **Comprehensive Change Analysis**: Shows template, parameter, tag, and resource changes
- **CloudFormation ChangeSet Integration**: Uses AWS ChangeSet API for accurate previews
- **Rich Diff Output**: Detailed comparison of current vs proposed infrastructure
- **Resource Impact Assessment**: Identifies which resources will be created, modified, or deleted
- **Replacement Warnings**: Highlights resources that require replacement during updates
- **Consistent Formatting**: Same preview format as the dedicated `diff` command

### Stack Information

- **Comprehensive Stack Details**: View complete information about deployed CloudFormation stacks
- **Status and Metadata**: Shows stack status, creation time, last update, and description
- **Parameter Display**: Current parameter values sorted alphabetically
- **Output Information**: Stack outputs with their current values
- **Tag Management**: All stack tags displayed in organised format
- **Human-Readable Format**: Clean, consistent formatting with proper indentation
- **Real-time Data**: Retrieves current information directly from AWS CloudFormation

### Template Validation

- Local CloudFormation template validation
- Parameter validation against template requirements
- Circular dependency detection

### Real-time Event Streaming

- See exactly what will change before applying
- Live CloudFormation events during deployment operations
- See resource creation, updates, and completion status in real-time
- Smart detection of create vs update operations
- Graceful handling of "no changes" scenarios

## Installation

### Using Go Install

```bash
go install github.com/orien/stackaroo@latest
```

### Download Binary

Download the latest release from the [releases page](https://github.com/orien/stackaroo/releases).

#### Linux/macOS

```bash
# Download and install (replace VERSION and ARCH as needed)
VERSION=1.0.0
ARCH=linux-x86_64
URL="https://github.com/orien/stackaroo/releases/download/v${VERSION}/stackaroo-${VERSION}-${ARCH}.tar.gz"
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
  - name: vpc
    template: templates/vpc.yaml
  - name: app
    template: templates/app.yaml
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

# Delete specific stack with confirmation
stackaroo delete development app

# Delete all stacks in context (reverse dependency order)
stackaroo delete development

# Use custom config file
stackaroo deploy production --config custom-config.yaml
```
