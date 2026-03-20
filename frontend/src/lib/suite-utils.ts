export function arrayLength(value: unknown): number {
  return Array.isArray(value) ? value.length : 0;
}

export function objectKeyCount(value: unknown): number {
  if (!value || typeof value !== 'object' || Array.isArray(value)) {
    return 0;
  }
  return Object.keys(value as Record<string, unknown>).length;
}

export function isEmptyObject(value: unknown): boolean {
  return objectKeyCount(value) === 0;
}

export function shortId(value?: string | null): string {
  return value ? `${value.slice(0, 8)}…` : '—';
}

export function percent(value: number, digits = 0): string {
  return `${value.toFixed(digits)}%`;
}

export function summarizeNamedRecords(value: unknown, maxItems = 2): string {
  if (!Array.isArray(value) || value.length === 0) {
    return '—';
  }

  const names = value
    .map((entry) => extractRecordLabel(entry))
    .filter((entry): entry is string => Boolean(entry));

  if (names.length === 0) {
    return `${value.length} record${value.length === 1 ? '' : 's'}`;
  }

  const visible = names.slice(0, maxItems);
  const remainder = names.length - visible.length;
  return remainder > 0 ? `${visible.join(', ')} +${remainder}` : visible.join(', ');
}

export function extractRecordLabel(value: unknown): string | null {
  if (typeof value === 'string') {
    return value;
  }
  if (!value || typeof value !== 'object' || Array.isArray(value)) {
    return null;
  }

  const record = value as Record<string, unknown>;
  const candidates = [
    'name',
    'full_name',
    'fullName',
    'title',
    'email',
    'label',
    'id',
  ];

  for (const key of candidates) {
    const candidate = record[key];
    if (typeof candidate === 'string' && candidate.trim() !== '') {
      return candidate.trim();
    }
  }

  return null;
}
