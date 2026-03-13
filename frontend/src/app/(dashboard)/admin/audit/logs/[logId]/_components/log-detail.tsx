"use client";

import Link from "next/link";
import {
  ArrowLeft,
  Globe,
  Monitor,
  Clock,
  Hash,
  Link2,
  Timer,
  User,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Separator } from "@/components/ui/separator";
import { Skeleton } from "@/components/ui/skeleton";
import { RelativeTime } from "@/components/shared/relative-time";
import { SeverityIndicator } from "@/components/shared/severity-indicator";
import { useAuditLogDetail } from "@/hooks/use-audit";
import { resolveAuditSeverity } from "@/lib/audit";
import { formatDateTime } from "@/lib/format";
import { ChangesDiff } from "./changes-diff";
import { JsonViewer } from "./json-viewer";
import type { AuditLogDetail } from "@/types/audit";

interface LogDetailProps {
  logId: string;
}

function DetailSkeleton() {
  return (
    <div className="space-y-6">
      <div className="flex items-center gap-3">
        <Skeleton className="h-9 w-20" />
        <Skeleton className="h-6 w-64" />
      </div>
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        <div className="lg:col-span-2 space-y-6">
          <Skeleton className="h-48 w-full rounded-lg" />
          <Skeleton className="h-64 w-full rounded-lg" />
        </div>
        <div className="space-y-6">
          <Skeleton className="h-64 w-full rounded-lg" />
        </div>
      </div>
    </div>
  );
}

export function LogDetail({ logId }: LogDetailProps) {
  const { data: log, isLoading, error, refetch } = useAuditLogDetail(logId);

  if (isLoading) {
    return <DetailSkeleton />;
  }

  if (error || !log) {
    return (
      <div className="flex flex-col items-center justify-center py-12 gap-3">
        <p className="text-sm text-muted-foreground">
          {error
            ? error instanceof Error
              ? error.message
              : "Failed to load audit log detail"
            : "Audit log not found"}
        </p>
        <div className="flex gap-2">
          <Button variant="outline" size="sm" asChild>
            <Link href="/admin/audit">Back to Audit Logs</Link>
          </Button>
          {error && (
            <Button variant="outline" size="sm" onClick={() => refetch()}>
              Retry
            </Button>
          )}
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
        <div className="flex items-center gap-2">
          <code className="text-sm font-mono bg-muted px-2 py-0.5 rounded">
            {log.action}
          </code>
          <SeverityIndicator
            severity={resolveAuditSeverity(log.action, log.severity)}
            size="sm"
          />
        </div>
      </div>

      {/* Main content */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Left column - wide */}
        <div className="lg:col-span-2 space-y-6">
          {/* Event Summary */}
          <Card>
            <CardHeader className="pb-3">
              <CardTitle className="text-sm font-semibold">
                Event Summary
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div className="grid grid-cols-2 gap-4 text-sm">
                <div>
                  <p className="text-xs text-muted-foreground font-medium uppercase tracking-wide">
                    Action
                  </p>
                  <code className="text-xs font-mono bg-muted px-1.5 py-0.5 rounded mt-1 inline-block">
                    {log.action}
                  </code>
                </div>
                <div>
                  <p className="text-xs text-muted-foreground font-medium uppercase tracking-wide">
                    Timestamp
                  </p>
                  <p className="mt-0.5">{formatDateTime(log.created_at)}</p>
                  <p className="text-xs text-muted-foreground">
                    <RelativeTime date={log.created_at} />
                  </p>
                </div>
                <div>
                  <p className="text-xs text-muted-foreground font-medium uppercase tracking-wide">
                    User
                  </p>
                  <p className="mt-0.5">
                    {log.user_email || (
                      <span className="text-muted-foreground">System</span>
                    )}
                  </p>
                </div>
                <div>
                  <p className="text-xs text-muted-foreground font-medium uppercase tracking-wide">
                    Resource
                  </p>
                  <div className="flex items-center gap-1.5 mt-0.5">
                    <Badge variant="outline" className="text-xs">
                      {log.resource_type}
                    </Badge>
                    {log.resource_id && (
                      <Link
                        href={`/admin/audit/timeline/${log.resource_id}`}
                        className="text-xs font-mono text-primary hover:underline"
                      >
                        {log.resource_id.slice(0, 8)}...
                      </Link>
                    )}
                  </div>
                </div>
                {log.service && (
                  <div>
                    <p className="text-xs text-muted-foreground font-medium uppercase tracking-wide">
                      Service
                    </p>
                    <p className="mt-0.5">{log.service}</p>
                  </div>
                )}
                {log.response_status !== null && (
                  <div>
                    <p className="text-xs text-muted-foreground font-medium uppercase tracking-wide">
                      Response Status
                    </p>
                    <Badge
                      variant={
                        log.response_status < 400
                          ? "default"
                          : "destructive"
                      }
                      className="mt-0.5"
                    >
                      {log.response_status}
                    </Badge>
                  </div>
                )}
              </div>
            </CardContent>
          </Card>

          {/* Changes Diff */}
          {log.changes?.length > 0 && (
            <Card>
              <CardHeader className="pb-3">
                <CardTitle className="text-sm font-semibold">
                  Changes ({log.changes?.length ?? 0})
                </CardTitle>
              </CardHeader>
              <CardContent>
                <ChangesDiff changes={log.changes} />
              </CardContent>
            </Card>
          )}

          {/* Request Body */}
          {log.request_body && (
            <Card>
              <CardHeader className="pb-3">
                <CardTitle className="text-sm font-semibold">
                  Request Body
                </CardTitle>
              </CardHeader>
              <CardContent>
                <JsonViewer data={log.request_body} />
              </CardContent>
            </Card>
          )}

          {/* Response Body */}
          {log.response_body && (
            <Card>
              <CardHeader className="pb-3">
                <CardTitle className="text-sm font-semibold">
                  Response Body
                </CardTitle>
              </CardHeader>
              <CardContent>
                <JsonViewer data={log.response_body} defaultCollapsed />
              </CardContent>
            </Card>
          )}
        </div>

        {/* Right column - narrow metadata */}
        <div className="space-y-6">
          <Card>
            <CardHeader className="pb-3">
              <CardTitle className="text-sm font-semibold">
                Metadata
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <MetadataRow icon={Globe} label="IP Address">
                <code className="text-xs font-mono">{log.ip_address}</code>
              </MetadataRow>
              <MetadataRow icon={Monitor} label="User Agent">
                <p className="text-xs text-muted-foreground break-all">
                  {log.user_agent}
                </p>
              </MetadataRow>
              {log.geo_location && (
                <MetadataRow icon={Globe} label="Location">
                  <p className="text-sm">
                    {log.geo_location.city}, {log.geo_location.country}
                  </p>
                </MetadataRow>
              )}
              {log.session_id && (
                <MetadataRow icon={User} label="Session ID">
                  <code className="text-xs font-mono break-all">
                    {log.session_id}
                  </code>
                </MetadataRow>
              )}
              {log.correlation_id && (
                <MetadataRow icon={Link2} label="Correlation ID">
                  <code className="text-xs font-mono break-all">
                    {log.correlation_id}
                  </code>
                </MetadataRow>
              )}
              {log.duration_ms !== null && (
                <MetadataRow icon={Timer} label="Duration">
                  <p className="text-sm tabular-nums">
                    {log.duration_ms}ms
                  </p>
                </MetadataRow>
              )}
              {log.entry_hash && (
                <MetadataRow icon={Hash} label="Entry Hash">
                  <code className="text-xs font-mono break-all text-muted-foreground">
                    {log.entry_hash}
                  </code>
                </MetadataRow>
              )}
              {log.prev_hash && (
                <MetadataRow icon={Hash} label="Previous Hash">
                  <code className="text-xs font-mono break-all text-muted-foreground">
                    {log.prev_hash}
                  </code>
                </MetadataRow>
              )}
            </CardContent>
          </Card>

          {/* Navigation */}
          {log.resource_id && (
            <Card>
              <CardHeader className="pb-3">
                <CardTitle className="text-sm font-semibold">
                  Navigation
                </CardTitle>
              </CardHeader>
              <CardContent>
                <Button variant="outline" size="sm" className="w-full" asChild>
                  <Link
                    href={`/admin/audit/timeline/${log.resource_id}`}
                  >
                    <Clock className="mr-2 h-4 w-4" />
                    View Resource Timeline
                  </Link>
                </Button>
              </CardContent>
            </Card>
          )}
        </div>
      </div>
    </div>
  );
}

function MetadataRow({
  icon: Icon,
  label,
  children,
}: {
  icon: React.ComponentType<{ className?: string }>;
  label: string;
  children: React.ReactNode;
}) {
  return (
    <div className="flex gap-3">
      <Icon className="h-4 w-4 text-muted-foreground shrink-0 mt-0.5" />
      <div className="min-w-0">
        <p className="text-xs text-muted-foreground font-medium">{label}</p>
        <div className="mt-0.5">{children}</div>
      </div>
    </div>
  );
}
