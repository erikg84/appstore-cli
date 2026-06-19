# Phase 2: Browser Runtime

## Objective

Implement the persistent headless browser runtime and the core page lifecycle.

## Scope

### Persistent Browser Context

Implement:

- Playwright bootstrap
- persistent Chromium user-data dir
- headless default mode
- explicit headed override for login/debug
- engine lifecycle management

### Page Navigation Service

Implement:

- page creation
- navigation with timeout policy
- configurable wait strategy
- final URL capture
- title capture
- HTML snapshot capture
- timing capture

### Browser Session Integration

Persist and reuse:

- cookies
- local storage
- session storage where Playwright profile retains it
- consent state

### Open Workflow

Implement `open` end-to-end:

- live navigation
- metadata capture
- raw snapshot persistence
- JSON/text output

## Constraints

- Do not add crawler logic yet
- Do not add summarization logic
- Do not bypass Playwright with raw `fetch` as the primary implementation

## Verification Targets

The pipeline should prove:

```bash
browser-cli open https://example.com --json
browser-cli open https://developer.android.com/topic/architecture --json
browser-cli cache inspect
```

Expected verifications:

- profile directory created
- raw page snapshot persisted
- page metadata persisted to SQLite
- subsequent runs reuse same profile

## Deliverables

- persistent browser runtime
- functional `open` command
- durable browser profile usage
- cache persistence for opened pages

## `agp` Notes

Keep browser-specific logic out of CLI argument parsing code. The runtime must be reusable by later search/read/crawl workflows.
