"use client";

import { useForm, FormProvider } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { Palette } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Label } from "@/components/ui/label";
import { FormField } from "@/components/shared/forms/form-field";
import { useUpdateTenantSettings } from "@/hooks/use-tenants";
import type { Tenant } from "@/types/tenant";

const brandingSchema = z.object({
  company_name: z.string().min(1, "Company name is required"),
  primary_color: z.string().regex(/^#[0-9a-fA-F]{6}$/, "Must be a valid hex color"),
  accent_color: z.string().regex(/^#[0-9a-fA-F]{6}$/, "Must be a valid hex color"),
  logo_url: z.string().url("Must be a valid URL").nullable().or(z.literal("")),
});

type BrandingFormData = z.infer<typeof brandingSchema>;

interface TenantBrandingFormProps {
  tenant: Tenant;
  onSuccess: () => void;
}

export function TenantBrandingForm({ tenant, onSuccess }: TenantBrandingFormProps) {
  const updateSettings = useUpdateTenantSettings();

  const branding = tenant.settings?.branding;

  const methods = useForm<BrandingFormData>({
    resolver: zodResolver(brandingSchema),
    defaultValues: {
      company_name: branding?.company_name ?? tenant.name,
      primary_color: branding?.primary_color ?? "#1B5E20",
      accent_color: branding?.accent_color ?? "#C6A962",
      logo_url: branding?.logo_url ?? "",
    },
  });

  const onSubmit = methods.handleSubmit(async (data) => {
    // Store branding inside the settings JSONB field
    const mergedSettings = {
      ...tenant.settings,
      branding: {
        ...data,
        logo_url: data.logo_url || null,
      },
    };

    await updateSettings.mutateAsync({
      tenantId: tenant.id,
      settings: mergedSettings,
    });
    onSuccess();
  });

  const primaryColor = methods.watch("primary_color");
  const accentColor = methods.watch("accent_color");

  return (
    <FormProvider {...methods}>
      <form onSubmit={onSubmit} className="space-y-6">
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Palette className="h-5 w-5" />
              Branding
            </CardTitle>
            <CardDescription>Customize the look and feel for this tenant</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <FormField name="company_name" label="Company Name" required>
              <Input
                {...methods.register("company_name")}
                disabled={updateSettings.isPending}
              />
            </FormField>

            <FormField name="logo_url" label="Logo URL">
              <Input
                {...methods.register("logo_url")}
                placeholder="https://example.com/logo.png"
                disabled={updateSettings.isPending}
              />
            </FormField>

            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div className="space-y-1.5">
                <Label htmlFor="primary_color">Primary Color</Label>
                <div className="flex items-center gap-2">
                  <input
                    type="color"
                    id="primary_color_picker"
                    value={primaryColor}
                    onChange={(e) =>
                      methods.setValue("primary_color", e.target.value, { shouldDirty: true })
                    }
                    className="h-10 w-10 rounded border border-input cursor-pointer"
                    disabled={updateSettings.isPending}
                  />
                  <Input
                    id="primary_color"
                    {...methods.register("primary_color")}
                    placeholder="#000000"
                    className="font-mono"
                    disabled={updateSettings.isPending}
                  />
                </div>
                {methods.formState.errors.primary_color && (
                  <p className="text-xs text-destructive" role="alert">
                    {methods.formState.errors.primary_color.message}
                  </p>
                )}
              </div>

              <div className="space-y-1.5">
                <Label htmlFor="accent_color">Accent Color</Label>
                <div className="flex items-center gap-2">
                  <input
                    type="color"
                    id="accent_color_picker"
                    value={accentColor}
                    onChange={(e) =>
                      methods.setValue("accent_color", e.target.value, { shouldDirty: true })
                    }
                    className="h-10 w-10 rounded border border-input cursor-pointer"
                    disabled={updateSettings.isPending}
                  />
                  <Input
                    id="accent_color"
                    {...methods.register("accent_color")}
                    placeholder="#000000"
                    className="font-mono"
                    disabled={updateSettings.isPending}
                  />
                </div>
                {methods.formState.errors.accent_color && (
                  <p className="text-xs text-destructive" role="alert">
                    {methods.formState.errors.accent_color.message}
                  </p>
                )}
              </div>
            </div>

            {/* Preview */}
            <div className="rounded-lg border p-4 space-y-3">
              <p className="text-sm font-medium text-muted-foreground">Preview</p>
              <div className="flex items-center gap-4">
                <div
                  className="h-10 w-10 rounded"
                  style={{ backgroundColor: primaryColor }}
                  aria-label="Primary color preview"
                />
                <div
                  className="h-10 w-10 rounded"
                  style={{ backgroundColor: accentColor }}
                  aria-label="Accent color preview"
                />
                <span className="text-sm font-medium">
                  {methods.watch("company_name") || "Company Name"}
                </span>
              </div>
            </div>
          </CardContent>
        </Card>

        <div className="flex justify-end">
          <Button type="submit" disabled={updateSettings.isPending || !methods.formState.isDirty}>
            {updateSettings.isPending ? "Saving..." : "Save Branding"}
          </Button>
        </div>
      </form>
    </FormProvider>
  );
}
