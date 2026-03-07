import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import { RelationshipGraph } from '@/app/(dashboard)/cyber/assets/[id]/_components/relationship-graph';
import type { AssetRelationship } from '@/types/cyber';

// Mock D3 dynamic import - use a proxy that handles any chained call
vi.mock('d3', async () => {
  const chain = (): Record<string, unknown> => new Proxy({}, {
    get(_: unknown, prop: string) {
      if (prop === 'then') return undefined; // prevent being treated as a Promise
      return () => chain();
    },
  }) as Record<string, unknown>;

  return {
    select: chain,
    forceSimulation: chain,
    forceManyBody: chain,
    forceLink: chain,
    forceCenter: chain,
    forceCollide: chain,
    zoom: chain,
    drag: chain,
    zoomIdentity: {},
  };
});

const mockRelationships: AssetRelationship[] = [
  {
    id: 'rel-1',
    source_asset_id: 'asset-1',
    source_asset_name: 'Web Server',
    source_asset_type: 'server',
    source_criticality: 'high',
    target_asset_id: 'asset-2',
    target_asset_name: 'Database',
    target_asset_type: 'database',
    target_criticality: 'critical',
    relationship_type: 'connects_to',
    created_at: '2024-01-01T00:00:00Z',
  },
];

describe('RelationshipGraph', () => {
  it('renders SVG element', () => {
    const { container } = render(
      <RelationshipGraph
        assetId="asset-1"
        assetName="Web Server"
        assetType="server"
        relationships={mockRelationships}
      />
    );
    expect(container.querySelector('svg')).toBeInTheDocument();
  });

  it('renders legend entries for asset types', () => {
    render(
      <RelationshipGraph
        assetId="asset-1"
        assetName="Web Server"
        assetType="server"
        relationships={[]}
      />
    );
    expect(screen.getByText('server')).toBeInTheDocument();
    expect(screen.getByText('database')).toBeInTheDocument();
  });

  it('renders with custom dimensions', () => {
    const { container } = render(
      <RelationshipGraph
        assetId="asset-1"
        assetName="Web Server"
        assetType="server"
        relationships={mockRelationships}
        width={800}
        height={500}
      />
    );
    const svg = container.querySelector('svg');
    expect(svg).toHaveAttribute('width', '800');
    expect(svg).toHaveAttribute('height', '500');
  });
});
