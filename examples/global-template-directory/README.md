# Global Template Directory Example

This example demonstrates the global template directory feature implemented in [ADR 0014: Global template directory](../../docs/architecture/decisions/0014-global-template-directory.md).

## Overview

The global template directory feature allows you to specify a single directory where all CloudFormation templates are stored, reducing configuration verbosity and establishing consistent project structure.

## Directory Structure

```
examples/global-template-directory/
├── stackaroo.yaml           # Configuration with global template directory
├── templates/               # Global template directory
│   ├── vpc.yaml            # VPC template
│   ├── security/
│   │   └── security-groups.yaml
│   ├── compute/
│   │   └── app.yaml
│   └── database/
│       └── rds.yaml
└── README.md               # This file
```

## Configuration Comparison

### Before: Individual Template Paths
```yaml
project: example-app
region: us-east-1

contexts:
  dev:
    account: "123456789012"
  prod:
    account: "987654321098"

stacks:
  - name: vpc
    template: templates/vpc.yaml
  - name: security-groups
    template: templates/security/security-groups.yaml
  - name: app
    template: templates/compute/app.yaml
  - name: database
    template: templates/database/rds.yaml
```

### After: Global Template Directory
```yaml
project: example-app
region: us-east-1

templates:
  directory: "templates/"

contexts:
  dev:
    account: "123456789012"
  prod:
    account: "987654321098"

stacks:
  - name: vpc
    template: vpc.yaml                    # Resolves to templates/vpc.yaml
  - name: security-groups
    template: security/security-groups.yaml  # Resolves to templates/security/security-groups.yaml
  - name: app
    template: compute/app.yaml            # Resolves to templates/compute/app.yaml
  - name: database
    template: database/rds.yaml           # Resolves to templates/database/rds.yaml
```

## Key Features Demonstrated

### Simple Template References
Stack templates use short, clean paths that are resolved relative to the global template directory:

```yaml
templates:
  directory: "templates/"

stacks:
  - name: vpc
    template: vpc.yaml  # Much cleaner than templates/vpc.yaml
```

### Subdirectory Support
Templates can be organised in subdirectories within the global template directory:

```yaml
stacks:
  - name: security-groups
    template: security/security-groups.yaml  # Resolves to templates/security/security-groups.yaml
  - name: app
    template: compute/app.yaml              # Resolves to templates/compute/app.yaml
```

### Absolute Path Override
Absolute paths bypass the global template directory entirely, providing flexibility when needed:

```yaml
stacks:
  - name: shared-template
    template: /absolute/path/to/shared.yaml  # Uses absolute path as-is
```

## Architecture Overview

This example demonstrates simple AWS resources to show the global template directory feature:

```
├── VPC                    ← vpc.yaml (Simple VPC + Internet Gateway)
├── Security Group         ← security/security-groups.yaml (Web Security Group)
├── S3 Bucket             ← compute/app.yaml (Application storage)
└── DynamoDB Table        ← database/rds.yaml (Application data)
```

## Stack Overview

The example includes four simple stacks to demonstrate template organisation:

1. **vpc** - Basic VPC with Internet Gateway
2. **security-groups** - Simple web security group
3. **app** - S3 bucket for application storage
4. **database** - DynamoDB table for application data

## Context-Specific Configurations

The example shows how global template directory works with different deployment contexts:

### Development Context
```yaml
contexts:
  dev:
    account: "123456789012"
```

### Production Context
```yaml
contexts:
  prod:
    account: "987654321098"
```

Each context deploys the same simple resources to different AWS accounts.
