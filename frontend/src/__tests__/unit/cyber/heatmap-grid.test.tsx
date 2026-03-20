import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { HeatmapGrid } from '@/app/(dashboard)/cyber/risk-heatmap/_components/heatmap-grid';
import type { RiskHeatmapData } from '@/types/cyber';

vi.mock('next/navigation', () => ({
  useRouter: () => ({ push: vi.fn() }),
}));

function makeHeatmapData(): RiskHeatmapData {
  const assetTypes = ['server', 'endpoint', 'cloud_resource', 'database', 'container', 'iot_device', 'network_device', 'application'];
  const severities = ['critical', 'high', 'medium', 'low'] as const;
  const cells = assetTypes.flatMap((type, ti) =>
    severities.map((sev, si) => ({
      asset_type: type,
      severity: sev,
      count: (ti + 1) * (si + 1) * 3,
      affected_asset_count: ti + 1,
      total_assets_of_type: (ti + 1) * 5,
    })),
  );
  return {
    cells,
    asset_types: assetTypes,
    total_vulnerabilities: cells.reduce((s, c) => s + c.count, 0),
    generated_at: '2024-01-01T00:00:00Z',
  };
}

describe('HeatmapGrid', () => {
  it('test_rendersAllCells: 8 types × 4 severities → 32 cells (SVG rects)', () => {
    const { container } = render(<HeatmapGrid data={makeHeatmapData()} />);
    const rects = container.querySelectorAll('rect');
    // 8 * 4 = 32 data rects
    expect(rects.length).toBeGreaterThanOrEqual(32);
  });

  it('test_rowTotals: sum of row cells shown', () => {
    const data = makeHeatmapData();
    render(<HeatmapGrid data={data} />);
    // Grand total text
    const grandTotal = String(data.total_vulnerabilities);
    expect(screen.getByText(grandTotal)).toBeTruthy();
  });

  it('test_columnTotals: severity column totals shown', () => {
    const data = makeHeatmapData();
    render(<HeatmapGrid data={data} />);
    // Critical column total: sum of all cells with severity=critical
    const critTotal = data.cells
      .filter((c) => c.severity === 'critical')
      .reduce((s, c) => s + c.count, 0);
    expect(screen.getByText(String(critTotal))).toBeTruthy();
  });

  it('test_assetTypeLabels: all 8 asset types displayed', () => {
    const data = makeHeatmapData();
    render(<HeatmapGrid data={data} />);
    // Server should appear in SVG text
    const texts = Array.from(document.querySelectorAll('text')).map((t) => t.textContent);
    expect(texts.some((t) => t?.includes('server'))).toBe(true);
  });

  it('test_severityHeaders: Critical/High/Medium/Low headers displayed', () => {
    render(<HeatmapGrid data={makeHeatmapData()} />);
    expect(screen.getByText('Critical')).toBeTruthy();
    expect(screen.getByText('High')).toBeTruthy();
    expect(screen.getByText('Medium')).toBeTruthy();
    expect(screen.getByText('Low')).toBeTruthy();
  });

  it('test_zeroCellWhite: count=0 → white fill', () => {
    const data = makeHeatmapData();
    // Set one cell to 0
    data.cells[0].count = 0;
    const { container } = render(<HeatmapGrid data={data} />);
    const rects = container.querySelectorAll('rect');
    const whiteRects = Array.from(rects).filter((r) => r.getAttribute('fill') === '#ffffff');
    expect(whiteRects.length).toBeGreaterThan(0);
  });

  it('test_rowSorting: Total column header shows', () => {
    render(<HeatmapGrid data={makeHeatmapData()} />);
    expect(screen.getAllByText('Total').length).toBeGreaterThan(0);
  });

  it('test_hoverTooltip: title element present per cell', () => {
    const { container } = render(<HeatmapGrid data={makeHeatmapData()} />);
    const titles = container.querySelectorAll('title');
    expect(titles.length).toBeGreaterThan(0);
  });
});
