# 11. Testing framework and strategy

Date: 2025-08-24

## Status

Accepted

## Context

We need to establish a consistent testing framework and strategy for Stackaroo to ensure high code quality, maintainability, and developer productivity. The project involves complex interactions with AWS services, CLI command processing, and file system operations that require comprehensive testing approaches.

Key testing requirements:
- Unit testing of business logic without external dependencies
- Mocking of AWS services and external systems
- Integration testing of CLI commands and workflows
- Fast, reliable test execution in CI/CD pipelines
- Clear test organisation and maintainability
- Support for Test-Driven Development (TDD) methodology

Several approaches were considered:
- **Testing frameworks**: Go standard library, testify, Ginkgo
- **Mocking strategies**: Manual mocks, testify/mock, gomock, interfaces vs concrete types
- **Architecture patterns**: Dependency injection, interface-based design, test pollution vs clean separation

The testing strategy needed to support our architectural goals of clean separation of concerns, SOLID principles, and maintainable code without compromising development velocity.

## Decision

We will adopt a comprehensive testing strategy based on **testify framework with testify/mock** for mocking, following **Test-Driven Development (TDD)** methodology with **interface-based dependency injection**.

**Core decisions:**

1. **Testing Framework: testify**
   - Use `github.com/stretchr/testify` as our primary testing framework
   - Leverage `testify/assert` and `testify/require` for expressive assertions
   - Utilise `testify/mock` for professional mocking capabilities

2. **Mocking Strategy: testify/mock with interfaces**
   - Use `testify/mock` for mocking external dependencies
   - Design interfaces for all external dependencies (AWS clients, file systems, etc.)
   - Implement dependency injection to enable mock substitution
   - No manual mock implementations or test-specific production code

3. **TDD Methodology**
   - Follow Red-Green-Refactor cycle for all new features
   - Write failing tests first, implement minimal code to pass, then refactor
   - Maintain high test coverage through TDD practice

4. **Architecture for Testability**
   - Use interfaces to abstract external dependencies
   - Implement dependency injection for clean test setup
   - Separate business logic from infrastructure concerns
   - No testing-specific code or pollution in production modules

5. **Test Organisation**
   - Unit tests alongside source files (`*_test.go`)
   - Integration tests in separate packages where appropriate
   - Mock implementations using testify/mock framework
   - Clear test naming conventions and documentation

## Consequences

**Positive:**
- **Professional testing approach** with industry-standard tools and practices
- **Fast, reliable tests** that don't depend on external services or credentials
- **High code quality** through TDD methodology and comprehensive test coverage
- **Maintainable test code** with clear mocking patterns and dependency injection
- **Developer productivity** through expressive assertions and good error messages
- **CI/CD friendly** with deterministic, fast-running tests
- **Refactoring confidence** with comprehensive test coverage protecting against regressions
- **Clean architecture** enforced through interface-based design for testability

**Negative:**
- **Additional complexity** in initial setup with interfaces and dependency injection
- **Learning curve** for team members unfamiliar with testify/mock patterns
- **More boilerplate code** for interface definitions and mock implementations
- **Potential over-abstraction** if interfaces are created unnecessarily
- **Initial development overhead** for TDD approach (offset by long-term benefits)

**Implementation Requirements:**
- All external dependencies must be abstracted behind interfaces
- Production code must not contain test-specific functionality
- Mock implementations must use testify/mock framework consistently
- Tests must be fast (< 5 seconds for full suite) and not require external resources
- TDD cycle must be followed for new feature development
- Test coverage should be maintained at high levels through automated tooling

**Examples of testing patterns:**
- Interface-based mocking: `mockClient.On("DeployStack", mock.Anything, stackName).Return(nil)`
- Dependency injection: `SetDeployer(mockDeployer)` for test setup
- TDD cycle: Write failing test → Minimal implementation → Refactor → Repeat
- Clean assertions: `assert.NoError(t, err)` and `require.NotNil(t, result)`

This testing strategy ensures high code quality while supporting rapid, confident development through comprehensive test coverage and clean architectural patterns.