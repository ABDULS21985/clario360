"use client";

import { useState } from "react";
import Link from "next/link";
import {
  ArrowLeft,
  Plus,
  Pencil,
  Trash2,
  Eye,
  ChevronDown,
  ChevronRight,
  Clock,
  Filter,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Separator } from "@/components/ui/separator";
import { Skeleton } from "@/components/ui/skeleton";
import { useAuditTimeline } from "@/hooks/use-audit";
import { formatDateTime } from "@/lib/format";
import { cn } from "@/lib/utils";
import type { AuditChange, AuditTimelineEvent, AuditTimelineParams } from "@/types/audit";

const ACTION_STYLES: Record<string, { color: string; icon: React.ComponentType<{ className?: string }> }> = {
  create: { color: "bg-green-500", icon: Plus },
  update: { color: "bg-blue-500", icon: Pencil },
  delete: { color: "bg-red-500", icon: Trash2 },
  access: { color: "bg-gray-400", icon: Eye },
};

function getActionStyle(action: string) {
  const lower = action.toLowerCase();
  if (lower.includes("create")) return ACTION_STYLES.create;
  if (lower.includes("update") || lower.includes("modify") || lower.includes("edit"))
    return ACTION_STYLES.update;
  if (lower.includes("delete") || lower.includes("remove"))
    return ACTION_STYLES.delete;
  return ACTION_STYLES.access;
}

function formatChangeValue(val: unknown): string {
  if (val === null || val === undefined) return "null";
  if (typeof val === "string") return `"${val}"`;
  if (typeof val === "object") return JSON.stringify(val);
  return String(val);
}

function TimelineEventCard({ event }: { event: AuditTimelineEvent }) {
  const [expanded, setExpanded] = useState(false);
  const style = getActionStyle(event.action);
  const Icon = style.icon;

  return (
    <div className="relative flex gap-4 pb-6 last:pb-0">
      {/* Timeline rail */}
      <div className="flex flex-col items-center">
        <div
          className={cn(
            "flex h-8 w-8 items-center justify-center rounded-full text-white shrink-0",
            style.color
          )}
        >
          <Icon className="h-4 w-4" />
        </div>
        <div className="flex-1 w-px bg-border mt-2" />
      </div>

      {/* Event content */}
      <div className="flex-1 min-w-0 pt-0.5">
        <div className="flex items-start justify-between gap-2">
          <div className="min-w-0">
            <div className="flex items-center gap-2 flex-wrap">
              <code className="text-xs font-mono bg-muted px-1.5 py-0.5 rounded">
                {event.action}
              </code>
              <span className="text-sm text-muted-foreground">by</span>
              <span className="text-sm font-medium truncate">
                {event.user_name || "System"}
              </span>
            </div>
            <p className="text-xs text-muted-foreground mt-1">
              {formatDateTime(event.timestamp)}
            </p>
            {event.summary && (
              <p className="text-sm text-muted-foreground mt-1">
                {event.summary}
              </p>
            )}
          </div>
          <div className="flex items-center gap-1 shrink-0">
            {event.changes.length > 0 && (
              <Button
                variant="ghost"
                size="sm"
                onClick={() => setExpanded(!expanded)}
                className="text-xs"
              >
                {expanded ? (
                  <ChevronDown className="h-3 w-3 mr-1" />
                ) : (
                  <ChevronRight className="h-3 w-3 mr-1" />
                )}
                {event.changes.length} change
                {event.changes.length > 1 ? "s" : ""}
              </Button>
            )}
            <Button variant="ghost" size="sm" asChild>
              <Link href={`/admin/audit/logs/${event.id}`}>
                <Eye className="h-3 w-3" />
              </Link>
            </Button>
          </div>
        </div>

        {expanded && event.changes.length > 0 && (
          <div className="mt-3 rounded-md border bg-muted/20 overflow-hidden">
            <table className="w-full text-xs font-mono">
              <tbody>
                {event.changes.map((change: AuditChange) => (
                  <tr
                    key={change.field}
                    className="border-b last:border-0"
                  >
                    <td className="px-2 py-1.5 font-semibold whitespace-nowrap w-32">
                      {change.field}
                    </td>
                    <td className="px-2 py-1.5 text-red-600 dark:text-red-400 break-all">
                      {formatChangeValue(change.old_value)}
                    </td>
                    <td className="px-2 py-1.5 text-muted-foreground w-6 text-center">
                      →
                    </td>
                    <td className="px-2 py-1.5 text-green-600 dark:text-green-400 break-all">
                      {formatChangeValue(change.new_value)}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  );
}

interface ResourceTimelineProps {
  resourceId: string;
}

export function ResourceTimeline({ resourceId }: ResourceTimelineProps) {
  const [params, setParams] = useState<AuditTimelineParams>({});
  const [showFilters, setShowFilters] = useState(false);

  const { data: timeline, isLoading, error, refetch } = useAuditTimeline(
    resourceId,
    params
  );

  if (isLoading) {
    return (
      <div className="space-y-6">
        <div className="flex items-center gap-3">
          <Skeleton className="h-9 w-20" />
          <Skeleton className="h-6 w-64" />
        </div>
        <div className="space-y-4">
          {Array.from({ length: 5 }).map((_, i) => (
            <div key={i} className="flex gap-4">
              <Skeleton className="h-8 w-8 rounded-full shrink-0" />
              <div className="flex-1 space-y-2">
                <Skeleton className="h-4 w-full" />
                <Skeleton className="h-3 w-48" />
              </div>
            </div>
          ))}
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="flex flex-col items-center justify-center py-12 gap-3">
        <p className="text-sm text-muted-foreground">
          {error instanceof Error
            ? error.message
            : "Failed to load resource timeline"}
        </p>
        <div className="flex gap-2">
          <Button variant="outline" size="sm" asChild>
            <Link href="/admin/audit">Back to Audit Logs</Link>
          </Button>
          <Button variant="outline" size="sm" onClick={() => refetch()}>
            Retry
          </Button>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center gap-3">
        <Button variant="ghost" size="sm" asChild>
          <Link href="/admin/audit">
            <ArrowLeft className="mr-1 h-4 w-4" />
            Back
          </Link>
        </Button>
        <Separator orientation="vertical" className="h-6" />
        <div>
          <h2 className="text-lg font-semibold">
            {timeline?.resource_name || resourceId}
          </h2>
          <div className="flex items-center gap-2 text-sm text-muted-foreground">
            {timeline?.resource_type && (
              <Badge variant="outline" className="text-xs">
                {timeline.resource_type}
              </Badge>
            )}
            <span>
              {timeline?.events.length ?? 0} event
              {(timeline?.events.length ?? 0) !== 1 ? "s" : ""}
            </span>
          </div>
        </div>
      </div>

      {/* Filters */}
      <div className="flex items-center gap-2">
        <Button
          variant="outline"
          size="sm"
          onClick={() => setShowFilters(!showFilters)}
        >
          <Filter className="mr-1 h-3.5 w-3.5" />
          Filters
        </Button>
      </div>

      {showFilters && (
        <Card>
          <CardContent className="pt-4">
            <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
              <div className="space-y-2">
                <Label htmlFor="timeline-action">Action</Label>
                <Input
                  id="timeline-action"
                  placeholder="e.g., user.update"
                  value={params.action ?? ""}
                  onChange={(e) =>
                    setParams((p) => ({
                      ...p,
                      action: e.target.value || undefined,
                    }))
                  }
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="timeline-from">From</Label>
                <Input
                  id="timeline-from"
                  type="date"
                  value={params.date_from ?? ""}
                  onChange={(e) =>
                    setParams((p) => ({
                      ...p,
                      date_from: e.target.value || undefined,
                    }))
                  }
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="timeline-to">To</Label>
                <Input
                  id="timeline-to"
                  type="date"
                  value={params.date_to ?? ""}
                  onChange={(e) =>
                    setParams((p) => ({
                      ...p,
                      date_to: e.target.value || undefined,
                    }))
                  }
                />
              </div>
            </div>
          </CardContent>
        </Card>
      )}

      {/* Timeline */}
      {!timeline?.events.length ? (
        <div className="flex flex-col items-center justify-center py-12 gap-2">
          <Clock className="h-8 w-8 text-muted-foreground/40" />
          <p className="text-sm text-muted-foreground">
            No events found for this resource.
          </p>
        </div>
      ) : (
        <div className="pl-2">
          {timeline.events.map((event) => (
            <TimelineEventCard key={event.id} event={event} />
          ))}
        </div>
      )}
    </div>
  );
}
