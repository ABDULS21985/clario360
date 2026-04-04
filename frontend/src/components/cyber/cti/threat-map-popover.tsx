'use client';

import { X } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import type { CTIGeoThreatHotspot } from '@/types/cti';

interface ThreatMapPopoverProps {
  hotspot: CTIGeoThreatHotspot;
  onClose: () => void;
  onViewEvents: (countryCode: string, city: string) => void;
}

function countryCodeToFlag(countryCode: string): string {
  if (!countryCode || countryCode.length !== 2) {
    return '🌐';
  }

  return countryCode
    .toUpperCase()
    .split('')
    .map((char) => String.fromCodePoint(127397 + char.charCodeAt(0)))
    .join('');
}

export function ThreatMapPopover({ hotspot, onClose, onViewEvents }: ThreatMapPopoverProps) {
  const total = hotspot.total_count;
  const bars = [
    { label: 'Critical', count: hotspot.severity_critical_count, color: '#FF3B5C' },
    { label: 'High', count: hotspot.severity_high_count, color: '#FF8C42' },
    { label: 'Medium', count: hotspot.severity_medium_count, color: '#FFD93D' },
    { label: 'Low', count: hotspot.severity_low_count, color: '#4ADE80' },
  ];

  return (
    <Card className="w-72 shadow-xl">
      <CardHeader className="flex flex-row items-center justify-between p-3">
        <CardTitle className="text-sm">
          {countryCodeToFlag(hotspot.country_code)} {hotspot.city}, {hotspot.country_code.toUpperCase()}
        </CardTitle>
        <Button variant="ghost" size="icon" className="h-6 w-6" onClick={onClose}>
          <X className="h-3 w-3" />
        </Button>
      </CardHeader>
      <CardContent className="space-y-3 p-3 pt-0">
        <div className="text-center">
          <span className="text-3xl font-bold tabular-nums">{total.toLocaleString()}</span>
          <p className="text-xs text-muted-foreground">Total Events</p>
        </div>

        {/* Severity breakdown bar */}
        <div className="space-y-1">
          <div className="flex h-3 overflow-hidden rounded-full bg-muted">
            {bars.map((bar) =>
              bar.count > 0 ? (
                <div
                  key={bar.label}
                  style={{ width: `${(bar.count / total) * 100}%`, backgroundColor: bar.color }}
                  title={`${bar.label}: ${bar.count}`}
                />
              ) : null,
            )}
          </div>
          <div className="grid grid-cols-2 gap-1 text-[10px]">
            {bars.map((bar) => (
              <span key={bar.label} className="flex items-center gap-1">
                <span className="h-1.5 w-1.5 rounded-full" style={{ backgroundColor: bar.color }} />
                {bar.label}: {bar.count}
              </span>
            ))}
          </div>
        </div>

        {hotspot.top_threat_type && (
          <p className="text-xs text-muted-foreground">
            Top type: <span className="font-medium text-foreground">{hotspot.top_threat_type}</span>
          </p>
        )}

        <Button
          size="sm"
          className="w-full text-xs"
          onClick={() => onViewEvents(hotspot.country_code, hotspot.city)}
        >
          View Events &rarr;
        </Button>
      </CardContent>
    </Card>
  );
}
