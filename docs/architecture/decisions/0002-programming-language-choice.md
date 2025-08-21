# 2. Programming language choice

Date: 2025-08-21

## Status

Accepted

## Context

We need to choose a programming language for implementing Stackaroo, a command-line tool for managing AWS CloudFormation stacks as code. The primary considerations are:

- Dependency management requirements for end users
- Development productivity and ecosystem maturity
- Performance characteristics for I/O-intensive operations
- Distribution and installation complexity
- AWS SDK quality and CloudFormation tooling availability

The main candidates considered were:
- **Python**: Mature AWS ecosystem (boto3), rich CLI libraries, but requires runtime dependencies
- **Go**: Single binary distribution, good AWS SDK, but more verbose development
- **Node.js/TypeScript**: Good AWS integration, but requires runtime dependencies
- **Rust**: High performance, single binary, but steeper learning curve

## Decision

We will use **Go** as the programming language for Stackaroo.

Key factors in this decision:
- Single static binary distribution eliminates dependency management for users
- No runtime requirements (Python interpreter, Node.js, etc.) on target systems
- Fast startup time suitable for frequent CLI usage
- AWS SDK for Go (v2) is mature and well-maintained
- Strong ecosystem for CLI development (Cobra framework)
- Aligns with common practices in infrastructure tooling

## Consequences

**Positive:**
- Users can install Stackaroo with a simple binary download or `go install`
- No version conflicts or virtual environment management required
- Fast execution and startup time
- Cross-platform compilation support
- Memory-efficient operation

**Negative:**
- More verbose code for configuration parsing and YAML handling compared to Python
- Smaller CloudFormation-specific ecosystem compared to Python
- Potentially longer initial development time for complex data transformations
- Team members may need to learn Go if not already familiar

We accept these trade-offs in favour of simplified distribution and reduced user friction.
