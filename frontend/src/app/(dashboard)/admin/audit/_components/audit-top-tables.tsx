"use client";

import Link from "next/link";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { Badge } from "@/components/ui/badge";
import { RelativeTime } from "@/components/shared/relative-time";
import { formatNumber } from "@/lib/format";
import type { AuditLogStats } from "@/types/audit";

interface AuditTopTablesProps {
  stats: AuditLogStats | undefined;
  loading: boolean;
}

export function AuditTopTables({ stats, loading }: AuditTopTablesProps) {
  return (
    <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
      <Card>
        <CardHeader className="pb-2">
          <CardTitle className="text-sm font-semibold">Top Users</CardTitle>
        </CardHeader>
        <CardContent>
          {loading ? (
            <div className="space-y-3">
              {Array.from({ length: 5 }).map((_, i) => (
                <div key={i} className="flex items-center gap-3">
                  <Skeleton className="h-8 w-8 rounded-full" />
                  <div className="flex-1 space-y-1">
                    <Skeleton className="h-3 w-32" />
                    <Skeleton className="h-2.5 w-48" />
                  </div>
                  <Skeleton className="h-4 w-12" />
                </div>
              ))}
            </div>
          ) : !stats?.top_users?.length ? (
            <p className="text-sm text-muted-foreground py-6 text-center">
              No user activity recorded yet.
            </p>
          ) : (
            <div className="space-y-3">
              {stats.top_users.map((user) => (
                <div
                  key={user.user_id}
                  className="flex items-center gap-3 text-sm"
                >
                  <div className="flex h-8 w-8 items-center justify-center rounded-full bg-muted text-xs font-medium">
                    {user.user_name
                      ? user.user_name
                          .split(" ")
                          .map((n) => n[0])
                          .join("")
                          .toUpperCase()
                          .slice(0, 2)
                      : "?"}
                  </div>
                  <div className="flex-1 min-w-0">
                    <p className="font-medium truncate">
                      {user.user_name || "Unknown"}
                    </p>
                    <p className="text-xs text-muted-foreground truncate">
                      {user.user_email}
                    </p>
                  </div>
                  <div className="text-right shrink-0">
                    <p className="font-medium tabular-nums">
                      {formatNumber(user.event_count)}
                    </p>
                    <p className="text-xs text-muted-foreground">
                      <RelativeTime date={user.last_event_at} />
                    </p>
                  </div>
                </div>
              ))}
            </div>
          )}
        </CardContent>
      </Card>

      <Card>
        <CardHeader className="pb-2">
          <CardTitle className="text-sm font-semibold">
            Top Resources
          </CardTitle>
        </CardHeader>
        <CardContent>
          {loading ? (
            <div className="space-y-3">
              {Array.from({ length: 5 }).map((_, i) => (
                <div key={i} className="flex items-center gap-3">
                  <div className="flex-1 space-y-1">
                    <Skeleton className="h-3 w-40" />
                    <Skeleton className="h-2.5 w-24" />
                  </div>
                  <Skeleton className="h-4 w-12" />
                </div>
              ))}
            </div>
          ) : !stats?.top_resources?.length ? (
            <p className="text-sm text-muted-foreground py-6 text-center">
              No resource activity recorded yet.
            </p>
          ) : (
            <div className="space-y-3">
              {stats.top_resources.map((resource) => (
                <div
                  key={`${resource.resource_type}-${resource.resource_id}`}
                  className="flex items-center gap-3 text-sm"
                >
                  <div className="flex-1 min-w-0">
                    <Link
                      href={`/admin/audit/timeline/${resource.resource_id}`}
                      className="font-medium truncate hover:underline"
                    >
                      {resource.resource_name || resource.resource_id}
                    </Link>
                    <div className="mt-0.5">
                      <Badge variant="outline" className="text-xs">
                        {resource.resource_type}
                      </Badge>
                    </div>
                  </div>
                  <p className="font-medium tabular-nums shrink-0">
                    {formatNumber(resource.event_count)}
                  </p>
                </div>
              ))}
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
