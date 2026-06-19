# Browser CLI Proposal

## Objective

Build `browser-cli` as a headless, persistent browser runtime for agents. It must act like a real browser without opening visible tabs during normal agent execution. It must preserve session state, browsing state, and retrieval history across runs, and expose a CLI surface suitable for agentic workflows.

This is **not** a search wrapper. It is a browser-backed research runtime.

## Why The Previous `search-cli` Was Insufficient

The deleted Go `search-cli` had the wrong center of gravity:

- it treated search as the product instead of browsing
- it required provider-specific logic for useful results
- it silently fell back to mock mode when not configured
- it had no persistent browser identity
- it had no browser session semantics
- it had no durable query/page cache
- it had no document extraction lifecycle
- it could not navigate authenticated or JS-heavy sites like a browser

The replacement must be designed around a persistent headless browser context.

## Product Definition

`browser-cli` is a command-line runtime that lets agents:

- search the web using real browser behavior
- open pages in a headless browser
- execute JavaScript and wait for rendered content
- preserve cookies, storage, and session state
- cache search results, raw pages, rendered DOM, and extracted text
- maintain browsing history and query history
- read cleaned page content from cache or network
- crawl sites with domain and depth constraints
- optionally enter a temporary headed mode for login/debug only

## Non-Goals

The first implementation should **not** attempt to do all of the following at once:

- full semantic summarization inside the CLI
- multi-user shared sync
- distributed crawling
- browser extension support
- arbitrary plugin execution
- cloud-hosted orchestration
- general-purpose scraping bypass systems

Those can be added later only if justified.

## Core Principles

1. Headless by default
2. Persistent state across runs
3. Deterministic CLI output
4. Explicit caching behavior
5. Agent-friendly structured output
6. Safe defaults for crawling and storage
7. Clear separation between browser state and extracted research data

## Recommended Technical Stack

### Runtime

Use **Node.js + TypeScript**.

Reasoning:

- Playwright is first-class there
- browser automation APIs are more mature than Go alternatives
- easier persistent context handling
- better ecosystem for DOM extraction and readability tooling
- lower friction for headless browser lifecycle management

### Browser Engine

Use **Playwright** with a persistent Chromium context.

Default mode:

- Chromium
- headless
- persistent user data dir

Optional modes:

- headed login mode
- alternate engine selection for compatibility diagnostics

### Storage

Use two storage layers:

1. **SQLite** for metadata and indexes
2. **Filesystem blob cache** for raw content and derived artifacts

Recommended root:

- `~/.browser-cli/`

Structure:

- `~/.browser-cli/profile/` browser user data
- `~/.browser-cli/cache/raw/` raw HTML and network artifacts
- `~/.browser-cli/cache/rendered/` DOM snapshots and extracted markdown/text
- `~/.browser-cli/cache/media/` screenshots or media previews if needed later
- `~/.browser-cli/index.db` metadata index
- `~/.browser-cli/config.json` runtime config
- `~/.browser-cli/logs/` operational logs

## Proposed CLI Surface

### Search

```bash
browser-cli search "current Android app architecture"
browser-cli search "kotlin 2.4.0 released" --engine duckduckgo --json
```

Responsibilities:

- run a browser-backed search workflow
- store query metadata
- cache SERP results
- return ranked result objects

### Open

```bash
browser-cli open https://developer.android.com/topic/architecture
```

Responsibilities:

- navigate headlessly
- wait for configured load conditions
- persist final URL, title, raw HTML snapshot, and timing data

### Read

```bash
browser-cli read https://developer.android.com/topic/architecture --format markdown
browser-cli read "current Android app architecture" --top 5
```

Responsibilities:

- fetch from cache or browser
- extract cleaned text/markdown
- return usable agent context

### Crawl

```bash
browser-cli crawl https://developer.android.com/topic/architecture --limit 20 --depth 2
```

Responsibilities:

- reuse same persistent session
- constrain host/depth
- dedupe visited pages
- cache outputs

### History

```bash
browser-cli history
browser-cli history --queries
browser-cli history --pages
```

Responsibilities:

- list prior search queries
- list visited pages
- support agent context continuity

### Login

```bash
browser-cli login example.com
```

Responsibilities:

- temporary headed workflow only when needed
- persist resulting auth state into the normal profile

### Cache Operations

```bash
browser-cli cache inspect
browser-cli cache prune
browser-cli cache clear
```

Responsibilities:

- inspect footprint
- remove stale entries
- reset runtime state safely

## Output Contracts

All commands intended for agents must support stable structured output.

Required formats:

- `json`
- `markdown`
- `text`

Recommended result envelope:

```json
{
  "ok": true,
  "command": "search",
  "timestamp": "2026-06-05T12:00:00Z",
  "data": {},
  "meta": {},
  "errors": []
}
```

## Internal Architecture

### 1. CLI Layer

Responsibilities:

- parse commands and flags
- validate input
- choose formatter
- call application services

### 2. Application Services

Responsibilities:

- search workflow orchestration
- open/read/crawl orchestration
- cache policy enforcement
- history tracking

### 3. Browser Runtime Layer

Responsibilities:

- create/reuse persistent Playwright context
- page creation and cleanup
- navigation policies
- engine-specific behavior

### 4. Extraction Layer

Responsibilities:

- title extraction
- main-content extraction
- markdown conversion
- boilerplate removal
- fallback heuristics for difficult pages

### 5. Persistence Layer

Responsibilities:

- SQLite metadata
- filesystem cache
- query/page history
- retention policies

## Database Schema Direction

Minimum tables:

- `queries`
- `search_results`
- `pages`
- `page_snapshots`
- `documents`
- `crawl_runs`
- `crawl_pages`
- `config_kv`

### `queries`

- `id`
- `query_text`
- `engine`
- `created_at`
- `result_count`
- `cache_key`

### `pages`

- `id`
- `url`
- `canonical_url`
- `domain`
- `title`
- `status_code`
- `content_type`
- `last_visited_at`
- `last_fetch_duration_ms`
- `hash`

### `documents`

- `id`
- `page_id`
- `format`
- `content_path`
- `created_at`
- `extractor_version`

## Search Strategy

The first implementation should support two search modes:

1. browser-driven web search
2. lightweight engine-specific search adapter if needed later

The browser-driven path is primary because the product goal is browser behavior, not provider plumbing.

## Authentication Strategy

Default execution must remain headless.

Only `login` should permit a temporary visible browser workflow. That mode should:

- be explicit
- persist resulting auth state
- return to headless operation afterward

## Cache Strategy

### What Must Be Cached

- query results
- page HTML snapshots
- rendered DOM snapshots where useful
- cleaned extracted markdown/text
- navigation timing metadata
- history metadata
- auth/session state via browser profile

### What Must Not Be Cached Indefinitely

- unrestricted raw media growth
- unlimited duplicate snapshots
- expired temporary artifacts

The workplan should define retention and pruning rules.

## Security And Privacy Considerations

- credentials and cookies live in the browser profile and must be treated as sensitive local state
- cache clearing must support both selective and full reset
- logs must avoid dumping sensitive cookie/header values
- history should be inspectable and deletable

## Agentic Workflow Fit

`browser-cli` is useful for agents because it provides:

- durable browsing identity
- repeatable CLI primitives
- stable cache semantics
- structured output for downstream synthesis
- separation between retrieval and reasoning

This is a better foundation than wiring agents directly to ad hoc raw `curl` workflows.

## Risks

1. Some sites will still resist headless automation
2. JS-heavy sites require disciplined wait strategies
3. Extraction quality will vary by site shape
4. Cache growth can become unbounded without retention policies
5. Search behavior may vary across engines and regions

## Implementation Recommendation

Build in phases:

1. foundation and storage
2. persistent browser runtime
3. search/open/read workflows
4. crawl and history
5. hardening and validation

## Delivery Constraint

This repository is planning-first. The implementation must be produced by the `agp` pipeline from the workplan. Humans or ad hoc manual coding should not bypass the plan unless explicitly authorized later.
