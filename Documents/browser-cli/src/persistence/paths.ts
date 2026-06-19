import { mkdirSync } from 'node:fs';
import { homedir } from 'node:os';
import path from 'node:path';

export interface StoragePaths {
  root: string;
  profileDir: string;
  cacheRawDir: string;
  cacheRenderedDir: string;
  cacheMediaDir: string;
  logsDir: string;
  dbPath: string;
  configPath: string;
}

export function getStoragePaths(): StoragePaths {
  const root = path.join(homedir(), '.browser-cli');
  const paths: StoragePaths = {
    root,
    profileDir: path.join(root, 'profile'),
    cacheRawDir: path.join(root, 'cache', 'raw'),
    cacheRenderedDir: path.join(root, 'cache', 'rendered'),
    cacheMediaDir: path.join(root, 'cache', 'media'),
    logsDir: path.join(root, 'logs'),
    dbPath: path.join(root, 'index.db'),
    configPath: path.join(root, 'config.json')
  };

  for (const value of Object.values(paths)) {
    if (value.endsWith('.db') || value.endsWith('.json')) continue;
    mkdirSync(value, { recursive: true });
  }

  return paths;
}
