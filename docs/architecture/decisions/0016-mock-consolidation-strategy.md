# 16. Mock consolidation strategy

Date: 2025-08-30

## Status

Accepted

Amends [ADR 0011: Testing framework and strategy](0011-testing-framework-and-strategy.md)

## Context

The Stackaroo codebase had grown to include extensive testing coverage across multiple internal packages (`aws`, `config`, `deploy`, `delete`, `diff`, `prompt`, `resolve`). However, the testing architecture suffered from significant mock code duplication and inconsistency issues:

**Problems Identified:**
- **Massive Code Duplication**: ~400 lines of duplicate mock implementations across test files
- **Inconsistent Mock Patterns**: Different packages used different mocking approaches and conventions
- **Maintenance Burden**: Interface changes required updating multiple identical mock implementations
- **Poor Discoverability**: Mock implementations were scattered across test files, making them hard to locate
- **Testing Pollution**: Production packages contained test-specific code and inline mock definitions
- **Cross-Package Testing Difficulty**: Sharing mocks between packages required copying code

**Specific Duplication Examples:**
- `MockCloudFormationOperations` was implemented identically in 5+ test files
- `MockClient` variations existed in multiple packages with slight differences
- `MockResolver`, `MockDeployer`, and other mocks were duplicated across test suites
- Each duplication meant maintaining identical testify/mock implementations separately

The existing approach violated DRY principles and created significant technical debt that hindered development velocity and code maintainability.

## Decision

We will implement a **consolidated mock architecture** using dedicated `testing.go` files co-located with their corresponding interfaces in each internal package.

**Core Strategy:**

1. **Co-located Testing Files**: Create `testing.go` files in each internal package (`internal/aws/testing.go`, `internal/deploy/testing.go`, etc.)

2. **Single Source of Truth**: Each mock implementation exists exactly once, alongside its corresponding interface

3. **Consistent testify/mock Usage**: All mocks use `github.com/stretchr/testify/mock` with professional expectations and assertions

4. **Cross-Package Reusability**: Mocks can be imported and shared across any test suite in the codebase

5. **Three-Tier Mock Architecture** (for AWS package):
   - `MockClient` - Top-level AWS client interface
   - `MockCloudFormationOperations` - High-level business operations
   - `MockCloudFormationClient` - Low-level AWS SDK interface

6. **Import Pattern**: Tests import mocks using standard Go import syntax: `import "github.com/orien/stackaroo/internal/aws"`

**Implementation Approach:**
- Migrate all duplicate mock code into appropriate `testing.go` files
- Remove inline mock definitions from test files
- Update all test files to use imported mocks from testing packages
- Ensure consistent mock naming and structure across all packages

## Consequences

**Positive:**

- **Zero Code Duplication**: Eliminated ~400 lines of duplicate mock implementations across the codebase
- **Perfect Co-location**: Every mock lives directly alongside its corresponding interface, improving discoverability
- **Maximum Reusability**: Any package can import and use mocks from any other internal package
- **Complete Consistency**: Uniform mock organisation and naming conventions across all internal packages
- **Reduced Maintenance**: Interface changes require updating only one mock implementation
- **Clean Architecture**: Removed testing pollution from production packages
- **Professional Testing**: Full testify/mock integration with expectations, assertions, and behaviour verification
- **Development Velocity**: Faster test writing using established, comprehensive mock implementations
- **Easier Refactoring**: Single point of change for mock evolution and enhancement

**Negative:**

- **Import Overhead**: Tests must import testing packages to access mocks (minimal impact)
- **Package Dependencies**: Test files depend on `testing.go` files in other internal packages
- **Learning Curve**: Developers must discover and learn the location of shared mocks
- **Potential Coupling**: Changes to shared mocks could theoretically impact multiple test suites (mitigated by interface stability)

**Implementation Requirements:**

- All new mock implementations must be placed in appropriate `testing.go` files
- Existing duplicate mocks must be consolidated during code maintenance
- Mock implementations must use consistent naming: `Mock[InterfaceName]`
- All mocks must implement their full interface contract with testify/mock
- Cross-package mock usage should be documented in architecture files

**Long-term Benefits:**

This consolidation strategy establishes a scalable foundation for testing as the codebase grows. New packages automatically inherit the pattern, and mock implementations become first-class architectural components that support comprehensive testing across the entire system.

The approach aligns with Go best practices for test organisation whilst providing the professional mocking capabilities required for complex AWS service interactions and business logic validation.
