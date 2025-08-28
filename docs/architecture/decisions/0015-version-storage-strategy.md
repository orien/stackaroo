# 15. Version storage strategy

Date: 2025-01-28

## Status

Accepted

## Context

Stackaroo requires a comprehensive version tracking system to support:
- **User identification** of installed binary version for support and debugging
- **Release management** with semantic versioning for public distribution
- **Development visibility** into exact build state and commit information
- **CI/CD integration** with automated build and release processes
- **Debugging support** with detailed build metadata (commit hash, build date, platform)

Currently, Stackaroo has **no version tracking mechanism**. The CLI framework (Cobra) expects version information, and there's already a placeholder test case for `--version` functionality that currently fails.

Several approaches were considered for version storage:

1. **Git-only approach**: Derive all version information from git tags and commits
   - **Pros**: No additional files, git is authoritative source
   - **Cons**: Requires git repository for builds, complex version logic, poor developer experience

2. **Static version file only**: Store complete version in a single file
   - **Pros**: Simple implementation, works without git
   - **Cons**: No development build differentiation, requires manual updates for every build

3. **Code constant approach**: Hard-code version in Go source files
   - **Pros**: Simple to implement, no external dependencies
   - **Cons**: Requires code changes for version updates, no build-time information

4. **Hybrid Git + VERSION file**: Combine semantic version file with git metadata enhancement
   - **Pros**: Best of both worlds, clear release process, detailed development info
   - **Cons**: More complex build system, multiple sources of truth to coordinate

The hybrid approach is widely adopted by mature CLI tools (kubectl, terraform, docker) and provides the best balance of usability and functionality.

## Decision

We will implement a **hybrid Git + VERSION file approach** for version storage and tracking.

**Core Components:**

1. **VERSION file**: Contains base semantic version
   ```
   1.0.0
   ```

2. **Build-time enhancement**: Git information enriches base version during compilation
   ```
   # Release builds (on tagged commits)
   v1.0.0

   # Development builds
   1.0.0+a1b2c3d        # Clean working directory
   1.0.0+a1b2c3d-dirty  # With uncommitted changes
   ```

3. **Build system integration**: Makefile and CI inject version via `-ldflags`
   ```makefile
   BASE_VERSION := $(shell cat VERSION)
   GIT_COMMIT := $(shell git rev-parse --short HEAD)
   VERSION := $(BASE_VERSION)+$(GIT_COMMIT)
   LDFLAGS := -ldflags="-X 'package.Version=$(VERSION)'"
   ```

4. **Code structure**: Dedicated version package with build-time variables
   ```go
   package version

   var (
       Version   = "dev"     // Injected via ldflags
       GitCommit = "unknown" // Injected via ldflags
       BuildDate = "unknown" // Injected via ldflags
   )
   ```

**Version Resolution Logic:**
- **VERSION file** provides semantic version base (1.0.0)
- **Git tag presence** determines clean vs development version format
- **Working directory state** adds dirty suffix if uncommitted changes exist
- **Build metadata** includes commit hash, build date, Go version, platform

**Release Workflow:**
1. Update VERSION file to target version (e.g., `1.1.0`)
2. Create matching git tag (`v1.1.0`)
3. Build release binaries (show clean `v1.1.0`)
4. Continue development (shows `1.1.0+<new-commits>`)

**CLI Integration:**
- Cobra `--version` flag displays comprehensive version information
- Custom version template shows all build metadata
- Version accessible programmatically for debugging and telemetry

## Consequences

**Positive:**
- **Clear version authority**: VERSION file provides single source of truth for semantic versions
- **Rich development information**: Git metadata enables precise build identification
- **Professional user experience**: Comprehensive version output similar to mature CLI tools
- **Automated release process**: Build system handles version formatting and injection automatically
- **Debugging support**: Full build context available for troubleshooting
- **CI/CD friendly**: Works seamlessly with automated build and release pipelines
- **Backward compatible**: Can be implemented without breaking existing functionality
- **Industry standard**: Follows patterns used by established infrastructure tools

**Negative:**
- **Build system complexity**: More sophisticated Makefile and CI configuration required
- **Multiple sources coordination**: VERSION file and git tags must be kept in sync during releases
- **Git dependency**: Full version information requires git repository (falls back gracefully)
- **Learning curve**: Team must understand version resolution rules and release workflow
- **Additional maintenance**: VERSION file requires manual updates during release process

**Implementation Requirements:**
- Create `stackaroo/VERSION` file with initial version (0.1.0)
- Implement `stackaroo/internal/version` package with build-time variables
- Update `stackaroo/cmd/root.go` to integrate version with Cobra CLI
- Enhance Makefile with version resolution and ldflags injection
- Update CI/CD pipeline to pass version information during builds
- Add comprehensive tests for version functionality
- Document release workflow and version management processes

**Example Version Output:**
```
$ stackaroo --version
stackaroo v1.0.0
  Git commit: a1b2c3d
  Build date: 2025-01-27 14:30:45 UTC
  Go version: go1.24.2
  Platform:   linux/amd64
```

This approach provides professional version management whilst maintaining simplicity for both users and developers, establishing a solid foundation for release management and user support.
