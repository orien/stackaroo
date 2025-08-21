# 3. Configuration format

Date: 2025-08-21

## Status

Accepted

## Context

We need to choose a configuration format for Stackaroo's primary configuration file and parameter files. The configuration will need to support:

- Stack definitions with templates and parameters
- Environment-specific overrides and configurations
- Hierarchical data structures for complex setups
- Human readability for maintainability
- Comment support for documentation
- Parsing efficiency in Go

The main candidates considered were:
- **YAML**: Human-readable, supports comments, widely used in DevOps
- **TOML**: Simple syntax, less ambiguous than YAML, good for configuration
- **JSON**: Simple parsing, CloudFormation native, but no comments
- **HCL**: Purpose-built for infrastructure, but additional complexity

## Decision

We will use **YAML** as the configuration format for Stackaroo.

Key factors in this decision:
- Widely adopted in infrastructure and DevOps tooling (Kubernetes, Ansible, etc.)
- Familiar to CloudFormation users who work with YAML templates
- Excellent comment support for documenting configuration choices
- Natural fit for hierarchical data structures (environments, stacks, parameters)
- Good Go library support with go-yaml
- Balances human readability with parsing efficiency

## Consequences

**Positive:**
- Familiar format for the target audience (DevOps engineers, cloud architects)
- Comments allow for self-documenting configuration files
- Hierarchical structure maps well to environment and stack organisation
- Consistent with many other infrastructure tools
- Good tooling support in editors and IDEs

**Negative:**
- Indentation-sensitive syntax can lead to parsing errors
- Some ambiguity with data types (strings vs numbers vs booleans)
- Slightly more complex parsing compared to JSON
- Potential security considerations with advanced YAML features

We will mitigate the negatives by:
- Providing clear documentation and examples
- Using strict parsing to avoid type ambiguity
- Implementing validation to catch common indentation errors
- Disabling potentially dangerous YAML features
