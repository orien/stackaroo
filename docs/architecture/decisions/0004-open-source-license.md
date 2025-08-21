# 4. Open source license

Date: 2025-08-21

## Status

Accepted

## Context

We need to choose an open source license for Stackaroo that balances openness with practical adoption considerations. The tool is intended for use by organisations managing AWS infrastructure, including commercial enterprises.

Key considerations:
- Encouraging wide adoption in enterprise environments
- Protecting the project name and contributors
- Balancing permissiveness with reasonable protections
- Compatibility with the Go ecosystem and AWS tools
- Simplicity and legal clarity for users

The main candidates considered were:
- **MIT License**: Very permissive, minimal restrictions, widely adopted
- **Apache License 2.0**: Permissive with patent protection, enterprise-friendly
- **GPL v3**: Strong copyleft, ensures derivatives remain open, may limit commercial adoption
- **BSD 3-Clause**: Permissive with name protection clause, business-friendly

## Decision

We will use the **BSD 3-Clause License** for Stackaroo.

Key factors in this decision:
- Highly permissive to encourage adoption in commercial environments
- Additional clause prevents misuse of project name for endorsement
- Simple and well-understood license terms
- Compatible with integration into proprietary workflows
- Widely accepted in the infrastructure and DevOps community
- Provides reasonable protection for contributors whilst remaining business-friendly

## Consequences

**Positive:**
- Encourages adoption by enterprises and commercial users
- Clear legal terms that are well-understood by legal departments
- Allows integration into proprietary systems and workflows
- Protects project name from unauthorised endorsement use
- Compatible with most other open source licenses
- Low barrier to contribution and usage

**Negative:**
- No explicit patent protection (unlike Apache 2.0)
- Derivatives can become proprietary (no copyleft protection)
- Minimal restrictions on how the code is used
- Commercial entities could create competing proprietary versions

We accept these trade-offs in favour of maximising adoption and practical utility for the infrastructure management community.
