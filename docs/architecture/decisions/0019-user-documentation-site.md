# 19. User documentation site with VitePress and GitHub Pages

Date: 2025/10/19

## Status

Accepted

## Context

Stackaroo currently ships internal-facing documentation (architecture notes, release records) under `docs/internal`. We lack a polished, user-facing documentation site that explains installation, commands, configuration patterns, and troubleshooting for practitioners adopting the CLI. The existing Markdown content and README snippets are scattered across the repository, making it difficult for new users to discover workflows or stay aligned with latest capabilities.

We evaluated several documentation toolchains:

- **MkDocs Material**: Simple Markdown workflow and wide adoption, but default styling was considered dated, and significant custom CSS would be required to achieve the brand quality we want.
- **Docusaurus**: Rich React ecosystem with versioning and internationalisation, but heavier dependency footprint and more boilerplate than our documentation needs justify.
- **Hugo (Docsy/Book)**: High performance and flexible, though Go templating adds complexity for contributors unfamiliar with the stack and theming requires more curation time.
- **VitePress**: Modern Vue-powered static site generator with fast feedback (Vite dev server), minimal configuration, Markdown-first authoring, and straightforward theming.

We also considered hosting options—Netlify, Vercel, Cloudflare Pages, and GitHub Pages. Given Stackaroo already lives on GitHub and needs a predictable, low-maintenance pipeline, GitHub Pages provides an acceptable default without adding another SaaS dependency.

Key requirements influencing the decision:

- Opinionated, modern styling out of the box that the maintainers consider visually appealing.
- Markdown-first authoring with support for Mermaid diagrams (already used in internal docs) and callouts.
- Ability to co-locate the source under `docs/` while keeping internal documents under `docs/internal`.
- Low operational overhead for builds and hosting, ideally leveraging GitHub Actions.
- Room for future enhancements such as interactive Vue components or dark mode support without committing to a full SPA rewrite.

## Decision

Produce Stackaroo's user-facing documentation site with **VitePress** sourced from the repository's `docs/` directory, and publish the generated static site to **GitHub Pages** via a CI workflow.

## Consequences

### Positive Consequences

- **Modern presentation**: VitePress's default theme delivers the contemporary look and feel the team expects, with easy palette and typography adjustments.
- **Efficient authoring**: Markdown plus Vue-powered extensions allows contributors to iterate quickly while retaining the option to add interactive components when necessary.
- **Fast local feedback**: The Vite dev server offers hot module replacement, reducing friction when refining content or styling.
- **Straightforward deployment**: GitHub Pages integrates cleanly with our existing repository, and VitePress includes documented workflows for publishing via Actions.
- **Clear separation**: Keeping internal documents under `docs/internal` avoids mixing contributor-only content with the public site.

### Negative Consequences

- **Additional toolchain**: Contributors must work with Node.js tooling (npm/pnpm) alongside Go build tools, increasing setup requirements.
- **Vue dependency**: Team members unfamiliar with Vue may face a learning curve when customising the theme or adding bespoke components.
- **Build step maintenance**: We must maintain the Pages publishing workflow, including caching dependencies and monitoring for upstream VitePress changes.
- **Static hosting limits**: GitHub Pages does not provide dynamic server-side features; any interactive demos require client-side implementation.

Overall, VitePress on GitHub Pages balances the desire for a modern, attractive site with a lightweight authoring experience aligned to Stackaroo's contributor base.
