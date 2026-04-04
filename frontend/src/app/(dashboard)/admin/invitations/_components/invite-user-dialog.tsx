"use client";

import { useForm, FormProvider } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { FormField } from "@/components/shared/forms/form-field";
import { useCreateInvitation } from "@/hooks/use-invitations";
import { useApiQuery } from "@/hooks/use-api";
import type { Role } from "@/types/models";

const inviteSchema = z.object({
  email: z.string().email("Invalid email address"),
  role_slug: z.string().min(1, "Role is required"),
  message: z.string().optional(),
});

type InviteFormData = z.infer<typeof inviteSchema>;

interface InviteUserDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSuccess: () => void;
}

export function InviteUserDialog({ open, onOpenChange, onSuccess }: InviteUserDialogProps) {
  const createMutation = useCreateInvitation();

  const { data: roles } = useApiQuery<Role[]>(["roles"], "/api/v1/roles");

  const methods = useForm<InviteFormData>({
    resolver: zodResolver(inviteSchema),
    defaultValues: {
      email: "",
      role_slug: "",
      message: "",
    },
  });

  const handleClose = () => {
    methods.reset();
    onOpenChange(false);
  };

  const onSubmit = methods.handleSubmit(async (data) => {
    await createMutation.mutateAsync({
      invitations: [
        {
          email: data.email,
          role_slug: data.role_slug,
          message: data.message || undefined,
        },
      ],
    });
    handleClose();
    onSuccess();
  });

  return (
    <Dialog open={open} onOpenChange={(o) => !o && handleClose()}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>Invite User</DialogTitle>
          <DialogDescription>
            Send an invitation email to join the platform
          </DialogDescription>
        </DialogHeader>

        <FormProvider {...methods}>
          <form onSubmit={onSubmit} className="space-y-4">
            <FormField name="email" label="Email Address" required>
              <Input
                type="email"
                {...methods.register("email")}
                placeholder="user@example.com"
                disabled={createMutation.isPending}
                autoFocus
              />
            </FormField>

            <FormField name="role_slug" label="Role" required>
              <Select
                value={methods.watch("role_slug")}
                onValueChange={(v) => methods.setValue("role_slug", v, { shouldValidate: true })}
                disabled={createMutation.isPending}
              >
                <SelectTrigger>
                  <SelectValue placeholder="Select a role" />
                </SelectTrigger>
                <SelectContent>
                  {(roles ?? []).map((role) => (
                    <SelectItem key={role.slug} value={role.slug}>
                      {role.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </FormField>

            <FormField name="message" label="Message (optional)">
              <Textarea
                {...methods.register("message")}
                placeholder="Add a personal message to the invitation..."
                rows={3}
                disabled={createMutation.isPending}
              />
            </FormField>

            <DialogFooter>
              <Button
                type="button"
                variant="outline"
                onClick={handleClose}
                disabled={createMutation.isPending}
              >
                Cancel
              </Button>
              <Button type="submit" disabled={createMutation.isPending}>
                {createMutation.isPending ? "Sending..." : "Send Invitation"}
              </Button>
            </DialogFooter>
          </form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}
