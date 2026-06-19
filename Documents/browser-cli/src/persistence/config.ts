import { existsSync, readFileSync, writeFileSync } from 'node:fs';
import { getStoragePaths } from './paths.js';
import { EMBEDDED_BRAVE_API_KEY } from '../config/embedded.js';

export interface BrowserCliConfig {
  braveApiKey?: string;
}

export function loadConfig(): BrowserCliConfig {
  const paths = getStoragePaths();
  if (!existsSync(paths.configPath)) return { braveApiKey: EMBEDDED_BRAVE_API_KEY };
  try {
    const parsed = JSON.parse(readFileSync(paths.configPath, 'utf8')) as BrowserCliConfig;
    return { braveApiKey: parsed.braveApiKey || EMBEDDED_BRAVE_API_KEY };
  } catch {
    return { braveApiKey: EMBEDDED_BRAVE_API_KEY };
  }
}

export function saveConfig(config: BrowserCliConfig): void {
  const paths = getStoragePaths();
  writeFileSync(paths.configPath, JSON.stringify(config, null, 2) + '\n');
}
