"use client";

import Link from "next/link";
import { ExternalLink, RefreshCw } from "lucide-react";
import { RelativeTime } from "@/components/shared/relative-time";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import type { ExternalTicketLinkRecord } from "@/types/integration";

export function TicketLinksTable({
  items,
  syncingId,
  onSync,
}: {
  items: ExternalTicketLinkRecord[];
  syncingId?: string | null;
  onSync?: (id: string) => void;
}) {
  if (items.length === 0) {
    return <div className="rounded-lg border border-dashed p-6 text-sm text-muted-foreground">No external ticket links are recorded for this integration.</div>;
  }

  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>External Ticket</TableHead>
          <TableHead>Entity</TableHead>
          <TableHead>Status</TableHead>
          <TableHead>Direction</TableHead>
          <TableHead>Last Synced</TableHead>
          <TableHead className="text-right">Actions</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {items.map((item) => (
          <TableRow key={item.id}>
            <TableCell>
              <div className="font-medium">{item.external_key}</div>
              <div className="text-xs text-muted-foreground">{item.external_system}</div>
            </TableCell>
            <TableCell>
              <div className="font-medium">{item.entity_type}</div>
              <div className="text-xs text-muted-foreground">{item.entity_id}</div>
            </TableCell>
            <TableCell>
              <Badge variant="outline">{item.external_status ?? "Unknown"}</Badge>
              {item.sync_error ? <div className="mt-1 text-xs text-destructive">{item.sync_error}</div> : null}
            </TableCell>
            <TableCell>{item.sync_direction}</TableCell>
            <TableCell>{item.last_synced_at ? <RelativeTime date={item.last_synced_at} /> : "Never"}</TableCell>
            <TableCell className="text-right">
              <div className="flex justify-end gap-2">
                <Button asChild variant="outline" size="sm">
                  <Link href={`/admin/integrations/ticket-links/${item.id}`}>Details</Link>
                </Button>
                <Button asChild variant="outline" size="sm">
                  <a href={item.external_url} target="_blank" rel="noreferrer">
                    <ExternalLink className="mr-2 h-4 w-4" />
                    Open
                  </a>
                </Button>
                {onSync ? (
                  <Button variant="outline" size="sm" onClick={() => onSync(item.id)} disabled={syncingId === item.id}>
                    <RefreshCw className={`mr-2 h-4 w-4 ${syncingId === item.id ? "animate-spin" : ""}`} />
                    Sync
                  </Button>
                ) : null}
              </div>
            </TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  );
}
