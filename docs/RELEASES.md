# Releases

This document describes the release process for Stackaroo, including automated releases via GitHub Actions and GoReleaser.

## Overview

Stackaroo uses a tag-based release system with automated builds and publishing:

- **Version Management**: Semantic versioning (e.g., `v1.2.3`)
- **Automated Builds**: GoReleaser creates cross-platform binaries
- **GitHub Releases**: Automated release creation with changelogs
- **Artefact Distribution**: Binaries, checksums, and archives
- **Quality Gates**: Tests, linting, and security scans before release

## Quick Release Process

### 1. Using the Helper Script (Recommended)

```bash
# Create and push a release tag
./scripts/release.sh 1.2.3

# Or with pre-release suffix
./scripts/release.sh 1.2.4-rc1

# Dry run to validate without creating tag
./scripts/release.sh --dry-run 1.2.3
```

### 2. Using Make Targets

```bash
# Check configuration and prepare for release
make release-prepare

# Create release (requires VERSION argument)
make release VERSION=1.2.3
```

### 3. Manual Process

```bash
# Update VERSION file
echo "1.2.3" > VERSION

# Commit and tag
git add VERSION
git commit -m "Release v1.2.3"
git tag -a v1.2.3 -m "Release v1.2.3"
git push origin HEAD
git push origin v1.2.3
```

## Release Workflow

### Automated Process

When a version tag (e.g., `v1.2.3`) is pushed to GitHub:

1. **Quality Checks**: Tests, linting, security scans
2. **Version Validation**: Tag format and VERSION file consistency
3. **GoReleaser Build**: Cross-platform binaries and archives
4. **GitHub Release**: Automated release creation with changelog
5. **Artefact Upload**: Binaries, checksums, and documentation

### Build Targets

GoReleaser builds for the following platforms:

- **Linux**: AMD64, ARM64
- **macOS**: AMD64, ARM64 (with Universal Binary)
- **Windows**: AMD64, ARM64

### Release Artefacts

Each release includes:

- **Binary Archives**: `.tar.gz` for Unix, `.zip` for Windows
- **Checksums**: SHA256 checksums for all artefacts
- **Documentation**: README, LICENSE, docs, and examples
- **Change Log**: Automatically generated from commit history

## Version Management

### Semantic Versioning

Stackaroo follows semantic versioning (`MAJOR.MINOR.PATCH`):

- **MAJOR**: Breaking changes to CLI interface or configuration
- **MINOR**: New features, backward-compatible
- **PATCH**: Bug fixes, backward-compatible

### Pre-release Versions

Pre-release versions use suffixes:

- `1.2.3-rc1`: Release candidate
- `1.2.3-beta1`: Beta version
- `1.2.3-alpha1`: Alpha version

### Version Sources

Version information comes from multiple sources:

1. **VERSION file**: Base version (e.g., `1.2.3`)
2. **Git tags**: Release tags (e.g., `v1.2.3`)
3. **Git commit**: Short commit hash for development builds
4. **Build timestamp**: When the binary was built

## Configuration Files

### .goreleaser.yml

GoReleaser configuration defining:

- Build targets and flags
- Archive formats and contents
- Release notes generation
- Validation rules

Key sections:

```yaml
builds:
  - binary: stackaroo
    goos: [linux, darwin, windows]
    goarch: [amd64, arm64]
    ldflags:
      - -X 'github.com/orien/stackaroo/internal/version.Version={{.Version}}'

archives:
  - files: [README.md, LICENSE, docs/**/*]

changelog:
  groups:
    - title: Features
    - title: Bug fixes
```

### GitHub Actions Workflow

`.github/workflows/release.yml` handles:

- Quality gates (test, lint, security)
- Version validation
- GoReleaser execution
- Post-release verification

## Using Released Binaries

### Installation Methods

#### 1. Download from GitHub Releases

```bash
# Download and extract (Linux/macOS)
curl -sL https://github.com/orien/stackaroo/releases/download/v1.2.3/stackaroo-1.2.3-linux-x86_64.tar.gz | tar -xz
cd stackaroo-1.2.3-linux-x86_64
sudo mv stackaroo /usr/local/bin/
cd .. && rm -rf stackaroo-1.2.3-linux-x86_64

# Or download manually from:
# https://github.com/orien/stackaroo/releases
```

#### 2. Using Go Install

```bash
go install github.com/orien/stackaroo@v1.2.3
```

#### 3. Using Checksums

Always verify downloads using the provided checksums:

```bash
# Download checksum file
curl -sL https://github.com/orien/stackaroo/releases/download/v1.2.3/checksums.txt

# Verify binary
sha256sum -c checksums.txt --ignore-missing
```

## Development Releases

### Snapshot Builds

Create snapshot builds without git tags:

```bash
# Build snapshot with GoReleaser
make goreleaser-snapshot

# Check output
ls -la dist/
```

### Manual Builds

Build specific platforms manually:

```bash
# Build for current platform
make build

# Build for multiple platforms
make release-build

# Build specific platform
GOOS=linux GOARCH=amd64 go build -o stackaroo-linux-amd64 .
```

## Troubleshooting Releases

### Common Issues

#### Tag Already Exists
```bash
# Delete local tag
git tag -d v1.2.3

# Delete remote tag (if needed)
git push origin --delete v1.2.3
```

#### VERSION File Mismatch
```bash
# Update VERSION file to match tag
echo "1.2.3" > VERSION
git add VERSION
git commit --amend --no-edit
```

#### Failed Quality Checks
```bash
# Run checks locally
make test
make lint
govulncheck ./...

# Fix issues and retry
git add .
git commit -m "Fix release issues"
git push
```

#### Release Workflow Failed
1. Check GitHub Actions logs
2. Fix issues in code or configuration
3. Delete and recreate tag if necessary

### Debugging Commands

```bash
# Check current version info
make version

# Validate GoReleaser config
make goreleaser-check

# Dry run release
make goreleaser-dry-run

# Check git status
make git-check
```

## Release Checklist

### Pre-Release

- [ ] All tests pass (`make test`)
- [ ] Code passes linting (`make lint`)
- [ ] No security vulnerabilities (`govulncheck ./...`)
- [ ] Documentation updated
- [ ] CHANGELOG.md updated (if maintained manually)
- [ ] VERSION file reflects new version

### Release

- [ ] Create release tag (`./scripts/release.sh X.Y.Z`)
- [ ] Verify GitHub Actions workflow succeeds
- [ ] Test downloaded binaries work correctly
- [ ] Verify checksums match

### Post-Release

- [ ] Update documentation with new version
- [ ] Announce release (if applicable)
- [ ] Monitor for issues from users

## Manual Release Process

If automated releases fail, you can create releases manually:

### 1. Build Locally

```bash
# Install GoReleaser
go install github.com/goreleaser/goreleaser@latest

# Create local builds
goreleaser release --skip=publish --clean
```

### 2. Create GitHub Release

```bash
# Using GitHub CLI
gh release create v1.2.3 \
  --title "Stackaroo v1.2.3" \
  --notes "Release notes here" \
  dist/*.tar.gz \
  dist/*.zip \
  dist/checksums.txt
```

### 3. Upload Artefacts

Upload the contents of `dist/` to the GitHub release.

## Configuration Reference

### Required Secrets

GitHub Actions workflow requires:

- `GITHUB_TOKEN`: Automatically provided by GitHub

### Optional Configuration

- **Homebrew**: Uncomment brew section in `.goreleaser.yml`
- **Docker**: Add Docker build configuration
- **Signing**: Add binary signing configuration

### Environment Variables

- `GORELEASER_CURRENT_TAG`: Current release tag
- `GITHUB_TOKEN`: GitHub API token for release creation

## Best Practices

1. **Always test releases** with `--dry-run` first
2. **Use semantic versioning** consistently
3. **Keep VERSION file** in sync with tags
4. **Write clear commit messages** for changelog generation
5. **Test binaries** after release
6. **Monitor GitHub Actions** for failures
7. **Keep release notes** informative and user-focused

## Getting Help

- **GitHub Issues**: Report release problems
- **GitHub Discussions**: Ask questions about releases
- **Logs**: Check GitHub Actions logs for detailed error information
- **Local Testing**: Use `make goreleaser-snapshot` for local testing