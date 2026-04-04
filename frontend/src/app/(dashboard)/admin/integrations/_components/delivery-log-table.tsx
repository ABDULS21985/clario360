"use client";

import { RelativeTime } from "@/components/shared/relative-time";
import { Badge } from "@/components/ui/badge";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import type { IntegrationDeliveryRecord } from "@/types/integration";

export function DeliveryLogTable({ items }: { items: IntegrationDeliveryRecord[] }) {
  if (items.length === 0) {
    return <div className="rounded-lg border border-dashed p-6 text-sm text-muted-foreground">No delivery records match the current filters.</div>;
  }

  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>Event</TableHead>
          <TableHead>Status</TableHead>
          <TableHead>Attempts</TableHead>
          <TableHead>Response</TableHead>
          <TableHead>Latency</TableHead>
          <TableHead>Next Retry</TableHead>
          <TableHead>Created</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {items.map((item) => (
          <TableRow key={item.id}>
            <TableCell>
              <div className="font-medium">{item.event_type}</div>
              <div className="text-xs text-muted-foreground">{item.event_id}</div>
              {item.last_error ? <div className="mt-1 text-xs text-destructive">{item.last_error}</div> : null}
            </TableCell>
            <TableCell>
              <Badge variant={badgeVariant(item.status)}>{item.status}</Badge>
            </TableCell>
            <TableCell>
              {item.attempts}/{item.max_attempts}
            </TableCell>
            <TableCell>{item.response_code ?? "—"}</TableCell>
            <TableCell>{item.latency_ms ? `${item.latency_ms} ms` : "—"}</TableCell>
            <TableCell>{item.next_retry_at ? <RelativeTime date={item.next_retry_at} /> : "—"}</TableCell>
            <TableCell>
              <RelativeTime date={item.created_at} />
            </TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  );
}

function badgeVariant(status: IntegrationDeliveryRecord["status"]): "default" | "secondary" | "destructive" | "outline" {
  switch (status) {
    case "delivered":
      return "default";
    case "retrying":
      return "secondary";
    case "failed":
      return "destructive";
    default:
      return "outline";
  }
}
