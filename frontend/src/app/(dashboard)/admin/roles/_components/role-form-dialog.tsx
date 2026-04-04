"use client";

import { useState, useEffect, useCallback } from "react";
import { useForm, FormProvider } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { ChevronRight, ChevronDown } from "lucide-react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { Checkbox } from "@/components/ui/checkbox";
import { Label } from "@/components/ui/label";
import { ScrollArea } from "@/components/ui/scroll-area";
import { FormField } from "@/components/shared/forms/form-field";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { toast } from "sonner";
import type { Role } from "@/types/models";

interface PermissionGroup {
  label: string;
  wildcard: string;
  permissions: Array<{ label: string; value: string }>;
}

const PERMISSION_TREE: PermissionGroup[] = [
  {
    label: "Cybersecurity",
    wildcard: "cyber:*",
    permissions: [
      { label: "Read data", value: "cyber:read" },
      { label: "Write data", value: "cyber:write" },
      { label: "Read alerts", value: "alerts:read" },
      { label: "Manage alerts", value: "alerts:write" },
      { label: "Execute remediation", value: "remediation:execute" },
      { label: "Approve remediation", value: "remediation:approve" },
    ],
  },
  {
    label: "Data Intelligence",
    wildcard: "data:*",
    permissions: [
      { label: "Read data", value: "data:read" },
      { label: "Write data", value: "data:write" },
      { label: "Manage pipelines", value: "pipelines:write" },
      { label: "View pipelines", value: "pipelines:read" },
      { label: "Data quality", value: "quality:read" },
      { label: "Data lineage", value: "lineage:read" },
    ],
  },
  {
    label: "Governance — Acta",
    wildcard: "acta:*",
    permissions: [
      { label: "View governance", value: "acta:read" },
      { label: "Manage governance", value: "acta:write" },
    ],
  },
  {
    label: "Legal — Lex",
    wildcard: "lex:*",
    permissions: [
      { label: "View legal", value: "lex:read" },
      { label: "Manage legal", value: "lex:write" },
    ],
  },
  {
    label: "Executive — Visus",
    wildcard: "visus:*",
    permissions: [
      { label: "View executive dashboards", value: "visus:read" },
    ],
  },
  {
    label: "Platform Admin",
    wildcard: "tenant:*",
    permissions: [
      { label: "View tenant settings", value: "tenant:read" },
      { label: "Manage tenant settings", value: "tenant:write" },
      { label: "View users", value: "users:read" },
      { label: "Manage users", value: "users:write" },
      { label: "View roles", value: "roles:read" },
      { label: "Manage roles", value: "roles:write" },
    ],
  },
  {
    label: "Reports",
    wildcard: "reports:*",
    permissions: [
      { label: "View reports", value: "reports:read" },
      { label: "Create reports", value: "reports:write" },
    ],
  },
  {
    label: "Audit",
    wildcard: "audit:*",
    permissions: [
      { label: "View audit logs", value: "audit:read" },
    ],
  },
];

const roleSchema = z.object({
  name: z.string().min(3, "Name must be at least 3 characters"),
  description: z.string().optional(),
  permissions: z.array(z.string()).min(1, "Select at least one permission"),
});

type RoleFormData = z.infer<typeof roleSchema>;

interface RoleFormDialogProps {
  role?: Role;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSuccess: () => void;
}

export function RoleFormDialog({ role, open, onOpenChange, onSuccess }: RoleFormDialogProps) {
  const isEdit = !!role;
  const [expandedGroups, setExpandedGroups] = useState<Set<string>>(new Set());

  const methods = useForm<RoleFormData>({
    resolver: zodResolver(roleSchema),
    defaultValues: {
      name: role?.name ?? "",
      description: role?.description ?? "",
      permissions: role?.permissions ?? [],
    },
  });

  useEffect(() => {
    if (role) {
      methods.reset({
        name: role.name,
        description: role.description,
        permissions: role.permissions,
      });
    } else {
      methods.reset({ name: "", description: "", permissions: [] });
    }
  }, [role, methods]);

  const selectedPermissions = new Set(methods.watch("permissions"));

  const getGroupState = useCallback(
    (group: PermissionGroup): "all" | "some" | "none" => {
      const groupPerms = group.permissions.map((p) => p.value);
      const selectedCount = groupPerms.filter((p) => selectedPermissions.has(p)).length;
      if (selectedCount === 0) return "none";
      if (selectedCount === groupPerms.length) return "all";
      return "some";
    },
    [selectedPermissions]
  );

  const toggleGroup = (group: PermissionGroup) => {
    const state = getGroupState(group);
    const groupPerms = group.permissions.map((p) => p.value);
    const current = new Set(methods.getValues("permissions"));
    if (state === "all") {
      groupPerms.forEach((p) => current.delete(p));
    } else {
      groupPerms.forEach((p) => current.add(p));
    }
    methods.setValue("permissions", [...current]);
  };

  const togglePermission = (value: string) => {
    const current = new Set(methods.getValues("permissions"));
    if (current.has(value)) current.delete(value);
    else current.add(value);
    methods.setValue("permissions", [...current]);
  };

  const toggleGroupExpanded = (label: string) => {
    setExpandedGroups((prev) => {
      const next = new Set(prev);
      if (next.has(label)) next.delete(label);
      else next.add(label);
      return next;
    });
  };

  function toSlug(name: string): string {
    return name
      .toLowerCase()
      .replace(/[^a-z0-9]+/g, "-")
      .replace(/^-+|-+$/g, "");
  }

  const mutation = useApiMutation<unknown, RoleFormData>(
    isEdit ? "put" : "post",
    isEdit ? `/api/v1/roles/${role!.id}` : "/api/v1/roles",
    {
      successMessage: isEdit ? "Role updated" : "Role created",
      invalidateKeys: ["roles"],
      onSuccess: () => {
        onOpenChange(false);
        onSuccess();
      },
    }
  );

  const onSubmit = methods.handleSubmit(async (data) => {
    if (isEdit && role?.is_system) {
      toast.error("Cannot modify system roles.");
      return;
    }
    const payload = isEdit
      ? data
      : { ...data, slug: toSlug(data.name) };
    await mutation.mutate(payload as RoleFormData);
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-lg max-h-[90vh] flex flex-col">
        <DialogHeader>
          <DialogTitle>{isEdit ? "Edit Role" : "Create Role"}</DialogTitle>
          <DialogDescription>
            {isEdit
              ? `Editing "${role!.name}"`
              : "Define a new role with specific permissions."}
          </DialogDescription>
        </DialogHeader>
        <FormProvider {...methods}>
          <form
            onSubmit={onSubmit}
            className="flex flex-col flex-1 overflow-hidden"
            noValidate
          >
            <div className="space-y-4 flex-shrink-0 px-1 pb-2">
              <FormField name="name" label="Role Name" required>
                <Input
                  {...methods.register("name")}
                  placeholder="e.g. Security Analyst"
                  disabled={mutation.isPending}
                />
              </FormField>
              <FormField name="description" label="Description">
                <Textarea
                  {...methods.register("description")}
                  placeholder="What can users with this role do?"
                  rows={2}
                  disabled={mutation.isPending}
                />
              </FormField>
            </div>

            <div className="flex-1 overflow-hidden px-1">
              <Label className="text-sm font-medium mb-2 block">
                Permissions
                {methods.formState.errors.permissions && (
                  <span className="text-destructive text-xs ml-2">
                    {methods.formState.errors.permissions.message}
                  </span>
                )}
              </Label>
              <ScrollArea className="h-64 border border-border rounded-md p-2">
                <div className="space-y-1">
                  {PERMISSION_TREE.map((group) => {
                    const state = getGroupState(group);
                    const isExpanded = expandedGroups.has(group.label);
                    return (
                      <div key={group.label}>
                        <div className="flex items-center gap-2 rounded px-2 py-1.5 hover:bg-muted/50">
                          <button
                            type="button"
                            className="p-0 focus:outline-none"
                            onClick={() => toggleGroupExpanded(group.label)}
                            aria-expanded={isExpanded}
                            aria-label={`${isExpanded ? "Collapse" : "Expand"} ${group.label}`}
                          >
                            {isExpanded ? (
                              <ChevronDown className="h-3.5 w-3.5 text-muted-foreground" />
                            ) : (
                              <ChevronRight className="h-3.5 w-3.5 text-muted-foreground" />
                            )}
                          </button>
                          <div
                            className="flex items-center gap-2 flex-1 cursor-pointer"
                            onClick={() => toggleGroup(group)}
                          >
                            <div
                              className={`flex h-4 w-4 items-center justify-center rounded border ${
                                state === "all"
                                  ? "bg-primary border-primary text-primary-foreground"
                                  : state === "some"
                                  ? "bg-primary/20 border-primary"
                                  : "border-input"
                              }`}
                            >
                              {state === "all" && (
                                <span className="text-[10px]">✓</span>
                              )}
                              {state === "some" && (
                                <span className="text-[10px] text-primary">—</span>
                              )}
                            </div>
                            <span className="text-sm font-medium">{group.label}</span>
                            <span className="text-xs text-muted-foreground ml-auto">
                              {
                                group.permissions.filter((p) =>
                                  selectedPermissions.has(p.value)
                                ).length
                              }
                              /{group.permissions.length}
                            </span>
                          </div>
                        </div>
                        {isExpanded && (
                          <div className="ml-8 space-y-1 mt-1">
                            {group.permissions.map((perm) => (
                              <div
                                key={perm.value}
                                className="flex items-center gap-2 rounded px-2 py-1 hover:bg-muted/30 cursor-pointer"
                                onClick={() => togglePermission(perm.value)}
                              >
                                <Checkbox
                                  checked={selectedPermissions.has(perm.value)}
                                  onCheckedChange={() => togglePermission(perm.value)}
                                  id={`perm-${perm.value}`}
                                  onClick={(e) => e.stopPropagation()}
                                />
                                <Label
                                  htmlFor={`perm-${perm.value}`}
                                  className="text-sm cursor-pointer flex-1"
                                  onClick={(e) => e.stopPropagation()}
                                >
                                  {perm.label}
                                </Label>
                                <code className="text-xs text-muted-foreground font-mono">
                                  {perm.value}
                                </code>
                              </div>
                            ))}
                          </div>
                        )}
                      </div>
                    );
                  })}
                </div>
              </ScrollArea>
            </div>

            <DialogFooter className="pt-4 flex-shrink-0">
              <Button
                type="button"
                variant="outline"
                onClick={() => onOpenChange(false)}
                disabled={mutation.isPending}
              >
                Cancel
              </Button>
              <Button type="submit" disabled={mutation.isPending}>
                {mutation.isPending
                  ? "Saving..."
                  : isEdit
                  ? "Save Changes"
                  : "Create Role"}
              </Button>
            </DialogFooter>
          </form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}
