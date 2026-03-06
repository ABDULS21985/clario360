import { clsx, type ClassValue } from 'clsx';
import { twMerge } from 'tailwind-merge';
import { format, formatDistanceToNow, parseISO } from 'date-fns';

export function cn(...inputs: ClassValue[]): string {
  return twMerge(clsx(inputs));
}

export function formatDate(
  date: string | Date | null | undefined,
  fmt = 'MMM d, yyyy',
): string {
  if (!date) return '—';
  try {
    const d = typeof date === 'string' ? parseISO(date) : date;
    return format(d, fmt);
  } catch {
    return '—';
  }
}

export function formatDateTime(date: string | Date | null | undefined): string {
  return formatDate(date, 'MMM d, yyyy HH:mm');
}

export function timeAgo(date: string | Date | null | undefined): string {
  if (!date) return '—';
  try {
    const d = typeof date === 'string' ? parseISO(date) : date;
    return formatDistanceToNow(d, { addSuffix: true });
  } catch {
    return '—';
  }
}

export function truncate(str: string, length: number): string {
  if (str.length <= length) return str;
  return str.slice(0, length) + '…';
}

export function capitalize(str: string): string {
  if (!str) return '';
  return str.charAt(0).toUpperCase() + str.slice(1);
}

export function slugToTitle(slug: string): string {
  return slug
    .split(/[-_]/)
    .map((word) => capitalize(word))
    .join(' ');
}

export async function copyToClipboard(text: string): Promise<boolean> {
  try {
    await navigator.clipboard.writeText(text);
    return true;
  } catch {
    return false;
  }
}

export function downloadTextFile(content: string, filename: string): void {
  const blob = new Blob([content], { type: 'text/plain;charset=utf-8' });
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = filename;
  document.body.appendChild(a);
  a.click();
  document.body.removeChild(a);
  URL.revokeObjectURL(url);
}

/** Promise-based delay — use only for staggered animations, never for fake loading states. */
export function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}
