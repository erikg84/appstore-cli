# Phase 4: Hardening And Validation

## Objective

Make the runtime robust enough for repeated agent use on developer machines.

## Scope

### Reliability

Implement:

- timeout configuration
- retry policy where appropriate
- cache freshness controls
- forced refresh paths
- clearer error envelopes

### Observability

Implement:

- structured logs
- debug mode
- navigation timing diagnostics
- cache hit/miss visibility

### Login Workflow

Implement:

- explicit `login` command
- temporary headed mode only for login/debug
- persisted auth state returned to headless runtime

### Validation Matrix

The pipeline should produce verification notes for:

- static site open/read
- JS-heavy site open/read
- repeat read from cache
- history persistence across runs
- crawl scope enforcement
- login-state reuse where possible

## Documentation Requirements

Before this phase is complete, update:

- root `README.md`
- operational usage docs
- cache/storage docs
- troubleshooting guidance

## Constraints

- do not ship visible browser mode as default
- do not leave debug-only behavior enabled by default
- do not store sensitive values in logs

## Acceptance Criteria

The runtime is acceptable for general agent use when:

- headless browser lifecycle is stable
- cache semantics are documented and inspectable
- commands return predictable structured output
- history and state survive process restarts
- developers are not forced to watch browser windows during normal operation

## `agp` Notes

Do not start adding unrelated product ideas in this phase. Finish operational quality and documentation first.
