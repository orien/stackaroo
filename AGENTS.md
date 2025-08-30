# AGENTS.md

## Project Overview

Stackaroo is a command-line tool for managing AWS CloudFormation stacks as code, written in Go. It provides declarative configuration, environment management, change preview, template validation, and dependency management for CloudFormation deployments.

**Key Technologies:**
- Go 1.24
- AWS SDK for Go v2
- Cobra CLI framework
- YAML configuration
- CloudFormation templates

## Setup Commands

- Install dependencies: `go mod download`
- Verify dependencies: `go mod verify && go mod tidy`
- Build main binary: `make build`
- Build all binaries: `make build-all`
- Install locally: `make install`
- Clean build artifacts: `make clean`

## Development Environment

### Required Tools
- Go 1.24 or later
- AWS CLI (for testing with real AWS resources)
- golangci-lint (install with `make install-golangci-lint`)

### Quick Start
```bash
make build              # Build the main binary
make test               # Run unit tests
make lint               # Run all linting tools
./bin/stackaroo --help  # Test the CLI
```

## Build System

The project uses a comprehensive Makefile with the following key targets:

### Build Targets
- `make build` - Build main stackaroo binary
- `make build-test-aws` - Build AWS test program
- `make build-all` - Build all binaries
- `make release-build` - Build cross-platform release binaries

### Test Targets
- `make test` or `make test-unit` - Run unit tests
- `make test-aws` - Test AWS module (dry-run)
- `make test-aws-live` - Test against real AWS (destructive!)

### Development Targets
- `make fmt` - Format Go code
- `make vet` - Run go vet
- `make golangci-lint` - Run golangci-lint
- `make lint` - Run all linting tools (fmt + vet + golangci-lint)
- `make tidy` - Tidy module dependencies

## Code Style and Conventions

### Go Standards
- Use `go fmt` for formatting (enforced in CI)
- Follow Go naming conventions (exported/unexported identifiers)
- Use Go 1.24 features where appropriate
- Prefer context.Context for cancellation and timeouts

### Project Structure
```
cmd/           - CLI commands and subcommands
internal/      - Internal packages (not importable by other projects)
  aws/         - AWS service interactions
  config/      - Configuration handling
  deploy/      - Deployment logic with integrated change preview
  resolve/     - Dependency resolution
examples/      - Usage examples
docs/          - Documentation
```

### Code Organization
- Keep main.go minimal - delegate to cmd/ package
- Use internal/ packages for core logic
- Each internal package should have a clear, single responsibility
- Write tests alongside code (e.g., `file.go` and `file_test.go`)

### Error Handling
- Always handle errors explicitly
- Use wrapped errors with context: `fmt.Errorf("operation failed: %w", err)`
- For CLI errors, return appropriate exit codes
- Use AWS SDK error types for AWS-specific error handling

### Testing Patterns
- Use testify/assert and testify/require for assertions
- Name test functions clearly: `TestFunctionName_Scenario`
- Use table-driven tests for multiple similar test cases
- Mock AWS services for unit tests, use integration tests for real AWS

### Language and Documentation Standards
- Use British English spelling throughout the codebase and documentation
- Examples: "colour" not "color", "organisation" not "organization", "optimise" not "optimize"
- Apply British spellings to comments, error messages, variable names, and documentation
- Follow British punctuation conventions (e.g., single quotes for nested quotes)
- Use ISO 8601 date formats (YYYY/MM/DD) where applicable

### Diagram Standards
- Use Mermaid for all diagrams and flowcharts
- Embed Mermaid diagrams in Markdown using code blocks with `mermaid` language identifier
- Common diagram types for this project:
  - Flowcharts for deployment processes
  - Sequence diagrams for AWS API interactions
  - State diagrams for stack lifecycle management
  - Class diagrams for internal package relationships
- Keep diagrams simple and focused on the specific concept being illustrated
- Use consistent naming and terminology that matches the codebase

## Testing Strategy

This project uses a hybrid testing approach that adapts to the type of work being done. Choose the most appropriate strategy based on the context.

### Testing Approaches by Context

#### Complex Business Logic (Tests First)
For complex algorithms, dependency resolution, and configuration parsing:
1. **Define the contract** - Write clear function signatures and expected behaviour
2. **Write failing tests** - Focus on edge cases and error conditions
3. **Implement to make tests pass** - Build the minimum viable solution
4. **Refactor** - Improve code whilst maintaining test coverage

#### AWS Integration (Implementation + Tests Together)
For AWS service interactions and external dependencies:
1. **Implement basic functionality** - Build the AWS client wrapper
2. **Create mocks alongside** - Mock AWS services as you build
3. **Test real and error scenarios** - Cover both success and failure paths
4. **Add integration tests** - Use dry-run mode for safety

#### Simple CLI Commands (Implementation First)
For straightforward command handlers and basic functionality:
1. **Implement the feature** - Build the basic command structure
2. **Add tests for edge cases** - Focus on error handling and validation
3. **Test integration points** - Verify command parsing and output

### Implementation Guidelines

#### Test Structure
```go
func TestFunctionName_Scenario_ExpectedBehaviour(t *testing.T) {
    // Arrange - Set up test data and mocks

    // Act - Execute the function under test

    // Assert - Verify the expected behaviour
    require.NoError(t, err)
    assert.Equal(t, expected, actual)
}
```

#### AWS Service Testing Pattern
```go
func TestCloudFormationService_CreateStack_Success(t *testing.T) {
    // Arrange
    mockCF := &mockCloudFormationAPI{}
    service := NewCloudFormationService(mockCF)

    expectedInput := &cloudformation.CreateStackInput{
        StackName: aws.String("test-stack"),
    }

    mockCF.On("CreateStack", mock.AnythingOfType("*context.emptyCtx"), expectedInput).
        Return(&cloudformation.CreateStackOutput{}, nil)

    // Act
    result, err := service.CreateStack(context.Background(), "test-stack", template)

    // Assert
    require.NoError(t, err)
    assert.NotNil(t, result)
    mockCF.AssertExpectations(t)
}
```

### Test Categories

#### Fast Tests (Unit Tests)
- Run in milliseconds
- No external dependencies
- Mock all AWS services, file systems, network calls
- Use `go test -short` to run only these tests

#### Slow Tests (Integration Tests)
- May take seconds to complete
- Test real AWS interactions (with dry-run when possible)
- Use build tags: `//go:build integration`
- Run with: `go test -tags=integration`

### Testing Best Practices

- **Test Behaviour, Not Implementation** - Focus on what the function should do, not how it does it
- **One Assertion Per Test** - Each test should verify one specific behaviour
- **Descriptive Test Names** - Test names should clearly describe the scenario and expected outcome
- **Test Data Builders** - Use builder patterns for complex test data setup
- **Clean Test Environment** - Each test should be independent and not rely on previous test state

### AWS-Specific Testing

#### CloudFormation Templates
- Validate template syntax before testing deployment logic
- Test parameter resolution and validation
- Verify dependency ordering in stack deployment

#### AWS SDK Mocking
```go
type mockCloudFormationAPI struct {
    mock.Mock
}

func (m *mockCloudFormationAPI) CreateStack(ctx context.Context, input *cloudformation.CreateStackInput, opts ...func(*cloudformation.Options)) (*cloudformation.CreateStackOutput, error) {
    args := m.Called(ctx, input)
    return args.Get(0).(*cloudformation.CreateStackOutput), args.Error(1)
}
```

#### Error Handling Tests
- Test AWS service errors (throttling, permissions, invalid parameters)
- Verify error wrapping and context preservation
- Test retry logic and backoff strategies

## Testing Instructions

### Unit Tests
```bash
make test                    # Run all unit tests
go test -v ./internal/...   # Run with verbose output
go test -race ./...         # Test with race detection
```

### AWS Integration Tests
```bash
make test-aws               # Safe dry-run tests
make test-aws-live          # Live tests (creates real resources!)
make aws-test-us-east-1     # Test in specific region
PROFILE=myprofile make aws-test-profile  # Test with AWS profile
```

### CI Pipeline
The GitHub Actions CI runs:
1. **Test** - Unit tests with race detection
2. **Lint** - golangci-lint with timeout
3. **Security** - govulncheck for vulnerabilities
4. **Build** - Cross-platform builds (Linux/macOS/Windows, AMD64/ARM64)
5. **Integration** - Basic CLI functionality tests

All tests must pass before merging. Run `make commit-check` before committing.

## AWS-Specific Development

### Authentication
- Uses AWS SDK v2 default credential chain
- Supports AWS profiles via `--profile` or `AWS_PROFILE`
- Supports role assumption and MFA

### Regions
- Default region from AWS config or `AWS_REGION`
- Override with `--region` flag
- Test with multiple regions using Makefile targets

### CloudFormation Integration
- Validate templates before deployment
- Handle stack lifecycle (create, update, delete)
- Support parameter files and overrides
- Implement dependency resolution between stacks

### Testing with AWS
- **NEVER** use production accounts for testing
- Use separate AWS account/profile for development
- Dry-run mode is default for safety
- Clean up resources after testing

## Configuration Management

### YAML Configuration
- Stack definitions in YAML format
- Context-specific parameter overrides
- Template path resolution
- Dependency declarations with `depends_on`

### File Structure Expectations
```
stackaroo.yml           # Main configuration
templates/
  vpc.yml              # CloudFormation templates
  app.yml
```

## Dependencies and External APIs

### Core Dependencies
- `github.com/aws/aws-sdk-go-v2` - AWS SDK
- `github.com/spf13/cobra` - CLI framework
- `gopkg.in/yaml.v3` - YAML parsing
- `github.com/stretchr/testify` - Testing framework

### Development Tools
- `golangci-lint` - Comprehensive Go linting
- `govulncheck` - Security vulnerability scanning

## Release and Deployment

### Building Releases
```bash
make release-build       # Build for all platforms
make version            # Show version info
```

### Supported Platforms
- Linux (AMD64, ARM64)
- macOS (AMD64, ARM64)
- Windows (AMD64, ARM64)

## Common Development Tasks

### Adding New Commands
1. Create new file in `cmd/` (e.g., `cmd/newcommand.go`)
2. Register with root command in `cmd/root.go`
3. Add tests in `cmd/newcommand_test.go`
4. Update help text and documentation

### Adding AWS Service Integration
1. Add service client to `internal/aws/`
2. Create service-specific functions
3. Add comprehensive error handling
4. Write unit tests with mocks
5. Add integration tests

### Modifying Configuration Schema
1. Update structs in `internal/config/`
2. Add validation logic
3. Update YAML parsing
4. Add migration logic if needed
5. Update examples and documentation

## Security Considerations

- Never commit AWS credentials or sensitive data
- Use IAM roles with minimal necessary permissions
- Validate all user inputs, especially file paths
- Sanitize CloudFormation template inputs
- Run `govulncheck` regularly for dependency vulnerabilities

## Git Workflow

### Before Committing
```bash
make lint               # Format and lint code
make test              # Run all tests
make commit-check      # Run comprehensive pre-commit checks
```

### Commit Messages
- Do use conventional commit format
- Include context about AWS resources or stacks affected
- Reference issue numbers when applicable

## Commit Workflow and Review Process

### Human Review Requirement
- **Every change** to any project file must be offered to the human for review before committing
- Never commit changes directly without explicit approval from the human
- Present all modifications clearly, explaining what was changed and why
- Wait for the human confirmation before proceeding with `git add` and `git commit`

### Change Presentation Format
When presenting changes to the human:
- Show the file path and nature of changes
- Explain the purpose and impact of each modification
- Highlight any potential breaking changes or dependencies
- Include relevant test results if applicable

### Commit Process
1. **Make Changes** - Implement the required functionality with tests
2. **Run Checks** - Execute `make commit-check` to ensure quality
3. **Present Changes** - Show all modifications to the human for review
4. **Wait for Approval** - Do not proceed until the human explicitly approves
5. **Commit** - Only after approval, stage and commit the changes

**Never push changes to remote repositories** - this is the human's responsibility.

### What Requires Review
All changes to project files, including but not limited to:
- Source code modifications (`.go` files)
- Configuration changes (YAML, Makefile, etc.)
- Documentation updates (README.md, AGENTS.md, etc.)
- Test additions or modifications
- Dependency updates (go.mod, go.sum)
- CI/CD pipeline changes (.github/workflows/)

### Exception Protocol
There are **no exceptions** to the human review requirement. Even minor changes such as:
- Typo fixes
- Comment updates
- Formatting changes
- Import organisation

Must be presented to and approved by the human before committing.

## Debugging and Troubleshooting

### Common Issues
- AWS credential configuration: Check `aws configure list`
- Template validation errors: Use `aws cloudformation validate-template`
- Dependency cycles: Review `depends_on` relationships
- Permission errors: Verify IAM policies

### Debug Flags
- Use `--verbose` flag for detailed output
- Enable AWS SDK logging with environment variables

### Logging
- Use structured logging for AWS operations
- Include request IDs for AWS API calls
- Log stack names and operations clearly
