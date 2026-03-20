"use client";

import { useForm, FormProvider } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Checkbox } from "@/components/ui/checkbox";
import { Label } from "@/components/ui/label";
import { FormField } from "@/components/shared/forms/form-field";
import { useUpdateTenantSettings } from "@/hooks/use-tenants";
import type { Tenant } from "@/types/tenant";

const settingsSchema = z.object({
  max_users: z.coerce.number().min(1, "Must be at least 1"),
  max_storage_gb: z.coerce.number().min(1, "Must be at least 1 GB"),
  mfa_required: z.boolean(),
  session_timeout_minutes: z.coerce.number().min(5, "Minimum 5 minutes").max(1440, "Maximum 24 hours"),
  enabled_suites: z.array(z.string()),
  ip_whitelist_raw: z.string(),
  custom_domain: z.string().nullable(),
  password_policy: z.object({
    min_length: z.coerce.number().min(8, "Minimum 8 characters"),
    require_uppercase: z.boolean(),
    require_lowercase: z.boolean(),
    require_numbers: z.boolean(),
    require_special: z.boolean(),
    max_age_days: z.coerce.number().min(0),
    history_count: z.coerce.number().min(0),
  }),
});

type SettingsFormData = z.infer<typeof settingsSchema>;

const AVAILABLE_SUITES = ["cyber", "data", "acta", "lex", "visus"];

const DEFAULT_PASSWORD_POLICY = {
  min_length: 8,
  require_uppercase: true,
  require_lowercase: true,
  require_numbers: true,
  require_special: false,
  max_age_days: 90,
  history_count: 5,
};

interface TenantSettingsFormProps {
  tenant: Tenant;
  onSuccess: () => void;
}

export function TenantSettingsForm({ tenant, onSuccess }: TenantSettingsFormProps) {
  const updateSettings = useUpdateTenantSettings();

  const methods = useForm<SettingsFormData>({
    resolver: zodResolver(settingsSchema),
    defaultValues: {
      max_users: tenant.settings?.max_users ?? 10,
      max_storage_gb: tenant.settings?.max_storage_gb ?? 10,
      mfa_required: tenant.settings?.mfa_required ?? false,
      session_timeout_minutes: tenant.settings?.session_timeout_minutes ?? 30,
      enabled_suites: tenant.settings?.enabled_suites ?? [],
      ip_whitelist_raw: (tenant.settings?.ip_whitelist ?? []).join("\n"),
      custom_domain: tenant.settings?.custom_domain ?? null,
      password_policy: tenant.settings?.password_policy ?? DEFAULT_PASSWORD_POLICY,
    },
  });

  const onSubmit = methods.handleSubmit(async (data) => {
    const { ip_whitelist_raw, ...rest } = data;
    const ip_whitelist = ip_whitelist_raw
      .split("\n")
      .map((ip) => ip.trim())
      .filter(Boolean);

    // Merge with existing settings to preserve branding and other fields
    const mergedSettings = {
      ...tenant.settings,
      ...rest,
      ip_whitelist,
    };

    await updateSettings.mutateAsync({
      tenantId: tenant.id,
      settings: mergedSettings,
    });
    onSuccess();
  });

  const enabledSuites = methods.watch("enabled_suites");

  const toggleSuite = (suite: string) => {
    const current = methods.getValues("enabled_suites");
    if (current.includes(suite)) {
      methods.setValue(
        "enabled_suites",
        current.filter((s) => s !== suite),
        { shouldDirty: true },
      );
    } else {
      methods.setValue("enabled_suites", [...current, suite], { shouldDirty: true });
    }
  };

  return (
    <FormProvider {...methods}>
      <form onSubmit={onSubmit} className="space-y-6">
        <Card>
          <CardHeader>
            <CardTitle>General Settings</CardTitle>
            <CardDescription>Configure tenant limits and access controls</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <FormField name="max_users" label="Max Users" required>
                <Input
                  type="number"
                  {...methods.register("max_users")}
                  disabled={updateSettings.isPending}
                />
              </FormField>
              <FormField name="max_storage_gb" label="Max Storage (GB)" required>
                <Input
                  type="number"
                  {...methods.register("max_storage_gb")}
                  disabled={updateSettings.isPending}
                />
              </FormField>
              <FormField name="session_timeout_minutes" label="Session Timeout (minutes)" required>
                <Input
                  type="number"
                  {...methods.register("session_timeout_minutes")}
                  disabled={updateSettings.isPending}
                />
              </FormField>
              <FormField name="custom_domain" label="Custom Domain">
                <Input
                  {...methods.register("custom_domain")}
                  placeholder="app.example.com"
                  disabled={updateSettings.isPending}
                />
              </FormField>
            </div>

            <div className="flex items-center gap-2">
              <Checkbox
                id="mfa_required"
                checked={methods.watch("mfa_required")}
                onCheckedChange={(checked) =>
                  methods.setValue("mfa_required", !!checked, { shouldDirty: true })
                }
                disabled={updateSettings.isPending}
              />
              <Label htmlFor="mfa_required">Require MFA for all users</Label>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Enabled Suites</CardTitle>
            <CardDescription>Select which platform suites this tenant can access</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 gap-3">
              {AVAILABLE_SUITES.map((suite) => (
                <div key={suite} className="flex items-center gap-2">
                  <Checkbox
                    id={`suite-${suite}`}
                    checked={enabledSuites.includes(suite)}
                    onCheckedChange={() => toggleSuite(suite)}
                    disabled={updateSettings.isPending}
                  />
                  <Label htmlFor={`suite-${suite}`} className="capitalize cursor-pointer">
                    {suite}
                  </Label>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Password Policy</CardTitle>
            <CardDescription>Configure password requirements for tenant users</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <FormField name="password_policy.min_length" label="Minimum Length" required>
                <Input
                  type="number"
                  {...methods.register("password_policy.min_length")}
                  disabled={updateSettings.isPending}
                />
              </FormField>
              <FormField name="password_policy.max_age_days" label="Max Age (days)">
                <Input
                  type="number"
                  {...methods.register("password_policy.max_age_days")}
                  disabled={updateSettings.isPending}
                />
              </FormField>
              <FormField name="password_policy.history_count" label="Password History Count">
                <Input
                  type="number"
                  {...methods.register("password_policy.history_count")}
                  disabled={updateSettings.isPending}
                />
              </FormField>
            </div>
            <div className="space-y-2">
              {(
                [
                  ["require_uppercase", "Require uppercase letter"],
                  ["require_lowercase", "Require lowercase letter"],
                  ["require_numbers", "Require number"],
                  ["require_special", "Require special character"],
                ] as const
              ).map(([key, label]) => (
                <div key={key} className="flex items-center gap-2">
                  <Checkbox
                    id={`pp-${key}`}
                    checked={methods.watch(`password_policy.${key}`)}
                    onCheckedChange={(checked) =>
                      methods.setValue(`password_policy.${key}`, !!checked, { shouldDirty: true })
                    }
                    disabled={updateSettings.isPending}
                  />
                  <Label htmlFor={`pp-${key}`} className="cursor-pointer">
                    {label}
                  </Label>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>IP Whitelist</CardTitle>
            <CardDescription>One IP or CIDR range per line. Leave empty to allow all.</CardDescription>
          </CardHeader>
          <CardContent>
            <textarea
              className="w-full rounded-md border border-input bg-background px-3 py-2 text-sm font-mono min-h-[100px] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
              {...methods.register("ip_whitelist_raw")}
              placeholder={"192.168.1.0/24\n10.0.0.1"}
              disabled={updateSettings.isPending}
            />
          </CardContent>
        </Card>

        <div className="flex justify-end">
          <Button type="submit" disabled={updateSettings.isPending || !methods.formState.isDirty}>
            {updateSettings.isPending ? "Saving..." : "Save Settings"}
          </Button>
        </div>
      </form>
    </FormProvider>
  );
}
