"use client";

import { useCallback, useMemo } from "react";
import { useRouter } from "next/navigation";
import {
  ShieldCheck,
  BarChart3,
  List,
  Download,
  HardDrive,
} from "lucide-react";
import { toast } from "sonner";
import { Badge } from "@/components/ui/badge";
import { PageHeader } from "@/components/common/page-header";
import { DataTable } from "@/components/shared/data-table/data-table";
import { SearchInput } from "@/components/shared/forms/search-input";
import { RelativeTime } from "@/components/shared/relative-time";
import { SeverityIndicator } from "@/components/shared/severity-indicator";
import {
  Tabs,
  TabsList,
  TabsTrigger,
  TabsContent,
} from "@/components/ui/tabs";
import { useDataTable } from "@/hooks/use-data-table";
import api from "@/lib/api";
import {
  AUDIT_SEVERITY_FILTER_OPTIONS,
  buildAuditLogQueryParams,
  getDefaultAuditDateRange,
  resolveAuditSeverity,
} from "@/lib/audit";
import type { ColumnDef } from "@tanstack/react-table";
import type { AuditLog } from "@/types/models";
import type { PaginatedResponse } from "@/types/api";
import type { FetchParams, FilterConfig } from "@/types/table";
import { AuditDashboard } from "./_components/audit-dashboard";
import { AuditExportForm } from "./_components/audit-export-form";
import { AuditVerifyPanel } from "./_components/audit-verify-panel";
import { AuditPartitions } from "./_components/audit-partitions";

async function fetchAuditLogs(
  params: FetchParams,
  defaultDateRange = getDefaultAuditDateRange()
): Promise<PaginatedResponse<AuditLog>> {
  const { data } = await api.get<PaginatedResponse<AuditLog>>(
    "/api/v1/audit/logs",
    {
      params: buildAuditLogQueryParams(params, defaultDateRange),
    }
  );
  return data;
}

const auditFilters: FilterConfig[] = [
  {
    key: "service",
    label: "Service",
    type: "select",
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
    type: "select",
    options: AUDIT_SEVERITY_FILTER_OPTIONS,
  },
];

const auditColumns: ColumnDef<AuditLog>[] = [
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
    enableSorting: false,
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
        severity={resolveAuditSeverity(
          row.original.action,
          row.original.severity
        )}
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

export default function AuditLogsPage() {
  const router = useRouter();
  const defaultDateRange = useMemo(() => getDefaultAuditDateRange(), []);
  const fetchAuditLogsWithDefaults = useCallback(
    (params: FetchParams) => fetchAuditLogs(params, defaultDateRange),
    [defaultDateRange]
  );

  const { tableProps } = useDataTable<AuditLog>({
    fetchFn: fetchAuditLogsWithDefaults,
    queryKey: "audit-logs",
    defaultPageSize: 50,
    defaultSort: { column: "created_at", direction: "desc" },
  });

  return (
    <div className="space-y-6">
      <PageHeader
        title="Audit Logs"
        description="Immutable record of all platform activity"
      />

      <Tabs defaultValue="dashboard">
        <TabsList>
          <TabsTrigger value="dashboard" className="gap-1.5">
            <BarChart3 className="h-4 w-4" />
            Dashboard
          </TabsTrigger>
          <TabsTrigger value="logs" className="gap-1.5">
            <List className="h-4 w-4" />
            Logs
          </TabsTrigger>
          <TabsTrigger value="export" className="gap-1.5">
            <Download className="h-4 w-4" />
            Export
          </TabsTrigger>
          <TabsTrigger value="integrity" className="gap-1.5">
            <ShieldCheck className="h-4 w-4" />
            Integrity
          </TabsTrigger>
          <TabsTrigger value="partitions" className="gap-1.5">
            <HardDrive className="h-4 w-4" />
            Partitions
          </TabsTrigger>
        </TabsList>

        <TabsContent value="dashboard">
          <AuditDashboard params={defaultDateRange} />
        </TabsContent>

        <TabsContent value="logs">
          <DataTable
            {...tableProps}
            columns={auditColumns}
            filters={auditFilters}
            onRowClick={(log) => router.push(`/admin/audit/logs/${log.id}`)}
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
              toast.info(
                `Exporting audit logs as ${format.toUpperCase()}...`
              )
            }
            emptyState={{
              icon: ShieldCheck,
              title: "No audit logs",
              description:
                "Audit events will appear here as actions are performed.",
            }}
            stickyHeader
          />
        </TabsContent>

        <TabsContent value="export">
          <AuditExportForm />
        </TabsContent>

        <TabsContent value="integrity">
          <AuditVerifyPanel />
        </TabsContent>

        <TabsContent value="partitions">
          <AuditPartitions />
        </TabsContent>
      </Tabs>
    </div>
  );
}
