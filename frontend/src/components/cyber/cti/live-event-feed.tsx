'use client';

import { useEffect, useState } from 'react';
import { Pause, Play } from 'lucide-react';
import { useRouter } from 'next/navigation';
import { formatDistanceToNow } from 'date-fns';
import { Button } from '@/components/ui/button';
import { ScrollArea } from '@/components/ui/scroll-area';
import { CTISeverityBadge } from './severity-badge';
import type { CTIThreatEvent } from '@/types/cti';

interface LiveEventFeedProps {
  events: CTIThreatEvent[];
  maxVisible?: number;
  className?: string;
}

export function LiveEventFeed({ events, maxVisible = 20, className }: LiveEventFeedProps) {
  const router = useRouter();
  const [paused, setPaused] = useState(false);
  const [snapshot, setSnapshot] = useState<CTIThreatEvent[]>([]);

  useEffect(() => {
    if (!paused) {
      setSnapshot(events.slice(0, maxVisible));
    }
  }, [events, maxVisible, paused]);

  const visible = paused ? snapshot.slice(0, maxVisible) : events.slice(0, maxVisible);

  return (
    <div className={className}>
      <div className="flex items-center gap-2 px-3 pb-2">
        <span className="relative flex h-2 w-2">
          <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-green-400 opacity-75" />
          <span className="relative inline-flex h-2 w-2 rounded-full bg-green-500" />
        </span>
        <span className="text-xs font-medium text-muted-foreground">Live Event Feed</span>
        <span className="text-xs text-muted-foreground">({events.length})</span>
        <Button
          type="button"
          variant="ghost"
          size="sm"
          className="ml-auto h-7 px-2 text-[11px]"
          onClick={() => setPaused((current) => !current)}
        >
          {paused ? <Play className="mr-1 h-3 w-3" /> : <Pause className="mr-1 h-3 w-3" />}
          {paused ? 'Resume' : 'Pause'}
        </Button>
      </div>
      <ScrollArea className="h-[400px]">
        <div className="space-y-1 px-2">
          {visible.length === 0 && (
            <p className="py-8 text-center text-xs text-muted-foreground">Waiting for events...</p>
          )}
          {visible.map((event) => (
            <button
              key={event.id}
              type="button"
              onClick={() => router.push(`/cyber/cti/events/${event.id}`)}
              className="flex w-full items-start gap-2 rounded-md border-l-2 bg-card/50 p-2 text-left text-xs animate-in slide-in-from-top-1 duration-300 hover:bg-card/80"
              style={{ borderLeftColor: severityBorderColor(event.severity_code) }}
            >
              <CTISeverityBadge severity={event.severity_code} size="sm" />
              <div className="min-w-0 flex-1">
                <p className="truncate font-medium">{event.title}</p>
                <p className="text-muted-foreground">
                  {event.origin_city && `${event.origin_city}, `}
                  {event.origin_country_code?.toUpperCase() ?? 'Unknown'}
                  {' · '}
                  {formatDistanceToNow(new Date(event.first_seen_at), { addSuffix: true })}
                </p>
              </div>
            </button>
          ))}
        </div>
      </ScrollArea>
    </div>
  );
}

function severityBorderColor(severity: string): string {
  switch (severity) {
    case 'critical': return '#FF3B5C';
    case 'high': return '#FF8C42';
    case 'medium': return '#FFD93D';
    case 'low': return '#4ADE80';
    default: return '#6B7280';
  }
}
