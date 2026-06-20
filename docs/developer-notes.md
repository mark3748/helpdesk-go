---
sidebar_position: 6
---

# Documentation Plan

The public Docusaurus site should be the source of truth for user-facing documentation. The app repository should keep developer-facing references that are useful close to code, such as OpenAPI files, chart examples, and implementation notes.

## Public Site Owns

- Quickstart and installation guides
- Helm installation and chart publishing instructions
- Discord setup and command reference
- Storage, mail, OIDC, and operational guides
- Screenshots and UI-oriented walkthroughs

## App Repository Owns

- `docs/openapi.yaml`
- Helm chart source and example values
- API implementation notes that change with code
- Developer setup, test, and contribution instructions

## Sync Process

Before a release:

1. Update app docs and OpenAPI when behavior changes.
2. Copy user-facing changes into the Docusaurus site.
3. Build the site with `npm run build`.
4. Link release notes to the matching site pages.

This keeps code-adjacent facts near code without making users browse a repository to install or configure the product. A small mercy, but civilization is built from such things.
