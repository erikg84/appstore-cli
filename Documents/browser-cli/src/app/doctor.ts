import { existsSync, readdirSync, statSync } from 'node:fs';
import Database from 'better-sqlite3';
import { chromium } from 'playwright';
import type { BrowserCliConfig } from '../persistence/config.js';
import { getStoragePaths } from '../persistence/paths.js';

export interface DoctorCheck {
  name: string;
  ok: boolean;
  details: string;
  suggestion?: string;
}

export interface DoctorResult {
  summary: {
    ok: boolean;
    healthyChecks: number;
    failedChecks: number;
  };
  environment: {
    nodeVersion: string;
    platform: string;
    cwd: string;
  };
  config: {
    braveConfigured: boolean;
    configPath: string;
    storageRoot: string;
    profileDir: string;
    dbPath: string;
  };
  checks: DoctorCheck[];
}

export function runDoctor(config: BrowserCliConfig): DoctorResult {
  const paths = getStoragePaths();
  const checks: DoctorCheck[] = [];

  checks.push(checkDirectory('storage_root', paths.root, 'Run any browser-cli command to bootstrap storage.'));
  checks.push(checkDirectory('profile_dir', paths.profileDir, 'Run `browser-cli open https://example.com` to initialize the persistent profile.'));
  checks.push(checkDirectory('raw_cache_dir', paths.cacheRawDir, 'Use `browser-cli open <url>` to populate raw HTML cache.'));
  checks.push(checkDirectory('rendered_cache_dir', paths.cacheRenderedDir, 'Use `browser-cli read <url>` to populate extracted content cache.'));
  checks.push(checkSqlite(paths.dbPath));
  checks.push(checkPlaywrightExecutable());
  checks.push(checkBraveConfig(config));
  checks.push(checkCacheFootprint(paths));

  const failedChecks = checks.filter((check) => !check.ok).length;
  return {
    summary: {
      ok: failedChecks === 0,
      healthyChecks: checks.length - failedChecks,
      failedChecks
    },
    environment: {
      nodeVersion: process.version,
      platform: process.platform,
      cwd: process.cwd()
    },
    config: {
      braveConfigured: Boolean(config.braveApiKey),
      configPath: paths.configPath,
      storageRoot: paths.root,
      profileDir: paths.profileDir,
      dbPath: paths.dbPath
    },
    checks
  };
}

function checkDirectory(name: string, dir: string, suggestion?: string): DoctorCheck {
  if (!existsSync(dir)) {
    return { name, ok: false, details: `Missing directory: ${dir}`, suggestion };
  }
  return { name, ok: true, details: `Directory exists: ${dir}` };
}

function checkSqlite(dbPath: string): DoctorCheck {
  try {
    const db = new Database(dbPath);
    db.prepare('SELECT 1').get();
    db.close();
    return { name: 'sqlite_index', ok: true, details: `SQLite index is readable: ${dbPath}` };
  } catch (error) {
    const message = error instanceof Error ? error.message : String(error);
    return {
      name: 'sqlite_index',
      ok: false,
      details: `SQLite index check failed: ${message}`,
      suggestion: 'Run `browser-cli cache clear` if the index is corrupted, then retry.'
    };
  }
}

function checkPlaywrightExecutable(): DoctorCheck {
  try {
    const executable = chromium.executablePath();
    if (!existsSync(executable)) {
      return {
        name: 'playwright_chromium',
        ok: false,
        details: `Chromium executable is missing: ${executable}`,
        suggestion: 'Run `npx playwright install chromium` in the repository.'
      };
    }
    return { name: 'playwright_chromium', ok: true, details: `Chromium executable found: ${executable}` };
  } catch (error) {
    const message = error instanceof Error ? error.message : String(error);
    return {
      name: 'playwright_chromium',
      ok: false,
      details: `Playwright runtime check failed: ${message}`,
      suggestion: 'Reinstall Playwright dependencies and run `npx playwright install chromium`.'
    };
  }
}

function checkBraveConfig(config: BrowserCliConfig): DoctorCheck {
  if (!config.braveApiKey) {
    return {
      name: 'brave_config',
      ok: false,
      details: 'Brave Search API key is not configured.',
      suggestion: 'Set a key in src/config/embedded.ts or ~/.browser-cli/config.json, or use --engine bing.'
    };
  }
  return { name: 'brave_config', ok: true, details: 'Brave Search API key is configured.' };
}

function checkCacheFootprint(paths: ReturnType<typeof getStoragePaths>): DoctorCheck {
  try {
    const raw = countEntries(paths.cacheRawDir);
    const rendered = countEntries(paths.cacheRenderedDir);
    return {
      name: 'cache_footprint',
      ok: true,
      details: `Cache contains ${raw} raw artifacts and ${rendered} rendered artifacts.`
    };
  } catch (error) {
    const message = error instanceof Error ? error.message : String(error);
    return {
      name: 'cache_footprint',
      ok: false,
      details: `Cache inspection failed: ${message}`,
      suggestion: 'Check filesystem permissions for ~/.browser-cli.'
    };
  }
}

function countEntries(dir: string): number {
  if (!existsSync(dir)) return 0;
  return readdirSync(dir).filter((name) => {
    try {
      return statSync(`${dir}/${name}`).isFile();
    } catch {
      return false;
    }
  }).length;
}
