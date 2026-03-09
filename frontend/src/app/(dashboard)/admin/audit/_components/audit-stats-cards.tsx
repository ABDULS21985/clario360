"use client";

import { Activity, CalendarDays, Users, Server } from "lucide-react";
import { Card, CardContent } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { formatCompactNumber } from "@/lib/format";
import type { AuditLogStats } from "@/types/audit";

interface AuditStatsCardsProps {
  stats: AuditLogStats | undefined;
  loading: boolean;
}

const cards = [
  {
    key: "total_events" as const,
    label: "Total Events",
    icon: Activity,
    color: "text-blue-600 dark:text-blue-400",
    bg: "bg-blue-100 dark:bg-blue-900/30",
  },
  {
    key: "events_today" as const,
    label: "Events Today",
    icon: CalendarDays,
    color: "text-green-600 dark:text-green-400",
    bg: "bg-green-100 dark:bg-green-900/30",
  },
  {
    key: "unique_users" as const,
    label: "Unique Users",
    icon: Users,
    color: "text-purple-600 dark:text-purple-400",
    bg: "bg-purple-100 dark:bg-purple-900/30",
  },
  {
    key: "unique_services" as const,
    label: "Unique Services",
    icon: Server,
    color: "text-orange-600 dark:text-orange-400",
    bg: "bg-orange-100 dark:bg-orange-900/30",
  },
] as const;

export function AuditStatsCards({ stats, loading }: AuditStatsCardsProps) {
  if (loading) {
    return (
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        {cards.map((card) => (
          <Card key={card.key}>
            <CardContent className="p-4">
              <div className="flex items-center gap-3">
                <Skeleton className="h-10 w-10 rounded-full" />
                <div className="space-y-2 flex-1">
                  <Skeleton className="h-3 w-20" />
                  <Skeleton className="h-6 w-16" />
                </div>
              </div>
            </CardContent>
          </Card>
        ))}
      </div>
    );
  }

  return (
    <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
      {cards.map((card) => {
        const Icon = card.icon;
        const value = stats?.[card.key] ?? 0;
        return (
          <Card key={card.key}>
            <CardContent className="p-4">
              <div className="flex items-center gap-3">
                <div
                  className={`flex h-10 w-10 items-center justify-center rounded-full ${card.bg}`}
                >
                  <Icon className={`h-5 w-5 ${card.color}`} />
                </div>
                <div>
                  <p className="text-xs text-muted-foreground font-medium">
                    {card.label}
                  </p>
                  <p className="text-xl font-bold tracking-tight">
                    {formatCompactNumber(value)}
                  </p>
                </div>
              </div>
            </CardContent>
          </Card>
        );
      })}
    </div>
  );
}
