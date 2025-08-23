# 8. Configuration abstraction

Date: 2025-08-23

## Status

Accepted

Amended by [ADR 0009: Configuration context abstraction](0009-configuration-context-abstraction.md)

Amends [ADR 0003: Configuration format](0003-configuration-format.md)

## Context

We need to design how Stackaroo will load and manage configuration for CloudFormation stacks across different environments. Different teams have varying requirements for configuration management:

- **Small teams** may prefer simple, single-file configuration
- **Large organisations** may require complex, multi-environment setups with separate files
- **Enterprise teams** may need configuration from external systems or APIs
- **DevOps teams** may want configuration versioned in Git with branch-based environments
- **Compliance-focused teams** may require configuration stored in secure, auditable systems

Key considerations:
- Flexibility to support different configuration strategies
- Consistency in how configuration is accessed and used
- Extensibility for future configuration sources
- Testability and mockability for different scenarios
- Clear separation between configuration loading and business logic

Rather than choosing a single configuration format or approach, we need an abstraction that allows teams to plug in different configuration providers based on their needs.

## Decision

We will implement a **configuration provider abstraction** that allows multiple configuration strategies through a common interface.

The core abstraction will define:
- `ConfigProvider` interface for loading configuration
- Standard data structures for configuration representation
- Plugin-like architecture for different provider implementations

Key interface methods:
- `LoadConfig(environment)` - Load configuration for specific environment
- `GetStack(stackName, environment)` - Get stack-specific configuration
- `ListEnvironments()` - Discover available environments
- `ListStacks(environment)` - Discover available stacks
- `Validate()` - Validate configuration integrity

This abstraction will allow teams to choose from different provider implementations without changing their Stackaroo workflows or commands.

## Consequences

**Positive:**
- Teams can choose configuration strategies that fit their operational model
- Easy to add new configuration sources without changing core logic
- Better testability through interface-based design
- Clear separation of concerns between configuration loading and stack operations
- Supports migration between configuration strategies as teams evolve
- Enables complex scenarios like multi-source configuration with fallbacks

**Negative:**
- Additional complexity compared to single, hard-coded configuration format
- More code to maintain across multiple provider implementations
- Learning curve for teams wanting to implement custom providers
- Potential for configuration inconsistencies across different providers
- Initial development effort higher than simple file-based approach

**Implementation Requirements:**
- Define clear interfaces and data structures for configuration
- Implement at least one provider (likely file-based) for initial release
- Provide clear documentation and examples for each provider type
- Ensure consistent behaviour across different provider implementations
- Design configuration validation that works across all provider types

We accept the additional complexity in favour of flexibility that supports diverse team requirements and allows Stackaroo to scale from simple to enterprise use cases.