# AGP Execution Guide

## Purpose

This file tells the agent pipeline how to execute the `browser-cli` plan without drifting from the intended architecture.

## Required Inputs

The pipeline must read these files before implementation starts:

1. `../BROWSER_CLI_PROPOSAL.md`
2. `00_INDEX.md`
3. `01_ARCHITECTURE_AND_GUARDRAILS.md`
4. `02_PHASE1_FOUNDATION.md`
5. `03_PHASE2_BROWSER_RUNTIME.md`
6. `04_PHASE3_RESEARCH_WORKFLOWS.md`
7. `05_PHASE4_HARDENING_AND_VALIDATION.md`

## Hard Rules

- Do not rewrite the product into a provider-backed search wrapper
- Do not make raw HTTP fetching the primary runtime model
- Do not make headed browser mode the default
- Do not implement silent mock fallback
- Do not skip Phase 1 and jump directly into search/crawl logic
- Do not mix persistence, extraction, and CLI parsing into the same modules

## Execution Strategy

### Phase Discipline

Implement phases in order.

Do not begin the next phase until the current phase has:

- working code
- passing build/test validation for that phase
- updated docs if the implementation surface changed

### Verification Discipline

For every phase, the pipeline must produce:

- commands run
- observed output summary
- unresolved issues summary

### Output Discipline

All agent-facing commands must support stable machine-readable output.

The pipeline should validate:

- JSON shape consistency
- deterministic error envelope shape
- clear success vs failure signaling

## Recommended Build Order

1. scaffold repo
2. bootstrap config and storage
3. bootstrap persistent Playwright runtime
4. implement `open`
5. implement `search`
6. implement `read`
7. implement `history`
8. implement `crawl`
9. implement `login`
10. harden logs/cache/docs

## Phase Exit Criteria

### Phase 1

- CLI builds
- commands parse
- storage root bootstraps
- SQLite bootstrap works

### Phase 2

- browser context persists across runs
- `open` works on at least one static and one JS-capable page
- page snapshots persist

### Phase 3

- `search`, `read`, `history`, and conservative `crawl` work
- extracted content persists independently from raw HTML

### Phase 4

- cache operations are safe and documented
- logs are useful and do not leak secrets
- login flow is explicit and temporary-headed only

## Escalation Rules

The pipeline should stop and report if it encounters:

- a fundamental runtime choice conflict with the plan
- a storage model that no longer matches the proposal
- a browser-engine limitation that requires a product decision
- a command contract conflict that breaks downstream agent usage

## Final Delivery Requirements

A complete delivery must include:

- implementation
- updated README
- usage docs
- cache/storage docs
- validation notes
- no hidden mock/developer-only production paths
