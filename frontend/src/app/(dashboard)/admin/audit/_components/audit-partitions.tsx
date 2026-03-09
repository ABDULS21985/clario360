"use client";

import { useState } from "react";
import { Plus, Archive, Trash2, HardDrive } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Label } from "@/components/ui/label";
import { Input } from "@/components/ui/input";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from "@/components/ui/dialog";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  useAuditPartitions,
  useCreateAuditPartition,
  useArchiveAuditPartition,
  useDeleteAuditPartition,
} from "@/hooks/use-audit";
import { formatDate, formatBytes, formatNumber } from "@/lib/format";
import type { AuditPartition, AuditPartitionStatus } from "@/types/audit";

const statusVariant: Record<AuditPartitionStatus, "default" | "secondary" | "outline"> = {
  active: "default",
  archived: "secondary",
  pending: "outline",
};

export function AuditPartitions() {
  const { data: partitions, isLoading, error, refetch } = useAuditPartitions();
  const createMutation = useCreateAuditPartition();
  const archiveMutation = useArchiveAuditPartition();
  const deleteMutation = useDeleteAuditPartition();

  const [createOpen, setCreateOpen] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<AuditPartition | null>(null);

  const [newName, setNewName] = useState("");
  const [newStart, setNewStart] = useState("");
  const [newEnd, setNewEnd] = useState("");

  const handleCreate = () => {
    createMutation.mutate(
      {
        name: newName,
        date_range_start: new Date(newStart).toISOString(),
        date_range_end: new Date(newEnd + "T23:59:59").toISOString(),
      },
      {
        onSuccess: () => {
          setCreateOpen(false);
          setNewName("");
          setNewStart("");
          setNewEnd("");
        },
      }
    );
  };

  if (error) {
    return (
      <div className="flex flex-col items-center justify-center py-12 gap-3">
        <p className="text-sm text-muted-foreground">
          Failed to load partitions.
        </p>
        <Button variant="outline" size="sm" onClick={() => refetch()}>
          Retry
        </Button>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Partition timeline bar */}
      {partitions && partitions.length > 0 && (
        <div className="rounded-lg border p-4">
          <p className="text-xs font-medium text-muted-foreground mb-3">
            Partition Coverage
          </p>
          <div className="flex gap-1 h-6">
            {partitions.map((p) => (
              <div
                key={p.id}
                className={`flex-1 rounded text-[10px] flex items-center justify-center text-white truncate px-1 ${
                  p.status === "active"
                    ? "bg-primary"
                    : p.status === "archived"
                    ? "bg-muted-foreground"
                    : "bg-muted"
                }`}
                title={`${p.name}: ${formatDate(p.date_range_start)} - ${formatDate(p.date_range_end)}`}
              >
                {p.name}
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Actions bar */}
      <div className="flex justify-end">
        <Button onClick={() => setCreateOpen(true)}>
          <Plus className="mr-2 h-4 w-4" />
          Create Partition
        </Button>
      </div>

      {/* Partitions table */}
      <div className="rounded-md border border-border overflow-hidden">
        <Table>
          <TableHeader>
            <TableRow className="hover:bg-transparent">
              <TableHead>Name</TableHead>
              <TableHead>Date Range</TableHead>
              <TableHead>Records</TableHead>
              <TableHead>Size</TableHead>
              <TableHead>Status</TableHead>
              <TableHead>Created</TableHead>
              <TableHead className="w-24">Actions</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              Array.from({ length: 3 }).map((_, i) => (
                <TableRow key={i}>
                  {Array.from({ length: 7 }).map((_, j) => (
                    <TableCell key={j}>
                      <Skeleton className="h-4 w-full" />
                    </TableCell>
                  ))}
                </TableRow>
              ))
            ) : !partitions?.length ? (
              <TableRow>
                <TableCell colSpan={7}>
                  <div className="flex flex-col items-center justify-center py-8 gap-2">
                    <HardDrive className="h-8 w-8 text-muted-foreground/40" />
                    <p className="text-sm text-muted-foreground">
                      No partitions created yet.
                    </p>
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => setCreateOpen(true)}
                    >
                      <Plus className="mr-2 h-4 w-4" />
                      Create Partition
                    </Button>
                  </div>
                </TableCell>
              </TableRow>
            ) : (
              partitions.map((partition) => (
                <TableRow key={partition.id} className="hover:bg-muted/40">
                  <TableCell className="font-medium">
                    {partition.name}
                  </TableCell>
                  <TableCell className="text-sm text-muted-foreground">
                    {formatDate(partition.date_range_start)} –{" "}
                    {formatDate(partition.date_range_end)}
                  </TableCell>
                  <TableCell className="tabular-nums">
                    {formatNumber(partition.record_count)}
                  </TableCell>
                  <TableCell className="text-sm">
                    {formatBytes(partition.size_bytes)}
                  </TableCell>
                  <TableCell>
                    <Badge variant={statusVariant[partition.status]}>
                      {partition.status}
                    </Badge>
                  </TableCell>
                  <TableCell className="text-sm text-muted-foreground">
                    {formatDate(partition.created_at)}
                  </TableCell>
                  <TableCell>
                    <div className="flex items-center gap-1">
                      {partition.status === "active" && (
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() =>
                            archiveMutation.mutate(partition.id)
                          }
                          disabled={archiveMutation.isPending}
                          aria-label={`Archive ${partition.name}`}
                        >
                          <Archive className="h-4 w-4" />
                        </Button>
                      )}
                      {partition.status === "archived" && (
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => setDeleteTarget(partition)}
                          aria-label={`Delete ${partition.name}`}
                        >
                          <Trash2 className="h-4 w-4 text-destructive" />
                        </Button>
                      )}
                    </div>
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </div>

      {/* Create Dialog */}
      <Dialog open={createOpen} onOpenChange={setCreateOpen}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>Create Partition</DialogTitle>
            <DialogDescription>
              Create a new audit log partition for a specific date range.
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="partition-name">Name</Label>
              <Input
                id="partition-name"
                placeholder="e.g., Q1 2026"
                value={newName}
                onChange={(e) => setNewName(e.target.value)}
              />
            </div>
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label htmlFor="partition-start">Start Date</Label>
                <Input
                  id="partition-start"
                  type="date"
                  value={newStart}
                  onChange={(e) => setNewStart(e.target.value)}
                  max={newEnd || undefined}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="partition-end">End Date</Label>
                <Input
                  id="partition-end"
                  type="date"
                  value={newEnd}
                  onChange={(e) => setNewEnd(e.target.value)}
                  min={newStart || undefined}
                />
              </div>
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setCreateOpen(false)}>
              Cancel
            </Button>
            <Button
              onClick={handleCreate}
              disabled={
                createMutation.isPending || !newName || !newStart || !newEnd
              }
            >
              {createMutation.isPending ? "Creating..." : "Create"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Delete Confirmation */}
      <AlertDialog
        open={!!deleteTarget}
        onOpenChange={(open) => !open && setDeleteTarget(null)}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete Partition</AlertDialogTitle>
            <AlertDialogDescription>
              Are you sure you want to delete the partition &ldquo;
              {deleteTarget?.name}&rdquo;? This action cannot be undone.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => {
                if (deleteTarget) {
                  deleteMutation.mutate(deleteTarget.id, {
                    onSuccess: () => setDeleteTarget(null),
                  });
                }
              }}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              {deleteMutation.isPending ? "Deleting..." : "Delete"}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
