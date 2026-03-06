"use client";

import { useState } from "react";
import { ShieldCheck, CheckCircle, AlertTriangle } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Separator } from "@/components/ui/separator";
import { PageHeader } from "@/components/common/page-header";
import { DataTable } from "@/components/shared/data-table/data-table";
import { SearchInput } from "@/components/shared/forms/search-input";
import { RelativeTime } from "@/components/shared/relative-time";
import { DetailPanel } from "@/components/shared/detail-panel";
import { SeverityIndicator } from "@/components/shared/severity-indicator";
import { useDataTable } from "@/hooks/use-data-table";
import api from "@/lib/api";
import type { ColumnDef } from "@tanstack/react-table";
import type { AuditLog } from "@/types/models";
import type { PaginatedResponse } from "@/types/api";
import type { FilterConfig } from "@/types/table";

async function fetchAuditLogs(params: {
  page: number;
  per_page: number;
  sort?: string;
  order?: string;
  search?: string;
  filters?: Record<string, string | string[]>;
}): Promise<PaginatedResponse<AuditLog>> {
  const { data } = await api.get<PaginatedResponse<AuditLog>>(
    "/api/v1/audit/logs",
    {
      params: {
        page: params.page,
        per_page: params.per_page,
        sort: params.sort ?? "created_at",
        order: params.order ?? "desc",
        search: params.search || undefined,
        service: params.filters?.service,
        severity: params.filters?.severity,
        user_id: params.filters?.user_id,
      },
    }
  );
  return data;
}

function getSeverityFromAction(
  action: string
): "critical" | "high" | "medium" | "low" | "info" {
  if (action.includes("delete") || action.includes("suspend")) return "high";
  if (action.includes("login.failed") || action.includes("unauthorized"))
    return "medium";
  if (action.includes("create") || action.includes("update")) return "low";
  return "info";
}

export default function AuditLogsPage() {
  const [selectedLog, setSelectedLog] = useState<AuditLog | null>(null);
  const [verifying, setVerifying] = useState(false);
  const [verifyResult, setVerifyResult] = useState<{
    valid: boolean;
    message: string;
  } | null>(null);

  const { tableProps } = useDataTable<AuditLog>({
    fetchFn: fetchAuditLogs,
    queryKey: "audit-logs",
    defaultPageSize: 50,
    defaultSort: { column: "created_at", direction: "desc" },
  });

  const filters: FilterConfig[] = [
    {
      key: "service",
      label: "Service",
      type: "multi-select",
      options: [
        { label: "IAM Service", value: "iam-service" },
        { label: "Cyber Service", value: "cyber-service" },
        { label: "Data Service", value: "data-service" },
        { label: "File Service", value: "file-service" },
        { label: "Notification Service", value: "notification-service" },
        { label: "Audit Service", value: "audit-service" },
      ],
    },
    {
      key: "severity",
      label: "Severity",
      type: "multi-select",
      options: [
        { label: "Critical", value: "critical" },
        { label: "High", value: "high" },
        { label: "Medium", value: "medium" },
        { label: "Low", value: "low" },
        { label: "Info", value: "info" },
      ],
    },
  ];

  const columns: ColumnDef<AuditLog>[] = [
    {
      id: "created_at",
      header: "Timestamp",
      accessorKey: "created_at",
      enableSorting: true,
      cell: ({ row }) => <RelativeTime date={row.original.created_at} />,
    },
    {
      id: "user_email",
      header: "User",
      accessorKey: "user_email",
      enableSorting: true,
      cell: ({ row }) => (
        <span className="text-sm">
          {row.original.user_email || (
            <span className="text-muted-foreground">System</span>
          )}
        </span>
      ),
    },
    {
      id: "action",
      header: "Action",
      accessorKey: "action",
      enableSorting: true,
      cell: ({ row }) => (
        <code className="text-xs font-mono bg-muted px-1.5 py-0.5 rounded">
          {row.original.action}
        </code>
      ),
    },
    {
      id: "resource_type",
      header: "Resource",
      enableSorting: false,
      cell: ({ row }) => (
        <div className="flex items-center gap-1.5">
          <Badge variant="outline" className="text-xs">
            {row.original.resource_type}
          </Badge>
          {row.original.resource_id && (
            <code className="text-xs text-muted-foreground font-mono">
              {row.original.resource_id.slice(0, 8)}
            </code>
          )}
        </div>
      ),
    },
    {
      id: "severity",
      header: "Severity",
      enableSorting: false,
      cell: ({ row }) => (
        <SeverityIndicator
          severity={getSeverityFromAction(row.original.action)}
          size="sm"
        />
      ),
    },
    {
      id: "ip_address",
      header: "IP",
      accessorKey: "ip_address",
      enableSorting: false,
      cell: ({ row }) => (
        <span className="text-xs font-mono text-muted-foreground">
          {row.original.ip_address}
        </span>
      ),
    },
  ];

  const handleVerifyChain = async () => {
    setVerifying(true);
    setVerifyResult(null);
    try {
      const { data } = await api.post<{
        valid: boolean;
        count: number;
        message?: string;
      }>("/api/v1/audit/verify", {});
      setVerifyResult({
        valid: data.valid,
        message: data.valid
          ? `Chain valid for ${data.count.toLocaleString()} records`
          : data.message ?? "Hash chain integrity check failed",
      });
    } catch {
      setVerifyResult({ valid: false, message: "Verification request failed" });
    } finally {
      setVerifying(false);
    }
  };

  return (
    <div className="space-y-6">
      <PageHeader
        title="Audit Logs"
        description="Immutable record of all platform activity"
        actions={
          <div className="flex items-center gap-2">
            {verifyResult && (
              <span
                className={`flex items-center gap-1.5 text-sm ${
                  verifyResult.valid ? "text-green-600" : "text-destructive"
                }`}
              >
                {verifyResult.valid ? (
                  <CheckCircle className="h-4 w-4" />
                ) : (
                  <AlertTriangle className="h-4 w-4" />
                )}
                {verifyResult.message}
              </span>
            )}
            <Button
              variant="outline"
              onClick={handleVerifyChain}
              disabled={verifying}
            >
              <ShieldCheck className="mr-2 h-4 w-4" />
              {verifying ? "Verifying..." : "Verify Hash Chain"}
            </Button>
          </div>
        }
      />

      <DataTable
        {...tableProps}
        columns={columns}
        filters={filters}
        onRowClick={(log) => setSelectedLog(log)}
        searchSlot={
          <SearchInput
            value={tableProps.searchValue ?? ""}
            onChange={tableProps.onSearchChange ?? (() => {})}
            placeholder="Search by action, user, or resource..."
            loading={tableProps.isLoading}
          />
        }
        enableExport
        onExport={(format) =>
          toast.info(`Exporting audit logs as ${format.toUpperCase()}...`)
        }
        emptyState={{
          icon: ShieldCheck,
          title: "No audit logs",
          description:
            "Audit events will appear here as actions are performed.",
        }}
        stickyHeader
      />

      {selectedLog && (
        <DetailPanel
          open={!!selectedLog}
          onOpenChange={(o) => !o && setSelectedLog(null)}
          title="Audit Entry"
          description={selectedLog.action}
        >
          <div className="space-y-4">
            <div className="grid grid-cols-2 gap-3 text-sm">
              {[
                {
                  label: "Action",
                  value: (
                    <code className="font-mono text-xs bg-muted px-1.5 py-0.5 rounded">
                      {selectedLog.action}
                    </code>
                  ),
                },
                {
                  label: "Timestamp",
                  value: <RelativeTime date={selectedLog.created_at} />,
                },
                {
                  label: "User",
                  value: selectedLog.user_email || "System",
                },
                {
                  label: "Resource Type",
                  value: (
                    <Badge variant="outline">{selectedLog.resource_type}</Badge>
                  ),
                },
                {
                  label: "Resource ID",
                  value: selectedLog.resource_id ? (
                    <code className="font-mono text-xs">{selectedLog.resource_id}</code>
                  ) : (
                    "—"
                  ),
                },
                {
                  label: "IP Address",
                  value: (
                    <code className="font-mono text-xs">{selectedLog.ip_address}</code>
                  ),
                },
              ].map(({ label, value }) => (
                <div key={label}>
                  <p className="text-xs text-muted-foreground font-medium uppercase tracking-wide">
                    {label}
                  </p>
                  <div className="mt-0.5">{value}</div>
                </div>
              ))}
            </div>
            {Object.keys(selectedLog.metadata).length > 0 && (
              <>
                <Separator />
                <div>
                  <p className="text-xs text-muted-foreground font-medium uppercase tracking-wide mb-2">
                    Metadata
                  </p>
                  <pre className="text-xs font-mono bg-muted rounded p-3 overflow-auto max-h-48">
                    {JSON.stringify(selectedLog.metadata, null, 2)}
                  </pre>
                </div>
              </>
            )}
          </div>
        </DetailPanel>
      )}
    </div>
  );
}
