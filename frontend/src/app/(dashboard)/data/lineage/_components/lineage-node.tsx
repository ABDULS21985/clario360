'use client';

import { type LineageNode as LineageNodeType } from '@/lib/data-suite';

interface LineageNodeProps {
  node: LineageNodeType;
  selected: boolean;
  dimmed: boolean;
  fill: string;
}

const WIDTH = 180;
const HEIGHT = 60;

export function LineageNode({
  node,
  selected,
  dimmed,
  fill,
}: LineageNodeProps) {
  const opacity = dimmed ? 0.2 : 1;
  const shape =
    node.type === 'data_source'
      ? 'cylinder'
      : node.type === 'pipeline'
        ? 'chevron'
        : node.type === 'quality_rule'
          ? 'diamond'
          : node.type === 'suite_consumer'
            ? 'hexagon'
            : node.type === 'report'
              ? 'document'
              : 'rounded-rect';

  return (
    <g opacity={opacity} data-shape={shape}>
      {node.type === 'data_source' ? (
        <>
          <ellipse cx={0} cy={-20} rx={WIDTH / 2} ry={12} fill={fill} />
          <rect x={-WIDTH / 2} y={-20} width={WIDTH} height={40} fill={fill} />
          <ellipse cx={0} cy={20} rx={WIDTH / 2} ry={12} fill={fill} />
        </>
      ) : node.type === 'pipeline' ? (
        <path d={`M ${-WIDTH / 2} ${-HEIGHT / 2} H ${WIDTH / 2 - 20} L ${WIDTH / 2} 0 L ${WIDTH / 2 - 20} ${HEIGHT / 2} H ${-WIDTH / 2} Z`} fill={fill} />
      ) : node.type === 'quality_rule' ? (
        <path d={`M 0 ${-HEIGHT / 2} L ${WIDTH / 2} 0 L 0 ${HEIGHT / 2} L ${-WIDTH / 2} 0 Z`} fill={fill} />
      ) : node.type === 'suite_consumer' ? (
        <path d={`M ${-WIDTH / 3} ${-HEIGHT / 2} H ${WIDTH / 3} L ${WIDTH / 2} 0 L ${WIDTH / 3} ${HEIGHT / 2} H ${-WIDTH / 3} L ${-WIDTH / 2} 0 Z`} fill={fill} />
      ) : node.type === 'report' ? (
        <path d={`M ${-WIDTH / 2} ${-HEIGHT / 2} H ${WIDTH / 4} L ${WIDTH / 2} ${-HEIGHT / 4} V ${HEIGHT / 2} H ${-WIDTH / 2} Z`} fill={fill} />
      ) : (
        <rect x={-WIDTH / 2} y={-HEIGHT / 2} width={WIDTH} height={HEIGHT} rx={18} fill={fill} />
      )}
      <text textAnchor="middle" y={selected ? -4 : 0} fill="white" fontSize="14" fontWeight="600">
        {node.name.length > 22 ? `${node.name.slice(0, 22)}…` : node.name}
      </text>
      <text textAnchor="middle" y={18} fill="rgba(255,255,255,0.82)" fontSize="11">
        {node.type.replace(/_/g, ' ')}
      </text>
    </g>
  );
}

export const LINEAGE_NODE_WIDTH = WIDTH;
export const LINEAGE_NODE_HEIGHT = HEIGHT;
