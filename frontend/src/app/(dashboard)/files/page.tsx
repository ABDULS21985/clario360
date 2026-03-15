'use client';

import { useMemo, useState } from 'react';
import {
  Database,
  Download,
  Eye,
  File as FileIcon,
  RefreshCw,
  ScanSearch,
  ShieldAlert,
  Trash2,
  Upload,
} from 'lucide-react';
import { useQuery } from '@tanstack/react-query';
import { toast } from 'sonner';
import { EmptyState } from '@/components/common/empty-state';
import { ErrorState } from '@/components/common/error-state';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import { PageHeader } from '@/components/common/page-header';
import { ConfirmDialog } from '@/components/shared/confirm-dialog';
import { FileUpload } from '@/components/shared/forms/file-upload';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Checkbox } from '@/components/ui/checkbox';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { useAuth } from '@/hooks/use-auth';
import { enterpriseApi } from '@/lib/enterprise/api';
import { downloadBlob, formatBytes, formatDateTime, formatRelativeTime, parseApiError, titleCase } from '@/lib/format';
import type {
  FileItem,
  FileLifecyclePolicy,
  FilePresignedDownload,
  FileQuarantineEntry,
  FileRecord,
  FileStorageStat,
  FileSuite,
  FileVirusScanStatus,
} from '@/types/models';

const FILE_SUITES: FileSuite[] = ['platform', 'cyber', 'data', 'acta', 'lex', 'visus', 'models'];
const FILE_LIFECYCLE_POLICIES: FileLifecyclePolicy[] = [
  'standard',
  'temporary',
  'archive',
  'audit_retention',
];
const QUARANTINE_ACTIONS = ['restored', 'deleted', 'false_positive'] as const;

type QuarantineAction = (typeof QUARANTINE_ACTIONS)[number];

function statusVariant(status: FileItem['status']): 'default' | 'success' | 'destructive' | 'warning' | 'outline' {
  switch (status) {
    case 'available':
      return 'success';
    case 'pending':
      return 'warning';
    case 'quarantined':
      return 'destructive';
    case 'processing':
      return 'default';
    default:
      return 'outline';
  }
}

function scanVariant(status: FileVirusScanStatus): 'default' | 'success' | 'destructive' | 'warning' | 'outline' {
  switch (status) {
    case 'clean':
    case 'skipped':
      return 'success';
    case 'infected':
      return 'destructive';
    case 'pending':
    case 'scanning':
      return 'warning';
    case 'error':
      return 'default';
    default:
      return 'outline';
  }
}

function prettyLabel(value: string | null | undefined): string {
  if (!value) return 'Not set';
  return titleCase(value);
}

function normalizeRoleKeys(fileRoles: Array<{ slug: string; name: string }>): Set<string> {
  return new Set(
    fileRoles.flatMap((role) => {
      const slug = role.slug.toLowerCase();
      const name = role.name.toLowerCase().replace(/\s+/g, '_');
      return [slug, name];
    }),
  );
}

function PaginationControls({
  page,
  totalPages,
  onPageChange,
}: {
  page: number;
  totalPages: number;
  onPageChange: (page: number) => void;
}) {
  return (
    <div className="flex items-center justify-between border-t px-4 py-3">
      <p className="text-xs text-muted-foreground">
        Page {page} of {totalPages}
      </p>
      <div className="flex gap-2">
        <Button variant="outline" size="sm" disabled={page <= 1} onClick={() => onPageChange(page - 1)}>
          Previous
        </Button>
        <Button
          variant="outline"
          size="sm"
          disabled={page >= totalPages}
          onClick={() => onPageChange(page + 1)}
        >
          Next
        </Button>
      </div>
    </div>
  );
}

function StorageSummaryCard({ title, value, caption }: { title: string; value: string; caption: string }) {
  return (
    <Card>
      <CardHeader className="space-y-2">
        <CardDescription>{title}</CardDescription>
        <CardTitle className="text-2xl">{value}</CardTitle>
      </CardHeader>
      <CardContent className="pt-0">
        <p className="text-sm text-muted-foreground">{caption}</p>
      </CardContent>
    </Card>
  );
}

function FileMetadataRow({ label, value }: { label: string; value: React.ReactNode }) {
  return (
    <div className="grid grid-cols-1 gap-1 sm:grid-cols-[160px_1fr]">
      <span className="text-sm text-muted-foreground">{label}</span>
      <div className="text-sm">{value}</div>
    </div>
  );
}

function FileDetailDialog({
  fileId,
  open,
  onOpenChange,
  isAdmin,
  busyKey,
  onDownload,
  onOpenPresigned,
  onRescan,
  onDelete,
}: {
  fileId: string | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  isAdmin: boolean;
  busyKey: string | null;
  onDownload: (file: FileRecord) => Promise<void>;
  onOpenPresigned: (file: FileRecord) => Promise<void>;
  onRescan: (file: FileRecord) => Promise<void>;
  onDelete: (file: FileRecord) => void;
}) {
  const [accessPage, setAccessPage] = useState(1);

  const fileQuery = useQuery({
    queryKey: ['file-detail', fileId],
    queryFn: () => enterpriseApi.files.get(fileId ?? ''),
    enabled: open && Boolean(fileId),
  });

  const versionsQuery = useQuery({
    queryKey: ['file-versions', fileId],
    queryFn: () => enterpriseApi.files.versions(fileId ?? ''),
    enabled: open && Boolean(fileId),
  });

  const accessLogQuery = useQuery({
    queryKey: ['file-access-log', fileId, accessPage],
    queryFn: () => enterpriseApi.files.accessLog(fileId ?? '', { page: accessPage, per_page: 10 }),
    enabled: open && Boolean(fileId),
  });

  const file = fileQuery.data;

  return (
    <Dialog
      open={open}
      onOpenChange={(nextOpen) => {
        if (!nextOpen) {
          setAccessPage(1);
        }
        onOpenChange(nextOpen);
      }}
    >
      <DialogContent className="max-h-[90vh] max-w-5xl overflow-y-auto">
        <DialogHeader>
          <DialogTitle>{file?.original_name ?? 'File details'}</DialogTitle>
          <DialogDescription>
            Inspect file metadata, version history, download activity, and admin controls.
          </DialogDescription>
        </DialogHeader>

        {fileQuery.isLoading ? (
          <LoadingSkeleton variant="text" count={4} />
        ) : fileQuery.isError || !file ? (
          <ErrorState
            title="Unable to load file details"
            message="The selected file could not be loaded."
            onRetry={() => void fileQuery.refetch()}
          />
        ) : (
          <div className="space-y-6">
            <div className="flex flex-wrap gap-2">
              <Badge variant={statusVariant(file.status)}>{prettyLabel(file.status)}</Badge>
              <Badge variant={scanVariant(file.virus_scan_status)}>{prettyLabel(file.virus_scan_status)}</Badge>
              <Badge variant="outline">v{file.version_number}</Badge>
              <Badge variant="outline">{prettyLabel(file.lifecycle_policy)}</Badge>
            </div>

            <div className="flex flex-wrap gap-2">
              <Button
                onClick={() => void onDownload(file)}
                disabled={busyKey === `download:${file.id}` || file.status === 'quarantined'}
              >
                <Download className="mr-2 h-4 w-4" />
                Download
              </Button>
              {!file.encrypted ? (
                <Button
                  variant="outline"
                  onClick={() => void onOpenPresigned(file)}
                  disabled={busyKey === `presigned:${file.id}` || file.status === 'quarantined'}
                >
                  <Eye className="mr-2 h-4 w-4" />
                  Open Presigned URL
                </Button>
              ) : null}
              {isAdmin ? (
                <Button
                  variant="outline"
                  onClick={() => void onRescan(file)}
                  disabled={busyKey === `rescan:${file.id}`}
                >
                  <ScanSearch className="mr-2 h-4 w-4" />
                  Queue Rescan
                </Button>
              ) : null}
              <Button
                variant="destructive"
                onClick={() => onDelete(file)}
                disabled={busyKey === `delete:${file.id}`}
              >
                <Trash2 className="mr-2 h-4 w-4" />
                Delete
              </Button>
            </div>

            <Card>
              <CardHeader>
                <CardTitle>Metadata</CardTitle>
                <CardDescription>Live metadata returned by file-service for this record.</CardDescription>
              </CardHeader>
              <CardContent className="space-y-3">
                <FileMetadataRow label="Suite" value={prettyLabel(file.suite)} />
                <FileMetadataRow label="Stored name" value={file.name} />
                <FileMetadataRow label="Sanitized name" value={file.sanitized_name} />
                <FileMetadataRow label="Content type" value={file.content_type} />
                <FileMetadataRow label="Detected type" value={file.detected_content_type || 'Not detected'} />
                <FileMetadataRow label="Size" value={formatBytes(file.size_bytes)} />
                <FileMetadataRow label="Uploaded by" value={file.uploaded_by} />
                <FileMetadataRow label="Checksum" value={<span className="break-all font-mono text-xs">{file.checksum_sha256}</span>} />
                <FileMetadataRow label="Entity link" value={file.entity_type && file.entity_id ? `${file.entity_type} / ${file.entity_id}` : 'Not linked'} />
                <FileMetadataRow label="Expires at" value={file.expires_at ? formatDateTime(file.expires_at) : 'No expiry'} />
                <FileMetadataRow label="Created" value={`${formatDateTime(file.created_at)} (${formatRelativeTime(file.created_at)})`} />
                <FileMetadataRow label="Updated" value={`${formatDateTime(file.updated_at)} (${formatRelativeTime(file.updated_at)})`} />
                <FileMetadataRow
                  label="Tags"
                  value={
                    file.tags.length > 0 ? (
                      <div className="flex flex-wrap gap-2">
                        {file.tags.map((tag) => (
                          <Badge key={tag} variant="outline">
                            {tag}
                          </Badge>
                        ))}
                      </div>
                    ) : (
                      'No tags'
                    )
                  }
                />
              </CardContent>
            </Card>

            <Tabs defaultValue="versions">
              <TabsList>
                <TabsTrigger value="versions">Versions</TabsTrigger>
                <TabsTrigger value="access-log">Access Log</TabsTrigger>
              </TabsList>

              <TabsContent value="versions">
                <Card>
                  <CardHeader>
                    <CardTitle>Version History</CardTitle>
                    <CardDescription>All versions returned by the file-service version lookup.</CardDescription>
                  </CardHeader>
                  <CardContent>
                    {versionsQuery.isLoading ? (
                      <LoadingSkeleton variant="list-item" count={3} />
                    ) : versionsQuery.isError ? (
                      <ErrorState
                        title="Unable to load versions"
                        message="Version history could not be loaded."
                        onRetry={() => void versionsQuery.refetch()}
                      />
                    ) : (
                      <Table>
                        <TableHeader>
                          <TableRow>
                            <TableHead>Version</TableHead>
                            <TableHead>Status</TableHead>
                            <TableHead>Scan</TableHead>
                            <TableHead>Created</TableHead>
                          </TableRow>
                        </TableHeader>
                        <TableBody>
                          {(versionsQuery.data ?? []).map((version) => (
                            <TableRow key={version.id}>
                              <TableCell className="font-medium">v{version.version_number}</TableCell>
                              <TableCell>
                                <Badge variant={statusVariant(version.status)}>{prettyLabel(version.status)}</Badge>
                              </TableCell>
                              <TableCell>
                                <Badge variant={scanVariant(version.virus_scan_status)}>
                                  {prettyLabel(version.virus_scan_status)}
                                </Badge>
                              </TableCell>
                              <TableCell>{formatDateTime(version.created_at)}</TableCell>
                            </TableRow>
                          ))}
                        </TableBody>
                      </Table>
                    )}
                  </CardContent>
                </Card>
              </TabsContent>

              <TabsContent value="access-log">
                <Card>
                  <CardHeader>
                    <CardTitle>Access Log</CardTitle>
                    <CardDescription>Download, view, and presigned operations recorded for this file.</CardDescription>
                  </CardHeader>
                  <CardContent className="space-y-4">
                    {accessLogQuery.isLoading ? (
                      <LoadingSkeleton variant="table-row" count={4} />
                    ) : accessLogQuery.isError ? (
                      <ErrorState
                        title="Unable to load access log"
                        message="The file access history could not be loaded."
                        onRetry={() => void accessLogQuery.refetch()}
                      />
                    ) : accessLogQuery.data && accessLogQuery.data.data.length > 0 ? (
                      <>
                        <Table>
                          <TableHeader>
                            <TableRow>
                              <TableHead>Action</TableHead>
                              <TableHead>User</TableHead>
                              <TableHead>IP Address</TableHead>
                              <TableHead>Time</TableHead>
                            </TableRow>
                          </TableHeader>
                          <TableBody>
                            {accessLogQuery.data.data.map((entry) => (
                              <TableRow key={entry.id}>
                                <TableCell>{prettyLabel(entry.action)}</TableCell>
                                <TableCell className="font-mono text-xs">{entry.user_id}</TableCell>
                                <TableCell>{entry.ip_address || 'Unknown'}</TableCell>
                                <TableCell>{formatDateTime(entry.created_at)}</TableCell>
                              </TableRow>
                            ))}
                          </TableBody>
                        </Table>
                        {accessLogQuery.data.meta.total_pages > 1 ? (
                          <PaginationControls
                            page={accessPage}
                            totalPages={accessLogQuery.data.meta.total_pages}
                            onPageChange={setAccessPage}
                          />
                        ) : null}
                      </>
                    ) : (
                      <EmptyState
                        icon={Database}
                        title="No access log entries"
                        description="This file does not have any recorded access operations yet."
                      />
                    )}
                  </CardContent>
                </Card>
              </TabsContent>
            </Tabs>
          </div>
        )}
      </DialogContent>
    </Dialog>
  );
}

export default function FilesPage() {
  const { tenant, user } = useAuth();
  const roleKeys = user ? normalizeRoleKeys(user.roles) : new Set<string>();
  const isAdmin =
    roleKeys.has('super_admin') ||
    roleKeys.has('security-manager') ||
    roleKeys.has('security_manager');

  const [page, setPage] = useState(1);
  const [suiteFilter, setSuiteFilter] = useState<string>('all');
  const [activeTab, setActiveTab] = useState<'library' | 'quarantine'>('library');
  const [quarantinePage, setQuarantinePage] = useState(1);
  const [selectedFileId, setSelectedFileId] = useState<string | null>(null);
  const [detailOpen, setDetailOpen] = useState(false);
  const [deleteCandidate, setDeleteCandidate] = useState<FileRecord | null>(null);
  const [quarantineResolution, setQuarantineResolution] = useState<{
    entry: FileQuarantineEntry;
    action: QuarantineAction;
  } | null>(null);
  const [busyKey, setBusyKey] = useState<string | null>(null);
  const [uploading, setUploading] = useState(false);
  const [uploadProgress, setUploadProgress] = useState(0);
  const [uploadConfig, setUploadConfig] = useState({
    suite: 'platform' as FileSuite,
    lifecycle_policy: 'standard' as FileLifecyclePolicy,
    tags: '',
    entity_type: '',
    entity_id: '',
    encrypt: false,
  });

  const filesQuery = useQuery({
    queryKey: ['files', page, suiteFilter],
    queryFn: () =>
      enterpriseApi.files.list({
        page,
        per_page: 25,
        suite: suiteFilter === 'all' ? undefined : suiteFilter,
      }),
  });

  const statsQuery = useQuery({
    queryKey: ['file-storage-stats'],
    queryFn: () => enterpriseApi.files.stats(),
    enabled: isAdmin,
  });

  const quarantineQuery = useQuery({
    queryKey: ['file-quarantine', quarantinePage],
    queryFn: () => enterpriseApi.files.quarantine({ page: quarantinePage, per_page: 20 }),
    enabled: isAdmin,
  });

  const tenantStats = useMemo(() => {
    const stats = statsQuery.data ?? [];
    if (!tenant?.id) return stats;
    const matching = stats.filter((stat) => stat.tenant_id === tenant.id);
    return matching.length > 0 ? matching : stats;
  }, [statsQuery.data, tenant?.id]);

  const totalFiles = tenantStats.reduce((sum, stat) => sum + stat.file_count, 0);
  const totalStorage = tenantStats.reduce((sum, stat) => sum + stat.total_bytes, 0);
  const suiteBreakdown = useMemo(() => {
    return tenantStats.reduce<Record<string, FileStorageStat>>((acc, stat) => {
      const existing = acc[stat.suite];
      if (existing) {
        existing.file_count += stat.file_count;
        existing.total_bytes += stat.total_bytes;
        return acc;
      }
      acc[stat.suite] = { ...stat };
      return acc;
    }, {});
  }, [tenantStats]);

  const refreshAll = async () => {
    await Promise.all([
      filesQuery.refetch(),
      isAdmin ? statsQuery.refetch() : Promise.resolve(null),
      isAdmin ? quarantineQuery.refetch() : Promise.resolve(null),
    ]);
  };

  const openDetail = (fileId: string) => {
    setSelectedFileId(fileId);
    setDetailOpen(true);
  };

  const handleUpload = async (files: File[]) => {
    setUploading(true);
    setUploadProgress(0);

    try {
      for (let index = 0; index < files.length; index += 1) {
        const file = files[index];
        await enterpriseApi.files.upload(
          file,
          {
            suite: uploadConfig.suite,
            lifecycle_policy: uploadConfig.lifecycle_policy,
            encrypt: String(uploadConfig.encrypt),
            ...(uploadConfig.tags.trim() ? { tags: uploadConfig.tags.trim() } : {}),
            ...(uploadConfig.entity_type.trim() ? { entity_type: uploadConfig.entity_type.trim() } : {}),
            ...(uploadConfig.entity_id.trim() ? { entity_id: uploadConfig.entity_id.trim() } : {}),
          },
          (progress) => {
            const completed = index / files.length;
            const current = progress / 100 / files.length;
            setUploadProgress(Math.round((completed + current) * 100));
          },
        );
      }

      toast.success(files.length === 1 ? 'File uploaded successfully' : `${files.length} files uploaded successfully`);
      await refreshAll();
    } catch (error) {
      toast.error(parseApiError(error));
    } finally {
      setUploading(false);
      setUploadProgress(0);
    }
  };

  const handleDownload = async (file: FileRecord) => {
    setBusyKey(`download:${file.id}`);
    try {
      const blob = await enterpriseApi.files.download(file.id);
      downloadBlob(blob, file.original_name || file.name);
    } catch (error) {
      toast.error(parseApiError(error));
    } finally {
      setBusyKey(null);
    }
  };

  const handleOpenPresigned = async (file: FileRecord) => {
    setBusyKey(`presigned:${file.id}`);
    try {
      const presigned: FilePresignedDownload = await enterpriseApi.files.getPresignedDownload(file.id);
      window.open(presigned.url, '_blank', 'noopener,noreferrer');
    } catch (error) {
      toast.error(parseApiError(error));
    } finally {
      setBusyKey(null);
    }
  };

  const handleRescan = async (file: FileRecord) => {
    setBusyKey(`rescan:${file.id}`);
    try {
      await enterpriseApi.files.rescan(file.id);
      toast.success('File rescan queued');
      await refreshAll();
    } catch (error) {
      toast.error(parseApiError(error));
    } finally {
      setBusyKey(null);
    }
  };

  const handleDelete = async (file: FileRecord) => {
    setBusyKey(`delete:${file.id}`);
    try {
      await enterpriseApi.files.delete(file.id);
      toast.success('File deleted');
      if (selectedFileId === file.id) {
        setDetailOpen(false);
        setSelectedFileId(null);
      }
      await refreshAll();
    } catch (error) {
      toast.error(parseApiError(error));
      throw error;
    } finally {
      setBusyKey(null);
    }
  };

  const handleResolveQuarantine = async (entry: FileQuarantineEntry, action: QuarantineAction) => {
    setBusyKey(`quarantine:${entry.id}:${action}`);
    try {
      await enterpriseApi.files.resolveQuarantine(entry.id, action);
      toast.success(`Quarantine entry marked ${prettyLabel(action)}`);
      await refreshAll();
    } catch (error) {
      toast.error(parseApiError(error));
      throw error;
    } finally {
      setBusyKey(null);
    }
  };

  return (
    <div className="space-y-6">
      <PageHeader
        title="Files"
        description="Operate the full file-service surface from the frontend: upload, inspect, download, rescan, and manage quarantine activity."
        actions={
          <Button variant="outline" onClick={() => void refreshAll()} disabled={filesQuery.isFetching || busyKey !== null}>
            <RefreshCw className={`mr-2 h-4 w-4 ${filesQuery.isFetching ? 'animate-spin' : ''}`} />
            Refresh
          </Button>
        }
      />

      {isAdmin ? (
        <section className="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-4">
          <StorageSummaryCard
            title="Tracked files"
            value={statsQuery.isLoading ? '...' : totalFiles.toString()}
            caption="Current tenant-visible file count across suites."
          />
          <StorageSummaryCard
            title="Storage used"
            value={statsQuery.isLoading ? '...' : formatBytes(totalStorage)}
            caption="Aggregated storage consumed by tracked file records."
          />
          <StorageSummaryCard
            title="Quarantine backlog"
            value={quarantineQuery.isLoading ? '...' : (quarantineQuery.data?.meta.total ?? 0).toString()}
            caption="Unresolved quarantine entries requiring admin action."
          />
          <StorageSummaryCard
            title="Active suites"
            value={Object.keys(suiteBreakdown).length.toString()}
            caption="Suites currently storing file records for this tenant."
          />
        </section>
      ) : null}

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Upload className="h-5 w-5" />
            Upload files
          </CardTitle>
          <CardDescription>
            Direct upload is already supported by file-service. Configure the suite metadata here so uploaded files land in the correct backend scope.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-5">
            <div className="space-y-2">
              <Label htmlFor="file-suite">Suite</Label>
              <Select
                value={uploadConfig.suite}
                onValueChange={(value) =>
                  setUploadConfig((current) => ({ ...current, suite: value as FileSuite }))
                }
              >
                <SelectTrigger id="file-suite">
                  <SelectValue placeholder="Select suite" />
                </SelectTrigger>
                <SelectContent>
                  {FILE_SUITES.map((suite) => (
                    <SelectItem key={suite} value={suite}>
                      {prettyLabel(suite)}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            <div className="space-y-2">
              <Label htmlFor="file-policy">Lifecycle policy</Label>
              <Select
                value={uploadConfig.lifecycle_policy}
                onValueChange={(value) =>
                  setUploadConfig((current) => ({
                    ...current,
                    lifecycle_policy: value as FileLifecyclePolicy,
                  }))
                }
              >
                <SelectTrigger id="file-policy">
                  <SelectValue placeholder="Select lifecycle policy" />
                </SelectTrigger>
                <SelectContent>
                  {FILE_LIFECYCLE_POLICIES.map((policy) => (
                    <SelectItem key={policy} value={policy}>
                      {prettyLabel(policy)}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            <div className="space-y-2">
              <Label htmlFor="file-entity-type">Entity type</Label>
              <Input
                id="file-entity-type"
                placeholder="contract, meeting, alert..."
                value={uploadConfig.entity_type}
                onChange={(event) =>
                  setUploadConfig((current) => ({ ...current, entity_type: event.target.value }))
                }
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="file-entity-id">Entity ID</Label>
              <Input
                id="file-entity-id"
                placeholder="Optional linked record ID"
                value={uploadConfig.entity_id}
                onChange={(event) =>
                  setUploadConfig((current) => ({ ...current, entity_id: event.target.value }))
                }
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="file-tags">Tags</Label>
              <Input
                id="file-tags"
                placeholder="Comma separated tags"
                value={uploadConfig.tags}
                onChange={(event) =>
                  setUploadConfig((current) => ({ ...current, tags: event.target.value }))
                }
              />
            </div>
          </div>

          <div className="flex items-center gap-3 rounded-xl border border-border/70 px-4 py-3">
            <Checkbox
              id="file-encrypt"
              checked={uploadConfig.encrypt}
              onCheckedChange={(checked) =>
                setUploadConfig((current) => ({ ...current, encrypt: checked === true }))
              }
            />
            <div className="space-y-1">
              <Label htmlFor="file-encrypt" className="cursor-pointer">
                Encrypt uploaded content at rest
              </Label>
              <p className="text-sm text-muted-foreground">
                Enable backend-managed file encryption before the object is stored.
              </p>
            </div>
          </div>

          <FileUpload
            accept="*/*"
            maxSizeMB={100}
            multiple={false}
            onUpload={handleUpload}
            uploading={uploading}
            progress={uploadProgress}
          />
        </CardContent>
      </Card>

      <Tabs value={activeTab} onValueChange={(value) => setActiveTab(value as 'library' | 'quarantine')}>
        <TabsList>
          <TabsTrigger value="library">Library</TabsTrigger>
          {isAdmin ? <TabsTrigger value="quarantine">Quarantine</TabsTrigger> : null}
        </TabsList>

        <TabsContent value="library" className="space-y-4">
          <div className="flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
            <div className="space-y-2">
              <Label htmlFor="suite-filter">Suite filter</Label>
              <Select
                value={suiteFilter}
                onValueChange={(value) => {
                  setSuiteFilter(value);
                  setPage(1);
                }}
              >
                <SelectTrigger id="suite-filter" className="w-[220px]">
                  <SelectValue placeholder="All suites" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">All suites</SelectItem>
                  {FILE_SUITES.map((suite) => (
                    <SelectItem key={suite} value={suite}>
                      {prettyLabel(suite)}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            <p className="text-sm text-muted-foreground">
              {filesQuery.data ? `${filesQuery.data.meta.total} files returned from file-service.` : 'Loading file inventory...'}
            </p>
          </div>

          {filesQuery.isLoading ? (
            <LoadingSkeleton variant="table-row" count={8} />
          ) : filesQuery.isError ? (
            <ErrorState message="Failed to load files" onRetry={() => void filesQuery.refetch()} />
          ) : !filesQuery.data || filesQuery.data.data.length === 0 ? (
            <EmptyState
              icon={FileIcon}
              title="No files found"
              description="No file records matched the current filter. Upload a file or switch suites."
            />
          ) : (
            <Card className="overflow-hidden">
              <CardContent className="p-0">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Name</TableHead>
                      <TableHead>Suite</TableHead>
                      <TableHead>Status</TableHead>
                      <TableHead>Scan</TableHead>
                      <TableHead>Size</TableHead>
                      <TableHead>Created</TableHead>
                      <TableHead className="w-[220px]">Actions</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {filesQuery.data.data.map((file) => (
                      <TableRow key={file.id}>
                        <TableCell>
                          <div className="space-y-1">
                            <div className="font-medium">{file.original_name}</div>
                            <div className="text-xs text-muted-foreground">
                              {file.entity_type && file.entity_id
                                ? `${file.entity_type} / ${file.entity_id}`
                                : file.name}
                            </div>
                          </div>
                        </TableCell>
                        <TableCell>{prettyLabel(file.suite)}</TableCell>
                        <TableCell>
                          <Badge variant={statusVariant(file.status)}>{prettyLabel(file.status)}</Badge>
                        </TableCell>
                        <TableCell>
                          <Badge variant={scanVariant(file.virus_scan_status)}>
                            {prettyLabel(file.virus_scan_status)}
                          </Badge>
                        </TableCell>
                        <TableCell>{formatBytes(file.size_bytes)}</TableCell>
                        <TableCell>
                          <div className="space-y-1">
                            <div>{formatDateTime(file.created_at)}</div>
                            <div className="text-xs text-muted-foreground">
                              {formatRelativeTime(file.created_at)}
                            </div>
                          </div>
                        </TableCell>
                        <TableCell>
                          <div className="flex flex-wrap gap-2">
                            <Button variant="outline" size="sm" onClick={() => openDetail(file.id)}>
                              <Eye className="mr-2 h-4 w-4" />
                              Inspect
                            </Button>
                            <Button
                              variant="outline"
                              size="sm"
                              disabled={busyKey === `download:${file.id}` || file.status === 'quarantined'}
                              onClick={() => void handleDownload(file)}
                            >
                              <Download className="mr-2 h-4 w-4" />
                              Download
                            </Button>
                          </div>
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>

                {filesQuery.data.meta.total_pages > 1 ? (
                  <PaginationControls
                    page={page}
                    totalPages={filesQuery.data.meta.total_pages}
                    onPageChange={setPage}
                  />
                ) : null}
              </CardContent>
            </Card>
          )}
        </TabsContent>

        {isAdmin ? (
          <TabsContent value="quarantine" className="space-y-4">
            {!statsQuery.isLoading && Object.keys(suiteBreakdown).length > 0 ? (
              <div className="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-4">
                {Object.values(suiteBreakdown).map((stat) => (
                  <Card key={stat.suite}>
                    <CardHeader className="space-y-2">
                      <CardDescription>{prettyLabel(stat.suite)}</CardDescription>
                      <CardTitle className="text-xl">{formatBytes(stat.total_bytes)}</CardTitle>
                    </CardHeader>
                    <CardContent className="pt-0">
                      <p className="text-sm text-muted-foreground">
                        {stat.file_count} tracked file{stat.file_count === 1 ? '' : 's'}
                      </p>
                    </CardContent>
                  </Card>
                ))}
              </div>
            ) : null}

            {quarantineQuery.isLoading ? (
              <LoadingSkeleton variant="table-row" count={6} />
            ) : quarantineQuery.isError ? (
              <ErrorState
                title="Unable to load quarantine queue"
                message="The admin quarantine list could not be loaded."
                onRetry={() => void quarantineQuery.refetch()}
              />
            ) : !quarantineQuery.data || quarantineQuery.data.data.length === 0 ? (
              <EmptyState
                icon={ShieldAlert}
                title="No quarantined files"
                description="The unresolved quarantine queue is empty."
              />
            ) : (
              <Card className="overflow-hidden">
                <CardHeader>
                  <CardTitle className="flex items-center gap-2">
                    <ShieldAlert className="h-5 w-5" />
                    Quarantine queue
                  </CardTitle>
                  <CardDescription>
                    Resolve infected-file events and clear the backend quarantine backlog.
                  </CardDescription>
                </CardHeader>
                <CardContent className="space-y-4">
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead>File ID</TableHead>
                        <TableHead>Virus</TableHead>
                        <TableHead>Quarantined</TableHead>
                        <TableHead>Action</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {quarantineQuery.data.data.map((entry) => (
                        <TableRow key={entry.id}>
                          <TableCell className="font-mono text-xs">{entry.file_id}</TableCell>
                          <TableCell>{entry.virus_name || 'Unknown'}</TableCell>
                          <TableCell>
                            <div className="space-y-1">
                              <div>{formatDateTime(entry.quarantined_at)}</div>
                              <div className="text-xs text-muted-foreground">
                                {formatRelativeTime(entry.quarantined_at)}
                              </div>
                            </div>
                          </TableCell>
                          <TableCell>
                            <div className="flex flex-wrap gap-2">
                              {QUARANTINE_ACTIONS.map((action) => (
                                <Button
                                  key={action}
                                  size="sm"
                                  variant={action === 'deleted' ? 'destructive' : 'outline'}
                                  disabled={busyKey === `quarantine:${entry.id}:${action}`}
                                  onClick={() => setQuarantineResolution({ entry, action })}
                                >
                                  {prettyLabel(action)}
                                </Button>
                              ))}
                            </div>
                          </TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>

                  {quarantineQuery.data.meta.total_pages > 1 ? (
                    <PaginationControls
                      page={quarantinePage}
                      totalPages={quarantineQuery.data.meta.total_pages}
                      onPageChange={setQuarantinePage}
                    />
                  ) : null}
                </CardContent>
              </Card>
            )}
          </TabsContent>
        ) : null}
      </Tabs>

      <FileDetailDialog
        fileId={selectedFileId}
        open={detailOpen}
        onOpenChange={setDetailOpen}
        isAdmin={isAdmin}
        busyKey={busyKey}
        onDownload={handleDownload}
        onOpenPresigned={handleOpenPresigned}
        onRescan={handleRescan}
        onDelete={setDeleteCandidate}
      />

      <ConfirmDialog
        open={Boolean(deleteCandidate)}
        onOpenChange={(open) => {
          if (!open) {
            setDeleteCandidate(null);
          }
        }}
        title="Delete file"
        description={
          deleteCandidate
            ? `Delete "${deleteCandidate.original_name}" from the file-service inventory? This cannot be undone.`
            : ''
        }
        confirmLabel="Delete file"
        variant="destructive"
        typeToConfirm={deleteCandidate?.original_name}
        loading={deleteCandidate ? busyKey === `delete:${deleteCandidate.id}` : false}
        onConfirm={async () => {
          if (deleteCandidate) {
            await handleDelete(deleteCandidate);
            setDeleteCandidate(null);
          }
        }}
      />

      <ConfirmDialog
        open={Boolean(quarantineResolution)}
        onOpenChange={(open) => {
          if (!open) {
            setQuarantineResolution(null);
          }
        }}
        title="Resolve quarantine entry"
        description={
          quarantineResolution
            ? `Mark this quarantine entry as ${prettyLabel(quarantineResolution.action)}?`
            : ''
        }
        confirmLabel={quarantineResolution ? prettyLabel(quarantineResolution.action) : 'Resolve'}
        variant={quarantineResolution?.action === 'deleted' ? 'destructive' : 'default'}
        loading={
          quarantineResolution
            ? busyKey === `quarantine:${quarantineResolution.entry.id}:${quarantineResolution.action}`
            : false
        }
        onConfirm={async () => {
          if (quarantineResolution) {
            await handleResolveQuarantine(quarantineResolution.entry, quarantineResolution.action);
            setQuarantineResolution(null);
          }
        }}
      />
    </div>
  );
}
