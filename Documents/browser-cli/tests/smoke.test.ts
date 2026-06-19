import { describe, expect, it } from 'vitest';
import { extractDocumentFromHtml } from '../src/extraction/extractor.js';
import { getStoragePaths } from '../src/persistence/paths.js';

describe('browser-cli foundation', () => {
  it('bootstraps storage paths', () => {
    const paths = getStoragePaths();
    expect(paths.root.endsWith('.browser-cli')).toBe(true);
    expect(paths.dbPath.endsWith('index.db')).toBe(true);
  });

  it('extracts readable content', () => {
    const result = extractDocumentFromHtml(
      '<html><head><title>Test</title></head><body><main><h1>Hello</h1><p>World</p></main></body></html>',
      'https://example.com'
    );
    expect(result.title).toContain('Test');
    expect(result.markdown).toContain('Hello');
    expect(result.text).toContain('World');
  });
});
