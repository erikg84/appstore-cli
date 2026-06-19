import type { Page } from 'playwright';
import type { BrowserRuntime } from './runtime.js';
import type {
  ClickResult,
  DomQueryResult,
  ElementInfo,
  EvalResult,
  FillResult,
  ScrollResult,
  SnapshotResult
} from '../types/index.js';

/**
 * Interaction layer. Drives the SAME persistent (possibly logged-in) Playwright
 * context the rest of the tool uses. Because each CLI invocation is a fresh
 * process, there is no live page to reuse across calls; instead we reuse the
 * on-disk persistent profile (cookies/localStorage => session stays auth'd) and
 * navigate to a caller-supplied URL or the last-known URL.
 */

const KEY_ATTRIBUTES = ['id', 'name', 'type', 'role', 'aria-label', 'href', 'value', 'placeholder', 'title', 'alt'];

async function navigateOrReuse(page: Page, url: string | undefined): Promise<void> {
  if (url) {
    await page.goto(url, { waitUntil: 'domcontentloaded' });
    await page.waitForLoadState('networkidle', { timeout: 8000 }).catch(() => undefined);
    return;
  }
  if (!page.url() || page.url() === 'about:blank') {
    throw new Error('No current page URL is available. Pass --url to navigate first.');
  }
}

// Serialized into the page context. Returns ElementInfo[] for matched nodes.
const SERIALIZE_FN = `(selector, all, includeHidden) => {
  const KEY_ATTRIBUTES = ${JSON.stringify(KEY_ATTRIBUTES)};
  const nodes = all ? Array.from(document.querySelectorAll(selector)) : (() => {
    const one = document.querySelector(selector);
    return one ? [one] : [];
  })();
  const describe = (el, index) => {
    const rect = el.getBoundingClientRect();
    const style = window.getComputedStyle(el);
    const visible = !!(rect.width || rect.height) && style.visibility !== 'hidden' && style.display !== 'none' && Number(style.opacity) !== 0;
    const attributes = {};
    for (const name of KEY_ATTRIBUTES) {
      const v = el.getAttribute(name);
      if (v != null) attributes[name] = v;
      else if (name === 'value' && 'value' in el && el.value != null && el.value !== '') attributes.value = String(el.value);
    }
    return {
      index,
      tag: el.tagName.toLowerCase(),
      text: (el.textContent || '').replace(/\\s+/g, ' ').trim().slice(0, 400),
      attributes,
      boundingBox: (rect.width || rect.height) ? { x: rect.x, y: rect.y, width: rect.width, height: rect.height } : null,
      visible
    };
  };
  return nodes.map(describe).filter((info) => includeHidden || info.visible);
}`;

export async function evalInPage(
  runtime: BrowserRuntime,
  js: string,
  url: string | undefined
): Promise<EvalResult> {
  const page = await runtime.newPage();
  try {
    await navigateOrReuse(page, url);
    // Wrap so both expressions ("document.title") and statements work.
    const wrapped = `(async () => { return (${js}); })()`;
    let result: unknown;
    try {
      result = await page.evaluate(wrapped);
    } catch {
      // Fall back to treating the input as a function body (statements).
      result = await page.evaluate(`(async () => { ${js} })()`);
    }
    return { url: page.url(), result };
  } finally {
    await page.close();
    await runtime.close();
  }
}

export async function domQuery(
  runtime: BrowserRuntime,
  selector: string,
  url: string | undefined,
  all: boolean,
  includeHidden: boolean
): Promise<DomQueryResult> {
  const page = await runtime.newPage();
  try {
    await navigateOrReuse(page, url);
    const elements = (await page.evaluate(
      `(${SERIALIZE_FN})(${JSON.stringify(selector)}, ${all}, ${includeHidden})`
    )) as ElementInfo[];
    return { url: page.url(), selector, count: elements.length, elements };
  } finally {
    await page.close();
    await runtime.close();
  }
}

export interface ClickTarget {
  css?: string;
  text?: string;
  role?: string;
}

export async function clickElement(
  runtime: BrowserRuntime,
  target: ClickTarget,
  url: string | undefined
): Promise<ClickResult> {
  const page = await runtime.newPage();
  try {
    await navigateOrReuse(page, url);
    const before = page.url();
    let locator;
    let matchedBy: 'css' | 'text' | 'role';
    let selector: string;
    if (target.css) {
      matchedBy = 'css';
      selector = target.css;
      locator = page.locator(target.css).first();
    } else if (target.role) {
      matchedBy = 'role';
      const [role, ...nameParts] = target.role.split(/\s+/);
      const name = nameParts.join(' ');
      selector = `role=${role}${name ? `[name="${name}"]` : ''}`;
      locator = name
        ? page.getByRole(role as Parameters<Page['getByRole']>[0], { name })
        : page.getByRole(role as Parameters<Page['getByRole']>[0]);
      locator = locator.first();
    } else if (target.text) {
      matchedBy = 'text';
      selector = `text=${target.text}`;
      locator = page.getByText(target.text, { exact: false }).first();
    } else {
      throw new Error('click requires one of: <css>, --text, or --role.');
    }

    await locator.scrollIntoViewIfNeeded({ timeout: 15000 });
    const element = (await locator.evaluate(`(el) => {
      const KEY_ATTRIBUTES = ${JSON.stringify(KEY_ATTRIBUTES)};
      const rect = el.getBoundingClientRect();
      const style = window.getComputedStyle(el);
      const visible = !!(rect.width || rect.height) && style.visibility !== 'hidden' && style.display !== 'none' && Number(style.opacity) !== 0;
      const attributes = {};
      for (const name of KEY_ATTRIBUTES) {
        const v = el.getAttribute(name);
        if (v != null) attributes[name] = v;
      }
      return {
        index: 0,
        tag: el.tagName.toLowerCase(),
        text: (el.textContent || '').replace(/\\s+/g, ' ').trim().slice(0, 400),
        attributes,
        boundingBox: (rect.width || rect.height) ? { x: rect.x, y: rect.y, width: rect.width, height: rect.height } : null,
        visible
      };
    }`).catch(() => null)) as ElementInfo | null;
    await locator.click({ timeout: 15000 });
    await page.waitForLoadState('domcontentloaded', { timeout: 8000 }).catch(() => undefined);
    const after = page.url();
    return {
      url: before,
      matchedBy,
      selector,
      clicked: true,
      newUrl: after,
      navigated: after !== before,
      element
    };
  } finally {
    await page.close();
    await runtime.close();
  }
}

export async function fillElement(
  runtime: BrowserRuntime,
  selector: string,
  value: string,
  mode: 'fill' | 'type',
  url: string | undefined
): Promise<FillResult> {
  const page = await runtime.newPage();
  try {
    await navigateOrReuse(page, url);
    const locator = page.locator(selector).first();
    await locator.scrollIntoViewIfNeeded({ timeout: 15000 });
    if (mode === 'fill') {
      await locator.fill(value, { timeout: 15000 });
    } else {
      await locator.click({ timeout: 15000 });
      await locator.pressSequentially(value, { delay: 15, timeout: 15000 });
    }
    return { url: page.url(), selector, value, mode };
  } finally {
    await page.close();
    await runtime.close();
  }
}

export interface ScrollTarget {
  to?: string;
  by?: number;
  bottom?: boolean;
  top?: boolean;
}

export async function scrollPage(
  runtime: BrowserRuntime,
  target: ScrollTarget,
  url: string | undefined
): Promise<ScrollResult> {
  const page = await runtime.newPage();
  try {
    await navigateOrReuse(page, url);
    let mode: ScrollResult['mode'];
    if (target.to) {
      mode = 'to';
      await page.locator(target.to).first().scrollIntoViewIfNeeded({ timeout: 15000 });
    } else if (target.bottom) {
      mode = 'bottom';
      await page.evaluate('window.scrollTo(0, document.body.scrollHeight)');
    } else if (target.top) {
      mode = 'top';
      await page.evaluate('window.scrollTo(0, 0)');
    } else if (typeof target.by === 'number') {
      mode = 'by';
      await page.evaluate(`window.scrollBy(0, ${target.by})`);
    } else {
      throw new Error('scroll requires one of: --to <css>, --by <px>, --bottom, --top.');
    }
    await page.waitForTimeout(200);
    const pos = (await page.evaluate('({ scrollX: window.scrollX, scrollY: window.scrollY })')) as {
      scrollX: number;
      scrollY: number;
    };
    return { url: page.url(), mode, scrollX: pos.scrollX, scrollY: pos.scrollY };
  } finally {
    await page.close();
    await runtime.close();
  }
}

export async function snapshotPage(
  runtime: BrowserRuntime,
  url: string | undefined,
  includeHidden: boolean
): Promise<SnapshotResult> {
  const page = await runtime.newPage();
  try {
    await navigateOrReuse(page, url);
    const interactiveSelector = 'a,button,input,select,textarea,[role],[onclick],[contenteditable],summary,label';
    const elements = (await page.evaluate(
      `(${SERIALIZE_FN})(${JSON.stringify(interactiveSelector)}, true, ${includeHidden})`
    )) as ElementInfo[];
    // file inputs + common custom upload widgets are surfaced even when hidden.
    const fileInputs = (await page.evaluate(
      `(${SERIALIZE_FN})('input[type=file],[data-upload],[class*=upload],[class*=Upload]', true, true)`
    )) as ElementInfo[];
    const title = await page.title();
    return { url: page.url(), title, includeHidden, elements, fileInputs };
  } finally {
    await page.close();
    await runtime.close();
  }
}
