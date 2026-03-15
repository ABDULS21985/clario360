"use client";

import { useState } from "react";
import {
  Key,
  Plus,
  RotateCw,
  Trash2,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { PageHeader } from "@/components/common/page-header";
import { DataTable } from "@/components/shared/data-table/data-table";
import { SearchInput } from "@/components/shared/forms/search-input";
import { StatusBadge } from "@/components/shared/status-badge";
import { RelativeTime } from "@/components/shared/relative-time";
import { ConfirmDialog } from "@/components/shared/confirm-dialog";
import { apiKeyStatusConfig } from "@/lib/status-configs";
import { useDataTable } from "@/hooks/use-data-table";
import { useRevokeApiKey, useRotateApiKey } from "@/hooks/use-api-keys";
import api from "@/lib/api";
import type { ColumnDef } from "@tanstack/react-table";
import type { PaginatedResponse } from "@/types/api";
import type { ApiKey } from "@/types/api-key";
import type { FilterConfig } from "@/types/table";
import { CreateKeyDialog } from "./_components/create-key-dialog";
import { KeySecretDialog } from "./_components/key-secret-dialog";

async function fetchApiKeys(params: {
  page: number;
  per_page: number;
  sort?: string;
  order?: string;
  search?: string;
  filters?: Record<string, string | string[]>;
}): Promise<PaginatedResponse<ApiKey>> {
  const { data } = await api.get<PaginatedResponse<ApiKey>>("/api/v1/api-keys", {
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

export default function ApiKeysPage() {
  const [createOpen, setCreateOpen] = useState(false);
  const [secretValue, setSecretValue] = useState<string | null>(null);
  const [revokeKey, setRevokeKey] = useState<ApiKey | null>(null);
  const [rotateKey, setRotateKey] = useState<ApiKey | null>(null);

  const revokeMutation = useRevokeApiKey();
  const rotateMutation = useRotateApiKey();

  const { tableProps, refetch } = useDataTable<ApiKey>({
    fetchFn: fetchApiKeys,
    queryKey: "api-keys-admin",
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
        { label: "Revoked", value: "revoked" },
        { label: "Expired", value: "expired" },
      ],
    },
  ];

  const columns: ColumnDef<ApiKey>[] = [
    {
      id: "name",
      header: "Name",
      accessorKey: "name",
      enableSorting: true,
      cell: ({ row }) => (
        <span className="font-medium text-sm">{row.original.name}</span>
      ),
    },
    {
      id: "prefix",
      header: "Key",
      accessorKey: "prefix",
      enableSorting: false,
      cell: ({ row }) => (
        <code className="text-xs font-mono text-muted-foreground">
          {row.original.prefix}••••••••
        </code>
      ),
    },
    {
      id: "scopes",
      header: "Scopes",
      enableSorting: false,
      cell: ({ row }) => {
        const scopes = row.original.scopes;
        const displayed = scopes.slice(0, 3);
        const extra = scopes.length - 3;
        return (
          <div className="flex flex-wrap gap-1 max-w-[250px]">
            {displayed.map((scope) => (
              <span
                key={scope}
                className="inline-flex items-center rounded-full bg-secondary text-secondary-foreground px-2 py-0.5 text-xs font-mono"
              >
                {scope}
              </span>
            ))}
            {extra > 0 && (
              <span className="text-xs text-muted-foreground">
                +{extra} more
              </span>
            )}
          </div>
        );
      },
    },
    {
      id: "status",
      header: "Status",
      accessorKey: "status",
      enableSorting: true,
      cell: ({ row }) => (
        <StatusBadge status={row.original.status} config={apiKeyStatusConfig} size="sm" />
      ),
    },
    {
      id: "last_used_at",
      header: "Last Used",
      accessorKey: "last_used_at",
      enableSorting: true,
      cell: ({ row }) =>
        row.original.last_used_at ? (
          <RelativeTime date={row.original.last_used_at} />
        ) : (
          <span className="text-xs text-muted-foreground">Never</span>
        ),
    },
    {
      id: "expires_at",
      header: "Expires",
      accessorKey: "expires_at",
      enableSorting: true,
      cell: ({ row }) =>
        row.original.expires_at ? (
          <RelativeTime date={row.original.expires_at} />
        ) : (
          <span className="text-xs text-muted-foreground">Never</span>
        ),
    },
  ];

  const rowActions = (key: ApiKey) => {
    const actions: { label: string; icon: typeof Trash2; onClick: (k: ApiKey) => void; variant?: "destructive" }[] = [];

    if (key.status === "active") {
      actions.push(
        {
          label: "Rotate",
          icon: RotateCw,
          onClick: (k: ApiKey) => setRotateKey(k),
        },
        {
          label: "Revoke",
          icon: Trash2,
          variant: "destructive" as const,
          onClick: (k: ApiKey) => setRevokeKey(k),
        },
      );
    }

    return actions;
  };

  return (
    <div className="space-y-6">
      <PageHeader
        title="API Keys"
        description="Manage API keys for programmatic platform access"
        actions={
          <Button onClick={() => setCreateOpen(true)}>
            <Plus className="mr-2 h-4 w-4" />
            Create API Key
          </Button>
        }
      />

      <DataTable
        {...tableProps}
        columns={columns}
        filters={filters}
        rowActions={rowActions}
        searchSlot={
          <SearchInput
            value={tableProps.searchValue ?? ""}
            onChange={tableProps.onSearchChange ?? (() => {})}
            placeholder="Search API keys..."
            loading={tableProps.isLoading}
          />
        }
        emptyState={{
          icon: Key,
          title: "No API keys yet",
          description: "Create your first API key to enable programmatic access.",
          action: { label: "Create API Key", onClick: () => setCreateOpen(true) },
        }}
      />

      <CreateKeyDialog
        open={createOpen}
        onOpenChange={setCreateOpen}
        onCreated={(secret) => {
          setSecretValue(secret);
          refetch();
        }}
      />

      <KeySecretDialog
        open={!!secretValue}
        onOpenChange={(o) => !o && setSecretValue(null)}
        secret={secretValue ?? ""}
      />

      {revokeKey && (
        <ConfirmDialog
          open={!!revokeKey}
          onOpenChange={(o) => !o && setRevokeKey(null)}
          title="Revoke API Key"
          description={`Revoke "${revokeKey.name}"? Any services using this key will lose access immediately.`}
          confirmLabel="Revoke"
          variant="destructive"
          loading={revokeMutation.isPending}
          onConfirm={async () => {
            await revokeMutation.mutateAsync(revokeKey.id);
            refetch();
          }}
        />
      )}

      {rotateKey && (
        <ConfirmDialog
          open={!!rotateKey}
          onOpenChange={(o) => !o && setRotateKey(null)}
          title="Rotate API Key"
          description={`Rotate the secret for "${rotateKey.name}"? The current secret will be immediately invalidated.`}
          confirmLabel="Rotate"
          variant="default"
          loading={rotateMutation.isPending}
          onConfirm={async () => {
            const result = await rotateMutation.mutateAsync(rotateKey.id);
            setSecretValue(result.secret);
            setRotateKey(null);
            refetch();
          }}
        />
      )}
    </div>
  );
}
