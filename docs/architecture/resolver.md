# Resolver Module Architecture

## Overview

The resolver module is responsible for transforming high-level configuration into deployment-ready artifacts. It serves as the bridge between the configuration system and the deployment system, handling dependency resolution, parameter inheritance, and template processing.

## Design Principles

### 1. **Format Agnostic**
- No knowledge of YAML, JSON, or other configuration formats
- Works with abstract configuration types from the config package
- Pure business logic focused on resolution and transformation

### 2. **Dependency Injection**
- Accepts `config.ConfigProvider` interface for configuration access
- Accepts `TemplateReader` interface for template file access
- Fully testable through interface mocking

### 3. **Single Responsibility**
- **Only** responsible for resolving configuration into deployment artifacts
- Does not handle deployment, file I/O, or format parsing
- Clear separation from configuration loading and AWS operations

### 4. **Deterministic Behavior**
- Dependency resolution produces consistent, repeatable ordering
- Parameter inheritance follows predictable precedence rules
- All operations are stateless and side-effect free

## Architecture Components

### Core Types

```go
type Resolver struct {
    configProvider config.ConfigProvider
    templateReader TemplateReader
}

type ResolvedStack struct {
    Name         string
    TemplateBody string
    Parameters   map[string]string
    Tags         map[string]string
    Capabilities []string
    Dependencies []string
}

type ResolvedStacks struct {
    Context         string
    Stacks          []*ResolvedStack
    DeploymentOrder []string
}
```

### Interfaces

#### ConfigProvider (from config package)
```go
type ConfigProvider interface {
    LoadConfig(ctx context.Context, context string) (*Config, error)
    GetStack(stackName, context string) (*StackConfig, error)
    ListContexts() ([]string, error)
    Validate() error
}
```

#### TemplateReader
```go
type TemplateReader interface {
    ReadTemplate(templateURI string) (string, error)
}
```

## Resolution Process

### 1. Single Stack Resolution

```mermaid
flowchart LR
    A[Load Config<br/>context] --> B[Get Stack<br/>Config]
    B --> C[Read Template<br/>File]
    C --> D[Resolve Stack<br/>Parameters/Tags]
```

**Steps:**
1. **Load Configuration** - Get global config for context
2. **Get Stack Config** - Retrieve stack-specific configuration
3. **Read Template** - Load CloudFormation template content from URI
4. **Merge Parameters** - Apply inheritance rules
5. **Merge Tags** - Combine global and stack tags
6. **Create ResolvedStack** - Package everything together

### 2. Multi-Stack Resolution

```mermaid
flowchart LR
    A[Resolve Each<br/>Stack<br/>Individually] --> B[Calculate<br/>Dependencies<br/>Order]
    B --> C[Create<br/>ResolvedStacks]
```

**Steps:**
1. **Resolve Individual Stacks** - Process each stack through single resolution
2. **Build Dependency Graph** - Create adjacency list from dependencies
3. **Topological Sort** - Use Kahn's algorithm for deployment order
4. **Detect Cycles** - Identify circular dependencies and fail fast
5. **Package Results** - Combine into final ResolvedStacks

## Dependency Management

### Topological Sorting Algorithm

The resolver uses **Kahn's algorithm** for dependency resolution:

1. **Build Graph** - Create adjacency list from stack dependencies
2. **Calculate In-Degrees** - Count incoming dependencies for each stack
3. **Initialize Queue** - Add stacks with zero dependencies
4. **Process Queue** - Remove nodes, update in-degrees, add newly eligible stacks
5. **Detect Cycles** - If not all stacks processed, circular dependency exists

### Dependency Features

- **Missing Dependencies** - Dependencies not in resolution set are ignored
- **Deterministic Ordering** - Queue is sorted for consistent results
- **Cycle Detection** - Fails fast with clear error messages
- **Complex Chains** - Handles deep dependency hierarchies (A→B→C→D)

### Example Dependency Resolution

```yaml
# Configuration
stacks:
  - name: vpc
    dependencies: []
  - name: security  
    dependencies: [vpc]
  - name: database
    dependencies: [security]
  - name: app
    dependencies: [database]

# Resolved Order: vpc → security → database → app
```

## Parameter and Tag Inheritance

### Inheritance Hierarchy

```mermaid
flowchart TD
    A[Global Config Tags] --> B[Stack Tags<br/>override global]
    B --> C[Final Resolved Tags]
```

### Merge Strategy

**Tags:**
1. Start with global tags from configuration
2. Add/override with stack-specific tags
3. Stack tags take precedence over global tags

**Parameters:**
1. Currently: Stack parameters only
2. **Future**: Global → Stack → Context parameter inheritance

### Example Inheritance

```yaml
# Global Config
tags:
  Project: "my-project"
  Environment: "dev"

# Stack Config  
tags:
  Project: "overridden-project"  # Overrides global
  Component: "web-server"        # New tag

# Result
tags:
  Project: "overridden-project"
  Environment: "dev"
  Component: "web-server"
```

## Integration Architecture

### Module Dependencies

```mermaid
flowchart LR
    A[Config<br/>providers] --> B[Resolver<br/>business<br/>logic]
    B --> C[Deploy<br/>AWS ops]
```

### Data Flow

1. **Config Provider** loads YAML, converts to config types
2. **Resolver** transforms config types to resolved artifacts
3. **Deploy** uses resolved artifacts for AWS API calls

### CLI Integration

```go
// Example usage in deploy command
configProvider := file.NewDefaultProvider()  // No hardcoded filename
templateReader := &resolve.FileTemplateReader{}
resolver := resolve.NewResolver(configProvider, templateReader)

resolved, err := resolver.Resolve(ctx, "dev", []string{"vpc", "app"})
if err != nil {
    return fmt.Errorf("resolution failed: %w", err)
}

// Deploy in dependency order
for _, stackName := range resolved.DeploymentOrder {
    stack := findStack(resolved.Stacks, stackName)
    err := deployer.DeployResolvedStack(ctx, stack)
    // ...
}
```

## Architectural Separation Improvements

### Responsibility Boundaries

The resolver module has been designed with clear separation of concerns to prevent responsibility leaks:

```mermaid
graph TD
    A[cmd/deploy<br/>CLI Orchestration] --> B[config/file<br/>Configuration & URI Resolution]
    A --> C[resolve<br/>Business Logic & Template Loading]
    
    B --> D[stackaroo.yaml<br/>File Knowledge]
    C --> E[URI Parsing<br/>file://, s3://, git://]
    
    style A fill:#e1f5fe
    style B fill:#f3e5f5
    style C fill:#f1f8e9
    style D fill:#fff3e0
    style E fill:#fff3e0
```

### Module Responsibilities

#### **cmd/deploy Module**
- **Pure CLI orchestration** - No file system knowledge
- **Uses factory methods** - `file.NewDefaultProvider()` instead of hardcoded filenames
- **Dependency injection** - Accepts resolver and configuration provider interfaces

#### **config/file Module**  
- **Owns "stackaroo.yaml" filename** - Via `NewDefaultProvider()` factory
- **Path-to-URI conversion** - Converts relative paths to `file://` URIs
- **Configuration file knowledge** - Understands YAML structure and resolution

#### **resolve Module**
- **URI-based template loading** - No assumptions about template sources
- **Business logic only** - Dependency resolution, parameter inheritance
- **Template preprocessing** - Handles multiple URI schemes (file://, s3://, git://)

### URI-Based Template Architecture

Templates are now handled as URIs throughout the system:

```go
// Before: Path-based (leaky abstraction)
type StackConfig struct {
    Template string  // Assumed to be file path
}

// After: URI-based (clean abstraction)  
type StackConfig struct {
    Template string  // URI: file://, s3://, git://, etc.
}

// Resolver handles URI parsing and loading
type TemplateReader interface {
    ReadTemplate(templateURI string) (string, error)
}
```

### Benefits of New Architecture

1. **No Hardcoded Filenames** - `cmd` module doesn't know about "stackaroo.yaml"
2. **URI Flexibility** - Templates can come from files, S3, Git, HTTP, etc.
3. **Clean Boundaries** - Each module has single, clear responsibility
4. **Testable Design** - Easy to mock template sources via URI interfaces
5. **Future Extensibility** - New template sources require no changes to cmd/config modules

## Error Handling

### Error Categories

1. **Configuration Errors**
   - Context not found
   - Stack not found
   - Invalid configuration

2. **Template Errors**
   - Template file not found
   - Template read permission errors

3. **Dependency Errors**
   - Circular dependencies
   - Invalid dependency references

4. **Resolution Errors**
   - Parameter inheritance conflicts
   - Missing required fields

### Error Propagation

- **Wrapped Errors** - All errors include context about what failed
- **Early Failure** - Stop resolution at first error
- **Clear Messages** - Error messages include stack names and contexts
- **No Partial Results** - Either complete success or complete failure

## Testing Strategy

### Mock-Based Testing

```go
type MockConfigProvider struct {
    mock.Mock
}

func (m *MockConfigProvider) LoadConfig(ctx context.Context, context string) (*config.Config, error) {
    args := m.Called(ctx, context)
    return args.Get(0).(*config.Config), args.Error(1)
}
```

### Test Coverage Areas

1. **Happy Path** - Successful single and multi-stack resolution
2. **Error Scenarios** - Config load, stack not found, template read failures
3. **Dependency Logic** - Complex chains, cycles, missing dependencies
4. **Inheritance** - Parameter and tag merging with overrides
5. **Edge Cases** - Empty stack lists, stacks with no dependencies

### Testing Principles

- **Interface-Based** - Mock all external dependencies
- **Deterministic** - All tests produce consistent results
- **Comprehensive** - Cover all code paths and error conditions
- **Fast** - No I/O operations, pure unit tests

## Performance Considerations

### Resolution Complexity

- **Single Stack**: O(1) - Constant time operations
- **Multi-Stack**: O(V + E) - Where V = stacks, E = dependencies
- **Topological Sort**: Linear time complexity
- **Memory Usage**: Proportional to number of stacks and dependencies

### Optimization Features

- **Stateless Operations** - No caching or state management overhead
- **Minimal Allocations** - Reuse maps and slices where possible
- **Early Termination** - Stop on first error or cycle detection
- **Sorted Processing** - Deterministic ordering without performance penalty

## Extension Points

### Adding New Resolution Logic

1. **Parameter Sources** - Extend parameter inheritance beyond stack level
2. **Template Processing** - Add template validation or transformation
3. **Dependency Types** - Support different dependency relationships
4. **Output Formats** - Generate different artifact types

### Template Reader Implementations

```go
// File-based templates (handles file:// URIs)
type FileTemplateReader struct{}

func (ftr *FileTemplateReader) ReadTemplate(templateURI string) (string, error) {
    // Parses file:// URI and reads from local filesystem
    // Supports both file://path and relative path formats
}

// S3-based templates (handles s3:// URIs)
type S3TemplateReader struct{}

func (str *S3TemplateReader) ReadTemplate(templateURI string) (string, error) {
    // Parses s3://bucket/key URI and reads from S3
}

// Git-based templates (handles git:// URIs)
type GitTemplateReader struct{}

func (gtr *GitTemplateReader) ReadTemplate(templateURI string) (string, error) {
    // Parses git://repo/branch/path URI and reads from Git
}
```

### Future Enhancements

1. **Context Parameter Inheritance** - Global → Stack → Context hierarchy
2. **Template Validation** - Validate CloudFormation syntax during resolution
3. **Conditional Dependencies** - Dependencies based on parameter values
4. **Template Preprocessing** - Jinja2-style template processing
5. **Parallel Resolution** - Resolve independent stacks concurrently
6. **Resolution Caching** - Cache resolved artifacts for performance

## Security Considerations

### Input Validation

- **Template URIs** - Validate URI schemes and prevent malicious URIs
- **Parameter Values** - Sanitize user inputs  
- **Stack Names** - Validate naming conventions

### Template Security

- **Template Sources** - Validate template URI schemes and origins
- **URI Parsing** - Safe parsing of file://, s3://, git:// URIs
- **Content Scanning** - Basic checks for malicious content
- **Size Limits** - Prevent memory exhaustion from large templates

### Dependency Validation

- **Cycle Prevention** - Robust circular dependency detection
- **Depth Limits** - Prevent excessive dependency chains
- **Name Validation** - Ensure dependency references are valid

This architecture provides a robust, testable, and extensible foundation for configuration resolution while maintaining clear separation of concerns and following clean architecture principles.