# Role Root

[Leia em Portugues](README.pt-BR.md)

Role Root is the workspace repository for the platform. It ties together deployment assets, smoke tooling, shared integration wiring, and the repositories that implement the security services.

This repository is meant to be the coordination point of the platform, not the only place where the code lives.

## Repositories

- [`role-root`](https://github.com/LCGant/role-root): workspace, deploy, docs, smoke tooling, and orchestration glue
- [`role-gateway`](https://github.com/LCGant/role-gateway): gateway foundation and public edge service
- [`role-auth`](https://github.com/LCGant/role-auth): authentication foundation
- [`role-pdp`](https://github.com/LCGant/role-pdp): authorization decision foundation
- [`role-pep`](https://github.com/LCGant/role-pep): policy enforcement library
- [`role-notification`](https://github.com/LCGant/role-notification): internal notification delivery foundation
- [`role-audit`](https://github.com/LCGant/role-audit): internal audit collection foundation

## What this repository contains

- `deploy`: Dockerfiles, Compose stacks, bootstrap scripts
- `tools/smoke`: smoke checks for the integrated stack
- `docs`: security invariants, production checklist, and operational notes
- workspace-level references to the service repositories listed above

## Project state

This platform is already a strong starting point for teams that want a serious auth/authz base in Go. It is not presented as fully finished software. Some integrations and operational pieces are still intentionally basic, which makes the codebase a good foundation rather than a turnkey platform.

## Suggested reading order

1. Read this repository first for the workspace view.
2. Read the service repositories in the order `role-gateway`, `role-auth`, `role-pdp`, `role-pep`.
3. Review `docs/SECURITY_INVARIANTS.md` and `docs/PRODUCTION_CHECKLIST.md` before changing security-sensitive flows.
