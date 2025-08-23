# Stackaroo

A command-line tool for managing AWS CloudFormation stacks as code.

## Overview

Stackaroo simplifies CloudFormation stack management by providing:

- **Declarative Configuration**: Define your stacks and parameters in YAML files
- **Environment Management**: Deploy the same templates across multiple environments
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

### Template Validation

- Local CloudFormation template validation
- Parameter validation against template requirements
- Circular dependency detection

## Installation

```bash
go install github.com/orien/stackaroo@latest
```

Or download a binary from the [releases page](https://github.com/orien/stackaroo/releases).
