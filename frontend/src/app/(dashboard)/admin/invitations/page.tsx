"use client";

import { useState } from "react";
import {
  Mail,
  Plus,
  Send,
  Trash2,
  Users,
  Clock,
  CheckCircle,
  XCircle,
} from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { PageHeader } from "@/components/common/page-header";
import { DataTable } from "@/components/shared/data-table/data-table";
import { SearchInput } from "@/components/shared/forms/search-input";
import { StatusBadge } from "@/components/shared/status-badge";
import { RelativeTime } from "@/components/shared/relative-time";
import { KpiCard } from "@/components/shared/kpi-card";
import { ConfirmDialog } from "@/components/shared/confirm-dialog";
import { invitationStatusConfig } from "@/lib/status-configs";
import { formatNumber, formatPercentage } from "@/lib/format";
import { useDataTable } from "@/hooks/use-data-table";
import {
  useInvitationStats,
  useResendInvitation,
  useDeleteInvitation,
} from "@/hooks/use-invitations";
import api from "@/lib/api";
import type { ColumnDef } from "@tanstack/react-table";
import type { PaginatedResponse } from "@/types/api";
import type { Invitation } from "@/types/invitation";
import type { FilterConfig } from "@/types/table";
import { InviteUserDialog } from "./_components/invite-user-dialog";

async function fetchInvitations(params: {
  page: number;
  per_page: number;
  sort?: string;
  order?: string;
  search?: string;
  filters?: Record<string, string | string[]>;
}): Promise<PaginatedResponse<Invitation>> {
  const { data } = await api.get<PaginatedResponse<Invitation>>("/api/v1/invitations", {
    params: {
      page: params.page,
      per_page: params.per_page,
      sort: params.sort,
      order: params.order,
      search: params.search || undefined,
      status: params.filters?.status,
    },
  });
  return data;
}

export default function InvitationsPage() {
  const [inviteOpen, setInviteOpen] = useState(false);
  const [deleteInvite, setDeleteInvite] = useState<Invitation | null>(null);

  const { data: stats, isLoading: statsLoading } = useInvitationStats();
  const resendMutation = useResendInvitation();
  const deleteMutation = useDeleteInvitation();

  const { tableProps, refetch } = useDataTable<Invitation>({
    fetchFn: fetchInvitations,
    queryKey: "invitations",
    defaultPageSize: 25,
    defaultSort: { column: "created_at", direction: "desc" },
  });

  const filters: FilterConfig[] = [
    {
      key: "status",
      label: "Status",
      type: "multi-select",
      options: [
        { label: "Pending", value: "pending" },
        { label: "Accepted", value: "accepted" },
        { label: "Expired", value: "expired" },
        { label: "Cancelled", value: "cancelled" },
      ],
    },
  ];

  const columns: ColumnDef<Invitation>[] = [
    {
      id: "email",
      header: "Email",
      accessorKey: "email",
      enableSorting: true,
      cell: ({ row }) => (
        <span className="text-sm font-medium">{row.original.email}</span>
      ),
    },
    {
      id: "role_name",
      header: "Role",
      accessorKey: "role_name",
      enableSorting: true,
      cell: ({ row }) => (
        <span className="inline-flex items-center rounded-full bg-secondary text-secondary-foreground px-2 py-0.5 text-xs font-medium">
          {row.original.role_name}
        </span>
      ),
    },
    {
      id: "status",
      header: "Status",
      accessorKey: "status",
      enableSorting: true,
      cell: ({ row }) => (
        <StatusBadge status={row.original.status} config={invitationStatusConfig} size="sm" />
      ),
    },
    {
      id: "invited_by_name",
      header: "Invited By",
      accessorKey: "invited_by_name",
      enableSorting: false,
      cell: ({ row }) => (
        <span className="text-sm text-muted-foreground">{row.original.invited_by_name}</span>
      ),
    },
    {
      id: "expires_at",
      header: "Expires",
      accessorKey: "expires_at",
      enableSorting: true,
      cell: ({ row }) => <RelativeTime date={row.original.expires_at} />,
    },
    {
      id: "created_at",
      header: "Sent",
      accessorKey: "created_at",
      enableSorting: true,
      cell: ({ row }) => <RelativeTime date={row.original.created_at} />,
    },
  ];

  const rowActions = (invitation: Invitation) => {
    const actions: Array<{
      label: string;
      icon?: typeof Send;
      variant?: "destructive";
      onClick: (inv: Invitation) => void;
      hidden?: (inv: Invitation) => boolean;
    }> = [];

    if (invitation.status === "pending") {
      actions.push({
        label: "Resend",
        icon: Send,
        onClick: async (inv: Invitation) => {
          await resendMutation.mutateAsync(inv.id);
        },
      });
    }

    if (invitation.status === "pending" || invitation.status === "expired") {
      actions.push({
        label: "Cancel",
        icon: Trash2,
        variant: "destructive",
        onClick: (inv: Invitation) => setDeleteInvite(inv),
      });
    }

    return actions;
  };

  return (
    <div className="space-y-6">
      <PageHeader
        title="Invitations"
        description="Manage user invitations to the platform"
        actions={
          <Button onClick={() => setInviteOpen(true)}>
            <Plus className="mr-2 h-4 w-4" />
            Invite User
          </Button>
        }
      />

      {/* Stats Cards */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        <KpiCard
          title="Total Sent"
          value={formatNumber(stats?.total_sent ?? 0)}
          icon={Mail}
          loading={statsLoading}
        />
        <KpiCard
          title="Pending"
          value={formatNumber(stats?.pending ?? 0)}
          icon={Clock}
          iconColor="text-yellow-600"
          loading={statsLoading}
        />
        <KpiCard
          title="Accepted"
          value={formatNumber(stats?.accepted ?? 0)}
          icon={CheckCircle}
          iconColor="text-green-600"
          loading={statsLoading}
        />
        <KpiCard
          title="Acceptance Rate"
          value={stats ? formatPercentage(stats.acceptance_rate) : "—"}
          icon={Users}
          loading={statsLoading}
        />
      </div>

      <DataTable
        {...tableProps}
        columns={columns}
        filters={filters}
        rowActions={rowActions}
        searchSlot={
          <SearchInput
            value={tableProps.searchValue ?? ""}
            onChange={tableProps.onSearchChange ?? (() => {})}
            placeholder="Search invitations..."
            loading={tableProps.isLoading}
          />
        }
        emptyState={{
          icon: Mail,
          title: "No invitations yet",
          description: "Invite users to join the platform.",
          action: { label: "Invite User", onClick: () => setInviteOpen(true) },
        }}
      />

      <InviteUserDialog
        open={inviteOpen}
        onOpenChange={setInviteOpen}
        onSuccess={refetch}
      />

      {deleteInvite && (
        <ConfirmDialog
          open={!!deleteInvite}
          onOpenChange={(o) => !o && setDeleteInvite(null)}
          title="Cancel Invitation"
          description={`Cancel the invitation to ${deleteInvite.email}? They will no longer be able to accept it.`}
          confirmLabel="Cancel Invitation"
          variant="destructive"
          loading={deleteMutation.isPending}
          onConfirm={async () => {
            await deleteMutation.mutateAsync(deleteInvite.id);
            refetch();
          }}
        />
      )}
    </div>
  );
}
