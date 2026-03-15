"use client";

import { useState, useEffect } from "react";
import { useForm, FormProvider } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { toast } from "sonner";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter } from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Checkbox } from "@/components/ui/checkbox";
import { Label } from "@/components/ui/label";
import { FormField } from "@/components/shared/forms/form-field";
import { MultiSelect } from "@/components/shared/forms/multi-select";
import { useApiQuery } from "@/hooks/use-api";
import api from "@/lib/api";
import type { Role, User } from "@/types/models";

const passwordRegex = /^(?=.*[A-Z])(?=.*[a-z])(?=.*\d)(?=.*[^A-Za-z\d]).{12,}$/;

const createUserSchema = z.object({
  first_name: z.string().min(2, "First name must be at least 2 characters"),
  last_name: z.string().min(2, "Last name must be at least 2 characters"),
  email: z.string().email("Please enter a valid email address"),
  password: z
    .string()
    .min(12, "Password must be at least 12 characters")
    .regex(passwordRegex, "Password must contain uppercase, lowercase, number, and special character"),
  status: z.enum(["active", "suspended", "inactive"]),
  role_ids: z.array(z.string()),
  send_welcome_email: z.boolean(),
});

type CreateUserFormData = z.infer<typeof createUserSchema>;

interface UserCreateDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSuccess: () => void;
}

export function UserCreateDialog({ open, onOpenChange, onSuccess }: UserCreateDialogProps) {
  const [submitting, setSubmitting] = useState(false);

  const methods = useForm<CreateUserFormData>({
    resolver: zodResolver(createUserSchema),
    defaultValues: {
      first_name: "",
      last_name: "",
      email: "",
      password: "",
      status: "active",
      role_ids: [],
      send_welcome_email: true,
    },
  });

  // Backend returns plain Role[] (not paginated)
  const { data: roles } = useApiQuery<Role[]>(
    ["roles"],
    "/api/v1/roles",
    { enabled: open }
  );

  const roleOptions = (roles ?? []).map((r) => ({ label: r.name, value: r.id }));

  const onSubmit = methods.handleSubmit(async (data) => {
    setSubmitting(true);
    try {
      // Admin create endpoint: POST /api/v1/users
      // Backend AdminCreateUserRequest: { email, password, first_name, last_name, status?, role_ids?, send_welcome_email? }
      await api.post<User>("/api/v1/users", {
        email: data.email,
        password: data.password,
        first_name: data.first_name,
        last_name: data.last_name,
        status: data.status,
        role_ids: data.role_ids.length > 0 ? data.role_ids : undefined,
        send_welcome_email: data.send_welcome_email,
      });

      toast.success("User created successfully");
      onOpenChange(false);
      methods.reset();
      onSuccess();
    } catch (err) {
      const message =
        err && typeof err === "object" && "message" in err
          ? String((err as { message: string }).message)
          : "Failed to create user";
      toast.error(message);
    } finally {
      setSubmitting(false);
    }
  });

  useEffect(() => {
    if (!open) methods.reset();
  }, [open, methods]);

  const password = methods.watch("password");
  const strength = {
    length: password.length >= 12,
    upper: /[A-Z]/.test(password),
    lower: /[a-z]/.test(password),
    digit: /\d/.test(password),
    special: /[^A-Za-z\d]/.test(password),
  };
  const strengthScore = Object.values(strength).filter(Boolean).length;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-lg max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>Add New User</DialogTitle>
          <DialogDescription>Create a new user account for your organization.</DialogDescription>
        </DialogHeader>

        <FormProvider {...methods}>
          <form onSubmit={onSubmit} className="space-y-5" noValidate>
            <div className="grid grid-cols-2 gap-4">
              <FormField name="first_name" label="First Name" required>
                <Input
                  {...methods.register("first_name")}
                  placeholder="John"
                  disabled={submitting}
                  aria-invalid={!!methods.formState.errors.first_name}
                />
              </FormField>
              <FormField name="last_name" label="Last Name" required>
                <Input
                  {...methods.register("last_name")}
                  placeholder="Doe"
                  disabled={submitting}
                  aria-invalid={!!methods.formState.errors.last_name}
                />
              </FormField>
            </div>

            <FormField name="email" label="Email Address" required>
              <Input
                {...methods.register("email")}
                type="email"
                placeholder="john@company.com"
                disabled={submitting}
                aria-invalid={!!methods.formState.errors.email}
              />
            </FormField>

            <FormField name="password" label="Password" required>
              <div className="space-y-2">
                <Input
                  {...methods.register("password")}
                  type="password"
                  placeholder="Create a strong password"
                  disabled={submitting}
                  aria-invalid={!!methods.formState.errors.password}
                />
                {password.length > 0 && (
                  <div className="space-y-1">
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
                    <ul className="grid grid-cols-2 gap-x-4 gap-y-0.5 text-xs text-muted-foreground">
                      {Object.entries({
                        "12+ characters": strength.length,
                        Uppercase: strength.upper,
                        Lowercase: strength.lower,
                        Number: strength.digit,
                        "Special character": strength.special,
                      }).map(([label, met]) => (
                        <li key={label} className={met ? "text-green-600" : ""}>
                          {met ? "✓" : "○"} {label}
                        </li>
                      ))}
                    </ul>
                  </div>
                )}
              </div>
            </FormField>

            <div className="grid grid-cols-2 gap-4">
              <FormField name="status" label="Initial Status">
                <Select
                  defaultValue="active"
                  onValueChange={(v) =>
                    methods.setValue("status", v as "active" | "suspended" | "inactive")
                  }
                  disabled={submitting}
                >
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="active">Active</SelectItem>
                    <SelectItem value="suspended">Suspended</SelectItem>
                    <SelectItem value="inactive">Inactive</SelectItem>
                  </SelectContent>
                </Select>
              </FormField>
            </div>

            <FormField name="role_ids" label="Assign Roles">
              <MultiSelect
                options={roleOptions}
                selected={methods.watch("role_ids")}
                onChange={(vals) => methods.setValue("role_ids", vals)}
                placeholder="Select roles..."
                disabled={submitting}
              />
            </FormField>

            <div className="flex items-center gap-2">
              <Checkbox
                id="send_welcome_email"
                checked={methods.watch("send_welcome_email")}
                onCheckedChange={(checked) =>
                  methods.setValue("send_welcome_email", checked === true)
                }
                disabled={submitting}
              />
              <Label htmlFor="send_welcome_email" className="text-sm font-normal cursor-pointer">
                Send welcome email to the new user
              </Label>
            </div>

            <DialogFooter>
              <Button
                type="button"
                variant="outline"
                onClick={() => onOpenChange(false)}
                disabled={submitting}
              >
                Cancel
              </Button>
              <Button type="submit" disabled={submitting}>
                {submitting ? "Creating..." : "Create User"}
              </Button>
            </DialogFooter>
          </form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}
