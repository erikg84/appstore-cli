import { createHash } from 'node:crypto';
import { readFileSync, rmSync, statSync, unlinkSync, writeFileSync, existsSync, readdirSync } from 'node:fs';
import path from 'node:path';
import readline from 'node:readline/promises';
import { stdin as input, stdout as output } from 'node:process';
import type { Page } from 'playwright';
import { JSDOM } from 'jsdom';
import { BrowserRuntime } from '../browser/runtime.js';
import {
  clickElement,
  domQuery,
  evalInPage,
  fillElement,
  scrollPage,
  snapshotPage,
  type ClickTarget,
  type ScrollTarget
} from '../browser/actions.js';
import { extractDocumentFromHtml } from '../extraction/extractor.js';
import { BrowserCliDb } from '../persistence/db.js';
import { loadConfig } from '../persistence/config.js';
import { getStoragePaths } from '../persistence/paths.js';
import { runDoctor } from './doctor.js';
import type {
  CacheInspectResult,
  CrawlResult,
  HistoryResult,
  OpenResult,
  PageRecord,
  ReadDocument,
  ReadResult,
  SearchCommandResult,
  SearchResult,
  ClickResult,
  DomQueryResult,
  EvalResult,
  FillResult,
  ScrollResult,
  SnapshotResult
} from '../types/index.js';

const STOP_WORDS = new Set([
  'a', 'an', 'and', 'are', 'as', 'at', 'be', 'best', 'by', 'current', 'for', 'from', 'how', 'in', 'is', 'it', 'latest', 'of', 'on',
  'or', 'popular', 'that', 'the', 'this', 'to', 'today', 'what', 'when', 'where', 'which', 'with'
]);

const LOW_SIGNAL_DOMAINS = ['play.google.com', 'apps.apple.com', 'wikipedia.org'];
const DOC_HINTS = ['docs', 'documentation', 'api', 'guide', 'reference', 'architecture'];
const INTENT_GENERIC_TOKENS = new Set(['app', 'architecture', 'api', 'docs', 'documentation', 'guide', 'popular', 'latest', 'current', 'reference']);

export class BrowserCliService {
  private readonly db = new BrowserCliDb();
  private readonly runtime = new BrowserRuntime();
  private readonly paths = getStoragePaths();
  private readonly config = loadConfig();

  async search(
    query: string,
    engine = 'bing',
    options: { maxAgeMinutes?: number; top?: number } = {}
  ): Promise<SearchCommandResult> {
    const selectedEngine = engine === 'auto' ? (this.config.braveApiKey ? 'brave' : 'bing') : engine;
    const maxAgeMinutes = options.maxAgeMinutes ?? 60;
    const cached = this.db.getRecentQueryResults(query, selectedEngine, maxAgeMinutes);
    if (cached && cached.length > 0) {
      return {
        query,
        engine: selectedEngine,
        cached: true,
        results: cached.slice(0, options.top ?? cached.length)
      };
    }

    const page = await this.runtime.newPage();
    try {
      const primary = selectedEngine === 'duckduckgo'
        ? await this.performDuckDuckGoSearch(page, query)
        : selectedEngine === 'brave'
          ? await this.performBraveSearch(query)
          : await this.performBingSearch(page, query);
      let finalResults = rerankResults(query, primary, selectedEngine);

      if (shouldRunQuotedFallback(query, finalResults)) {
        const fallbackSets: SearchResult[][] = [];
        fallbackSets.push(selectedEngine === 'brave' ? await this.performBraveSearch(`"${query}"`) : await this.performBingSearch(page, `"${query}"`));
        const simplified = simplifyQuery(query);
        if (simplified && simplified.toLowerCase() != query.toLowerCase()) {
          fallbackSets.push(selectedEngine === 'brave' ? await this.performBraveSearch(simplified) : await this.performBingSearch(page, simplified));
        }
        for (const variant of buildSiteConstrainedVariants(query)) {
          fallbackSets.push(selectedEngine === 'brave' ? await this.performBraveSearch(variant) : await this.performBingSearch(page, variant));
        }
        finalResults = rerankResults(query, mergeResults(finalResults, ...fallbackSets), 'bing');
      }

      const limited = finalResults.slice(0, options.top ?? 10).map((item, index) => ({ ...item, rank: index + 1 }));
      this.db.insertQuery(query, selectedEngine, limited);
      return { query, engine: selectedEngine, cached: false, results: limited };
    } finally {
      await page.close();
      await this.runtime.close();
    }
  }

  async open(
    url: string,
    force = false,
    options: { maxAgeMinutes?: number } = {}
  ): Promise<OpenResult> {
    const existing = this.db.getPage(url);
    const maxAgeMinutes = options.maxAgeMinutes ?? 24 * 60;
    if (existing && existing.htmlPath && existsSync(existing.htmlPath) && !force && !isStale(existing.visitedAt, maxAgeMinutes)) {
      return { page: existing, cached: true };
    }

    const page = await this.runtime.newPage();
    const started = Date.now();
    try {
      const response = await page.goto(url, { waitUntil: 'domcontentloaded' });
      await page.waitForLoadState('networkidle', { timeout: 10000 }).catch(() => undefined);
      const html = await page.content();
      const finalUrl = page.url();
      const title = await page.title();
      const duration = Date.now() - started;
      const hash = createHash('sha256').update(html).digest('hex');
      const htmlPath = path.join(this.paths.cacheRawDir, `${hash}.html`);
      writeFileSync(htmlPath, html);

      const record = this.db.upsertPage({
        url,
        canonicalUrl: finalUrl,
        domain: new URL(finalUrl).hostname,
        title: title || finalUrl,
        statusCode: response?.status() ?? null,
        contentType: response?.headers()['content-type'] ?? null,
        htmlPath,
        visitedAt: new Date().toISOString(),
        fetchDurationMs: duration,
        contentHash: hash
      });

      return { page: record, cached: false };
    } finally {
      await page.close();
      await this.runtime.close();
    }
  }

  async read(
    inputValue: string,
    top = 3,
    force = false,
    options: { maxAgeMinutes?: number; engine?: string } = {}
  ): Promise<ReadResult> {
    const isUrl = looksLikeUrl(inputValue);
    const maxAgeMinutes = options.maxAgeMinutes ?? 24 * 60;
    if (isUrl) {
      const opened = await this.open(inputValue, force, { maxAgeMinutes });
      const docs = this.ensureDocuments(opened.page);
      return { source: 'url', documents: [docs] };
    }

    const searchResult = await this.search(inputValue, options.engine ?? 'bing', {
      maxAgeMinutes: Math.min(maxAgeMinutes, 120),
      top: Math.max(top * 3, 8)
    });

    const candidates: ReadDocument[] = [];
    for (const result of searchResult.results.slice(0, Math.max(top * 3, 8))) {
      try {
        const opened = await this.open(result.url, force, { maxAgeMinutes });
        const doc = this.ensureDocuments(opened.page);
        const score = scoreDocumentForQuery(inputValue, doc.page, doc.text, result);
        candidates.push({ ...doc, score });
      } catch {
        continue;
      }
    }

    const viable = candidates.filter((candidate) => !isBlockedOrErrorPage(candidate));
    const rankedPool = viable.length >= top ? viable : candidates;
    const ranked = rankedPool.sort((left, right) => (right.score ?? 0) - (left.score ?? 0)).slice(0, top);
    return { source: 'query', query: inputValue, documents: ranked };
  }

  async crawl(seedUrl: string, limit = 10, depthLimit = 1, force = false): Promise<CrawlResult> {
    const seedHost = new URL(seedUrl).hostname;
    const crawlRunId = this.db.startCrawl(seedUrl);
    const queue: Array<{ url: string; depth: number }> = [{ url: seedUrl, depth: 0 }];
    const seen = new Set<string>();
    const pages: PageRecord[] = [];

    while (queue.length > 0 && pages.length < limit) {
      const current = queue.shift();
      if (!current || seen.has(current.url) || current.depth > depthLimit) continue;
      seen.add(current.url);
      try {
        const opened = await this.open(current.url, force);
        pages.push(opened.page);
        this.db.recordCrawlPage(crawlRunId, opened.page.id, current.depth);
        if (current.depth >= depthLimit) continue;
        const links = await this.extractLinks(current.url);
        for (const link of links) {
          if (pages.length + queue.length >= limit) break;
          if (new URL(link).hostname === seedHost && !seen.has(link)) {
            queue.push({ url: link, depth: current.depth + 1 });
          }
        }
      } catch {
        continue;
      }
    }

    return { seedUrl, visited: pages.length, pages };
  }

  doctor() {
    return runDoctor(this.config);
  }

  history(limit = 20): HistoryResult {
    return {
      queries: this.db.listQueries(limit),
      pages: this.db.listPages(limit)
    };
  }

  cacheInspect(): CacheInspectResult {
    return {
      root: this.paths.root,
      profileDir: this.paths.profileDir,
      rawFiles: countFiles(this.paths.cacheRawDir),
      renderedFiles: countFiles(this.paths.cacheRenderedDir),
      mediaFiles: countFiles(this.paths.cacheMediaDir),
      dbPath: this.paths.dbPath
    };
  }

  cacheClear(): CacheInspectResult {
    this.db.close();
    rmSync(this.paths.cacheRawDir, { recursive: true, force: true });
    rmSync(this.paths.cacheRenderedDir, { recursive: true, force: true });
    rmSync(this.paths.cacheMediaDir, { recursive: true, force: true });
    rmSync(this.paths.profileDir, { recursive: true, force: true });
    rmSync(this.paths.dbPath, { force: true });
    getStoragePaths();
    return this.cacheInspect();
  }

  cachePrune(days = 7): CacheInspectResult {
    const cutoff = Date.now() - days * 24 * 60 * 60 * 1000;
    for (const dir of [this.paths.cacheRawDir, this.paths.cacheRenderedDir, this.paths.cacheMediaDir]) {
      for (const name of readdirSync(dir)) {
        const filePath = path.join(dir, name);
        const stats = statSync(filePath);
        if (stats.mtimeMs < cutoff) unlinkSync(filePath);
      }
    }
    return this.cacheInspect();
  }

  async login(urlOrDomain: string): Promise<{ message: string; profileDir: string }> {
    const target = looksLikeUrl(urlOrDomain) ? urlOrDomain : `https://${urlOrDomain}`;
    const page = await this.runtime.newPage({ headed: true });
    try {
      await page.goto(target, { waitUntil: 'domcontentloaded' });
      const rl = readline.createInterface({ input, output });
      await rl.question('Complete login in the browser window, then press Enter here to persist session.');
      rl.close();
      return { message: 'Login session persisted.', profileDir: this.paths.profileDir };
    } finally {
      await page.close();
      await this.runtime.close();
    }
  }

  /** Resolve a target URL: explicit --url wins, else the most recent visited page. */
  private resolveTargetUrl(url?: string): string | undefined {
    if (url) return url;
    const recent = this.db.listPages(1)[0];
    return recent?.canonicalUrl ?? recent?.url;
  }

  async evalJs(js: string, url?: string): Promise<EvalResult> {
    return evalInPage(this.runtime, js, this.resolveTargetUrl(url));
  }

  async domQuery(
    selector: string,
    options: { url?: string; all?: boolean; includeHidden?: boolean } = {}
  ): Promise<DomQueryResult> {
    return domQuery(
      this.runtime,
      selector,
      this.resolveTargetUrl(options.url),
      options.all ?? false,
      options.includeHidden ?? false
    );
  }

  async click(target: ClickTarget, url?: string): Promise<ClickResult> {
    return clickElement(this.runtime, target, this.resolveTargetUrl(url));
  }

  async fill(selector: string, value: string, mode: 'fill' | 'type', url?: string): Promise<FillResult> {
    return fillElement(this.runtime, selector, value, mode, this.resolveTargetUrl(url));
  }

  async scroll(target: ScrollTarget, url?: string): Promise<ScrollResult> {
    return scrollPage(this.runtime, target, this.resolveTargetUrl(url));
  }

  async snapshot(options: { url?: string; includeHidden?: boolean } = {}): Promise<SnapshotResult> {
    return snapshotPage(this.runtime, this.resolveTargetUrl(options.url), options.includeHidden ?? false);
  }

  async shutdown(): Promise<void> {
    await this.runtime.close();
    this.db.close();
  }

  private ensureDocuments(page: PageRecord): ReadDocument {
    if (!page.htmlPath) throw new Error(`No HTML snapshot available for ${page.url}`);
    const markdownDoc = this.db.getDocument(page.id, 'markdown');
    const textDoc = this.db.getDocument(page.id, 'text');

    if (markdownDoc && textDoc && existsSync(markdownDoc.contentPath) && existsSync(textDoc.contentPath)) {
      return {
        page,
        markdown: readFileSync(markdownDoc.contentPath, 'utf8'),
        text: readFileSync(textDoc.contentPath, 'utf8')
      };
    }

    const html = readFileSync(page.htmlPath, 'utf8');
    const extracted = extractDocumentFromHtml(html, page.canonicalUrl);
    const baseName = page.contentHash ?? createHash('sha256').update(page.url).digest('hex');
    const markdownPath = path.join(this.paths.cacheRenderedDir, `${baseName}.md`);
    const textPath = path.join(this.paths.cacheRenderedDir, `${baseName}.txt`);
    writeFileSync(markdownPath, extracted.markdown);
    writeFileSync(textPath, extracted.text);
    this.db.upsertDocument(page.id, 'markdown', markdownPath);
    this.db.upsertDocument(page.id, 'text', textPath);

    return {
      page: { ...page, title: extracted.title || page.title },
      markdown: extracted.markdown,
      text: extracted.text
    };
  }

  private async performDuckDuckGoSearch(page: Page, query: string): Promise<SearchResult[]> {
    const searchUrl = `https://html.duckduckgo.com/html/?q=${encodeURIComponent(query)}`;
    await page.goto(searchUrl, { waitUntil: 'domcontentloaded' });
    await page.waitForTimeout(1000);
    const bodyText = (await page.textContent('body')) ?? '';
    if (bodyText.includes('Unfortunately, bots use DuckDuckGo too.')) {
      throw new Error('DuckDuckGo blocked this search request with a bot challenge. Use --engine bing.');
    }
    const html = await page.content();
    const dom = new JSDOM(html, { url: searchUrl });
    const anchors = Array.from(dom.window.document.querySelectorAll('a.result__a')) as HTMLAnchorElement[];
    return anchors
      .map((anchor, index) => {
        const row = anchor.closest('.result');
        const snippet = row?.querySelector('.result__snippet')?.textContent?.trim() ?? '';
        return {
          rank: index + 1,
          title: anchor.textContent?.trim() ?? anchor.href,
          url: anchor.href,
          snippet,
          sourceEngine: 'duckduckgo'
        };
      })
      .filter((item) => item.url.startsWith('http'))
      .slice(0, 10);
  }

  private async performBraveSearch(query: string): Promise<SearchResult[]> {
    const apiKey = this.config.braveApiKey || process.env.BRAVE_SEARCH_API_KEY;
    if (!apiKey) throw new Error('Brave API key is not configured.');
    const url = `https://api.search.brave.com/res/v1/web/search?q=${encodeURIComponent(query)}&count=10`;
    const response = await fetch(url, {
      headers: {
        'Accept': 'application/json',
        'X-Subscription-Token': apiKey
      }
    });
    if (!response.ok) {
      throw new Error(`Brave search failed: ${response.status} ${response.statusText}`);
    }
    const payload = await response.json() as { web?: { results?: Array<{ title?: string; url?: string; description?: string }> } };
    const results = payload.web?.results ?? [];
    return results
      .map((item, index) => ({
        rank: index + 1,
        title: item.title ?? '',
        url: item.url ?? '',
        snippet: item.description ?? '',
        sourceEngine: 'brave'
      }))
      .filter((item) => item.url.startsWith('http') && item.title)
      .slice(0, 10);
  }

  private async performBingSearch(page: Page, query: string): Promise<SearchResult[]> {
    const searchUrl = `https://www.bing.com/search?q=${encodeURIComponent(query)}`;
    await page.goto(searchUrl, { waitUntil: 'domcontentloaded' });
    await page.waitForTimeout(1000);
    await page.waitForLoadState('networkidle', { timeout: 10000 }).catch(() => undefined);
    const html = await page.content();
    const dom = new JSDOM(html, { url: searchUrl });
    const results = Array.from(dom.window.document.querySelectorAll('li.b_algo')).slice(0, 10).map((item, index) => ({
      rank: index + 1,
      title: item.querySelector('h2')?.textContent?.trim() ?? '',
      url: (item.querySelector('h2 a') as HTMLAnchorElement | null)?.href ?? '',
      snippet: item.querySelector('.b_caption p')?.textContent?.trim() ?? '',
      sourceEngine: 'bing'
    }));
    return results
      .filter((item) => item.url.startsWith('http') && item.title)
      .map((item) => ({ ...item, url: normalizeBingUrl(item.url) }));
  }

  private async extractLinks(url: string): Promise<string[]> {
    const page = await this.runtime.newPage();
    try {
      await page.goto(url, { waitUntil: 'domcontentloaded' });
      await page.waitForLoadState('networkidle', { timeout: 5000 }).catch(() => undefined);
      const links = await page.evaluate(() =>
        Array.from(document.querySelectorAll('a[href]'))
          .map((anchor) => (anchor as HTMLAnchorElement).href)
          .filter(Boolean)
      );
      return Array.from(new Set(links)).filter((link) => {
        try {
          const parsed = new URL(link);
          return parsed.protocol === 'http:' || parsed.protocol === 'https:';
        } catch {
          return false;
        }
      });
    } finally {
      await page.close();
      await this.runtime.close();
    }
  }
}

function countFiles(dir: string): number {
  return readdirSync(dir).length;
}

function looksLikeUrl(input: string): boolean {
  try {
    const url = input.startsWith('http://') || input.startsWith('https://') ? new URL(input) : null;
    return Boolean(url);
  } catch {
    return false;
  }
}

function isStale(visitedAt: string, maxAgeMinutes: number): boolean {
  return Date.now() - new Date(visitedAt).getTime() > maxAgeMinutes * 60 * 1000;
}

function mergeResults(...sets: SearchResult[][]): SearchResult[] {
  const seen = new Set<string>();
  const merged: SearchResult[] = [];
  for (const result of sets.flat()) {
    if (seen.has(result.url)) continue;
    seen.add(result.url);
    merged.push(result);
  }
  return merged;
}

function shouldRunQuotedFallback(query: string, results: SearchResult[]): boolean {
  if (results.length === 0) return true;
  const topScore = results[0]?.score ?? 0;
  const queryTokens = tokenize(query);
  return queryTokens.length >= 2 && topScore < 8;
}

function rerankResults(query: string, results: SearchResult[], engine: string): SearchResult[] {
  return results
    .map((result) => ({ ...result, score: scoreSearchResult(query, result), sourceEngine: result.sourceEngine ?? engine }))
    .sort((left, right) => (right.score ?? 0) - (left.score ?? 0))
    .map((result, index) => ({ ...result, rank: index + 1 }));
}

function scoreSearchResult(query: string, result: SearchResult): number {
  const queryTokens = tokenize(query);
  const haystack = `${result.title} ${result.url} ${result.snippet ?? ''}`.toLowerCase();
  let score = 0;
  for (const token of queryTokens) {
    if (haystack.includes(token)) score += 3;
    if (result.title.toLowerCase().includes(token)) score += 2;
    if (result.url.toLowerCase().includes(token)) score += 2;
  }

  const url = safeUrl(result.url);
  const host = url?.hostname ?? '';
  const pathName = url?.pathname.toLowerCase() ?? '';

  const docsIntent = DOC_HINTS.some((hint) => query.toLowerCase().includes(hint));
  if (docsIntent && (host.includes('developer.') || pathName.includes('/docs') || pathName.includes('/api') || pathName.includes('/reference'))) {
    score += 8;
  }

  if (host.includes('github.com')) score += 2;
  if (host.includes('developer.') || host.includes('docs.')) score += 4;
  if (docsIntent && (host.includes('smapply.org') || haystack.includes('grant') || haystack.includes('apply for api credits') || haystack.includes('sign up'))) score -= 10;
  if (LOW_SIGNAL_DOMAINS.some((domain) => host.includes(domain))) score -= 4;
  if (pathName.includes('/search')) score -= 2;
  if (result.title.length < 4) score -= 2;
  if (/\bcurrent\b/i.test(result.title) && !query.toLowerCase().includes('current ')) score -= 3;

  return score;
}

function scoreDocumentForQuery(query: string, page: PageRecord, text: string, searchResult: SearchResult): number {
  const queryTokens = tokenize(query);
  const title = page.title.toLowerCase();
  const url = page.canonicalUrl.toLowerCase();
  const head = text.toLowerCase().slice(0, 4000);
  let score = searchResult.score ?? 0;

  for (const token of queryTokens) {
    if (title.includes(token)) score += 3;
    if (url.includes(token)) score += 2;
    if (head.includes(token)) score += 1.5;
  }

  const docsIntent = DOC_HINTS.some((hint) => query.toLowerCase().includes(hint));
  if (docsIntent && (url.includes('/docs') || url.includes('/api') || url.includes('/reference'))) {
    score += 10;
  }
  if (docsIntent && (head.includes('grant') || head.includes('apply for api credits') || head.includes('sign up'))) score -= 12;

  if ((page.statusCode ?? 200) >= 400) score -= 25;
  if (title.includes('just a moment')) score -= 25;
  if (head.includes('cloudflare') || head.includes('incorrect device time') || head.includes('security verification process')) score -= 25;
  if (head.length < 150) score -= 8;
  if (head.includes('sign in') || head.includes('log in')) score -= 6;
  if (head.includes('quiz me') || head.includes('plan a')) score -= 10;

  return score;
}

function tokenize(value: string): string[] {
  return value
    .toLowerCase()
    .split(/[^a-z0-9]+/)
    .filter((token) => token.length > 1 && !STOP_WORDS.has(token));
}

function safeUrl(value: string): URL | null {
  try {
    return new URL(value);
  } catch {
    return null;
  }
}

function normalizeBingUrl(url: string): string {
  try {
    const parsed = new URL(url);
    if (parsed.hostname !== 'www.bing.com') return url;
    const encoded = parsed.searchParams.get('u');
    if (!encoded) return url;
    const payload = encoded.startsWith('a1') ? encoded.slice(2) : encoded;
    return Buffer.from(payload, 'base64').toString('utf8') || url;
  } catch {
    return url;
  }
}

function simplifyQuery(query: string): string | null {
  const tokens = tokenize(query);
  if (tokens.length === 0) return null;
  return tokens.join(' ');
}

function isBlockedOrErrorPage(document: ReadDocument): boolean {
  const head = document.text.toLowerCase().slice(0, 1200);
  const title = document.page.title.toLowerCase();
  return (document.page.statusCode ?? 200) >= 400
    || title.includes('just a moment')
    || head.includes('cloudflare')
    || head.includes('security verification process')
    || head.includes('incorrect device time');
}

function buildSiteConstrainedVariants(query: string): string[] {
  const variants: string[] = [];
  const docsIntent = DOC_HINTS.some((hint) => query.toLowerCase().includes(hint));
  if (!docsIntent) return variants;
  const simplified = simplifyQuery(query);
  if (!simplified) return variants;
  const brandToken = getBrandToken(query);
  if (!brandToken) return variants;
  variants.push(`site:${brandToken}.com ${simplified}`);
  variants.push(`site:developer.${brandToken}.com ${simplified}`);
  variants.push(`site:docs.${brandToken}.com ${simplified}`);
  return Array.from(new Set(variants));
}

function getBrandToken(query: string): string | null {
  const tokens = tokenize(query).filter((token) => !INTENT_GENERIC_TOKENS.has(token));
  return tokens[0] ?? null;
}
