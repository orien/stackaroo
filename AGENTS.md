# AGENTS.md

## Project Overview

Stackaroo is a Go CLI tool for managing AWS CloudFormation stacks as code. Provides declarative configuration, environment management, change preview, template validation, and dependency management.

**Tech Stack:** Go 1.24, AWS SDK v2, Cobra CLI, YAML configuration, CloudFormation

## Quick Start

```bash
make build              # Build binary
make test               # Run tests  
make lint               # Run linting
./bin/stackaroo --help  # Test CLI
```

## Build System

**Essential Commands:**
- `make build` - Build main binary
- `make build-all` - Build all binaries  
- `make test` - Unit tests
- `make test-aws` - AWS integration (dry-run)
- `make test-aws-live` - Live AWS tests (destructive!)
- `make lint` - Format + vet + golangci-lint
- `make commit-check` - Pre-commit validation

## Project Structure

```
cmd/           - CLI commands
internal/      - Core packages
  aws/         - AWS service interactions
  config/      - Configuration handling
  deploy/      - Deployment logic
  resolve/     - Dependency resolution
examples/      - Usage examples
docs/          - Documentation
```

## Development Standards

### Code Style
- Use `go fmt` (enforced in CI)
- Follow Go naming conventions
- Handle errors explicitly with wrapped context
- Use `context.Context` for cancellation/timeouts
- Write tests alongside code (`file.go` + `file_test.go`)

### Testing Strategy

Choose approach based on context:

**Complex Logic (TDD):** Write failing tests → implement → refactor  
**AWS Integration:** Build + mock together, test success/error paths  
**Simple CLI:** Implement → test edge cases and validation

**Test Categories:**
- **Unit Tests:** Fast, mocked dependencies, run with `go test -short`
- **Integration Tests:** Real AWS (dry-run preferred), use `//go:build integration`

**Test Structure:**
```go
func TestFunction_Scenario_Expected(t *testing.T) {
    // Arrange: setup data/mocks
    // Act: execute function
    // Assert: verify behaviour
    require.NoError(t, err)
    assert.Equal(t, expected, actual)
}
```

**AWS Mocking Pattern:**
```go
type mockCloudFormationAPI struct {
    mock.Mock
}

func (m *mockCloudFormationAPI) CreateStack(ctx context.Context, input *cloudformation.CreateStackInput, opts ...func(*cloudformation.Options)) (*cloudformation.CreateStackOutput, error) {
    args := m.Called(ctx, input)
    return args.Get(0).(*cloudformation.CreateStackOutput), args.Error(1)
}
```

### Language Standards
- Use British English throughout (colour, organisation, optimise)
- Apply to code comments, errors, variables, and documentation
- Use Mermaid for diagrams (flowcharts, sequence, state, class)
- ISO 8601 dates (YYYY/MM/DD)

## AWS Development

### Configuration
- Uses AWS SDK v2 credential chain
- Supports profiles (`--profile` or `AWS_PROFILE`)
- Region from config/environment or `--region` flag

### CloudFormation Integration
- Validate templates before deployment
- Handle stack lifecycle (create/update/delete)
- Support parameter files and dependency resolution
- Implement `depends_on` relationships

### Testing Guidelines
- **Never use production accounts**
- Use separate dev account/profile
- Dry-run mode is default for safety
- Clean up resources after testing

```bash
make test-aws                    # Safe dry-run
PROFILE=dev make aws-test-profile # With specific profile
make aws-test-us-east-1         # Specific region
```

## Configuration Management

**File Structure:**
```
stackaroo.yml           # Main configuration
templates/
  vpc.yml              # CloudFormation templates
  app.yml
```

**Features:**
- Stack definitions in YAML
- Context-specific parameter overrides
- Template path resolution
- Dependency declarations with `depends_on`

## Dependencies

**Core:**
- `github.com/aws/aws-sdk-go-v2` - AWS SDK
- `github.com/spf13/cobra` - CLI framework  
- `gopkg.in/yaml.v3` - YAML parsing
- `github.com/stretchr/testify` - Testing

**Dev Tools:**
- `golangci-lint` - Linting
- `govulncheck` - Security scanning

## Common Tasks

### Adding Commands
1. Create `cmd/newcommand.go`
2. Register in `cmd/root.go` 
3. Add tests `cmd/newcommand_test.go`
4. Update documentation

### Adding AWS Services
1. Add client to `internal/aws/`
2. Create service functions with error handling
3. Write unit tests with mocks
4. Add integration tests

### Modifying Configuration
1. Update structs in `internal/config/`
2. Add validation and YAML parsing
3. Add migration logic if needed
4. Update examples

## CI/CD Pipeline

GitHub Actions runs:
1. **Test** - Unit tests with race detection
2. **Lint** - golangci-lint 
3. **Security** - govulncheck
4. **Build** - Cross-platform (Linux/macOS/Windows, AMD64/ARM64)
5. **Integration** - Basic CLI functionality

All checks must pass before merge.

## Security
- Never commit credentials
- Use minimal IAM permissions
- Validate all inputs (especially file paths)
- Sanitise CloudFormation inputs
- Run `govulncheck` regularly

## Git Workflow

### Pre-commit
```bash
make lint          # Format and lint
make test         # Run tests  
make commit-check # Full validation
```

### Commit Standards
- Use conventional commit format
- Include AWS resource context
- Reference issue numbers

## Human Review Requirement

**All changes require human approval before committing:**
- Present modifications with file paths and explanations
- Highlight breaking changes and dependencies  
- Include test results if applicable
- Wait for explicit approval before `git add` and `git commit`
- **No exceptions** - applies to all commits including amendments
- Never push to remote (human responsibility)

**Applies to all changes:**
- Source code (`.go` files)
- Configuration (YAML, Makefile)
- Documentation (README.md, AGENTS.md)
- Tests and dependencies
- CI/CD pipeline changes

## Debugging

**Common Issues:**
- Credentials: `aws configure list`
- Templates: `aws cloudformation validate-template`  
- Dependencies: Review `depends_on` cycles
- Permissions: Verify IAM policies

**Debug Tools:**
- Use `--verbose` flag
- Enable AWS SDK logging via environment
- Structured logging with request IDs
- Clear stack name/operation logging

**Release:**
```bash
make release-build  # All platforms
make version       # Version info
```
