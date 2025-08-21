# 6. License headers in source files

Date: 2025-08-22

## Status

Accepted

## Context

We need to decide on the format and extent of license headers in source code files. Currently, our Go files contain the full BSD 3-Clause license text (29 lines each), which creates significant visual clutter and takes up substantial screen space.

Options considered:
- **Full license text**: Complete license in every file (current approach)
- **No headers**: Rely solely on LICENSE file at repository root
- **Minimal copyright**: Just copyright line
- **SPDX identifier**: Copyright + standardised machine-readable license reference
- **Brief reference**: Copyright + short reference to LICENSE file

Key considerations:
- Legal clarity and attribution
- Code readability and maintainability
- Industry standards and best practices
- Tool compatibility and automation
- Developer experience

## Decision

We will use **minimal headers with SPDX license identifiers** in all source code files.

Format:
```
/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
```

Key factors in this decision:
- SPDX identifiers are industry standard and machine-readable
- Minimal visual impact (4 lines vs 29 lines)
- Clear legal attribution maintained
- Compatible with automated license scanning tools
- Widely adopted by major open source projects
- Balances legal clarity with code readability

## Consequences

**Positive:**
- Significantly reduced visual clutter in source files
- Standardised, machine-readable license identification
- Maintains clear copyright attribution
- Compatible with automated compliance tools
- Faster file loading and easier code review
- Industry standard approach

**Negative:**
- Full license text not immediately visible in each file
- Requires understanding of SPDX identifier system
- Dependency on LICENSE file remaining in repository root

We will ensure the LICENSE file remains prominently available and clearly referenced in project documentation to mitigate the trade-offs.
