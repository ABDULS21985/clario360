"use client";

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
import type { ColumnDef } from "@tanstack/react-table";
import type { AuditLog } from "@/types/models";
import type { PaginatedResponse } from "@/types/api";
import type { FilterConfig } from "@/types/table";
import { AuditDashboard } from "./_components/audit-dashboard";
import { AuditExportForm } from "./_components/audit-export-form";
import { AuditVerifyPanel } from "./_components/audit-verify-panel";
import { AuditPartitions } from "./_components/audit-partitions";

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

const auditFilters: FilterConfig[] = [
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

export default function AuditLogsPage() {
  const router = useRouter();

  const { tableProps } = useDataTable<AuditLog>({
    fetchFn: fetchAuditLogs,
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
          <AuditDashboard />
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
