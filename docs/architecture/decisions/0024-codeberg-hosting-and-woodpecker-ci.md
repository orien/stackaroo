# 24. Codeberg hosting and Woodpecker CI migration

Date: 2026-01-08

## Status

Accepted

Supersedes [12. CI/Release platform choice](0012-cicd-platform.md)

## Context

Stackaroo is currently hosted on GitHub with GitHub Actions providing CI/CD infrastructure. As a Free and Open Source Software (FOSS) project, we should consider whether our infrastructure aligns with FOSS principles:

- **Digital Sovereignty**: Dependency on proprietary platforms (GitHub/Microsoft) conflicts with FOSS values
- **Community Infrastructure**: Supporting non-profit alternatives strengthens the ecosystem
- **Privacy**: GitHub includes tracking; contributors deserve privacy-respecting infrastructure
- **Values Alignment**: Infrastructure choices should reflect project principles

**Codeberg** (codeberg.org) offers a FOSS alternative:
- Non-profit organisation (Codeberg e.V., Berlin, Germany)
- Powered by Forgejo (FOSS fork of Gitea)
- No tracking, no profiteering, community-first
- Woodpecker CI for automated workflows
- 255,000+ users, 387,000+ projects

## Decision

We will **migrate primary development to Codeberg** completely. We will **migrate only the primary CI workflow** (ci.yml) to Woodpecker CI initially, keeping documentation and release workflows on GitHub Actions temporarily to reduce migration risk.

**Primary Platform: Codeberg**
- Git repository: `codeberg.org/orien/stackaroo`
- CI/CD: Woodpecker CI for testing, linting, security, building
- Issue tracking and pull requests
- All development activity

**Temporary GitHub Usage:**
- Documentation building (GitHub Pages via docs.yml)
- Release automation (GoReleaser via release.yml)

**Migration Scope:**
- **Now**: Convert `.github/workflows/ci.yml` → `.woodpecker/ci.yaml`
- **Later**: Evaluate migrating docs.yml and release.yml based on CI migration success

**Transition Plan:**
1. Import repository to Codeberg with full history
2. Convert CI workflow to Woodpecker CI and test thoroughly
3. Turn off issuses in GitHub repository
4. Keep GitHub Actions running for docs/releases temporarily

## Consequences

### Positive

- **Values alignment**: Infrastructure reflects FOSS principles and digital sovereignty
- **Community support**: Strengthens non-profit, community-driven alternatives
- **Privacy**: No tracking or data collection on contributors
- **Technical independence**: Forgejo/Woodpecker are self-hostable; reduces vendor lock-in
- **Simplicity**: No complex mirroring strategy needed (CLI tool, not library)
- **Gradual migration**: Low-risk approach; docs/releases remain on proven infrastructure

### Negative

- **Reduced discoverability**: Smaller platform means less organic project discovery
- **Learning curve**: Team and contributors must learn Woodpecker CI syntax
- **Split infrastructure**: Managing CI on Codeberg whilst docs/releases on GitHub adds temporary complexity
- **Migration effort**: 2-4 hours for workflow conversion and testing
- **Potential contributor friction**: Some may prefer GitHub ecosystem
- **Community transition**: Existing contributors need to adapt to new platform

## References

- [Codeberg](https://codeberg.org)
- [Woodpecker CI](https://woodpecker-ci.org/)
- [Give Up GitHub Campaign](https://sfconservancy.org/GiveUpGitHub/)
