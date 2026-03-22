'use client';

import { useState } from 'react';
import { useForm, Controller } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { toast } from 'sonner';
import { Key, Plus, Copy, AlertTriangle } from 'lucide-react';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '@/components/ui/dialog';
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Badge } from '@/components/ui/badge';
import { Skeleton } from '@/components/ui/skeleton';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { ConfirmDialog } from '@/components/shared/confirm-dialog';
import { PermissionTree } from '@/app/(dashboard)/admin/roles/_components/permission-tree';
import { createApiKeySchema, type CreateApiKeyFormData } from '@/lib/validators/settings-validators';
import { apiGet, apiPost, apiDelete } from '@/lib/api';
import { isApiError } from '@/types/api';
import { formatDate, copyToClipboard } from '@/lib/utils';
import { RelativeTime } from '@/components/shared/relative-time';

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

interface CreateApiKeyResponse {
  key: ApiKey;
  secret: string;
}

function ApiKeyCreateDialog({
  open,
  onOpenChange,
  onSuccess,
}: {
  open: boolean;
  onOpenChange: (o: boolean) => void;
  onSuccess: () => void;
}) {
  const [loading, setLoading] = useState(false);
  const [createdKey, setCreatedKey] = useState<string | null>(null);
  const [confirmClose, setConfirmClose] = useState(false);

  const {
    register,
    handleSubmit,
    reset,
    control,
    formState: { errors },
  } = useForm<CreateApiKeyFormData>({
    resolver: zodResolver(createApiKeySchema),
    defaultValues: { scopes: [], no_expiry: true },
  });

  const handleClose = (o: boolean) => {
    if (!o && createdKey) {
      setConfirmClose(true);
      return;
    }
    if (!o) {
      reset();
      setCreatedKey(null);
    }
    onOpenChange(o);
  };

  const onSubmit = async (data: CreateApiKeyFormData) => {
    setLoading(true);
    try {
      const res = await apiPost<CreateApiKeyResponse>('/api/v1/api-keys', {
        name: data.name,
        scopes: data.scopes,
        expires_at: data.no_expiry ? null : data.expires_at,
      });
      setCreatedKey(res.secret);
      onSuccess();
    } catch (err) {
      const msg = isApiError(err) ? err.message : 'Failed to create API key.';
      toast.error(msg);
    } finally {
      setLoading(false);
    }
  };

  return (
    <>
      <Dialog open={open} onOpenChange={handleClose}>
        <DialogContent className="sm:max-w-lg max-h-[90vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle>Create API Key</DialogTitle>
            <DialogDescription>
              API keys provide programmatic access to the platform.
            </DialogDescription>
          </DialogHeader>

          {createdKey ? (
            <div className="space-y-4">
              <Alert>
                <AlertTriangle className="h-4 w-4" />
                <AlertDescription>
                  Copy your API key now. It won&apos;t be shown again.
                </AlertDescription>
              </Alert>
              <div className="rounded-md bg-muted p-3 font-mono text-sm break-all flex items-start gap-2">
                <span className="flex-1">{createdKey}</span>
                <button
                  onClick={async () => {
                    await copyToClipboard(createdKey);
                    toast.success('API key copied.');
                  }}
                  className="shrink-0 text-muted-foreground hover:text-foreground"
                >
                  <Copy className="h-4 w-4" />
                </button>
              </div>
              <DialogFooter>
                <Button onClick={() => { setCreatedKey(null); reset(); onOpenChange(false); }}>
                  Done
                </Button>
              </DialogFooter>
            </div>
          ) : (
            <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="key-name">Name *</Label>
                <Input
                  id="key-name"
                  {...register('name')}
                  placeholder="e.g. CI/CD Bot"
                  disabled={loading}
                />
                {errors.name && (
                  <p className="text-sm text-destructive">{errors.name.message}</p>
                )}
              </div>

              <div className="space-y-2">
                <Label>Scopes *</Label>
                {errors.scopes && (
                  <p className="text-sm text-destructive">{errors.scopes.message}</p>
                )}
                <Controller
                  name="scopes"
                  control={control}
                  render={({ field }) => (
                    <PermissionTree value={field.value} onChange={field.onChange} />
                  )}
                />
              </div>

              <DialogFooter>
                <Button type="button" variant="outline" onClick={() => handleClose(false)} disabled={loading}>
                  Cancel
                </Button>
                <Button type="submit" disabled={loading}>
                  {loading ? 'Creating...' : 'Create API Key'}
                </Button>
              </DialogFooter>
            </form>
          )}
        </DialogContent>
      </Dialog>

      <ConfirmDialog
        open={confirmClose}
        onOpenChange={setConfirmClose}
        title="Close without saving?"
        description="Have you saved your API key? It will not be shown again."
        confirmLabel="Close Anyway"
        variant="destructive"
        onConfirm={() => {
          setCreatedKey(null);
          reset();
          onOpenChange(false);
        }}
      />
    </>
  );
}

export function ApiKeysSection() {
  const queryClient = useQueryClient();
  const [createOpen, setCreateOpen] = useState(false);
  const [revokeKey, setRevokeKey] = useState<ApiKey | null>(null);

  const { data: keysResponse, isLoading } = useQuery<ApiKeysResponse>({
    queryKey: ['api-keys'],
    queryFn: () => apiGet<ApiKeysResponse>('/api/v1/api-keys'),
  });
  const keys = keysResponse?.data;

  const refetch = () => queryClient.invalidateQueries({ queryKey: ['api-keys'] });

  const handleRevoke = async () => {
    if (!revokeKey) return;
    try {
      await apiDelete(`/api/v1/api-keys/${revokeKey.id}`);
      toast.success('API key revoked.');
      setRevokeKey(null);
      refetch();
    } catch (err) {
      const msg = isApiError(err) ? err.message : 'Failed to revoke API key.';
      toast.error(msg);
    }
  };

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between">
          <div>
            <CardTitle className="text-base">API Keys</CardTitle>
            <CardDescription>Manage programmatic access to the platform.</CardDescription>
          </div>
          <Button size="sm" variant="outline" onClick={() => setCreateOpen(true)}>
            <Plus className="mr-2 h-4 w-4" />
            Create API Key
          </Button>
        </div>
      </CardHeader>
      <CardContent>
        {isLoading ? (
          <div className="space-y-2">
            {Array.from({ length: 2 }).map((_, i) => (
              <Skeleton key={i} className="h-12 w-full" />
            ))}
          </div>
        ) : !keys || keys.length === 0 ? (
          <div className="text-center py-6 space-y-2">
            <Key className="h-8 w-8 text-muted-foreground mx-auto" />
            <p className="text-sm text-muted-foreground">No API keys yet.</p>
          </div>
        ) : (
          <div className="space-y-2">
            {keys.map((key) => (
              <div
                key={key.id}
                className="flex items-center gap-3 rounded-md border px-3 py-2.5"
              >
                <Key className="h-4 w-4 text-muted-foreground shrink-0" />
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2 flex-wrap">
                    <p className="text-sm font-medium">{key.name}</p>
                    <span className="font-mono text-xs text-muted-foreground">
                      {key.prefix}...
                    </span>
                    {key.expires_at && (
                      <Badge variant="outline" className="text-xs">
                        Expires {formatDate(key.expires_at)}
                      </Badge>
                    )}
                  </div>
                  <p className="text-xs text-muted-foreground">
                    Created {formatDate(key.created_at)} ·{' '}
                    {key.last_used_at ? (
                      <>Last used <RelativeTime date={key.last_used_at} /></>
                    ) : (
                      'Never used'
                    )}
                  </p>
                </div>
                <Button
                  variant="ghost"
                  size="sm"
                  className="text-xs h-7 text-destructive hover:text-destructive"
                  onClick={() => setRevokeKey(key)}
                >
                  Revoke
                </Button>
              </div>
            ))}
          </div>
        )}
      </CardContent>

      <ApiKeyCreateDialog
        open={createOpen}
        onOpenChange={setCreateOpen}
        onSuccess={refetch}
      />

      <ConfirmDialog
        open={!!revokeKey}
        onOpenChange={(o) => { if (!o) setRevokeKey(null); }}
        title="Revoke API Key"
        description={`Are you sure you want to revoke "${revokeKey?.name}"? Any applications using this key will lose access immediately.`}
        confirmLabel="Revoke Key"
        variant="destructive"
        onConfirm={handleRevoke}
      />
    </Card>
  );
}
