'use client';

import type { JsonValue, VisusWidget, VisusWidgetType } from '@/types/suites';

export function parseCommaSeparatedList(value: string): string[] {
  return value
    .split(',')
    .map((item) => item.trim())
    .filter(Boolean);
}

export function formatCommaSeparatedList(values: string[] | null | undefined): string {
  return (values ?? []).join(', ');
}

export function formatJsonInput(value: unknown): string {
  try {
    return JSON.stringify(value ?? {}, null, 2);
  } catch {
    return '{}';
  }
}

export function parseJsonInput(value: string): Record<string, JsonValue> {
  const trimmed = value.trim();
  if (!trimmed) {
    return {};
  }

  const parsed = JSON.parse(trimmed) as unknown;
  if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) {
    throw new Error('JSON configuration must be an object.');
  }

  return parsed as Record<string, JsonValue>;
}

export function compactWidgetPositions(widgets: VisusWidget[]): Array<{
  widget_id: string;
  x: number;
  y: number;
  w: number;
  h: number;
}> {
  const sorted = [...widgets].sort((left, right) => {
    if (left.position.y === right.position.y) {
      return left.position.x - right.position.x;
    }
    return left.position.y - right.position.y;
  });

  let x = 0;
  let y = 0;
  let rowHeight = 0;

  return sorted.map((widget) => {
    const width = Math.max(1, Math.min(12, widget.position.w));
    const height = Math.max(1, Math.min(8, widget.position.h));

    if (x + width > 12) {
      x = 0;
      y += Math.max(1, rowHeight);
      rowHeight = 0;
    }

    const next = {
      widget_id: widget.id,
      x,
      y,
      w: width,
      h: height,
    };

    x += width;
    rowHeight = Math.max(rowHeight, height);
    return next;
  });
}

export function widgetSupportsKpi(type: VisusWidgetType): boolean {
  return ['kpi_card', 'gauge', 'sparkline', 'trend_indicator'].includes(type);
}
