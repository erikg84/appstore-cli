import { chromium, type BrowserContext, type Page } from 'playwright';
import { getStoragePaths } from '../persistence/paths.js';

export interface BrowserRuntimeOptions {
  headed?: boolean;
}

export class BrowserRuntime {
  private context: BrowserContext | null = null;

  async getContext(options: BrowserRuntimeOptions = {}): Promise<BrowserContext> {
    if (this.context) return this.context;
    const paths = getStoragePaths();
    this.context = await chromium.launchPersistentContext(paths.profileDir, {
      headless: !options.headed,
      viewport: { width: 1440, height: 960 },
      userAgent:
        'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0 Safari/537.36'
    });
    return this.context;
  }

  async newPage(options: BrowserRuntimeOptions = {}): Promise<Page> {
    const context = await this.getContext(options);
    const page = await context.newPage();
    page.setDefaultNavigationTimeout(45000);
    page.setDefaultTimeout(45000);
    return page;
  }

  async close(): Promise<void> {
    if (this.context) {
      await this.context.close();
      this.context = null;
    }
  }
}
