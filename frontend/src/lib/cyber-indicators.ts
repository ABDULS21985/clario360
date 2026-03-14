import type {
  IndicatorSource,
  IndicatorType,
  ThreatFeedAuthType,
  ThreatFeedConfig,
  ThreatFeedInterval,
  ThreatFeedType,
  ThreatIndicator,
  ThreatSeverity,
} from '@/types/cyber';
import { downloadTextFile, slugToTitle } from '@/lib/utils';

export interface IndicatorPreview {
  type: IndicatorType;
  value: string;
  severity?: ThreatSeverity;
  source?: IndicatorSource;
  description?: string;
  tags?: string[];
}

export interface CsvPreview {
  headers: string[];
  rows: Record<string, string>[];
}

export const INDICATOR_SOURCE_OPTIONS: Array<{ label: string; value: IndicatorSource }> = [
  { label: 'Manual', value: 'manual' },
  { label: 'STIX Feed', value: 'stix_feed' },
  { label: 'OSINT', value: 'osint' },
  { label: 'Internal', value: 'internal' },
  { label: 'Vendor', value: 'vendor' },
];

export const THREAT_FEED_TYPE_OPTIONS: Array<{ label: string; value: ThreatFeedType }> = [
  { label: 'STIX Bundle', value: 'stix' },
  { label: 'TAXII 2.1', value: 'taxii' },
  { label: 'MISP', value: 'misp' },
  { label: 'CSV URL', value: 'csv_url' },
  { label: 'Manual', value: 'manual' },
];

export const THREAT_FEED_AUTH_OPTIONS: Array<{ label: string; value: ThreatFeedAuthType }> = [
  { label: 'None', value: 'none' },
  { label: 'API Key', value: 'api_key' },
  { label: 'Basic Auth', value: 'basic' },
  { label: 'Certificate', value: 'certificate' },
];

export const THREAT_FEED_INTERVAL_OPTIONS: Array<{ label: string; value: ThreatFeedInterval }> = [
  { label: 'Hourly', value: 'hourly' },
  { label: 'Every 6 Hours', value: 'every_6h' },
  { label: 'Daily', value: 'daily' },
  { label: 'Weekly', value: 'weekly' },
  { label: 'Manual', value: 'manual' },
];

export const INDICATOR_TYPE_BADGE_CLASSES: Record<IndicatorType, string> = {
  ip: 'bg-blue-100 text-blue-700',
  domain: 'bg-violet-100 text-violet-700',
  url: 'bg-red-100 text-red-700',
  email: 'bg-yellow-100 text-yellow-800',
  file_hash_md5: 'bg-orange-100 text-orange-700',
  file_hash_sha1: 'bg-orange-100 text-orange-700',
  file_hash_sha256: 'bg-orange-100 text-orange-700',
  certificate: 'bg-teal-100 text-teal-700',
  registry_key: 'bg-slate-200 text-slate-700',
  user_agent: 'bg-pink-100 text-pink-700',
  cidr: 'bg-indigo-100 text-indigo-700',
};

const domainPattern = /^(?=.{1,253}$)(?!-)(?:[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z]{2,63}$/i;
const emailPattern = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
const md5Pattern = /^[a-f0-9]{32}$/i;
const sha1Pattern = /^[a-f0-9]{40}$/i;
const sha256Pattern = /^[a-f0-9]{64}$/i;

export function getIndicatorSourceLabel(value: string): string {
  return INDICATOR_SOURCE_OPTIONS.find((option) => option.value === value)?.label ?? slugToTitle(value);
}

export function getThreatFeedTypeLabel(value: string): string {
  return THREAT_FEED_TYPE_OPTIONS.find((option) => option.value === value)?.label ?? slugToTitle(value);
}

export function getThreatFeedIntervalLabel(value: string): string {
  return THREAT_FEED_INTERVAL_OPTIONS.find((option) => option.value === value)?.label ?? slugToTitle(value);
}

export function detectIndicatorType(rawValue: string): IndicatorType | null {
  const value = rawValue.trim();
  if (!value) {
    return null;
  }
  if (value.includes('/') && isCIDR(value)) {
    return 'cidr';
  }
  if (isIP(value)) {
    return 'ip';
  }
  if (isURL(value)) {
    return 'url';
  }
  if (emailPattern.test(value)) {
    return 'email';
  }
  if (sha256Pattern.test(value)) {
    return 'file_hash_sha256';
  }
  if (sha1Pattern.test(value)) {
    return 'file_hash_sha1';
  }
  if (md5Pattern.test(value)) {
    return 'file_hash_md5';
  }
  if (domainPattern.test(value)) {
    return 'domain';
  }
  return null;
}

export function validateIndicatorValue(type: IndicatorType, rawValue: string): string | null {
  const value = rawValue.trim();
  if (!value) {
    return 'Value is required';
  }

  switch (type) {
    case 'ip':
      return isIP(value) ? null : 'Enter a valid IPv4 or IPv6 address';
    case 'domain':
      return domainPattern.test(value) ? null : 'Enter a valid domain';
    case 'url':
      return isURL(value) ? null : 'Enter a valid URL';
    case 'email':
      return emailPattern.test(value) ? null : 'Enter a valid email address';
    case 'cidr':
      return isCIDR(value) ? null : 'Enter a valid CIDR block';
    case 'file_hash_md5':
      return md5Pattern.test(value) ? null : 'Enter a valid MD5 hash';
    case 'file_hash_sha1':
      return sha1Pattern.test(value) ? null : 'Enter a valid SHA1 hash';
    case 'file_hash_sha256':
      return sha256Pattern.test(value) ? null : 'Enter a valid SHA256 hash';
    default:
      return null;
  }
}

export function parseTagsInput(value: string): string[] {
  return value
    .split(',')
    .map((item) => item.trim())
    .filter(Boolean);
}

export function parseCsvText(text: string): CsvPreview {
  const rows = text
    .split(/\r?\n/)
    .map((line) => line.trim())
    .filter(Boolean)
    .map(parseCsvLine);

  if (rows.length === 0) {
    return { headers: [], rows: [] };
  }

  const [headerRow, ...bodyRows] = rows;
  const headers = headerRow.map((item) => item.trim());
  const normalizedRows = bodyRows.map((row) => {
    const entry: Record<string, string> = {};
    headers.forEach((header, index) => {
      entry[header] = row[index] ?? '';
    });
    return entry;
  });

  return {
    headers,
    rows: normalizedRows,
  };
}

export function parseStixPreview(payload: string): IndicatorPreview[] {
  try {
    const parsed = JSON.parse(payload) as { objects?: Array<Record<string, unknown>> };
    const objects = Array.isArray(parsed.objects) ? parsed.objects : [];
    const previews: IndicatorPreview[] = [];

    for (const object of objects) {
      if (object.type !== 'indicator') {
        continue;
      }
      const pattern = typeof object.pattern === 'string' ? object.pattern : '';
      const description = typeof object.description === 'string' ? object.description : undefined;
      const values = extractStixPatternValues(pattern);
      values.forEach((entry) => {
        previews.push({
          type: entry.type,
          value: entry.value,
          description,
          source: 'stix_feed',
        });
      });
    }

    return dedupeIndicatorPreviews(previews);
  } catch {
    return [];
  }
}

export function exportIndicatorsAsCsv(indicators: ThreatIndicator[], filename = 'indicators.csv'): void {
  const header = [
    'type',
    'value',
    'severity',
    'source',
    'confidence',
    'threat_name',
    'active',
    'first_seen_at',
    'last_seen_at',
    'expires_at',
    'tags',
  ];
  const rows = indicators.map((indicator) => [
    indicator.type,
    indicator.value,
    indicator.severity,
    indicator.source,
    String(Math.round(indicator.confidence * 100)),
    indicator.threat_name ?? '',
    String(indicator.active),
    indicator.first_seen_at ?? '',
    indicator.last_seen_at ?? '',
    indicator.expires_at ?? '',
    (indicator.tags ?? []).join('|'),
  ]);

  const content = [header, ...rows]
    .map((row) => row.map(escapeCsvValue).join(','))
    .join('\n');
  downloadTextFile(content, filename);
}

export function exportIndicatorsAsJson(indicators: ThreatIndicator[], filename = 'indicators.json'): void {
  downloadTextFile(JSON.stringify(indicators, null, 2), filename);
}

export function exportIndicatorsAsStix(indicators: ThreatIndicator[], filename = 'indicators.stix.json'): void {
  const bundle = {
    type: 'bundle',
    id: `bundle--${crypto.randomUUID()}`,
    objects: indicators.map((indicator) => ({
      type: 'indicator',
      spec_version: '2.1',
      id: `indicator--${indicator.id}`,
      created: indicator.created_at,
      modified: indicator.updated_at,
      name: `${indicator.type}:${indicator.value}`,
      description: indicator.description,
      pattern_type: 'stix',
      pattern: toStixPattern(indicator.type, indicator.value),
      labels: indicator.tags ?? [],
    })),
  };
  downloadTextFile(JSON.stringify(bundle, null, 2), filename);
}

export function getFeedLastImportedCount(feed: ThreatFeedConfig, history: Array<{ indicators_imported: number }> = []): number | null {
  if (history.length === 0) {
    return null;
  }
  return history[0]?.indicators_imported ?? null;
}

function isIP(value: string): boolean {
  if (typeof window === 'undefined') {
    return /^(\d{1,3}\.){3}\d{1,3}$/.test(value) || /^[0-9a-f:]+$/i.test(value);
  }
  return value.includes(':')
    ? Boolean(value.match(/^[0-9a-f:]+$/i))
    : value.split('.').length === 4 && value.split('.').every((part) => Number(part) >= 0 && Number(part) <= 255);
}

function isCIDR(value: string): boolean {
  const [ip, prefix] = value.split('/');
  const parsedPrefix = Number(prefix);
  if (!isIP(ip) || !Number.isInteger(parsedPrefix)) {
    return false;
  }
  const maxPrefix = ip.includes(':') ? 128 : 32;
  return parsedPrefix >= 0 && parsedPrefix <= maxPrefix;
}

function isURL(value: string): boolean {
  try {
    const parsed = new URL(value);
    return Boolean(parsed.protocol && parsed.hostname);
  } catch {
    return false;
  }
}

function parseCsvLine(line: string): string[] {
  const result: string[] = [];
  let current = '';
  let quoted = false;

  for (let index = 0; index < line.length; index += 1) {
    const char = line[index];
    if (char === '"') {
      if (quoted && line[index + 1] === '"') {
        current += '"';
        index += 1;
      } else {
        quoted = !quoted;
      }
      continue;
    }
    if (char === ',' && !quoted) {
      result.push(current);
      current = '';
      continue;
    }
    current += char;
  }

  result.push(current);
  return result;
}

function extractStixPatternValues(pattern: string): IndicatorPreview[] {
  const mappings: Array<{ matcher: RegExp; type: IndicatorType }> = [
    { matcher: /ipv4-addr:value\s*=\s*'([^']+)'/gi, type: 'ip' },
    { matcher: /ipv6-addr:value\s*=\s*'([^']+)'/gi, type: 'ip' },
    { matcher: /domain-name:value\s*=\s*'([^']+)'/gi, type: 'domain' },
    { matcher: /url:value\s*=\s*'([^']+)'/gi, type: 'url' },
    { matcher: /email-addr:value\s*=\s*'([^']+)'/gi, type: 'email' },
    { matcher: /file:hashes\.'MD5'\s*=\s*'([^']+)'/gi, type: 'file_hash_md5' },
    { matcher: /file:hashes\.'SHA-1'\s*=\s*'([^']+)'/gi, type: 'file_hash_sha1' },
    { matcher: /file:hashes\.'SHA-256'\s*=\s*'([^']+)'/gi, type: 'file_hash_sha256' },
  ];

  const results: IndicatorPreview[] = [];
  for (const mapping of mappings) {
    const matches = pattern.matchAll(mapping.matcher);
    for (const match of matches) {
      if (match[1]) {
        results.push({ type: mapping.type, value: match[1] });
      }
    }
  }
  return results;
}

function dedupeIndicatorPreviews(items: IndicatorPreview[]): IndicatorPreview[] {
  const seen = new Set<string>();
  return items.filter((item) => {
    const key = `${item.type}:${item.value}`;
    if (seen.has(key)) {
      return false;
    }
    seen.add(key);
    return true;
  });
}

function escapeCsvValue(value: string): string {
  if (value.includes(',') || value.includes('"') || value.includes('\n')) {
    return `"${value.replaceAll('"', '""')}"`;
  }
  return value;
}

function toStixPattern(type: IndicatorType, value: string): string {
  switch (type) {
    case 'ip':
      return `[ipv4-addr:value = '${value}']`;
    case 'domain':
      return `[domain-name:value = '${value}']`;
    case 'url':
      return `[url:value = '${value}']`;
    case 'email':
      return `[email-addr:value = '${value}']`;
    case 'file_hash_md5':
      return `[file:hashes.'MD5' = '${value}']`;
    case 'file_hash_sha1':
      return `[file:hashes.'SHA-1' = '${value}']`;
    case 'file_hash_sha256':
      return `[file:hashes.'SHA-256' = '${value}']`;
    default:
      return `[x-clario:value = '${value}']`;
  }
}
