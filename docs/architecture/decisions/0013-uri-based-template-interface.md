# 13. URI-based template interface

Date: 2025-08-24

## Status

Accepted

## Context

The original architecture had several responsibility leaks between the configuration and resolution modules:

1. **File path assumptions**: `StackConfig.Template` was assumed to be a file path, creating tight coupling between configuration and file system
2. **Hardcoded file knowledge**: The `cmd/deploy` module contained hardcoded knowledge of "stackaroo.yaml" filename
3. **Limited extensibility**: Template loading was restricted to local file system only
4. **Leaky abstractions**: The resolve module was performing file I/O operations, mixing business logic with infrastructure concerns

These issues violated clean architecture principles and prevented the system from supporting multiple template sources (S3, Git repositories, HTTP endpoints, etc.).

The existing interface between configuration and resolution was:
```go
type StackConfig struct {
    Template string  // Assumed to be file path
}

type TemplateReader interface {
    ReadTemplate(templatePath string) (string, error)  // File-specific
}
```

This design made several problematic assumptions:
- Templates must be files on the local file system
- Configuration providers must know about file system structure
- Resolution module must handle file I/O directly
- Extension to new template sources requires changes across multiple modules

## Decision

We will use **URI-based template references** in the interface between configuration and resolution modules.

The new interface design:
```go
type StackConfig struct {
    Template string  // URI: file://, s3://, git://, http://, etc.
}

type TemplateReader interface {
    ReadTemplate(templateURI string) (string, error)  // URI-agnostic
}
```

Key architectural changes:
1. **`StackConfig.Template` becomes a URI** - No assumptions about template source
2. **Configuration providers convert paths to URIs** - `file.Provider` converts relative paths to `file://` URIs
3. **Resolution module handles URI parsing** - `resolve.FileTemplateReader` parses `file://` URIs
4. **Factory methods encapsulate filenames** - `file.NewDefaultProvider()` hides "stackaroo.yaml" knowledge

URI scheme examples:
- `file:///project/templates/vpc.yaml` - Local file system
- `s3://bucket/templates/vpc.yaml` - AWS S3 bucket
- `git://repo.com/templates/vpc.yaml` - Git repository
- `http://server.com/templates/vpc.yaml` - HTTP endpoint

## Consequences

**Positive:**
- **Clean separation of concerns** - Configuration providers handle URI generation, resolution handles URI parsing
- **Extensible template sources** - Easy to add S3, Git, HTTP template readers without changing interfaces
- **No hardcoded filenames** - cmd module uses factory methods instead of hardcoded "stackaroo.yaml"
- **Protocol flexibility** - Same interface supports multiple transport mechanisms
- **Future-proof design** - New template sources require no changes to existing modules
- **Better testability** - Easy to mock different URI schemes for testing
- **Consistent abstraction** - All template references are treated uniformly regardless of source

**Negative:**
- **Additional complexity** - URI parsing and scheme handling adds implementation overhead
- **Backward compatibility** - Must support both URI and legacy path formats during transition
- **URI validation** - Need to validate URI schemes and prevent malicious URIs
- **Error handling complexity** - Different URI schemes may have different failure modes
- **Performance overhead** - URI parsing adds small computational cost

**Implementation Requirements:**
- URI parsing in resolve module with scheme detection
- Backward compatibility for relative paths in file provider
- Security validation for URI schemes and paths
- Clear error messages for invalid URIs or unsupported schemes
- Documentation of supported URI schemes and their semantics

**Migration Impact:**
- File provider automatically converts existing path configurations to file:// URIs
- Existing YAML configurations continue to work without changes
- Tests updated to expect URI format instead of plain paths
- Documentation updated to reflect URI-based architecture

This change establishes a clean architectural boundary between configuration and resolution, enabling future extensibility while maintaining compatibility with existing configurations.