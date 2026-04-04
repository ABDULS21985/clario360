'use client';

import { ArrowRight, Globe2 } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Skeleton } from '@/components/ui/skeleton';
import { formatNumber, severityToColor } from '@/lib/cti-utils';
import type { CTIGeoThreatHotspot } from '@/types/cti';

interface CTIThreatMapMiniWidgetProps {
  hotspots: CTIGeoThreatHotspot[];
  isLoading: boolean;
  onExpand: () => void;
}

const W = 480;
const H = 240;
const CONTINENT_PATHS = [
  'M65,40 L115,32 L140,48 L146,70 L132,86 L118,100 L98,108 L84,122 L70,118 L50,100 L40,76 L44,54 Z',
  'M100,126 L116,122 L132,134 L140,154 L136,176 L124,196 L108,204 L96,194 L90,174 L86,154 L92,136 Z',
  'M214,36 L244,32 L260,42 L266,54 L256,68 L242,72 L226,72 L216,62 L210,52 Z',
  'M214,82 L240,76 L260,86 L270,108 L266,138 L256,160 L242,176 L226,170 L216,150 L210,126 L204,102 Z',
  'M260,28 L328,24 L366,38 L390,52 L394,74 L380,90 L360,96 L342,100 L322,96 L302,90 L280,84 L266,72 L260,58 Z',
  'M360,144 L394,140 L414,150 L420,166 L410,182 L390,186 L372,176 L360,160 Z',
];

function toMiniSVG(lat: number, lng: number): { x: number; y: number } {
  return {
    x: ((lng + 180) / 360) * W,
    y: ((90 - lat) / 180) * H,
  };
}

function hotspotRadius(total: number): number {
  return Math.min(Math.max(2.5, Math.log2(total + 1) * 1.6), 8);
}

export function CTIThreatMapMiniWidget({
  hotspots,
  isLoading,
  onExpand,
}: CTIThreatMapMiniWidgetProps) {
  const topHotspots = [...hotspots].sort((left, right) => right.total_count - left.total_count).slice(0, 10);

  if (isLoading) {
    return (
      <Card>
        <CardHeader>
          <Skeleton className="h-4 w-40" />
          <Skeleton className="h-3 w-52" />
        </CardHeader>
        <CardContent className="space-y-4">
          <Skeleton className="h-[220px] w-full rounded-xl" />
          <div className="grid grid-cols-2 gap-3">
            <Skeleton className="h-10 w-full" />
            <Skeleton className="h-10 w-full" />
          </div>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card>
      <CardHeader className="flex flex-row items-start justify-between gap-4 space-y-0">
        <div>
          <CardTitle className="text-base">Threat Map</CardTitle>
          <CardDescription>Top ten global CTI hotspots over the last 24 hours.</CardDescription>
        </div>
        <Globe2 className="h-5 w-5 text-muted-foreground" />
      </CardHeader>
      <CardContent className="space-y-4">
        {topHotspots.length === 0 ? (
          <div className="rounded-lg border border-dashed px-4 py-8 text-center text-sm text-muted-foreground">
            No hotspot data is available for this tenant yet.
          </div>
        ) : (
          <>
            <div className="overflow-hidden rounded-xl border bg-slate-950/95">
              <svg viewBox={`0 0 ${W} ${H}`} className="w-full" preserveAspectRatio="xMidYMid meet" aria-label="Mini CTI threat map">
                {Array.from({ length: 8 }, (_, index) => (
                  <line
                    key={`v-${index}`}
                    x1={index * 60}
                    y1={0}
                    x2={index * 60}
                    y2={H}
                    stroke="white"
                    strokeOpacity={0.05}
                  />
                ))}
                {Array.from({ length: 4 }, (_, index) => (
                  <line
                    key={`h-${index}`}
                    x1={0}
                    y1={index * 60}
                    x2={W}
                    y2={index * 60}
                    stroke="white"
                    strokeOpacity={0.05}
                  />
                ))}
                {CONTINENT_PATHS.map((path, index) => (
                  <path
                    key={index}
                    d={path}
                    fill="#89A7B0"
                    fillOpacity={0.12}
                    stroke="#89A7B0"
                    strokeOpacity={0.18}
                    strokeWidth={0.6}
                  />
                ))}
                {topHotspots.map((hotspot) => {
                  const point = toMiniSVG(hotspot.latitude ?? 0, hotspot.longitude ?? 0);
                  const color = severityToColor(
                    hotspot.severity_critical_count > 0
                      ? 'critical'
                      : hotspot.severity_high_count > 0
                        ? 'high'
                        : hotspot.severity_medium_count > 0
                          ? 'medium'
                          : 'low',
                  );

                  return (
                    <g key={hotspot.id}>
                      <circle cx={point.x} cy={point.y} r={hotspotRadius(hotspot.total_count) + 2} fill={color} fillOpacity={0.16} />
                      <circle cx={point.x} cy={point.y} r={hotspotRadius(hotspot.total_count)} fill={color} />
                    </g>
                  );
                })}
              </svg>
            </div>

            <div className="grid gap-2 sm:grid-cols-2">
              {topHotspots.slice(0, 4).map((hotspot) => (
                <div key={hotspot.id} className="rounded-lg border px-3 py-2">
                  <div className="flex items-center justify-between gap-2">
                    <p className="truncate text-sm font-medium">
                      {hotspot.city}, {hotspot.country_code}
                    </p>
                    <span className="text-xs text-muted-foreground">{formatNumber(hotspot.total_count)}</span>
                  </div>
                </div>
              ))}
            </div>
          </>
        )}

        <Button variant="ghost" size="sm" className="px-0" onClick={onExpand}>
          Expand
          <ArrowRight className="ml-1 h-4 w-4" />
        </Button>
      </CardContent>
    </Card>
  );
}
