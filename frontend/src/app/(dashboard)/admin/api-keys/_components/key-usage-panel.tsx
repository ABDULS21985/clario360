"use client";

import { X, BarChart3 } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetDescription,
} from "@/components/ui/sheet";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { ErrorState } from "@/components/common/error-state";
import { formatNumber, formatCompactNumber } from "@/lib/format";
import { useApiKeyUsage } from "@/hooks/use-api-keys";

interface KeyUsagePanelProps {
  keyId: string;
  keyName: string;
  open: boolean;
  onClose: () => void;
}

export function KeyUsagePanel({ keyId, keyName, open, onClose }: KeyUsagePanelProps) {
  const { data: usage, isLoading, error, refetch } = useApiKeyUsage(keyId);

  return (
    <Sheet open={open} onOpenChange={(o) => !o && onClose()}>
      <SheetContent className="sm:max-w-lg overflow-y-auto">
        <SheetHeader>
          <SheetTitle className="flex items-center gap-2">
            <BarChart3 className="h-5 w-5" />
            Usage: {keyName}
          </SheetTitle>
          <SheetDescription>
            API call statistics for this key
          </SheetDescription>
        </SheetHeader>

        <div className="mt-6 space-y-6">
          {isLoading ? (
            <div className="space-y-4">
              <Skeleton className="h-20" />
              <Skeleton className="h-40" />
              <Skeleton className="h-32" />
            </div>
          ) : error ? (
            <ErrorState
              message="Failed to load usage data"
              onRetry={() => refetch()}
            />
          ) : usage ? (
            <>
              <Card>
                <CardHeader className="pb-2">
                  <CardTitle className="text-sm text-muted-foreground">Total API Calls</CardTitle>
                </CardHeader>
                <CardContent>
                  <p className="text-3xl font-bold">{formatCompactNumber(usage.total_calls)}</p>
                  <p className="text-xs text-muted-foreground mt-1">
                    Period: {usage.period}
                  </p>
                </CardContent>
              </Card>

              {/* Calls by day chart (simple bar representation) */}
              {usage.calls_by_day.length > 0 && (
                <Card>
                  <CardHeader className="pb-2">
                    <CardTitle className="text-sm">Calls by Day</CardTitle>
                  </CardHeader>
                  <CardContent>
                    <div className="space-y-2">
                      {usage.calls_by_day.slice(-14).map((day) => {
                        const maxCalls = Math.max(
                          ...usage.calls_by_day.map((d) => d.count),
                          1,
                        );
                        const widthPercent = (day.count / maxCalls) * 100;
                        return (
                          <div key={day.date} className="flex items-center gap-2 text-xs">
                            <span className="w-20 text-muted-foreground shrink-0 font-mono">
                              {day.date.slice(5)}
                            </span>
                            <div className="flex-1 h-5 bg-muted rounded-sm overflow-hidden">
                              <div
                                className="h-full bg-primary/60 rounded-sm transition-all"
                                style={{ width: `${widthPercent}%` }}
                              />
                            </div>
                            <span className="w-14 text-right text-muted-foreground">
                              {formatNumber(day.count)}
                            </span>
                          </div>
                        );
                      })}
                    </div>
                  </CardContent>
                </Card>
              )}

              {/* Calls by scope */}
              {Object.keys(usage.calls_by_scope).length > 0 && (
                <Card>
                  <CardHeader className="pb-2">
                    <CardTitle className="text-sm">Calls by Scope</CardTitle>
                  </CardHeader>
                  <CardContent>
                    <div className="space-y-2">
                      {Object.entries(usage.calls_by_scope)
                        .sort(([, a], [, b]) => b - a)
                        .map(([scope, count]) => (
                          <div
                            key={scope}
                            className="flex items-center justify-between text-sm"
                          >
                            <code className="text-xs font-mono">{scope}</code>
                            <span className="text-muted-foreground">
                              {formatNumber(count)}
                            </span>
                          </div>
                        ))}
                    </div>
                  </CardContent>
                </Card>
              )}
            </>
          ) : null}
        </div>
      </SheetContent>
    </Sheet>
  );
}
