"use client";

import { useState } from "react";
import { useForm, FormProvider } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { Key, Shield, Laptop, Trash2, Plus, CheckCircle } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { ConnectedAccounts } from "@/components/auth/connected-accounts";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
  CardDescription,
} from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { PageHeader } from "@/components/common/page-header";
import { FormField } from "@/components/shared/forms/form-field";
import { RelativeTime } from "@/components/shared/relative-time";
import { CopyButton } from "@/components/shared/copy-button";
import { UserAvatar } from "@/components/shared/user-avatar";
import { ConfirmDialog } from "@/components/shared/confirm-dialog";
import { useAuth } from "@/hooks/use-auth";
import { useApiQuery } from "@/hooks/use-api";
import { useApiMutation } from "@/hooks/use-api-mutation";
import api from "@/lib/api";

const passwordSchema = z
  .object({
    current_password: z.string().min(1, "Current password is required"),
    new_password: z
      .string()
      .min(12, "Must be at least 12 characters")
      .regex(
        /^(?=.*[A-Z])(?=.*[a-z])(?=.*\d)(?=.*[^A-Za-z\d]).{12,}$/,
        "Must contain uppercase, lowercase, number, and special character"
      ),
    confirm_password: z.string(),
  })
  .refine((d) => d.new_password === d.confirm_password, {
    message: "Passwords do not match",
    path: ["confirm_password"],
  });

type PasswordFormData = z.infer<typeof passwordSchema>;

interface Session {
  id: string;
  user_agent: string;
  ip_address: string;
  created_at: string;
  last_active_at: string;
  is_current: boolean;
}

interface ApiKey {
  id: string;
  name: string;
  prefix: string;
  scopes: string[];
  status: string;
  created_at: string;
  last_used_at: string | null;
  expires_at: string | null;
  created_by: string | null;
}

interface ApiKeysResponse {
  data: ApiKey[];
  meta: { page: number; per_page: number; total: number; total_pages: number };
}

export default function SettingsPage() {
  const { user } = useAuth();
  const [newKeyName, setNewKeyName] = useState("");
  const [newKeyVisible, setNewKeyVisible] = useState<string | null>(null);
  const [revokeSession, setRevokeSession] = useState<Session | null>(null);
  const [revokeKey, setRevokeKey] = useState<ApiKey | null>(null);

  const {
    data: sessionsData,
    isLoading: sessionsLoading,
    refetch: refetchSessions,
  } = useApiQuery<Session[]>(["sessions"], "/api/v1/users/me/sessions");

  const {
    data: apiKeysRaw,
    isLoading: apiKeysLoading,
    refetch: refetchKeys,
  } = useApiQuery<ApiKeysResponse>(["api-keys"], "/api/v1/api-keys");

  const passwordMethods = useForm<PasswordFormData>({
    resolver: zodResolver(passwordSchema),
    defaultValues: {
      current_password: "",
      new_password: "",
      confirm_password: "",
    },
  });

  const passwordMutation = useApiMutation<unknown, PasswordFormData>(
    "post",
    "/api/v1/users/me/password",
    {
      successMessage: "Password changed successfully",
      onSuccess: () => passwordMethods.reset(),
    }
  );

  const onPasswordSubmit = passwordMethods.handleSubmit(async (data) => {
    await passwordMutation.mutate(data);
  });

  const newPassword = passwordMethods.watch("new_password");
  const strength = {
    length: newPassword.length >= 12,
    upper: /[A-Z]/.test(newPassword),
    lower: /[a-z]/.test(newPassword),
    digit: /\d/.test(newPassword),
    special: /[^A-Za-z\d]/.test(newPassword),
  };
  const strengthScore = Object.values(strength).filter(Boolean).length;

  const handleCreateApiKey = async () => {
    if (!newKeyName.trim()) {
      toast.error("Please enter a key name");
      return;
    }
    try {
      const { data } = await api.post<{
        key: ApiKey;
        secret: string;
      }>("/api/v1/api-keys", { name: newKeyName.trim() });
      setNewKeyVisible(data.secret);
      setNewKeyName("");
      toast.success("API key created — copy it now, it won't be shown again");
      refetchKeys();
    } catch {
      toast.error("Failed to create API key");
    }
  };

  const sessions = sessionsData ?? [];
  const apiKeys = apiKeysRaw?.data ?? [];

  return (
    <div className="w-full space-y-6">
      <PageHeader
        title="Account Settings"
        description="Manage your profile, security, and API access"
      />

      {/* Profile */}
      <Card>
        <CardHeader>
          <CardTitle>Profile Information</CardTitle>
          <CardDescription>Your account details and identity</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex items-center gap-4">
            {user ? (
              <UserAvatar user={user} size="lg" />
            ) : (
              <Skeleton className="h-10 w-10 rounded-full" />
            )}
            <div>
              <p className="font-semibold">
                {user?.first_name} {user?.last_name}
              </p>
              <p className="text-sm text-muted-foreground">{user?.email}</p>
            </div>
          </div>
          <p className="text-xs text-muted-foreground">
            To update your name or avatar, contact your administrator.
          </p>
        </CardContent>
      </Card>

      {/* Change Password */}
      <Card>
        <CardHeader>
          <CardTitle>Change Password</CardTitle>
          <CardDescription>Use a strong password with 12+ characters</CardDescription>
        </CardHeader>
        <CardContent>
          <FormProvider {...passwordMethods}>
            <form onSubmit={onPasswordSubmit} className="space-y-4" noValidate>
              <FormField name="current_password" label="Current Password" required>
                <Input
                  type="password"
                  {...passwordMethods.register("current_password")}
                  disabled={passwordMutation.isPending}
                  aria-invalid={
                    !!passwordMethods.formState.errors.current_password
                  }
                />
              </FormField>
              <FormField name="new_password" label="New Password" required>
                <div className="space-y-2">
                  <Input
                    type="password"
                    {...passwordMethods.register("new_password")}
                    disabled={passwordMutation.isPending}
                    aria-invalid={
                      !!passwordMethods.formState.errors.new_password
                    }
                  />
                  {newPassword.length > 0 && (
                    <div className="flex gap-1 h-1.5">
                      {Array.from({ length: 5 }).map((_, i) => (
                        <div
                          key={i}
                          className={`flex-1 rounded-full transition-colors ${
                            i < strengthScore
                              ? strengthScore <= 2
                                ? "bg-red-500"
                                : strengthScore <= 4
                                ? "bg-yellow-500"
                                : "bg-green-500"
                              : "bg-muted"
                          }`}
                        />
                      ))}
                    </div>
                  )}
                </div>
              </FormField>
              <FormField
                name="confirm_password"
                label="Confirm New Password"
                required
              >
                <Input
                  type="password"
                  {...passwordMethods.register("confirm_password")}
                  disabled={passwordMutation.isPending}
                  aria-invalid={
                    !!passwordMethods.formState.errors.confirm_password
                  }
                />
              </FormField>
              <Button type="submit" disabled={passwordMutation.isPending}>
                {passwordMutation.isPending ? "Changing..." : "Change Password"}
              </Button>
            </form>
          </FormProvider>
        </CardContent>
      </Card>

      {/* Two-Factor Authentication */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Shield className="h-5 w-5" />
            Two-Factor Authentication
          </CardTitle>
          <CardDescription>
            Add an extra layer of security to your account
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="flex items-center justify-between">
            <div>
              <p className="font-medium text-sm">
                {user?.mfa_enabled ? "MFA is enabled" : "MFA is not enabled"}
              </p>
              <p className="text-xs text-muted-foreground mt-0.5">
                {user?.mfa_enabled
                  ? "Your account is protected with two-factor authentication."
                  : "Enable 2FA to protect your account with an authenticator app."}
              </p>
            </div>
            <Badge variant={user?.mfa_enabled ? "default" : "secondary"}>
              {user?.mfa_enabled ? "Enabled" : "Disabled"}
            </Badge>
          </div>
        </CardContent>
      </Card>

      {/* Connected Accounts (OAuth) */}
      <ConnectedAccounts />

      {/* Active Sessions */}
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div>
              <CardTitle className="flex items-center gap-2">
                <Laptop className="h-5 w-5" />
                Active Sessions
              </CardTitle>
              <CardDescription>
                Devices currently signed in to your account
              </CardDescription>
            </div>
            {sessions.filter((s) => !s.is_current).length > 0 && (
              <Button
                size="sm"
                variant="destructive"
                onClick={async () => {
                  try {
                    await api.delete("/api/v1/users/me/sessions?exclude_current=true");
                    toast.success("All other sessions revoked");
                    refetchSessions();
                  } catch {
                    toast.error("Failed to revoke sessions");
                  }
                }}
              >
                Revoke all others
              </Button>
            )}
          </div>
        </CardHeader>
        <CardContent>
          <div className="space-y-3">
            {sessionsLoading ? (
              Array.from({ length: 2 }).map((_, i) => (
                <Skeleton key={i} className="h-14 rounded" />
              ))
            ) : sessions.length === 0 ? (
              <p className="text-sm text-muted-foreground">No active sessions.</p>
            ) : (
              sessions.map((session) => (
                <div
                  key={session.id}
                  className="flex items-start justify-between gap-4 rounded-lg border border-border p-3"
                >
                  <div>
                    <div className="flex items-center gap-2">
                      <p className="text-sm font-medium">{labelSessionDevice(session.user_agent)}</p>
                      {session.is_current && (
                        <Badge variant="outline" className="text-xs">
                          Current
                        </Badge>
                      )}
                    </div>
                    <p className="text-xs text-muted-foreground mt-0.5">
                      {session.ip_address} ·{" "}
                      <RelativeTime date={session.last_active_at} />
                    </p>
                  </div>
                  {!session.is_current && (
                    <Button
                      size="sm"
                      variant="outline"
                      onClick={() => setRevokeSession(session)}
                      className="shrink-0"
                    >
                      Revoke
                    </Button>
                  )}
                </div>
              ))
            )}
          </div>
        </CardContent>
      </Card>

      {/* API Keys */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Key className="h-5 w-5" />
            API Keys
          </CardTitle>
          <CardDescription>
            Manage programmatic access to the platform
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          {newKeyVisible && (
            <div className="rounded-lg border border-green-300 bg-green-50 dark:bg-green-900/20 p-4 space-y-2">
              <p className="text-sm font-medium text-green-800 dark:text-green-200 flex items-center gap-2">
                <CheckCircle className="h-4 w-4" />
                Copy your new API key — it won&apos;t be shown again
              </p>
              <div className="flex items-center gap-2">
                <code className="flex-1 text-xs font-mono bg-white dark:bg-gray-900 border rounded px-3 py-2 overflow-auto">
                  {newKeyVisible}
                </code>
                <CopyButton value={newKeyVisible} label="Copy API key" />
              </div>
              <Button
                size="sm"
                variant="outline"
                onClick={() => setNewKeyVisible(null)}
              >
                Done
              </Button>
            </div>
          )}

          {/* Create new key */}
          <div className="flex gap-2">
            <Input
              value={newKeyName}
              onChange={(e) => setNewKeyName(e.target.value)}
              placeholder="Key name (e.g. CI/CD Pipeline)"
              className="flex-1"
              onKeyDown={(e) => {
                if (e.key === "Enter") {
                  e.preventDefault();
                  handleCreateApiKey();
                }
              }}
            />
            <Button
              onClick={handleCreateApiKey}
              disabled={!newKeyName.trim()}
            >
              <Plus className="mr-2 h-4 w-4" />
              Create
            </Button>
          </div>

          {/* Key list */}
          <div className="space-y-2">
            {apiKeysLoading ? (
              Array.from({ length: 2 }).map((_, i) => (
                <Skeleton key={i} className="h-12 rounded" />
              ))
            ) : apiKeys.length === 0 ? (
              <p className="text-sm text-muted-foreground">No API keys yet.</p>
            ) : (
              apiKeys.map((key) => (
                <div
                  key={key.id}
                  className="flex items-center justify-between gap-4 rounded-lg border border-border p-3"
                >
                  <div>
                    <p className="text-sm font-medium">{key.name}</p>
                    <p className="text-xs text-muted-foreground font-mono mt-0.5">
                      {key.prefix}••••••••
                    </p>
                    <p className="text-xs text-muted-foreground mt-0.5">
                      Created <RelativeTime date={key.created_at} />
                      {key.last_used_at && (
                        <>
                          {" "}
                          · Last used <RelativeTime date={key.last_used_at} />
                        </>
                      )}
                    </p>
                  </div>
                  <Button
                    size="sm"
                    variant="destructive"
                    onClick={() => setRevokeKey(key)}
                  >
                    <Trash2 className="h-4 w-4" />
                  </Button>
                </div>
              ))
            )}
          </div>
        </CardContent>
      </Card>

      {/* Confirm revoke session */}
      {revokeSession && (
        <ConfirmDialog
          open={!!revokeSession}
          onOpenChange={(o) => !o && setRevokeSession(null)}
          title="Revoke Session"
          description={`Sign out session from ${labelSessionDevice(revokeSession.user_agent)} (${revokeSession.ip_address})?`}
          confirmLabel="Revoke"
          variant="destructive"
          onConfirm={async () => {
            await api.delete(`/api/v1/users/me/sessions/${revokeSession.id}`);
            toast.success("Session revoked");
            refetchSessions();
          }}
        />
      )}

      {/* Confirm revoke API key */}
      {revokeKey && (
        <ConfirmDialog
          open={!!revokeKey}
          onOpenChange={(o) => !o && setRevokeKey(null)}
          title="Revoke API Key"
          description={`Revoke "${revokeKey.name}"? Any services using this key will lose access immediately.`}
          confirmLabel="Revoke"
          variant="destructive"
          onConfirm={async () => {
            await api.delete(`/api/v1/api-keys/${revokeKey.id}`);
            toast.success("API key revoked");
            refetchKeys();
          }}
        />
      )}
    </div>
  );
}

function labelSessionDevice(userAgent: string): string {
  const value = userAgent.trim();
  if (!value) {
    return "Unknown device";
  }
  if (value.length <= 80) {
    return value;
  }
  return `${value.slice(0, 77)}...`;
}
