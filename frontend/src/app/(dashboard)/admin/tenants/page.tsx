"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { Plus, Building2 } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { PageHeader } from "@/components/common/page-header";
import { DataTable } from "@/components/shared/data-table/data-table";
import { SearchInput } from "@/components/shared/forms/search-input";
import { StatusBadge } from "@/components/shared/status-badge";
import { RelativeTime } from "@/components/shared/relative-time";
import { ConfirmDialog } from "@/components/shared/confirm-dialog";
import { tenantStatusConfig, tenantPlanConfig } from "@/lib/status-configs";
import { parseApiError } from "@/lib/format";
import { useDataTable } from "@/hooks/use-data-table";
import { useDeprovisionTenant } from "@/hooks/use-tenants";
import api from "@/lib/api";
import type { ColumnDef } from "@tanstack/react-table";
import type { PaginatedResponse } from "@/types/api";
import type { Tenant } from "@/types/tenant";
import type { FilterConfig } from "@/types/table";

async function fetchTenants(params: {
  page: number;
  per_page: number;
  sort?: string;
  order?: string;
  search?: string;
  filters?: Record<string, string | string[]>;
}): Promise<PaginatedResponse<Tenant>> {
  const { data } = await api.get<PaginatedResponse<Tenant>>("/api/v1/tenants", {
    params: {
      page: params.page,
      per_page: params.per_page,
      sort: params.sort,
      order: params.order,
      search: params.search || undefined,
      status: params.filters?.status,
      subscription_tier: params.filters?.subscription_tier,
    },
  });
  return data;
}

export default function TenantsPage() {
  const router = useRouter();
  const [deprovisionTenant, setDeprovisionTenant] = useState<Tenant | null>(null);
  const deprovisionMutation = useDeprovisionTenant();

  const { tableProps, refetch } = useDataTable<Tenant>({
    fetchFn: fetchTenants,
    queryKey: "tenants",
    defaultPageSize: 25,
    defaultSort: { column: "created_at", direction: "desc" },
  });

  const filters: FilterConfig[] = [
    {
      key: "status",
      label: "Status",
      type: "multi-select",
      options: [
        { label: "Active", value: "active" },
        { label: "Inactive", value: "inactive" },
        { label: "Suspended", value: "suspended" },
        { label: "Trial", value: "trial" },
        { label: "Onboarding", value: "onboarding" },
        { label: "Deprovisioned", value: "deprovisioned" },
      ],
    },
    {
      key: "subscription_tier",
      label: "Plan",
      type: "multi-select",
      options: [
        { label: "Free", value: "free" },
        { label: "Starter", value: "starter" },
        { label: "Professional", value: "professional" },
        { label: "Enterprise", value: "enterprise" },
      ],
    },
  ];

  const columns: ColumnDef<Tenant>[] = [
    {
      id: "name",
      header: "Name",
      accessorKey: "name",
      enableSorting: true,
      cell: ({ row }) => (
        <button
          className="font-medium text-sm hover:underline text-left"
          onClick={(e) => {
            e.stopPropagation();
            router.push(`/admin/tenants/${row.original.id}`);
          }}
        >
          {row.original.name}
        </button>
      ),
    },
    {
      id: "slug",
      header: "Slug",
      accessorKey: "slug",
      enableSorting: true,
      cell: ({ row }) => (
        <code className="text-xs font-mono text-muted-foreground">
          {row.original.slug}
        </code>
      ),
    },
    {
      id: "status",
      header: "Status",
      accessorKey: "status",
      enableSorting: true,
      cell: ({ row }) => (
        <StatusBadge status={row.original.status} config={tenantStatusConfig} size="sm" />
      ),
    },
    {
      id: "subscription_tier",
      header: "Plan",
      accessorKey: "subscription_tier",
      enableSorting: true,
      cell: ({ row }) => (
        <StatusBadge status={row.original.subscription_tier} config={tenantPlanConfig} variant="outline" size="sm" />
      ),
    },
    {
      id: "created_at",
      header: "Created",
      accessorKey: "created_at",
      enableSorting: true,
      cell: ({ row }) => <RelativeTime date={row.original.created_at} />,
    },
  ];

  const rowActions = (tenant: Tenant) => {
    const actions = [
      {
        label: "View",
        onClick: (t: Tenant) => router.push(`/admin/tenants/${t.id}`),
      },
      {
        label: "Edit",
        onClick: (t: Tenant) => router.push(`/admin/tenants/${t.id}?tab=settings`),
      },
    ];

    if (tenant.status === "active") {
      actions.push({
        label: "Suspend",
        onClick: async (t: Tenant) => {
          try {
            await api.put(`/api/v1/tenants/${t.id}/status`, { status: "suspended" });
            toast.success("Tenant suspended");
            refetch();
          } catch (error) {
            toast.error(parseApiError(error));
          }
        },
      });
    } else if (tenant.status === "suspended") {
      actions.push({
        label: "Activate",
        onClick: async (t: Tenant) => {
          try {
            await api.put(`/api/v1/tenants/${t.id}/status`, { status: "active" });
            toast.success("Tenant activated");
            refetch();
          } catch (error) {
            toast.error(parseApiError(error));
          }
        },
      });
    }

    if (tenant.status !== "deprovisioned") {
      actions.push({
        label: "Deprovision",
        variant: "destructive" as const,
        onClick: (t: Tenant) => setDeprovisionTenant(t),
      } as { label: string; onClick: (t: Tenant) => void; variant?: "destructive" });
    }

    return actions;
  };

  return (
    <div className="space-y-6">
      <PageHeader
        title="Tenant Management"
        description="Manage tenants, plans, and provisioning"
        actions={
          <Button onClick={() => router.push("/admin/tenants/new")}>
            <Plus className="mr-2 h-4 w-4" />
            Provision Tenant
          </Button>
        }
      />

      <DataTable
        {...tableProps}
        columns={columns}
        filters={filters}
        rowActions={rowActions}
        onRowClick={(tenant) => router.push(`/admin/tenants/${tenant.id}`)}
        searchSlot={
          <SearchInput
            value={tableProps.searchValue ?? ""}
            onChange={tableProps.onSearchChange ?? (() => {})}
            placeholder="Search tenants..."
            loading={tableProps.isLoading}
          />
        }
        emptyState={{
          icon: Building2,
          title: "No tenants found",
          description: "Get started by provisioning your first tenant.",
          action: {
            label: "Provision Tenant",
            onClick: () => router.push("/admin/tenants/new"),
          },
        }}
      />

      {deprovisionTenant && (
        <ConfirmDialog
          open={!!deprovisionTenant}
          onOpenChange={(o) => !o && setDeprovisionTenant(null)}
          title="Deprovision Tenant"
          description={`This will permanently deprovision "${deprovisionTenant.name}" and all associated data. This action cannot be undone.`}
          confirmLabel="Deprovision"
          typeToConfirm={deprovisionTenant.name}
          variant="destructive"
          loading={deprovisionMutation.isPending}
          onConfirm={async () => {
            await deprovisionMutation.mutateAsync(deprovisionTenant.id);
            refetch();
          }}
        />
      )}
    </div>
  );
}
