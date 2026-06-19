# Phase 3: Research Workflows

## Objective

Implement agent-useful retrieval workflows on top of the browser runtime.

## Scope

### Search Workflow

Implement `search` using browser-backed search behavior.

Requirements:

- query execution
- result parsing
- result ranking preservation
- result cache persistence
- JSON/markdown/text output

### Read Workflow

Implement `read`.

Requirements:

- accept URL or query
- query mode should search then read top N results
- extract meaningful page content
- convert to clean markdown/text
- persist extracted documents separately from raw HTML

### Extraction Layer

Implement:

- readability-style main-content extraction
- boilerplate reduction
- title normalization
- markdown conversion
- fallback handling for weak extraction

### History Workflow

Implement:

- query history
- opened page history
- recent read activity

### Cache Workflow

Implement:

- cache inspection
- cache pruning
- cache clearing

## Crawl Workflow

Implement a conservative crawler.

Requirements:

- same-session crawl
- host scoping
- depth limit
- page limit
- dedupe
- metadata persistence

## Constraints

- no aggressive parallel crawling in first version
- no unrestricted off-domain expansion
- no silent extraction failure

## Verification Targets

The pipeline should prove:

```bash
browser-cli search "current Android app architecture" --json
browser-cli read https://developer.android.com/topic/architecture --format markdown
browser-cli read "kotlin 2.4.0 released" --top 3 --json
browser-cli history --queries
browser-cli crawl https://developer.android.com/topic/architecture --limit 5 --depth 1 --json
```

## Deliverables

- functional `search`
- functional `read`
- functional `history`
- functional conservative `crawl`
- extracted document cache

## `agp` Notes

The pipeline must preserve stable output contracts. Any extraction heuristics should be encapsulated, not spread through command handlers.
