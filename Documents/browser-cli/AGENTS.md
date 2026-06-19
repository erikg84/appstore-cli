# Repository Guidelines — browser-cli

## What this is

A headless persistent browser runtime for agent-driven web research.

## Core architecture

- `src/index.ts` — CLI surface and help UX
- `src/app/` — orchestration logic for search/open/read/crawl/history/login/doctor
- `src/browser/` — Playwright runtime and browser lifecycle
- `src/persistence/` — storage paths, config, SQLite metadata
- `src/extraction/` — HTML → readable text/markdown extraction
- `src/formatting/` — output envelope formatting for json/markdown/text
- `src/types/` — shared contracts

## Non-negotiable rules

- Default execution stays headless
- Normal agent execution must not open visible browser windows
- `login` is the only command allowed to use headed mode
- Keep browser runtime code separate from CLI parsing
- Keep persistence separate from extraction and ranking
- Agent-facing commands must return structured output with useful metadata, warnings, and actionable errors
- Do not add silent mock fallbacks

## Search policy

- Prefer `auto`, which uses Brave when configured and otherwise falls back
- Treat search as discovery, not final truth
- Preserve useful metadata so agents can decide whether to retry, narrow the query, or switch engines

## Build and validation

```bash
npm run typecheck
npm run build
npm test
node dist/index.js --help
node dist/index.js doctor
```

## Documentation rule

If you add or materially change a command, update:

- `README.md`
- command help text in `src/index.ts`
- structured error handling if the command introduces new failure modes
