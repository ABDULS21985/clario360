"use client";

import { useEffect } from "react";
import { useForm, FormProvider } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter } from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { FormField } from "@/components/shared/forms/form-field";
import { useApiMutation } from "@/hooks/use-api-mutation";
import type { User } from "@/types/models";

// Backend UpdateUserRequest only accepts: { first_name?, last_name?, avatar_url? }
// Status changes go through PUT /api/v1/users/{id}/status (separate endpoint)
const editUserSchema = z.object({
  first_name: z.string().min(2, "First name must be at least 2 characters"),
  last_name: z.string().min(2, "Last name must be at least 2 characters"),
});

type EditUserFormData = z.infer<typeof editUserSchema>;

interface UserEditDialogProps {
  user: User;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSuccess: () => void;
}

export function UserEditDialog({ user, open, onOpenChange, onSuccess }: UserEditDialogProps) {
  const methods = useForm<EditUserFormData>({
    resolver: zodResolver(editUserSchema),
    defaultValues: {
      first_name: user.first_name,
      last_name: user.last_name,
    },
  });

  useEffect(() => {
    if (user) {
      methods.reset({
        first_name: user.first_name,
        last_name: user.last_name,
      });
    }
  }, [user, methods]);

  const updateMutation = useApiMutation<unknown, EditUserFormData>(
    "put",
    `/api/v1/users/${user.id}`,
    {
      successMessage: "User updated successfully",
      invalidateKeys: ["users"],
      onSuccess: () => {
        onOpenChange(false);
        onSuccess();
      },
    }
  );

  const onSubmit = methods.handleSubmit(async (data) => {
    await updateMutation.mutate(data);
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>Edit User</DialogTitle>
          <DialogDescription>
            Update user information for {user.first_name} {user.last_name}.
          </DialogDescription>
        </DialogHeader>
        <FormProvider {...methods}>
          <form onSubmit={onSubmit} className="space-y-4" noValidate>
            <div className="grid grid-cols-2 gap-4">
              <FormField name="first_name" label="First Name" required>
                <Input
                  {...methods.register("first_name")}
                  disabled={updateMutation.isPending}
                  aria-invalid={!!methods.formState.errors.first_name}
                />
              </FormField>
              <FormField name="last_name" label="Last Name" required>
                <Input
                  {...methods.register("last_name")}
                  disabled={updateMutation.isPending}
                  aria-invalid={!!methods.formState.errors.last_name}
                />
              </FormField>
            </div>
            <FormField name="email" label="Email Address">
              <Input value={user.email} disabled className="bg-muted" />
            </FormField>
            <DialogFooter>
              <Button
                type="button"
                variant="outline"
                onClick={() => onOpenChange(false)}
                disabled={updateMutation.isPending}
              >
                Cancel
              </Button>
              <Button type="submit" disabled={updateMutation.isPending}>
                {updateMutation.isPending ? "Saving..." : "Save Changes"}
              </Button>
            </DialogFooter>
          </form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}
