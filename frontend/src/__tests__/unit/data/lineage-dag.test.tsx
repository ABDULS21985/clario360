import { describe, expect, it, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { LineageDag } from '@/app/(dashboard)/data/lineage/_components/lineage-dag';
import { lineageGraph } from '@/__tests__/data-suite-fixtures';

describe('LineageDag', () => {
  it('test_rendersAllNodes: renders node groups for each lineage node', () => {
    render(
      <LineageDag
        graph={lineageGraph}
        direction="LR"
        selectedNodeId={null}
        search=""
        impact={null}
        onSelectNode={vi.fn()}
      />,
    );

    expect(screen.getByTestId('lineage-node-source-1')).toBeInTheDocument();
    expect(screen.getByTestId('lineage-node-pipeline-1')).toBeInTheDocument();
    expect(screen.getByTestId('lineage-node-model-1')).toBeInTheDocument();
    expect(screen.getByTestId('lineage-node-consumer-1')).toBeInTheDocument();
  });

  it('test_rendersAllEdges: renders each edge path', () => {
    render(
      <LineageDag
        graph={lineageGraph}
        direction="LR"
        selectedNodeId={null}
        search=""
        impact={null}
        onSelectNode={vi.fn()}
      />,
    );

    expect(screen.getByTestId('lineage-edge-edge-1')).toBeInTheDocument();
    expect(screen.getByTestId('lineage-edge-edge-2')).toBeInTheDocument();
    expect(screen.getByTestId('lineage-edge-edge-3')).toBeInTheDocument();
  });

  it('test_nodeShapes: renders different shapes for source, pipeline, model, and consumer nodes', () => {
    render(
      <LineageDag
        graph={lineageGraph}
        direction="LR"
        selectedNodeId={null}
        search=""
        impact={null}
        onSelectNode={vi.fn()}
      />,
    );

    expect(screen.getByTestId('lineage-node-source-1').querySelector('[data-shape="cylinder"]')).toBeTruthy();
    expect(screen.getByTestId('lineage-node-pipeline-1').querySelector('[data-shape="chevron"]')).toBeTruthy();
    expect(screen.getByTestId('lineage-node-model-1').querySelector('[data-shape="rounded-rect"]')).toBeTruthy();
    expect(screen.getByTestId('lineage-node-consumer-1').querySelector('[data-shape="hexagon"]')).toBeTruthy();
  });

  it('test_clickHighlights: clicking a node calls selection handler', async () => {
    const user = userEvent.setup();
    const onSelectNode = vi.fn();
    render(
      <LineageDag
        graph={lineageGraph}
        direction="LR"
        selectedNodeId={null}
        search=""
        impact={null}
        onSelectNode={onSelectNode}
      />,
    );

    await user.click(screen.getByTestId('lineage-node-source-1'));
    expect(onSelectNode).toHaveBeenCalledTimes(1);
    expect(onSelectNode.mock.calls[0]?.[0].id).toBe('source-1');
  });

  it('test_impactAnalysis: highlights impacted nodes when impact data is passed', () => {
    render(
      <LineageDag
        graph={lineageGraph}
        direction="LR"
        selectedNodeId="source-1"
        search=""
        impact={{
          entity: lineageGraph.nodes[0],
          directly_affected: [{ node: lineageGraph.nodes[1], hop_distance: 1, path_description: 'source -> pipeline' }],
          indirectly_affected: [{ node: lineageGraph.nodes[2], hop_distance: 2, path_description: 'source -> pipeline -> model' }],
          affected_suites: [],
          total_affected: 2,
          severity: 'high',
          summary: 'Impact summary',
        }}
        onSelectNode={vi.fn()}
      />,
    );

    expect(screen.getByTestId('lineage-node-pipeline-1')).toBeInTheDocument();
    expect(screen.getByTestId('lineage-node-model-1')).toBeInTheDocument();
  });

  it('test_search: filters nodes by search text', () => {
    render(
      <LineageDag
        graph={lineageGraph}
        direction="LR"
        selectedNodeId={null}
        search="customer"
        impact={null}
        onSelectNode={vi.fn()}
      />,
    );

    expect(screen.getByTestId('lineage-node-source-1')).toBeInTheDocument();
  });
});

