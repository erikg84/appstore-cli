import type { CliErrorInfo, OutputEnvelope, OutputFormat } from '../types/index.js';

export function formatOutput<T>(format: OutputFormat, envelope: OutputEnvelope<T>): string {
  if (format === 'json') return JSON.stringify(envelope, null, 2);
  if (format === 'markdown') return toMarkdown(envelope);
  return toText(envelope);
}

function toMarkdown<T>(envelope: OutputEnvelope<T>): string {
  const lines = [
    '# browser-cli',
    '',
    `- command: ${envelope.command}`,
    `- ok: ${envelope.ok}`
  ];
  if (Object.keys(envelope.meta).length > 0) {
    lines.push('- meta:');
    for (const [key, value] of Object.entries(envelope.meta)) {
      lines.push(`  - ${key}: ${stringifyInline(value)}`);
    }
  }
  if (envelope.warnings.length > 0) {
    lines.push('- warnings:');
    for (const warning of envelope.warnings) lines.push(`  - ${warning}`);
  }
  if (envelope.errors.length > 0) {
    lines.push('- errors:');
    for (const error of envelope.errors) {
      lines.push(`  - [${error.code}] ${error.message}`);
      if (error.suggestion) lines.push(`    - suggestion: ${error.suggestion}`);
    }
  }
  lines.push('', '```json', JSON.stringify(envelope.data, null, 2), '```');
  return lines.join('\n');
}

function toText<T>(envelope: OutputEnvelope<T>): string {
  const lines = [`command=${envelope.command}`, `ok=${envelope.ok}`];
  if (Object.keys(envelope.meta).length > 0) {
    lines.push('meta:');
    for (const [key, value] of Object.entries(envelope.meta)) {
      lines.push(`  ${key}: ${stringifyInline(value)}`);
    }
  }
  if (envelope.warnings.length > 0) {
    lines.push('warnings:');
    for (const warning of envelope.warnings) lines.push(`  - ${warning}`);
  }
  if (envelope.errors.length > 0) {
    lines.push('errors:');
    for (const error of envelope.errors) {
      lines.push(`  - [${error.code}] ${error.message}`);
      if (error.suggestion) lines.push(`    suggestion: ${error.suggestion}`);
    }
  }
  lines.push('data:', JSON.stringify(envelope.data, null, 2));
  return lines.join('\n');
}

function stringifyInline(value: unknown): string {
  if (typeof value === 'string') return value;
  return JSON.stringify(value);
}

export function envelope<T>(
  command: string,
  data: T,
  meta: Record<string, unknown> = {},
  warnings: string[] = [],
  errors: CliErrorInfo[] = []
): OutputEnvelope<T> {
  return {
    ok: errors.length === 0,
    command,
    timestamp: new Date().toISOString(),
    data,
    meta,
    warnings,
    errors
  };
}
