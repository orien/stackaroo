# 25. Codeberg release platform

Date: 2026-01-24

## Status

Accepted

Complements [24. Codeberg hosting and Woodpecker CI migration](0024-codeberg-hosting-and-woodpecker-ci.md)

## Context

Following the migration to Codeberg for primary development (ADR 0024), we needed to decide where to host and automate software releases. Release distribution has unique considerations beyond general CI/CD:

- **Discoverability**: Users need to find and download releases easily
- **Automation**: Release builds, artefact generation, and publishing must be automated
- **Trust and Security**: Checksums, reproducible builds, and secure distribution
- **Consistency**: Aligning infrastructure with project values (FOSS principles)

**Release Platform Options:**

1. **GitHub Releases**: Established ecosystem, good discoverability, GitHub Actions + GoReleaser integration
2. **Codeberg Releases**: Aligned with FOSS values, Forgejo/Gitea native, GoReleaser support via Gitea mode
3. **Hybrid Approach**: CI on Codeberg, releases on GitHub (maintains discoverability)

GoReleaser supports Gitea/Forgejo natively, Codeberg provides a compatible release API, and `go install` works identically for both platforms.

## Decision

We will **perform releases exclusively on Codeberg** using Woodpecker CI and GoReleaser.

**Release Infrastructure:**

- **Automation**: Woodpecker CI (`.woodpecker/release.yaml`) triggers on version tags with quality gates
- **Build and Publish**: GoReleaser (`.goreleaser.yaml`) builds cross-platform binaries and publishes to Codeberg Releases API
- **Distribution**: Codeberg Releases at `https://codeberg.org/orien/stackaroo/releases`

We will **not** maintain GitHub releases or mirrors. All releases exclusively on Codeberg.

## Consequences

### Positive

- **Value Consistency**: Release infrastructure fully aligned with FOSS principles
- **Simplicity**: Single platform for code, CI, issues, and releases
- **No Platform Fragmentation**: Avoids confusion about canonical release location
- **Community Support**: Strengthens Codeberg ecosystem by using it fully
- **Technical Sovereignty**: Complete control via self-hostable Forgejo if needed
- **Go Module Transparency**: `go install` works identically regardless of hosting

### Negative

- **Discoverability**: Codeberg's smaller user base reduces organic discovery
  - **Mitigation**: Clear documentation, README installation instructions, prominent download links
  
- **Familiarity**: Some users accustomed to GitHub releases
  - **Mitigation**: Documentation explains Codeberg usage, identical download UX

- **Ecosystem Integration**: Fewer third-party tools integrate with Codeberg
  - **Mitigation**: Most tools use `go install` or direct binary downloads which work fine

- **Search Engine Ranking**: GitHub releases may rank higher in search results
  - **Mitigation**: SEO-optimised documentation, direct links in README

## References

- [Codeberg](https://codeberg.org)
- [GoReleaser Gitea Support](https://goreleaser.com/customization/gitea/)
