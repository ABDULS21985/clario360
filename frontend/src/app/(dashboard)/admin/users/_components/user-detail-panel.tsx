"use client";

import { ShieldCheck, Edit, Ban, CheckCircle, Clock } from "lucide-react";
import { DetailPanel } from "@/components/shared/detail-panel";
import { UserAvatar } from "@/components/shared/user-avatar";
import { StatusBadge } from "@/components/shared/status-badge";
import { RelativeTime } from "@/components/shared/relative-time";
import { Timeline } from "@/components/shared/timeline";
import { Button } from "@/components/ui/button";
import { Separator } from "@/components/ui/separator";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { userStatusConfig } from "@/lib/status-configs";
import { useApiQuery } from "@/hooks/use-api";
import { useApiMutation } from "@/hooks/use-api-mutation";
import type { User, AuditLog } from "@/types/models";
import type { PaginatedResponse } from "@/types/api";

interface UserDetailPanelProps {
  user: User;
  open: boolean;
  /** Called when the panel is closed */
  onClose: () => void;
  onEdit: () => void;
  onAssignRoles: () => void;
}

export function UserDetailPanel({
  user,
  open,
  onClose,
  onEdit,
  onAssignRoles,
}: UserDetailPanelProps) {
  const { data: auditData, isLoading: auditLoading } = useApiQuery<PaginatedResponse<AuditLog>>(
    ["audit-logs", "user", user.id],
    "/api/v1/audit/logs",
    { enabled: open }
  );

  const statusMutation = useApiMutation<unknown, { status: string }>(
    "put",
    `/api/v1/users/${user.id}/status`,
    { successMessage: "User status updated", invalidateKeys: ["users"] }
  );

  const toggleStatus = async () => {
    const newStatus = user.status === "active" ? "suspended" : "active";
    await statusMutation.mutate({ status: newStatus });
  };

  const auditItems =
    auditData?.data?.slice(0, 10).map((log) => ({
      id: log.id,
      title: log.action,
      description: `${log.resource_type}${log.resource_id ? ` · ${log.resource_id.slice(0, 8)}` : ""}`,
      timestamp: new Date(log.created_at).toLocaleString(),
    })) ?? [];

  return (
    <DetailPanel
      open={open}
      onOpenChange={(o) => {
        if (!o) onClose();
      }}
      title="User Profile"
      description={`${user.first_name} ${user.last_name}`}
      width="lg"
    >
      <div className="space-y-6">
        {/* Profile */}
        <div className="flex items-start gap-4">
          <UserAvatar user={user} size="lg" />
          <div className="flex-1 min-w-0">
            <h3 className="text-base font-semibold">
              {user.first_name} {user.last_name}
            </h3>
            <p className="text-sm text-muted-foreground">{user.email}</p>
            <div className="flex items-center gap-2 mt-2">
              <StatusBadge status={user.status} config={userStatusConfig} size="sm" />
              {user.mfa_enabled && (
                <span className="inline-flex items-center gap-1 text-xs text-green-600">
                  <ShieldCheck className="h-3 w-3" />
                  MFA
                </span>
              )}
            </div>
          </div>
        </div>

        <div className="flex flex-wrap gap-2">
          <Button size="sm" variant="outline" onClick={onEdit}>
            <Edit className="mr-2 h-4 w-4" />
            Edit
          </Button>
          <Button size="sm" variant="outline" onClick={onAssignRoles}>
            <ShieldCheck className="mr-2 h-4 w-4" />
            Manage Roles
          </Button>
          <Button
            size="sm"
            variant={user.status === "active" ? "destructive" : "default"}
            onClick={toggleStatus}
            disabled={statusMutation.isPending}
          >
            {user.status === "active" ? (
              <>
                <Ban className="mr-2 h-4 w-4" />
                Suspend
              </>
            ) : (
              <>
                <CheckCircle className="mr-2 h-4 w-4" />
                Activate
              </>
            )}
          </Button>
        </div>

        <Separator />

        {/* Roles */}
        <div className="space-y-2">
          <h4 className="text-sm font-semibold">Assigned Roles</h4>
          {user.roles.length === 0 ? (
            <p className="text-sm text-muted-foreground">No roles assigned.</p>
          ) : (
            <div className="flex flex-wrap gap-2">
              {user.roles.map((role) => (
                <Badge key={role.id} variant="secondary" className="gap-1">
                  {role.is_system && <ShieldCheck className="h-3 w-3" />}
                  {role.name}
                </Badge>
              ))}
            </div>
          )}
        </div>

        <Separator />

        {/* Info grid */}
        <div className="grid grid-cols-2 gap-3 text-sm">
          <div>
            <p className="text-xs text-muted-foreground font-medium uppercase tracking-wide">
              Last Login
            </p>
            <p className="mt-0.5">
              {user.last_login_at ? <RelativeTime date={user.last_login_at} /> : "Never"}
            </p>
          </div>
          <div>
            <p className="text-xs text-muted-foreground font-medium uppercase tracking-wide">
              Member Since
            </p>
            <p className="mt-0.5">
              <RelativeTime date={user.created_at} />
            </p>
          </div>
          <div>
            <p className="text-xs text-muted-foreground font-medium uppercase tracking-wide">
              MFA
            </p>
            <p className="mt-0.5">{user.mfa_enabled ? "Enabled" : "Disabled"}</p>
          </div>
        </div>

        <Separator />

        {/* Recent Activity */}
        <div className="space-y-3">
          <h4 className="text-sm font-semibold flex items-center gap-2">
            <Clock className="h-4 w-4 text-muted-foreground" />
            Recent Activity
          </h4>
          {auditLoading ? (
            <div className="space-y-2">
              {Array.from({ length: 4 }).map((_, i) => (
                <Skeleton key={i} className="h-10 rounded" />
              ))}
            </div>
          ) : auditItems.length === 0 ? (
            <p className="text-sm text-muted-foreground">No recent activity.</p>
          ) : (
            <Timeline items={auditItems} />
          )}
        </div>
      </div>
    </DetailPanel>
  );
}
