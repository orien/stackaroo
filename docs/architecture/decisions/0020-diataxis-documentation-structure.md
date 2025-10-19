# 20. Adopt the Diátaxis documentation framework

Date: 2025/10/20

## Status

Accepted

## Context

Stackaroo’s user-facing documentation is being migrated to a dedicated VitePress site (ADR 0019). We need a cohesive information architecture that scales as the CLI adds new commands, configuration patterns, and operational guidance. Without a structured framework, existing and future pages risk mixing conceptual background with procedural steps, making it harder for practitioners to find the right level of detail quickly.

We evaluated several documentation frameworks:

- **Diátaxis**: Four complementary modes (Tutorials, How-to guides, Explanation, Reference) tailored to user intent, emphasising reuse and clarity for both novice and advanced readers.
- **DITA**: Provides rigorous topic typing and conditional publishing, but demands XML authoring and introduces a steeper tooling curve than our Markdown-first workflow.
- **Every Page is Page One**: Encourages self-contained topics and heavy hyperlinking, yet offers less explicit guidance for structuring practical tasks versus conceptual material.
- **Minimalism (Carroll)**: Strong task orientation, though it under-specifies how to handle conceptual background and canonical references alongside procedures.

Diátaxis emerged as the best fit for a lean team authoring Markdown within VitePress, while still encouraging intentional separation between learning and doing content.

## Decision

Structure Stackaroo’s user documentation according to the **Diátaxis** framework:

- Tutorials guide first-time users through complete, goal-driven scenarios.
- How-to guides solve narrowly scoped tasks for readers who already know the basics.
- Explanations provide conceptual and architectural insight.
- Reference pages deliver authoritative, exhaustive detail (e.g. CLI flags, configuration schemas).

Each section will live under dedicated directories in `docs/user/` and cross-link where appropriate. New documentation must identify the intended Diátaxis mode before drafting.

## Consequences

### Positive Consequences

- **Improved discoverability**: Readers can jump directly to the mode that matches their intent, reducing time spent scanning unrelated material.
- **Consistent authoring workflow**: Contributors know up-front which template and voice to adopt, simplifying reviews and keeping tone consistent.
- **Scalable structure**: As Stackaroo gains features, content can expand within the relevant Diátaxis mode without reshuffling the entire navigation.
- **Supports automation**: The clear directory breakdown enables future automation (linting or CI checks) to verify coverage across modes.

### Negative Consequences

- **Upfront rework**: Existing pages require auditing, rewriting, or relocating to align with the four modes.
- **Editorial overhead**: Authors must consciously choose the correct mode, potentially increasing planning time for small changes.
- **Navigation complexity**: Users unfamiliar with Diátaxis may initially need orientation, requiring onboarding notes or site cues.
