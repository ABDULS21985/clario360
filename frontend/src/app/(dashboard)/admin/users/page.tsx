"use client";

import { useState } from "react";
import { Plus, Trash2, Users } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { PageHeader } from "@/components/common/page-header";
import { DataTable } from "@/components/shared/data-table/data-table";
import { SearchInput } from "@/components/shared/forms/search-input";
import { StatusBadge } from "@/components/shared/status-badge";
import { RelativeTime } from "@/components/shared/relative-time";
import { UserAvatar } from "@/components/shared/user-avatar";
import { CopyButton } from "@/components/shared/copy-button";
import { ConfirmDialog } from "@/components/shared/confirm-dialog";
import { userStatusConfig } from "@/lib/status-configs";
import { useDataTable } from "@/hooks/use-data-table";
import api from "@/lib/api";
import type { ColumnDef } from "@tanstack/react-table";
import type { User } from "@/types/models";
import type { PaginatedResponse } from "@/types/api";
import type { FilterConfig, BulkAction } from "@/types/table";
import { UserCreateDialog } from "./_components/user-create-dialog";
import { UserEditDialog } from "./_components/user-edit-dialog";
import { UserDetailPanel } from "./_components/user-detail-panel";
import { UserRoleAssignDialog } from "./_components/user-role-assign-dialog";
import { CheckCircle, Ban, ShieldCheck } from "lucide-react";

async function fetchUsers(params: {
  page: number;
  per_page: number;
  sort?: string;
  order?: string;
  search?: string;
  filters?: Record<string, string | string[]>;
}): Promise<PaginatedResponse<User>> {
  const { data } = await api.get<PaginatedResponse<User>>("/api/v1/users", {
    params: {
      page: params.page,
      per_page: params.per_page,
      sort: params.sort,
      order: params.order,
      search: params.search || undefined,
      status: params.filters?.status,
      role: params.filters?.role,
    },
  });
  return data;
}

export default function UserManagementPage() {
  const [createOpen, setCreateOpen] = useState(false);
  const [editUser, setEditUser] = useState<User | null>(null);
  const [detailUser, setDetailUser] = useState<User | null>(null);
  const [assignRoleUser, setAssignRoleUser] = useState<User | null>(null);
  const [deleteUser, setDeleteUser] = useState<User | null>(null);
  const [selectedIds, setSelectedIds] = useState<string[]>([]);
  const [bulkDeleteOpen, setBulkDeleteOpen] = useState(false);

  const { tableProps, refetch } = useDataTable<User>({
    fetchFn: fetchUsers,
    queryKey: "users",
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
        { label: "Suspended", value: "suspended" },
        { label: "Deactivated", value: "deactivated" },
        { label: "Pending", value: "pending_verification" },
      ],
    },
  ];

  const columns: ColumnDef<User>[] = [
    {
      id: "select",
      header: ({ table }) => (
        <input
          type="checkbox"
          className="h-4 w-4 rounded border-input"
          checked={table.getIsAllPageRowsSelected()}
          onChange={(e) => table.toggleAllPageRowsSelected(e.target.checked)}
          aria-label="Select all"
        />
      ),
      cell: ({ row }) => (
        <input
          type="checkbox"
          className="h-4 w-4 rounded border-input"
          checked={row.getIsSelected()}
          onChange={(e) => row.toggleSelected(e.target.checked)}
          onClick={(e) => e.stopPropagation()}
          aria-label="Select row"
        />
      ),
      enableSorting: false,
      enableHiding: false,
      size: 40,
    },
    {
      id: "name",
      header: "Name",
      enableSorting: true,
      cell: ({ row }) => {
        const user = row.original;
        return (
          <div className="flex items-center gap-2">
            <UserAvatar user={user} size="sm" />
            <button
              className="font-medium text-sm hover:underline text-left"
              onClick={(e) => {
                e.stopPropagation();
                setDetailUser(user);
              }}
            >
              {user.first_name} {user.last_name}
            </button>
          </div>
        );
      },
    },
    {
      id: "email",
      header: "Email",
      accessorKey: "email",
      enableSorting: true,
      cell: ({ row }) => (
        <div className="flex items-center gap-1 group">
          <span className="text-sm text-muted-foreground">{row.original.email}</span>
          <CopyButton value={row.original.email} label="Copy email" />
        </div>
      ),
    },
    {
      id: "roles",
      header: "Roles",
      enableSorting: false,
      cell: ({ row }) => (
        <div className="flex flex-wrap gap-1 max-w-[200px]">
          {(row.original.roles ?? []).length === 0 ? (
            <span className="text-xs text-muted-foreground">No roles</span>
          ) : (
            (row.original.roles ?? []).slice(0, 2).map((role) => (
              <span
                key={role.id}
                className="inline-flex items-center rounded-full bg-secondary text-secondary-foreground px-2 py-0.5 text-xs font-medium"
              >
                {role.name}
              </span>
            ))
          )}
          {(row.original.roles ?? []).length > 2 && (
            <span className="text-xs text-muted-foreground">
              +{(row.original.roles ?? []).length - 2}
            </span>
          )}
        </div>
      ),
    },
    {
      id: "status",
      header: "Status",
      accessorKey: "status",
      enableSorting: true,
      cell: ({ row }) => (
        <StatusBadge status={row.original.status} config={userStatusConfig} size="sm" />
      ),
    },
    {
      id: "mfa_enabled",
      header: "MFA",
      accessorKey: "mfa_enabled",
      enableSorting: false,
      cell: ({ row }) =>
        row.original.mfa_enabled ? (
          <CheckCircle className="h-4 w-4 text-green-600" aria-label="MFA enabled" />
        ) : (
          <span className="text-xs text-muted-foreground">—</span>
        ),
    },
    {
      id: "last_login_at",
      header: "Last Login",
      accessorKey: "last_login_at",
      enableSorting: true,
      cell: ({ row }) =>
        row.original.last_login_at ? (
          <RelativeTime date={row.original.last_login_at} />
        ) : (
          <span className="text-xs text-muted-foreground">Never</span>
        ),
    },
  ];

  const rowActions = (user: User) => [
    {
      label: "Edit",
      icon: undefined,
      onClick: (u: User) => setEditUser(u),
    },
    {
      label: "Assign Roles",
      icon: ShieldCheck,
      onClick: (u: User) => setAssignRoleUser(u),
    },
    {
      label: user.status === "active" ? "Suspend" : "Activate",
      icon: user.status === "active" ? Ban : CheckCircle,
      variant: user.status === "active" ? ("destructive" as const) : ("default" as const),
      onClick: async (u: User) => {
        const newStatus = u.status === "active" ? "suspended" : "active";
        try {
          await api.put(`/api/v1/users/${u.id}/status`, { status: newStatus });
          toast.success(
            `User ${newStatus === "active" ? "activated" : "suspended"}`
          );
          refetch();
        } catch {
          toast.error("Failed to update user status");
        }
      },
    },
    {
      label: "Delete",
      icon: Trash2,
      variant: "destructive" as const,
      onClick: (u: User) => setDeleteUser(u),
    },
  ];

  const bulkActions: BulkAction[] = [
    {
      label: "Suspend Selected",
      icon: Ban,
      variant: "destructive",
      onClick: async (ids) => {
        await Promise.all(
          ids.map((id) =>
            api.put(`/api/v1/users/${id}/status`, { status: "suspended" })
          )
        );
        toast.success(`${ids.length} users suspended`);
        refetch();
      },
    },
    {
      label: "Delete Selected",
      icon: Trash2,
      variant: "destructive",
      onClick: async () => {
        setBulkDeleteOpen(true);
      },
    },
  ];

  return (
    <div className="space-y-6">
      <PageHeader
        title="User Management"
        description="Manage users, roles, and permissions"
        actions={
          <Button onClick={() => setCreateOpen(true)}>
            <Plus className="mr-2 h-4 w-4" />
            Add User
          </Button>
        }
      />

      <DataTable
        {...tableProps}
        columns={columns}
        filters={filters}
        enableSelection
        onSelectionChange={setSelectedIds}
        rowActions={rowActions}
        onRowClick={(user) => setDetailUser(user)}
        searchSlot={
          <SearchInput
            value={tableProps.searchValue ?? ""}
            onChange={tableProps.onSearchChange ?? (() => {})}
            placeholder="Search users..."
            loading={tableProps.isLoading}
          />
        }
        bulkActions={bulkActions}
        enableExport
        onExport={(format) => {
          toast.info(`Exporting as ${format.toUpperCase()}...`);
        }}
        emptyState={{
          icon: Users,
          title: "No users found",
          description: "Get started by adding your first user.",
          action: { label: "Add User", onClick: () => setCreateOpen(true) },
        }}
      />

      <UserCreateDialog
        open={createOpen}
        onOpenChange={setCreateOpen}
        onSuccess={refetch}
      />

      {editUser && (
        <UserEditDialog
          user={editUser}
          open={!!editUser}
          onOpenChange={(o) => !o && setEditUser(null)}
          onSuccess={refetch}
        />
      )}

      {detailUser && (
        <UserDetailPanel
          user={detailUser}
          open={!!detailUser}
          onClose={() => setDetailUser(null)}
          onEdit={() => { setEditUser(detailUser); setDetailUser(null); }}
          onAssignRoles={() => { setAssignRoleUser(detailUser); setDetailUser(null); }}
        />
      )}

      {assignRoleUser && (
        <UserRoleAssignDialog
          user={assignRoleUser}
          open={!!assignRoleUser}
          onOpenChange={(o) => { if (!o) setAssignRoleUser(null); }}
          onSuccess={refetch}
        />
      )}

      {deleteUser && (
        <ConfirmDialog
          open={!!deleteUser}
          onOpenChange={(o) => !o && setDeleteUser(null)}
          title="Delete User"
          description={`Are you sure you want to delete ${deleteUser.first_name} ${deleteUser.last_name}? This action cannot be undone.`}
          confirmLabel="Delete"
          typeToConfirm="DELETE"
          variant="destructive"
          onConfirm={async () => {
            await api.delete(`/api/v1/users/${deleteUser.id}`);
            toast.success("User deleted");
            refetch();
          }}
        />
      )}

      <ConfirmDialog
        open={bulkDeleteOpen}
        onOpenChange={setBulkDeleteOpen}
        title={`Delete ${selectedIds.length} Users`}
        description="This will permanently delete all selected users. This action cannot be undone."
        confirmLabel="Delete All"
        typeToConfirm="DELETE"
        variant="destructive"
        onConfirm={async () => {
          await Promise.all(selectedIds.map((id) => api.delete(`/api/v1/users/${id}`)));
          toast.success(`${selectedIds.length} users deleted`);
          refetch();
        }}
      />
    </div>
  );
}
