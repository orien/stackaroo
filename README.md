# Stackaroo

A command-line tool for managing AWS CloudFormation stacks as code.

## Overview

Stackaroo simplifies CloudFormation stack management by providing:

- **Declarative Configuration**: Define your stacks and parameters in YAML files
- **Environment Management**: Deploy the same templates across multiple environments
- **Change Preview**: See exactly what changes will be made before deployment
- **Template Validation**: Validate CloudFormation templates before deployment
- **Stack Lifecycle**: Deploy, update, delete, and monitor stack status
- **Parameter Management**: Organize parameters by environment and stack

## Features

### Environment Management
- Deploy the same templates across multiple environments
- Environment-specific parameter overrides
- Different AWS regions and profiles per environment

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

### Template Validation

- Local CloudFormation template validation
- Parameter validation against template requirements
- Circular dependency detection

### Real-time Event Streaming

- **Change Preview Before Deployment**: See exactly what will change before applying
- Live CloudFormation events during deployment operations
- See resource creation, updates, and completion status in real-time
- Smart detection of create vs update operations
- Graceful handling of "no changes" scenarios

## Installation

```bash
go install github.com/orien/stackaroo@latest
```

Or download a binary from the [releases page](https://github.com/orien/stackaroo/releases).
