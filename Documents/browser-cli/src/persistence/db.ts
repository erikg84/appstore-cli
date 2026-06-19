import Database from 'better-sqlite3';
import { getStoragePaths } from './paths.js';
import type { DocumentRecord, PageRecord, SearchResult } from '../types/index.js';

export class BrowserCliDb {
  readonly db: Database.Database;

  constructor() {
    const paths = getStoragePaths();
    this.db = new Database(paths.dbPath);
    this.db.pragma('journal_mode = WAL');
    this.migrate();
  }

  private migrate(): void {
    this.db.exec(`
      CREATE TABLE IF NOT EXISTS queries (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        query_text TEXT NOT NULL,
        engine TEXT NOT NULL,
        created_at TEXT NOT NULL,
        result_count INTEGER NOT NULL DEFAULT 0
      );

      CREATE TABLE IF NOT EXISTS search_results (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        query_id INTEGER NOT NULL,
        rank INTEGER NOT NULL,
        title TEXT NOT NULL,
        url TEXT NOT NULL,
        snippet TEXT,
        score REAL,
        source_engine TEXT,
        FOREIGN KEY(query_id) REFERENCES queries(id)
      );

      CREATE TABLE IF NOT EXISTS pages (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        url TEXT NOT NULL UNIQUE,
        canonical_url TEXT NOT NULL,
        domain TEXT NOT NULL,
        title TEXT NOT NULL,
        status_code INTEGER,
        content_type TEXT,
        html_path TEXT,
        visited_at TEXT NOT NULL,
        fetch_duration_ms INTEGER NOT NULL,
        content_hash TEXT
      );

      CREATE TABLE IF NOT EXISTS documents (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        page_id INTEGER NOT NULL,
        format TEXT NOT NULL,
        content_path TEXT NOT NULL,
        created_at TEXT NOT NULL,
        UNIQUE(page_id, format),
        FOREIGN KEY(page_id) REFERENCES pages(id)
      );

      CREATE TABLE IF NOT EXISTS crawl_runs (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        seed_url TEXT NOT NULL,
        started_at TEXT NOT NULL,
        visited_count INTEGER NOT NULL DEFAULT 0
      );

      CREATE TABLE IF NOT EXISTS crawl_pages (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        crawl_run_id INTEGER NOT NULL,
        page_id INTEGER NOT NULL,
        depth INTEGER NOT NULL,
        FOREIGN KEY(crawl_run_id) REFERENCES crawl_runs(id),
        FOREIGN KEY(page_id) REFERENCES pages(id)
      );
    `);

    const columns = this.db.prepare(`PRAGMA table_info(search_results)`).all() as Array<{ name: string }>;
    const columnNames = new Set(columns.map((column) => column.name));
    if (!columnNames.has('score')) this.db.exec('ALTER TABLE search_results ADD COLUMN score REAL');
    if (!columnNames.has('source_engine')) this.db.exec('ALTER TABLE search_results ADD COLUMN source_engine TEXT');
  }

  insertQuery(query: string, engine: string, results: SearchResult[]): number {
    const createdAt = new Date().toISOString();
    const insertQuery = this.db.prepare(
      'INSERT INTO queries (query_text, engine, created_at, result_count) VALUES (?, ?, ?, ?)'
    );
    const queryId = Number(insertQuery.run(query, engine, createdAt, results.length).lastInsertRowid);
    const insertResult = this.db.prepare(
      'INSERT INTO search_results (query_id, rank, title, url, snippet, score, source_engine) VALUES (?, ?, ?, ?, ?, ?, ?)'
    );
    const tx = this.db.transaction((items: SearchResult[]) => {
      for (const item of items) {
        insertResult.run(queryId, item.rank, item.title, item.url, item.snippet ?? null, item.score ?? null, item.sourceEngine ?? engine);
      }
    });
    tx(results);
    return queryId;
  }

  getRecentQueryResults(query: string, engine: string, maxAgeMinutes: number): SearchResult[] | null {
    const cutoff = new Date(Date.now() - maxAgeMinutes * 60 * 1000).toISOString();
    const queryRow = this.db.prepare(`
      SELECT id FROM queries
      WHERE query_text = ? AND engine = ? AND created_at >= ?
      ORDER BY id DESC LIMIT 1
    `).get(query, engine, cutoff) as { id: number } | undefined;
    if (!queryRow) return null;
    const rows = this.db.prepare(`
      SELECT rank, title, url, snippet, score, source_engine as sourceEngine
      FROM search_results WHERE query_id = ? ORDER BY rank ASC
    `).all(queryRow.id) as SearchResult[];
    if (rows.length === 0) return null;
    if (rows.some((row) => row.score == null || !row.sourceEngine)) return null;
    return rows;
  }

  upsertPage(input: Omit<PageRecord, 'id'>): PageRecord {
    const existing = this.db.prepare('SELECT id FROM pages WHERE url = ?').get(input.url) as { id: number } | undefined;

    if (existing) {
      this.db.prepare(`
        UPDATE pages
        SET canonical_url = ?, domain = ?, title = ?, status_code = ?, content_type = ?, html_path = ?,
            visited_at = ?, fetch_duration_ms = ?, content_hash = ?
        WHERE id = ?
      `).run(
        input.canonicalUrl,
        input.domain,
        input.title,
        input.statusCode,
        input.contentType,
        input.htmlPath,
        input.visitedAt,
        input.fetchDurationMs,
        input.contentHash,
        existing.id
      );
      return { id: existing.id, ...input };
    }

    const result = this.db.prepare(`
      INSERT INTO pages (url, canonical_url, domain, title, status_code, content_type, html_path, visited_at, fetch_duration_ms, content_hash)
      VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
    `).run(
      input.url,
      input.canonicalUrl,
      input.domain,
      input.title,
      input.statusCode,
      input.contentType,
      input.htmlPath,
      input.visitedAt,
      input.fetchDurationMs,
      input.contentHash
    );

    return { id: Number(result.lastInsertRowid), ...input };
  }

  upsertDocument(pageId: number, format: 'markdown' | 'text', contentPath: string): DocumentRecord {
    const existing = this.db.prepare('SELECT id FROM documents WHERE page_id = ? AND format = ?').get(pageId, format) as { id: number } | undefined;
    const createdAt = new Date().toISOString();
    if (existing) {
      this.db.prepare('UPDATE documents SET content_path = ?, created_at = ? WHERE id = ?').run(contentPath, createdAt, existing.id);
      return { id: existing.id, pageId, format, contentPath, createdAt };
    }
    const result = this.db.prepare('INSERT INTO documents (page_id, format, content_path, created_at) VALUES (?, ?, ?, ?)').run(pageId, format, contentPath, createdAt);
    return { id: Number(result.lastInsertRowid), pageId, format, contentPath, createdAt };
  }

  getPage(url: string): PageRecord | null {
    const row = this.db.prepare(`
      SELECT id, url, canonical_url as canonicalUrl, domain, title, status_code as statusCode,
             content_type as contentType, html_path as htmlPath, visited_at as visitedAt,
             fetch_duration_ms as fetchDurationMs, content_hash as contentHash
      FROM pages WHERE url = ?
    `).get(url) as PageRecord | undefined;
    return row ?? null;
  }

  getDocument(pageId: number, format: 'markdown' | 'text'): DocumentRecord | null {
    const row = this.db.prepare(`
      SELECT id, page_id as pageId, format, content_path as contentPath, created_at as createdAt
      FROM documents WHERE page_id = ? AND format = ?
    `).get(pageId, format) as DocumentRecord | undefined;
    return row ?? null;
  }

  listQueries(limit = 20): Array<{ query: string; engine: string; createdAt: string; resultCount: number }> {
    return this.db.prepare(`
      SELECT query_text as query, engine, created_at as createdAt, result_count as resultCount
      FROM queries ORDER BY id DESC LIMIT ?
    `).all(limit) as Array<{ query: string; engine: string; createdAt: string; resultCount: number }>;
  }

  listPages(limit = 20): PageRecord[] {
    return this.db.prepare(`
      SELECT id, url, canonical_url as canonicalUrl, domain, title, status_code as statusCode,
             content_type as contentType, html_path as htmlPath, visited_at as visitedAt,
             fetch_duration_ms as fetchDurationMs, content_hash as contentHash
      FROM pages ORDER BY id DESC LIMIT ?
    `).all(limit) as PageRecord[];
  }

  startCrawl(seedUrl: string): number {
    const result = this.db.prepare('INSERT INTO crawl_runs (seed_url, started_at, visited_count) VALUES (?, ?, 0)').run(seedUrl, new Date().toISOString());
    return Number(result.lastInsertRowid);
  }

  recordCrawlPage(crawlRunId: number, pageId: number, depth: number): void {
    this.db.prepare('INSERT INTO crawl_pages (crawl_run_id, page_id, depth) VALUES (?, ?, ?)').run(crawlRunId, pageId, depth);
    this.db.prepare('UPDATE crawl_runs SET visited_count = visited_count + 1 WHERE id = ?').run(crawlRunId);
  }

  close(): void {
    this.db.close();
  }
}
