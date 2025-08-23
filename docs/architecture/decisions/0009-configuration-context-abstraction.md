# 9. Configuration context abstraction

Date: 2025-08-23

## Status

Accepted

Amends [ADR 0008: Configuration abstraction](0008-configuration-abstraction.md)

## Context

In ADR 0008, we decided to implement a configuration provider abstraction with a pluggable system. However, the initial interface design assumed that all teams would organise their configuration around an "environment" concept (dev, staging, prod, etc.).

Further analysis revealed that teams use various organisational strategies for their infrastructure configuration:
- **Environment-based**: dev, staging, production
- **Git branch-based**: main, develop, feature branches
- **Geographic**: us-east-1, eu-west-1, ap-southeast-2
- **Account-based**: dev-account, prod-account
- **Directory-based**: separate folders per deployment target
- **Multi-dimensional**: combinations of environment, region, team, etc.

The proposed interface:
```go
LoadConfig(ctx context.Context, environment string) (*Config, error)
GetStack(stackName, environment string) (*StackConfig, error)
ListEnvironments() ([]string, error)
```

This creates an **abstraction leak** where the configuration interface imposes a specific organisational model ("environments") on all provider implementations, reducing the flexibility we intended to achieve.

Different configuration providers need the freedom to interpret deployment contexts in ways that match their teams' operational models, without being constrained by a predetermined "environment" concept.

## Decision

We will replace the environment-specific interface with a **generic context-based abstraction**.

The configuration interface will use a simple `context` string parameter instead of assuming "environment" semantics:

```go
type ConfigProvider interface {
    LoadConfig(ctx context.Context, context string) (*Config, error)
    GetStack(stackName, context string) (*StackConfig, error)
    ListContexts() ([]string, error)
    Validate() error
}
```

Key principles:
- **Provider flexibility**: Each provider interprets `context` according to its organisational model
- **CLI consistency**: Users still use familiar commands, but providers map contexts as needed
- **No semantic assumptions**: The abstraction layer doesn't impose meaning on context values

## Consequences

**Positive:**
- Maximum flexibility for different organisational approaches to configuration
- Providers can implement complex context resolution (multi-dimensional selectors, hierarchical contexts)
- No abstraction leakage of specific organisational concepts
- Teams can migrate between different context models without changing the core interface
- Simple, clean interface that focuses on the essential operation: selecting configuration

**Negative:**
- Less semantic clarity in the interface (generic "context" vs specific "environment")
- Documentation and examples must explain how different providers interpret context
- Potential confusion for users switching between providers with different context models
- Loss of built-in validation for common environment patterns

**Implementation Impact:**
- CLI commands like `--environment` become provider-agnostic context selectors
- Provider implementations have full control over context interpretation
- Configuration validation becomes provider-specific rather than framework-level
- Documentation must clearly explain context semantics for each provider type

**Examples of context interpretation:**
- **File provider**: `context="prod"` loads `stackaroo-prod.yaml`
- **Git provider**: `context="main"` checks out main branch
- **Directory provider**: `context="us-east-1"` loads from `./us-east-1/` directory
- **Multi-selector provider**: `context="prod:us-east-1"` uses composite selection logic

We accept the trade-off of reduced semantic clarity for increased flexibility, enabling the configuration abstraction to truly support diverse team requirements without imposing architectural constraints.