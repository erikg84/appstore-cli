# Phase 1: Foundation

## Objective

Create the repository and runtime foundation required for browser-backed features.

## Scope

### Project Bootstrap

Create:

- `package.json`
- `tsconfig.json`
- ESLint / formatting config if appropriate
- test runner configuration
- Playwright dependency setup
- SQLite dependency setup

### Source Layout

Create these modules:

- `src/cli`
- `src/app`
- `src/browser`
- `src/extraction`
- `src/persistence`
- `src/formatting`
- `src/types`

### Storage Abstractions

Implement:

- storage root resolver
- directory bootstrapping
- SQLite connection bootstrap
- path helpers for cache/profile/log locations

### Command Stubs

Wire top-level commands with placeholder application-service integration:

- `search`
- `open`
- `read`
- `crawl`
- `history`
- `login`
- `cache`

These can initially fail with explicit `not implemented` errors, but they must be real commands with stable argument parsing.

## Required Technical Decisions

- ESM vs CJS: choose one and keep it consistent
- formatter/output abstraction must be centralized
- error envelope shape must be standardized now
- runtime config loading must be centralized now

## Verification Targets

The pipeline should deliver commands such as:

```bash
npm run build
npm test
node dist/index.js --help
node dist/index.js search --help
node dist/index.js cache inspect
```

## Deliverables

- compilable CLI scaffold
- storage bootstrap working locally
- basic SQLite schema migration path
- documented local development instructions

## `agp` Notes

Do not start implementing page extraction or crawling logic in this phase. Keep Phase 1 narrow and verifiable.
