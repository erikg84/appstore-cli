import type { CliErrorInfo } from '../types/index.js';

export function toCliError(error: unknown): CliErrorInfo {
  const message = error instanceof Error ? error.message : String(error);
  const lower = message.toLowerCase();

  if (lower.includes('brave api key')) {
    return {
      code: 'BRAVE_AUTH_MISSING',
      message,
      suggestion: 'Use --engine bing or configure a Brave API key in src/config/embedded.ts or ~/.browser-cli/config.json.'
    };
  }

  if (lower.includes('cloudflare') || lower.includes('security verification') || lower.includes('incorrect device time')) {
    return {
      code: 'SITE_VERIFICATION_BLOCK',
      message,
      suggestion: 'Retry with a different result, prefer GitHub/docs domains, or use a site-constrained query.'
    };
  }

  if (lower.includes('duckduckgo blocked')) {
    return {
      code: 'SEARCH_ENGINE_BLOCKED',
      message,
      suggestion: 'Retry with --engine brave or --engine bing.'
    };
  }

  if (lower.includes('browsertype.launchpersistentcontext') || lower.includes('playwright')) {
    return {
      code: 'BROWSER_RUNTIME_ERROR',
      message,
      suggestion: 'Run `npx playwright install chromium` in the repository and retry.'
    };
  }

  if (lower.includes('timed out') || lower.includes('timeout')) {
    return {
      code: 'TIMEOUT',
      message,
      suggestion: 'Retry with --force, narrow the query, or target a specific domain.'
    };
  }

  if (lower.includes('fetch failed') || lower.includes('getaddrinfo') || lower.includes('enotfound')) {
    return {
      code: 'NETWORK_ERROR',
      message,
      suggestion: 'Check connectivity or retry with a different engine.'
    };
  }

  return {
    code: 'RUNTIME_ERROR',
    message,
    suggestion: 'Inspect the command meta/errors and retry with a narrower query or different engine.'
  };
}
