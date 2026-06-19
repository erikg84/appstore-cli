#!/usr/bin/env node
import { Command } from 'commander';
import { BrowserCliService } from './app/service.js';
import { toCliError } from './cli/errors.js';
import { envelope, formatOutput } from './formatting/output.js';
import type { OutputFormat } from './types/index.js';

const program = new Command();
const service = new BrowserCliService();

program
  .name('browser-cli')
  .description('Headless persistent browser runtime for agent-driven web research')
  .showHelpAfterError()
  .option('--format <format>', 'Output format: json|markdown|text', 'json')
  .addHelpText('after', `
Examples:
  browser-cli doctor
  browser-cli search "OpenAI API docs"
  browser-cli read "Android app architecture" --engine brave --top 3
  browser-cli open https://developer.android.com/topic/architecture
  browser-cli crawl https://example.com --limit 5 --depth 1
  browser-cli cache inspect
  browser-cli login appstoreconnect.apple.com
  browser-cli eval "document.title" --url https://example.com
  browser-cli dom-query "input[type=file]" --include-hidden
  browser-cli click --text "Save"
  browser-cli fill "input[name=email]" you@example.com
  browser-cli scroll --bottom

Guidance:
  - Use 'doctor' first to confirm browser, storage, and engine readiness.
  - Use 'search' to discover candidate URLs.
  - Use 'open' to fetch and snapshot a specific page.
  - Use 'read' to extract clean text/markdown from a URL or query.
  - Use 'crawl' for conservative same-host discovery.
  - Use '--format json' for agents and automation.
`);

program
  .command('doctor')
  .summary('Inspect runtime readiness for agent use')
  .description('Check config, Playwright installation, storage directories, SQLite index, and engine readiness.')
  .addHelpText('after', `
Examples:
  browser-cli doctor
  browser-cli doctor --format markdown

What it checks:
  - storage root and cache directories
  - persistent browser profile path
  - SQLite index readability
  - Playwright Chromium executable
  - Brave Search key presence
  - cache footprint
`)
  .action(async () => {
    await runWithOutput('doctor', async (format) => {
      const result = service.doctor();
      const warnings = result.summary.failedChecks > 0
        ? ['One or more runtime checks failed. Review the failed checks and suggestions before relying on agent execution.']
        : ['Runtime checks passed. The CLI appears ready for agent workflows.'];
      return formatOutput(
        format,
        envelope('doctor', result, {
          ok: result.summary.ok,
          healthyChecks: result.summary.healthyChecks,
          failedChecks: result.summary.failedChecks,
          braveConfigured: result.config.braveConfigured
        }, warnings)
      );
    });
  });

program
  .command('search')
  .summary('Search the web and return ranked URLs with snippets')
  .description('Search the web using the selected engine, rerank the results, and return structured candidates.')
  .argument('<query>', 'Search query')
  .option('--engine <engine>', 'Search engine: auto|brave|bing|duckduckgo', 'auto')
  .option('--top <count>', 'Maximum results to return', '10')
  .option('--max-age-minutes <minutes>', 'Reuse cached query results newer than this age', '60')
  .addHelpText('after', `
Examples:
  browser-cli search "OpenAI API docs"
  browser-cli search "site:developer.android.com architecture" --engine brave --top 5
  browser-cli search "Kotlin 2.4.0 released" --max-age-minutes 0

Agent notes:
  - Prefer --format json.
  - If topScore is low, narrow the query or constrain the site.
  - If cached=true and freshness matters, set --max-age-minutes 0.
`)
  .action(async (query: string, options: { engine: string; top: string; maxAgeMinutes: string }) => {
    await runWithOutput('search', async (format) => {
      const result = await service.search(query, options.engine, {
        top: Number(options.top),
        maxAgeMinutes: Number(options.maxAgeMinutes)
      });
      const warnings = buildSearchWarnings(result);
      return formatOutput(
        format,
        envelope(
          'search',
          result,
          {
            engineRequested: options.engine,
            engineUsed: result.engine,
            top: Number(options.top),
            maxAgeMinutes: Number(options.maxAgeMinutes),
            cached: result.cached,
            resultCount: result.results.length,
            topScore: result.results[0]?.score ?? null
          },
          warnings
        )
      );
    });
  });

program
  .command('open')
  .summary('Open and snapshot a specific URL')
  .description('Open a URL in the persistent headless browser, snapshot the HTML, and persist metadata to cache/index.')
  .argument('<url>', 'Target URL')
  .option('--force', 'Force refresh', false)
  .option('--max-age-minutes <minutes>', 'Reuse cached page snapshots newer than this age', `${24 * 60}`)
  .addHelpText('after', `
Examples:
  browser-cli open https://example.com
  browser-cli open https://developer.android.com/topic/architecture --force

Agent notes:
  - Use this when you already know the target URL.
  - Inspect statusCode and fetchDurationMs in meta.
  - A non-2xx page can still produce extracted text, but confidence is lower.
`)
  .action(async (url: string, options: { force: boolean; maxAgeMinutes: string }) => {
    await runWithOutput('open', async (format) => {
      const result = await service.open(url, options.force, { maxAgeMinutes: Number(options.maxAgeMinutes) });
      const warnings = result.page.statusCode && result.page.statusCode >= 400 ? [`Fetched page returned HTTP ${result.page.statusCode}.`] : [];
      return formatOutput(
        format,
        envelope(
          'open',
          result,
          {
            force: options.force,
            maxAgeMinutes: Number(options.maxAgeMinutes),
            cached: result.cached,
            domain: result.page.domain,
            statusCode: result.page.statusCode,
            fetchDurationMs: result.page.fetchDurationMs
          },
          warnings
        )
      );
    });
  });

program
  .command('read')
  .summary('Read extracted content from a URL or query')
  .description('Read clean text/markdown from a direct URL or search query. Query mode searches, fetches multiple candidates, and reranks extracted documents.')
  .argument('<input>', 'URL or search query')
  .option('--top <count>', 'Top results for query mode', '3')
  .option('--force', 'Force refresh', false)
  .option('--engine <engine>', 'Search engine for query mode: auto|brave|bing|duckduckgo', 'auto')
  .option('--max-age-minutes <minutes>', 'Reuse cached query/page data newer than this age', `${24 * 60}`)
  .addHelpText('after', `
Examples:
  browser-cli read https://developer.android.com/topic/architecture
  browser-cli read "OpenAI API docs" --engine brave --top 2
  browser-cli read "Kotlin multiplatform persistence guide" --max-age-minutes 0

Agent notes:
  - Use URL mode when you already trust the source.
  - Use query mode to build a small candidate set and rerank extracted documents.
  - Inspect documentCount and topScore in meta.
`)
  .action(async (inputValue: string, options: { top: string; force: boolean; engine: string; maxAgeMinutes: string }) => {
    await runWithOutput('read', async (format) => {
      const result = await service.read(inputValue, Number(options.top), options.force, {
        engine: options.engine,
        maxAgeMinutes: Number(options.maxAgeMinutes)
      });
      const warnings = buildReadWarnings(result);
      return formatOutput(
        format,
        envelope(
          'read',
          result,
          {
            source: result.source,
            top: Number(options.top),
            force: options.force,
            engine: options.engine,
            maxAgeMinutes: Number(options.maxAgeMinutes),
            documentCount: result.documents.length,
            topScore: result.documents[0]?.score ?? null
          },
          warnings
        )
      );
    });
  });

program
  .command('crawl')
  .summary('Crawl a site conservatively within one host')
  .description('Crawl a seed URL with depth and page limits, reusing the same headless session and cache.')
  .argument('<url>', 'Seed URL')
  .option('--limit <count>', 'Max pages', '10')
  .option('--depth <count>', 'Max depth', '1')
  .option('--force', 'Force refresh', false)
  .addHelpText('after', `
Examples:
  browser-cli crawl https://example.com --limit 5 --depth 1
  browser-cli crawl https://developer.android.com/topic/architecture --limit 10 --depth 2

Agent notes:
  - Crawl stays on the same host.
  - Keep limit/depth small unless you need broader discovery.
  - Use read/open on high-value pages returned by crawl.
`)
  .action(async (url: string, options: { limit: string; depth: string; force: boolean }) => {
    await runWithOutput('crawl', async (format) => {
      const result = await service.crawl(url, Number(options.limit), Number(options.depth), options.force);
      const warnings = result.visited === 0 ? ['Crawler visited zero pages. Retry with --force or a simpler seed URL.'] : [];
      return formatOutput(
        format,
        envelope(
          'crawl',
          result,
          {
            limit: Number(options.limit),
            depth: Number(options.depth),
            visited: result.visited
          },
          warnings
        )
      );
    });
  });

program
  .command('history')
  .summary('Inspect recent query and page history')
  .description('Inspect recent query history and opened page history from the local index.')
  .option('--limit <count>', 'Number of items', '20')
  .addHelpText('after', `
Examples:
  browser-cli history
  browser-cli history --limit 50

Agent notes:
  - Use history to recover previous queries and page targets.
  - Use cache inspect to understand artifact volume.
`)
  .action(async (options: { limit: string }) => {
    await runWithOutput('history', async (format) => {
      const result = service.history(Number(options.limit));
      return formatOutput(
        format,
        envelope('history', result, {
          limit: Number(options.limit),
          queryCount: result.queries.length,
          pageCount: result.pages.length
        })
      );
    });
  });

const cache = program.command('cache').description('Inspect and manage cache state').showHelpAfterError();

cache
  .command('inspect')
  .summary('Show cache/index/profile locations and counts')
  .description('Inspect storage locations and artifact counts for raw, rendered, and media cache.')
  .action(async () => {
    await runWithOutput('cache.inspect', async (format) => {
      const result = service.cacheInspect();
      return formatOutput(format, envelope('cache.inspect', result, {
        rawFiles: result.rawFiles,
        renderedFiles: result.renderedFiles,
        mediaFiles: result.mediaFiles,
        dbPath: result.dbPath
      }));
    });
  });

cache
  .command('clear')
  .summary('Delete cache, profile, and local index')
  .description('Delete all local cached content, the persistent browser profile, and the SQLite index.')
  .action(async () => {
    await runWithOutput('cache.clear', async (format) => formatOutput(format, envelope('cache.clear', service.cacheClear(), {}, ['Cache, profile, and index were fully reset.'])));
  });

cache
  .command('prune')
  .summary('Delete cache artifacts older than N days')
  .description('Delete raw/rendered/media artifacts older than the given retention window.')
  .option('--days <days>', 'Delete artifacts older than N days', '7')
  .action(async (options: { days: string }) => {
    await runWithOutput('cache.prune', async (format) =>
      formatOutput(format, envelope('cache.prune', service.cachePrune(Number(options.days)), { days: Number(options.days) }))
    );
  });

program
  .command('login')
  .summary('Open a visible browser only for authentication')
  .description('Launch a temporary headed browser to complete an authentication flow and persist session state to the normal profile.')
  .argument('<urlOrDomain>', 'Domain or URL to authenticate against')
  .addHelpText('after', `
Examples:
  browser-cli login example.com
  browser-cli login https://example.com/login

Agent notes:
  - This is the only command that intentionally uses a visible browser window.
  - Use it when a target site requires human authentication once.
`)
  .action(async (urlOrDomain: string) => {
    await runWithOutput('login', async (format) => formatOutput(format, envelope('login', await service.login(urlOrDomain), { headed: true })));
  });

program
  .command('eval')
  .summary('Run JavaScript in the page and return the JSON-serializable result')
  .description('Evaluate a JS expression (or statements) in the persistent, possibly logged-in page and return the result. Navigates to --url first if given, else reuses the last visited page.')
  .argument('<js>', 'JavaScript expression or statements to evaluate')
  .option('--url <url>', 'Navigate to this URL before evaluating')
  .addHelpText('after', `
Examples:
  browser-cli eval "document.title"
  browser-cli eval "window.scrollY"
  browser-cli eval "document.querySelectorAll('input').length" --url https://example.com

Agent notes:
  - Highest-leverage verb: enables arbitrary page interaction.
  - The result must be JSON-serializable (DOM nodes are not; return primitives/objects).
  - Runs on the persistent profile, so an authenticated ASC/Play session is reused.
`)
  .action(async (js, options) => {
    await runWithOutput('eval', async (format) => {
      const result = await service.evalJs(js, options.url);
      return formatOutput(format, envelope('eval', result, { url: result.url, hasUrl: Boolean(options.url) }));
    });
  });

program
  .command('dom-query')
  .summary('Return matching DOM element(s) with attributes, bbox, and visibility')
  .description('Query the page for elements matching a CSS selector. Returns tag, text, key attributes, bounding box, and a visible flag. By default returns visible elements only.')
  .argument('<css>', 'CSS selector')
  .option('--url <url>', 'Navigate to this URL first')
  .option('--all', 'Return all matches (default: first match)', false)
  .option('--include-hidden', 'Include hidden/offscreen elements (e.g. hidden file inputs)', false)
  .addHelpText('after', `
Examples:
  browser-cli dom-query "h1"
  browser-cli dom-query "input" --all --include-hidden
  browser-cli dom-query "input[type=file]" --include-hidden --url https://appstoreconnect.apple.com

Agent notes:
  - Use --include-hidden to find hidden upload widgets / file inputs (the ASC & Play gap).
`)
  .action(async (css, options) => {
    await runWithOutput('dom-query', async (format) => {
      const result = await service.domQuery(css, { url: options.url, all: options.all, includeHidden: options.includeHidden });
      const warnings = result.count === 0 ? ['No elements matched. Try --include-hidden or a different selector.'] : [];
      return formatOutput(format, envelope('dom-query', result, { url: result.url, selector: css, count: result.count, all: options.all, includeHidden: options.includeHidden }, warnings));
    });
  });

program
  .command('click')
  .summary('Scroll an element into view and click it')
  .description('Click an element matched by CSS selector, by visible text (contains), or by ARIA role+name. Scrolls into view first.')
  .argument('[css]', 'CSS selector (optional if using --text or --role)')
  .option('--url <url>', 'Navigate to this URL first')
  .option('--text <text>', 'Match by visible text (contains)')
  .option('--role <role>', 'Match by ARIA role, optionally with a name: --role "button Submit"')
  .addHelpText('after', `
Examples:
  browser-cli click "a.more"
  browser-cli click --text "More information"
  browser-cli click --role "button Save"

Agent notes:
  - Exactly one of <css>, --text, or --role is used (css takes precedence).
  - Reports navigated=true and the newUrl if the click triggered navigation.
`)
  .action(async (css, options) => {
    await runWithOutput('click', async (format) => {
      const target = { css: css || undefined, text: options.text, role: options.role };
      const result = await service.click(target, options.url);
      return formatOutput(format, envelope('click', result, { url: result.url, matchedBy: result.matchedBy, navigated: result.navigated, newUrl: result.newUrl }));
    });
  });

program
  .command('fill')
  .summary("Set an input's value (clears then fills)")
  .description("Fill an input/textarea/contenteditable with a value, dispatching proper input events. Clears existing content first.")
  .argument('<css>', 'CSS selector for the input')
  .argument('<value>', 'Value to set')
  .option('--url <url>', 'Navigate to this URL first')
  .addHelpText('after', `
Examples:
  browser-cli fill "input[name=email]" "test@mail.com"
  browser-cli fill "#search" "hello world" --url https://example.com
`)
  .action(async (css, value, options) => {
    await runWithOutput('fill', async (format) => {
      const result = await service.fill(css, value, 'fill', options.url);
      return formatOutput(format, envelope('fill', result, { url: result.url, selector: css }));
    });
  });

program
  .command('type')
  .summary('Type text into an input character-by-character')
  .description('Focus an input and type text key-by-key (dispatches keystroke + input events). Use for fields that require real keystrokes.')
  .argument('<css>', 'CSS selector for the input')
  .argument('<text>', 'Text to type')
  .option('--url <url>', 'Navigate to this URL first')
  .addHelpText('after', `
Examples:
  browser-cli type "input[name=q]" "query text"
`)
  .action(async (css, text, options) => {
    await runWithOutput('type', async (format) => {
      const result = await service.fill(css, text, 'type', options.url);
      return formatOutput(format, envelope('type', result, { url: result.url, selector: css }));
    });
  });

program
  .command('scroll')
  .summary('Scroll the page (to element, by px, bottom, or top)')
  .description('Scroll the page to an element, by a pixel delta, to the bottom, or to the top. Returns the resulting scroll position.')
  .option('--url <url>', 'Navigate to this URL first')
  .option('--to <css>', 'Scroll the matching element into view')
  .option('--by <px>', 'Scroll vertically by this many pixels')
  .option('--bottom', 'Scroll to the bottom of the page', false)
  .option('--top', 'Scroll to the top of the page', false)
  .addHelpText('after', `
Examples:
  browser-cli scroll --bottom
  browser-cli scroll --by 800
  browser-cli scroll --to "footer"
`)
  .action(async (options) => {
    await runWithOutput('scroll', async (format) => {
      const target = { to: options.to, by: options.by != null ? Number(options.by) : undefined, bottom: options.bottom, top: options.top };
      const result = await service.scroll(target, options.url);
      return formatOutput(format, envelope('scroll', result, { url: result.url, mode: result.mode, scrollY: result.scrollY }));
    });
  });

program
  .command('snapshot')
  .summary('Snapshot interactive elements on the page')
  .description('List interactive elements (links, buttons, inputs, role/onclick widgets) with attributes, bbox, and visibility. --include-hidden also surfaces hidden file inputs and custom upload widgets.')
  .option('--url <url>', 'Navigate to this URL first')
  .option('--include-hidden', 'Include hidden elements + hidden file inputs / custom upload widgets', false)
  .addHelpText('after', `
Examples:
  browser-cli snapshot
  browser-cli snapshot --include-hidden --url https://appstoreconnect.apple.com

Agent notes:
  - fileInputs is always populated (incl. hidden) to expose upload widgets on ASC/Play.
`)
  .action(async (options) => {
    await runWithOutput('snapshot', async (format) => {
      const result = await service.snapshot({ url: options.url, includeHidden: options.includeHidden });
      return formatOutput(format, envelope('snapshot', result, { url: result.url, title: result.title, elementCount: result.elements.length, fileInputCount: result.fileInputs.length, includeHidden: options.includeHidden }));
    });
  });

program.parseAsync(process.argv).finally(async () => {
  await service.shutdown();
});

async function runWithOutput(commandName: string, fn: (format: OutputFormat) => Promise<string>): Promise<void> {
  try {
    const format = (program.opts().format ?? 'json') as OutputFormat;
    const output = await fn(format);
    process.stdout.write(`${output}\n`);
  } catch (error) {
    const cliError = toCliError(error);
    process.stderr.write(
      `${JSON.stringify(envelope(commandName, {}, { suggestion: cliError.suggestion }, [], [cliError]), null, 2)}\n`
    );
    process.exitCode = 1;
  }
}

function buildSearchWarnings(result: Awaited<ReturnType<BrowserCliService['search']>>): string[] {
  const warnings: string[] = [];
  if (result.cached) warnings.push('Results were served from the local query cache. Use --max-age-minutes 0 for a fresh search.');
  if (result.results.length === 0) warnings.push('Search returned zero results. Retry with a narrower query or a different engine.');
  const lowTopScore = result.results[0]?.score != null && (result.results[0]?.score ?? 0) < 8;
  if (lowTopScore) warnings.push('Top result confidence is low. Prefer a site-constrained query or use read on a known URL.');
  return warnings;
}

function buildReadWarnings(result: Awaited<ReturnType<BrowserCliService['read']>>): string[] {
  const warnings: string[] = [];
  if (result.documents.length === 0) warnings.push('No readable documents were extracted. Retry with a narrower query or a known URL.');
  const weakTop = result.documents[0]?.score != null && (result.documents[0]?.score ?? 0) < 12;
  if (weakTop) warnings.push('Top extracted document confidence is low. Consider using a site-constrained query or direct URL.');
  return warnings;
}
