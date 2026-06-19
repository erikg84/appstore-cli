# browser-cli

Headless persistent browser runtime for agent-driven web research.

## Status

Implemented working runtime with:

- persistent headless Playwright browser state
- SQLite-backed history/index storage
- raw and rendered artifact caches
- search/open/read/crawl/history/login workflows
- Brave-backed search with embedded key support
- agent-oriented help, diagnostics, warnings, and structured errors

## Commands

- `browser-cli doctor`
- `browser-cli search <query>`
- `browser-cli open <url>`
- `browser-cli read <url-or-query>`
- `browser-cli crawl <url>`
- `browser-cli history`
- `browser-cli login <domain-or-url>`
- `browser-cli cache inspect|clear|prune`

## Agent Workflow

Recommended sequence for agents:

1. `browser-cli doctor`
2. `browser-cli search "..." --format json`
3. `browser-cli read "..." --format json`
4. `browser-cli open <known-url> --format json`
5. `browser-cli crawl <seed-url> --format json`

### Why `doctor` matters

`doctor` is the first command agents should use when they need to know whether the CLI is operational. It verifies:

- storage root and cache directories
- persistent browser profile path
- SQLite index readability
- Playwright Chromium executable presence
- Brave Search key presence
- cache footprint

## Output Contract

All agent-facing commands support:

- `--format json`
- `--format markdown`
- `--format text`

Structured output includes:

- `ok`
- `command`
- `timestamp`
- `data`
- `meta`
- `warnings`
- `errors`

### Warnings

Warnings are used for situations where the command succeeded but the result may need caution, for example:

- cached search results were reused
- top search confidence is low
- fetched page returned non-2xx status
- extracted document quality appears weak

### Errors

Errors are structured and agent-actionable. Each error includes:

- `code`
- `message`
- `suggestion`

Examples of handled failure classes:

- missing Brave auth
- DuckDuckGo challenge/blocking
- Playwright/browser runtime issues
- network failures
- timeout failures
- site verification/Cloudflare blocks

## Search Behavior

Search engines supported:

- `auto`
- `brave`
- `bing`
- `duckduckgo`

### Default engine

`auto` prefers Brave Search using the embedded repository key and falls back to Bing when needed.

### Ranking behavior

Search results are not returned raw. The CLI applies:

- heuristic reranking
- quoted-query fallback
- simplified-query fallback
- site-constrained fallback generation for docs-style queries
- document reranking after extraction in query-based `read`

### Practical guidance

For agent use:

- use `search` for discovery
- use `read` for query-to-document workflows
- use `open` when a known URL is already trusted
- inspect `meta.topScore` and `warnings`
- if confidence is low, narrow the query or constrain the site

## Storage Layout

All runtime state lives under `~/.browser-cli/`:

- `profile/` persistent browser profile
- `cache/raw/` raw HTML snapshots
- `cache/rendered/` extracted markdown/text
- `cache/media/` reserved for future media artifacts
- `index.db` SQLite metadata index
- `config.json` local override config
- `logs/` reserved for future runtime logs

## Install

```bash
npm install
npx playwright install chromium
npm run build
```

## Examples

```bash
browser-cli doctor
browser-cli search "OpenAI API docs" --engine brave --format json
browser-cli search "site:developer.android.com architecture" --format json
browser-cli open https://developer.android.com/topic/architecture --format json
browser-cli read https://developer.android.com/topic/architecture --format markdown
browser-cli read "OpenAI API docs" --top 2 --format json
browser-cli crawl https://example.com --limit 5 --depth 1 --format json
browser-cli history --limit 20 --format json
browser-cli cache inspect --format json
```

## Help UX

The CLI is designed so agents can learn it from `--help` alone.

- global help explains the overall workflow
- each command has examples
- command summaries are concise
- agent notes are embedded in help text where useful

Recommended checks:

```bash
browser-cli --help
browser-cli doctor --help
browser-cli search --help
browser-cli read --help
browser-cli cache inspect --help
```

## Validation

```bash
npm run typecheck
npm run build
npm test
browser-cli doctor
```

## Repository Guidance

- `AGENTS.md`
- `docs/BROWSER_CLI_PROPOSAL.md`
- `docs/workplan/00_INDEX.md`

## Notes

- Normal execution is headless.
- `login` is the only command intended to use a visible browser window.
- `duckduckgo` can still be challenged by anti-bot behavior in this environment.
- Search quality is improved, but still bounded by the underlying engine and target-site protections.
