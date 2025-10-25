# 10. File provider configuration structure

Date: 2025-08-23

## Status

Accepted

Amended by [ADR 0014: Global template directory](0014-global-template-directory.md), [ADR 0017: Parameter resolution system enhancement](0017-parameter-resolution-system-enhancement.md), [ADR 0021: Consistent YAML schema structure](0021-consistent-yaml-schema-structure.md)

## Context

We need to define the specific configuration structure for the file-based configuration provider. This is the first concrete implementation of our configuration abstraction ([ADR 0008: Configuration abstraction](0008-configuration-abstraction.md)) using the generic context approach ([ADR 0009: Configuration context abstraction](0009-configuration-context-abstraction.md)).

Different configuration structures offer varying levels of complexity and capability:

**Simple context-as-override approach**:
- Single file with basic context sections
- Contexts only provide parameter overrides
- No deployment target specification
- Minimal complexity but limited functionality

**Multi-file approach**:
- Separate files per context (stackaroo-dev.yaml, stackaroo-prod.yaml)
- Complete isolation between contexts
- Simple structure but potential duplication

**CDK/Pulumi-style approach**:
- Contexts specify both logical grouping AND deployment targets
- Account, region, and profile mapping per context
- Shared stack definitions with context-specific overrides
- Parameter and tag inheritance hierarchy
- Matches patterns from mature infrastructure-as-code tools

Key requirements for the file provider:
- Support for multiple AWS accounts and regions per context
- Parameter override hierarchy (global → stack → context)
- Template reuse across contexts
- Clear deployment target specification
- Intuitive structure for teams familiar with modern IaC tools

## Decision

We will implement a **CDK/Pulumi-style configuration structure** for the file provider.

The configuration will include:

1. **Global defaults** - Project-wide settings, tags, and AWS configuration
2. **Context definitions** - Logical grouping combined with deployment targets
3. **Stack definitions** - Shared templates with context-specific overrides

Structure:
```yaml
project: myapp
region: us-east-1    # Global default
tags:                # Global tags
  Project: myapp

contexts:
  dev:
    account: "123456789012"
    region: us-west-2
    tags:
      Environment: dev

stacks:
  - name: vpc
    template: templates/vpc.yaml
    parameters:
      VpcCidr: 10.0.0.0/16
    contexts:
      dev:
        parameters:
          VpcCidr: 10.1.0.0/16
```

Key characteristics:
- **Deployment targets**: Each context specifies account and region
- **Inheritance hierarchy**: Global → Stack → Context for parameters and tags
- **Shared templates**: Same CloudFormation templates deployed across contexts
- **Clear mapping**: Context name maps to complete deployment specification

Configuration constraints:
- **Single region per context**: Each context is confined to exactly one AWS region
- **Single account per context**: Each context targets exactly one AWS account
- **Unique context names**: Context names must be unique within a configuration file
- **Template path resolution**: Template paths are resolved relative to configuration file location

## Consequences

**Positive:**
- Familiar pattern for teams using CDK, Pulumi, or Terraform
- Complete deployment target specification eliminates ambiguity
- Natural support for multi-account, multi-region deployments
- Parameter inheritance reduces duplication whilst allowing customisation
- Single source of truth for all deployment contexts
- Clear audit trail of what gets deployed where
- Relies on AWS SDK credential chain for authentication (no profile management needed)

**Negative:**
- More complex than simple override-based approaches
- Potential for large configuration files in complex scenarios
- Learning curve for teams unfamiliar with infrastructure-as-code patterns
- Risk of configuration drift if contexts become too different

**Implementation Requirements:**
- YAML parsing with proper inheritance resolution
- Validation of AWS account/region combinations
- Context resolution logic that merges global, stack, and context-specific settings
- Clear error messages when context or stack references are invalid
- Support for both absolute and relative template paths

**Usage Impact:**
- `stackaroo deploy vpc --context dev` resolves to specific AWS account, region, and parameters
- Teams can maintain separate AWS accounts for different contexts
- AWS authentication handled automatically via SDK credential chain
- Configuration supports both simple (single account) and complex (multi-account) scenarios
- Easy migration path from other infrastructure-as-code tools

This structure provides the flexibility needed for enterprise deployments whilst remaining approachable for simpler use cases, aligning with our goal of supporting diverse team requirements through the configuration abstraction.
