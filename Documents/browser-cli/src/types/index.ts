export type OutputFormat = 'json' | 'markdown' | 'text';

export interface CliErrorInfo {
  code: string;
  message: string;
  suggestion?: string;
}

export interface OutputEnvelope<T> {
  ok: boolean;
  command: string;
  timestamp: string;
  data: T;
  meta: Record<string, unknown>;
  warnings: string[];
  errors: CliErrorInfo[];
}

export interface SearchResult {
  rank: number;
  title: string;
  url: string;
  snippet?: string;
  score?: number;
  sourceEngine?: string;
}

export interface PageRecord {
  id: number;
  url: string;
  canonicalUrl: string;
  domain: string;
  title: string;
  statusCode: number | null;
  contentType: string | null;
  htmlPath: string | null;
  visitedAt: string;
  fetchDurationMs: number;
  contentHash: string | null;
}

export interface DocumentRecord {
  id: number;
  pageId: number;
  format: 'markdown' | 'text';
  contentPath: string;
  createdAt: string;
}

export interface OpenResult {
  page: PageRecord;
  cached: boolean;
}

export interface ReadDocument {
  page: PageRecord;
  markdown: string;
  text: string;
  score?: number;
}

export interface ReadResult {
  source: 'url' | 'query';
  query?: string;
  documents: ReadDocument[];
}

export interface SearchCommandResult {
  query: string;
  engine: string;
  cached: boolean;
  results: SearchResult[];
}

export interface HistoryResult {
  queries: Array<{ query: string; engine: string; createdAt: string; resultCount: number }>;
  pages: PageRecord[];
}

export interface CacheInspectResult {
  root: string;
  profileDir: string;
  rawFiles: number;
  renderedFiles: number;
  mediaFiles: number;
  dbPath: string;
}

export interface CrawlResult {
  seedUrl: string;
  visited: number;
  pages: PageRecord[];
}

export interface ElementInfo {
  index: number;
  tag: string;
  text: string;
  attributes: Record<string, string>;
  boundingBox: { x: number; y: number; width: number; height: number } | null;
  visible: boolean;
}

export interface DomQueryResult {
  url: string;
  selector: string;
  count: number;
  elements: ElementInfo[];
}

export interface EvalResult {
  url: string;
  result: unknown;
}

export interface ClickResult {
  url: string;
  matchedBy: "css" | "text" | "role";
  selector: string;
  clicked: boolean;
  newUrl: string;
  navigated: boolean;
  element: ElementInfo | null;
}

export interface FillResult {
  url: string;
  selector: string;
  value: string;
  mode: "fill" | "type";
}

export interface ScrollResult {
  url: string;
  mode: "to" | "by" | "bottom" | "top";
  scrollX: number;
  scrollY: number;
}

export interface SnapshotResult {
  url: string;
  title: string;
  includeHidden: boolean;
  elements: ElementInfo[];
  fileInputs: ElementInfo[];
}
