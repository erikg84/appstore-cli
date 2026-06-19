# Browser CLI Workplan Index

## Purpose

This directory contains the implementation plan for `browser-cli`.

The implementation is intended to be executed by the agent pipeline (`agp`). The plan is written to minimize architectural drift and prevent the pipeline from inventing a different product.

## Required Reading Order

1. `../BROWSER_CLI_PROPOSAL.md`
2. `01_ARCHITECTURE_AND_GUARDRAILS.md`
3. `02_PHASE1_FOUNDATION.md`
4. `03_PHASE2_BROWSER_RUNTIME.md`
5. `04_PHASE3_RESEARCH_WORKFLOWS.md`
6. `05_PHASE4_HARDENING_AND_VALIDATION.md`
7. `06_AGP_EXECUTION_GUIDE.md`

## Global Rules

- Do not write implementation that bypasses the persistent browser runtime
- Do not replace the browser-backed model with raw HTTP-only shortcuts as the primary design
- Do not add visible browser UI for normal agent execution
- Do not introduce mock fallback behavior for production commands
- Keep command output stable and machine-readable
- Prefer small verifiable increments per phase
- Keep state under `~/.browser-cli/`
- Preserve a clean separation between browser profile state and extracted document cache

## Delivery Rule

Every phase must end with:

- implementation notes
- explicit verification commands
- updated docs where relevant
- no unresolved TODO comments for critical path behavior
