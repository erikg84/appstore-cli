import TurndownService from 'turndown';
import { JSDOM } from 'jsdom';
import { Readability } from '@mozilla/readability';

export interface ExtractedDocument {
  title: string;
  markdown: string;
  text: string;
}

export function extractDocumentFromHtml(html: string, url: string): ExtractedDocument {
  const dom = new JSDOM(html, { url });
  const reader = new Readability(dom.window.document);
  const article = reader.parse();
  const turndown = new TurndownService();

  if (!article) {
    const bodyHtml = dom.window.document.body?.innerHTML ?? html;
    const markdown = turndown.turndown(bodyHtml);
    const text = dom.window.document.body?.textContent?.replace(/\s+/g, ' ').trim() ?? '';
    const title = dom.window.document.title || url;
    return { title, markdown, text };
  }

  const markdown = turndown.turndown(article.content ?? '');
  const text = (article.textContent ?? '').replace(/\s+/g, ' ').trim();
  return { title: article.title || url, markdown, text };
}
