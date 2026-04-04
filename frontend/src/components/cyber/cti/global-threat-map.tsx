'use client';

import { useMemo, useState } from 'react';
import { PeriodSelector } from './period-selector';
import type { CTIGeoThreatHotspot, CTIThreatEvent } from '@/types/cti';

interface GlobalThreatMapProps {
  hotspots: CTIGeoThreatHotspot[];
  period: string;
  onPeriodChange: (period: string) => void;
  onHotspotClick: (hotspot: CTIGeoThreatHotspot) => void;
  selectedHotspot: CTIGeoThreatHotspot | null;
  liveEvents?: CTIThreatEvent[];
  className?: string;
}

const W = 960;
const H = 480;

function toSVG(lat: number, lng: number): { x: number; y: number } {
  return {
    x: ((lng + 180) / 360) * W,
    y: ((90 - lat) / 180) * H,
  };
}

function hotspotColor(h: CTIGeoThreatHotspot): string {
  if (h.severity_critical_count > 0) return '#FF3B5C';
  if (h.severity_high_count > 0) return '#FF8C42';
  if (h.severity_medium_count > 0) return '#FFD93D';
  return '#4ADE80';
}

function hotspotRadius(total: number): number {
  return Math.min(Math.max(4, Math.log2(total + 1) * 3), 16);
}

// Simplified continent outlines as SVG paths
const CONTINENT_PATHS = [
  // North America (simplified)
  'M130,80 L230,65 L270,100 L280,140 L260,170 L230,200 L200,210 L170,240 L140,230 L100,200 L80,150 L90,110 Z',
  // South America
  'M200,250 L230,240 L260,260 L280,300 L270,340 L250,380 L220,400 L190,380 L180,340 L170,300 L180,270 Z',
  // Europe
  'M430,70 L490,60 L520,80 L530,100 L510,130 L480,140 L450,140 L430,120 L420,100 Z',
  // Africa
  'M430,160 L480,150 L520,170 L540,210 L530,270 L510,320 L480,350 L450,340 L430,300 L420,250 L410,200 Z',
  // Asia
  'M520,50 L660,40 L740,70 L780,100 L790,140 L760,170 L720,180 L680,190 L640,180 L600,170 L560,160 L530,140 L520,110 Z',
  // Australia
  'M720,290 L790,280 L830,300 L840,330 L820,360 L780,370 L740,350 L720,320 Z',
];

export function GlobalThreatMap({
  hotspots,
  period,
  onPeriodChange,
  onHotspotClick,
  selectedHotspot,
  liveEvents = [],
  className,
}: GlobalThreatMapProps) {
  const [hoveredId, setHoveredId] = useState<string | null>(null);

  // Target point (Saudi Arabia — tenant default)
  const target = useMemo(() => toSVG(24.7136, 46.6753), []);

  // Top 5 hotspots for connection lines
  const top5 = useMemo(
    () => [...hotspots].sort((a, b) => b.total_count - a.total_count).slice(0, 5),
    [hotspots],
  );

  const flashingHotspots = useMemo(() => {
    const now = Date.now();
    const flashIds = new Set<string>();

    for (const event of liveEvents.slice(0, 12)) {
      const timestamp = Date.parse(event.created_at || event.first_seen_at);
      if (Number.isNaN(timestamp) || now - timestamp > 30_000) {
        continue;
      }

      const match = hotspots.find((hotspot) =>
        hotspot.country_code.toLowerCase() === (event.origin_country_code ?? '').toLowerCase()
          && hotspot.city.toLowerCase() === (event.origin_city ?? '').toLowerCase(),
      );

      if (match) {
        flashIds.add(match.id);
      }
    }

    return flashIds;
  }, [hotspots, liveEvents]);

  return (
    <div className={className}>
      <div className="mb-3 flex items-center justify-between">
        <h3 className="text-sm font-semibold">Global Threat Map</h3>
        <PeriodSelector value={period} onChange={onPeriodChange} />
      </div>
      <div className="relative overflow-hidden rounded-lg border bg-slate-950/50">
        <svg viewBox={`0 0 ${W} ${H}`} className="w-full" preserveAspectRatio="xMidYMid meet">
          {/* Grid lines */}
          {Array.from({ length: 12 }, (_, i) => (
            <line key={`vg-${i}`} x1={i * 80} y1={0} x2={i * 80} y2={H} stroke="currentColor" strokeOpacity={0.04} />
          ))}
          {Array.from({ length: 6 }, (_, i) => (
            <line key={`hg-${i}`} x1={0} y1={i * 80} x2={W} y2={i * 80} stroke="currentColor" strokeOpacity={0.04} />
          ))}

          {/* Continent fills */}
          {CONTINENT_PATHS.map((d, i) => (
            <path key={`c-${i}`} d={d} fill="currentColor" fillOpacity={0.06} stroke="currentColor" strokeOpacity={0.1} strokeWidth={0.5} />
          ))}

          {/* Connection lines from top hotspots to target */}
          {top5.map((h) => {
            const from = toSVG(h.latitude ?? 0, h.longitude ?? 0);
            return (
              <line
                key={`line-${h.id}`}
                x1={from.x} y1={from.y}
                x2={target.x} y2={target.y}
                stroke={hotspotColor(h)}
                strokeOpacity={0.15}
                strokeWidth={1}
                strokeDasharray="4 4"
              >
                <animate attributeName="stroke-dashoffset" values="8;0" dur="2s" repeatCount="indefinite" />
              </line>
            );
          })}

          {/* Target marker */}
          <circle cx={target.x} cy={target.y} r={5} fill="#0D4B4F" stroke="#C6A962" strokeWidth={1.5} />

          {/* Hotspot circles */}
          {hotspots.map((h) => {
            const pos = toSVG(h.latitude ?? 0, h.longitude ?? 0);
            const r = hotspotRadius(h.total_count);
            const color = hotspotColor(h);
            const isSelected = selectedHotspot?.id === h.id;
            const isHovered = hoveredId === h.id;
            const shouldPulse = h.severity_critical_count > 0 || h.severity_high_count > 0;
            const isFlashing = flashingHotspots.has(h.id);

            return (
              <g key={h.id}>
                {isFlashing && (
                  <circle cx={pos.x} cy={pos.y} r={r + 3} fill={color} fillOpacity={0.3}>
                    <animate attributeName="r" values={`${r + 2};${r + 12};${r + 2}`} dur="0.9s" repeatCount="2" />
                    <animate attributeName="opacity" values="0.8;0;0.8" dur="0.9s" repeatCount="2" />
                  </circle>
                )}
                {/* Pulse ring for critical/high */}
                {shouldPulse && (
                  <circle cx={pos.x} cy={pos.y} r={r} fill="none" stroke={color} strokeWidth={1}>
                    <animate attributeName="r" values={`${r};${r + 6};${r}`} dur="2s" repeatCount="indefinite" />
                    <animate attributeName="opacity" values="0.6;0;0.6" dur="2s" repeatCount="indefinite" />
                  </circle>
                )}
                {/* Main circle */}
                <circle
                  cx={pos.x}
                  cy={pos.y}
                  r={isHovered || isSelected ? r + 2 : r}
                  fill={color}
                  fillOpacity={isSelected ? 1 : 0.7}
                  stroke={isSelected ? '#fff' : 'none'}
                  strokeWidth={isSelected ? 2 : 0}
                  className="cursor-pointer transition-all duration-150"
                  onMouseEnter={() => setHoveredId(h.id)}
                  onMouseLeave={() => setHoveredId(null)}
                  onClick={() => onHotspotClick(h)}
                />
                {/* Label for hovered/selected */}
                {(isHovered || isSelected) && (
                  <text
                    x={pos.x}
                    y={pos.y - r - 6}
                    textAnchor="middle"
                    className="fill-foreground"
                    style={{ fontSize: 9, fontWeight: 600 }}
                  >
                    {h.city} ({h.total_count})
                  </text>
                )}
              </g>
            );
          })}
        </svg>

        {/* Legend */}
        <div className="absolute bottom-2 left-2 flex items-center gap-3 rounded bg-background/80 px-2 py-1 text-[10px] backdrop-blur">
          <span className="flex items-center gap-1"><span className="h-2 w-2 rounded-full" style={{ backgroundColor: '#FF3B5C' }} /> Critical</span>
          <span className="flex items-center gap-1"><span className="h-2 w-2 rounded-full" style={{ backgroundColor: '#FF8C42' }} /> High</span>
          <span className="flex items-center gap-1"><span className="h-2 w-2 rounded-full" style={{ backgroundColor: '#FFD93D' }} /> Medium</span>
          <span className="flex items-center gap-1"><span className="h-2 w-2 rounded-full" style={{ backgroundColor: '#4ADE80' }} /> Low</span>
        </div>
      </div>
    </div>
  );
}
