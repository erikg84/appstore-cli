# Phase 0: Architecture And Guardrails

## Objective

Define the implementation boundaries so the pipeline builds the correct product.

## Must-Have Architectural Decisions

### Runtime Choice

Use:

- Node.js
- TypeScript
- Playwright
- SQLite

Do not use Go as the primary runtime.

Reason:

The product center is browser execution and persistent browser state. Node + Playwright is the most direct implementation path.

### Browser Execution Model

Default behavior must be:

- headless
- persistent profile
- no visible browser UI

Only explicit login/debug workflows may temporarily use headed mode.

### Storage Model

Use:

- SQLite metadata index
- filesystem cache
- persistent browser user-data directory

Storage root:

- `~/.browser-cli/`

### CLI Philosophy

The CLI must expose small composable commands rather than one huge command.

Required first-class commands:

- `search`
- `open`
- `read`
- `crawl`
- `history`
- `login`
- `cache`

## Repository Skeleton To Create

Expected high-level structure:

```text
browser-cli/
  package.json
  tsconfig.json
  src/
    cli/
    app/
    browser/
    extraction/
    persistence/
    formatting/
    types/
  scripts/
  tests/
  docs/
```

## Guardrails

### No Mock Production Mode

The deleted `search-cli` silently returned fake results. That must never happen here.

If a command cannot perform its job, it must fail explicitly.

### No Provider-Centric Design

Do not build around Tavily, SerpAPI, or any paid provider as the primary dependency.

Optional adapters can exist later. The core product must work with no subscription.

### No Manual Browser Tabs During Normal Use

The developer should not see browser windows during normal agent operation.

### No Hidden Network Magic

Commands must document whether they are using:

- cached output
- live browsing
- forced refresh

## Acceptance Criteria

Before Phase 1 is considered complete, the repository must contain:

- runtime scaffolding
- package management configuration
- lint/test/build scripts
- clear storage root abstraction
- documented command contract stubs

## `agp` Instructions

The pipeline must:

- follow the directory structure above
- prefer strongly typed boundaries
- keep browser-specific code isolated from CLI parsing
- keep persistence logic isolated from extraction logic
- avoid premature feature branching
