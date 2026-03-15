"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { useForm, FormProvider } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import {
  ArrowLeft,
  ArrowRight,
  Building2,
  User,
  Settings,
  CheckCircle,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Checkbox } from "@/components/ui/checkbox";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { PageHeader } from "@/components/common/page-header";
import { FormField } from "@/components/shared/forms/form-field";
import { useProvisionTenant } from "@/hooks/use-tenants";
import { cn } from "@/lib/utils";
import type { SubscriptionTier } from "@/types/tenant";
import Link from "next/link";

const provisionSchema = z.object({
  name: z.string().min(2, "Name must be at least 2 characters"),
  slug: z
    .string()
    .min(2, "Slug must be at least 2 characters")
    .max(50)
    .regex(/^[a-z0-9-]+$/, "Only lowercase letters, numbers, and hyphens"),
  subscription_tier: z.enum(["free", "starter", "professional", "enterprise"]),
  owner_email: z.string().email("Invalid email address"),
  owner_name: z.string().min(1, "Owner name is required"),
  max_users: z.coerce.number().min(1).default(10),
  max_storage_gb: z.coerce.number().min(1).default(10),
  mfa_required: z.boolean().default(false),
  enabled_suites: z.array(z.string()).default([]),
});

type ProvisionFormData = z.infer<typeof provisionSchema>;

const STEPS = [
  { id: 1, label: "Tenant Info", icon: Building2 },
  { id: 2, label: "Owner", icon: User },
  { id: 3, label: "Settings", icon: Settings },
  { id: 4, label: "Review", icon: CheckCircle },
];

const AVAILABLE_SUITES = [
  { value: "cyber", label: "Cybersecurity" },
  { value: "data", label: "Data Intelligence" },
  { value: "acta", label: "Acta (Governance)" },
  { value: "lex", label: "Lex (Legal)" },
  { value: "visus", label: "Visus (Executive)" },
];

export default function ProvisionTenantPage() {
  const router = useRouter();
  const [step, setStep] = useState(1);
  const provisionMutation = useProvisionTenant();

  const methods = useForm<ProvisionFormData>({
    resolver: zodResolver(provisionSchema),
    mode: "onBlur",
    defaultValues: {
      name: "",
      slug: "",
      subscription_tier: "professional",
      owner_email: "",
      owner_name: "",
      max_users: 10,
      max_storage_gb: 10,
      mfa_required: false,
      enabled_suites: ["cyber", "data"],
    },
  });

  const generateSlug = (name: string) => {
    return name
      .toLowerCase()
      .replace(/[^a-z0-9]+/g, "-")
      .replace(/^-|-$/g, "");
  };

  const canAdvance = async (): Promise<boolean> => {
    switch (step) {
      case 1:
        return methods.trigger(["name", "slug", "subscription_tier"]);
      case 2:
        return methods.trigger(["owner_email", "owner_name"]);
      case 3:
        return methods.trigger(["max_users", "max_storage_gb"]);
      default:
        return true;
    }
  };

  const handleNext = async () => {
    if (await canAdvance()) {
      setStep((s) => Math.min(s + 1, 4));
    }
  };

  const handleBack = () => {
    setStep((s) => Math.max(s - 1, 1));
  };

  const onSubmit = methods.handleSubmit(async (data) => {
    const result = await provisionMutation.mutateAsync({
      name: data.name,
      slug: data.slug,
      subscription_tier: data.subscription_tier as SubscriptionTier,
      owner_email: data.owner_email,
      owner_name: data.owner_name,
      settings: {
        max_users: data.max_users,
        max_storage_gb: data.max_storage_gb,
        mfa_required: data.mfa_required,
        enabled_suites: data.enabled_suites,
      },
    });
    router.push(`/admin/tenants/${result.id}`);
  });

  const values = methods.watch();

  const tierLabel = (tier: string) => {
    const labels: Record<string, string> = {
      free: "Free",
      starter: "Starter",
      professional: "Professional",
      enterprise: "Enterprise",
    };
    return labels[tier] ?? tier;
  };

  return (
    <div className="w-full space-y-6">
      <div className="flex items-center gap-2">
        <Button variant="ghost" size="icon" asChild>
          <Link href="/admin/tenants" aria-label="Back to tenants">
            <ArrowLeft className="h-4 w-4" />
          </Link>
        </Button>
        <PageHeader
          title="Provision New Tenant"
          description="Set up a new tenant environment"
        />
      </div>

      {/* Stepper */}
      <nav aria-label="Provision steps" className="flex items-center justify-between">
        {STEPS.map((s, i) => {
          const Icon = s.icon;
          const isActive = step === s.id;
          const isCompleted = step > s.id;
          return (
            <div key={s.id} className="flex items-center flex-1">
              <div className="flex flex-col items-center gap-1.5">
                <div
                  className={cn(
                    "flex h-10 w-10 items-center justify-center rounded-full border-2 transition-colors",
                    isActive && "border-primary bg-primary text-primary-foreground",
                    isCompleted && "border-primary bg-primary/10 text-primary",
                    !isActive && !isCompleted && "border-muted-foreground/30 text-muted-foreground",
                  )}
                >
                  {isCompleted ? (
                    <CheckCircle className="h-5 w-5" />
                  ) : (
                    <Icon className="h-5 w-5" />
                  )}
                </div>
                <span
                  className={cn(
                    "text-xs font-medium",
                    isActive ? "text-primary" : "text-muted-foreground",
                  )}
                >
                  {s.label}
                </span>
              </div>
              {i < STEPS.length - 1 && (
                <div
                  className={cn(
                    "flex-1 h-0.5 mx-3 mt-[-20px]",
                    isCompleted ? "bg-primary" : "bg-muted",
                  )}
                />
              )}
            </div>
          );
        })}
      </nav>

      <FormProvider {...methods}>
        <form onSubmit={onSubmit}>
          {/* Step 1: Tenant Info */}
          {step === 1 && (
            <Card>
              <CardHeader>
                <CardTitle>Tenant Information</CardTitle>
                <CardDescription>Basic details for the new tenant</CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                <FormField name="name" label="Tenant Name" required>
                  <Input
                    {...methods.register("name", {
                      onChange: (e) => {
                        const slug = generateSlug(e.target.value);
                        if (!methods.formState.dirtyFields.slug) {
                          methods.setValue("slug", slug);
                        }
                      },
                    })}
                    placeholder="Acme Corporation"
                  />
                </FormField>

                <FormField name="slug" label="Slug" required description="Used in URLs and API references">
                  <Input
                    {...methods.register("slug")}
                    placeholder="acme-corp"
                    className="font-mono"
                  />
                </FormField>

                <FormField name="subscription_tier" label="Plan" required>
                  <Select
                    value={methods.watch("subscription_tier")}
                    onValueChange={(v) =>
                      methods.setValue("subscription_tier", v as SubscriptionTier, { shouldValidate: true })
                    }
                  >
                    <SelectTrigger>
                      <SelectValue placeholder="Select a plan" />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="free">Free</SelectItem>
                      <SelectItem value="starter">Starter</SelectItem>
                      <SelectItem value="professional">Professional</SelectItem>
                      <SelectItem value="enterprise">Enterprise</SelectItem>
                    </SelectContent>
                  </Select>
                </FormField>
              </CardContent>
            </Card>
          )}

          {/* Step 2: Owner */}
          {step === 2 && (
            <Card>
              <CardHeader>
                <CardTitle>Tenant Owner</CardTitle>
                <CardDescription>The primary administrator for this tenant</CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                <FormField name="owner_name" label="Owner Name" required>
                  <Input {...methods.register("owner_name")} placeholder="John Doe" />
                </FormField>

                <FormField name="owner_email" label="Owner Email" required>
                  <Input
                    type="email"
                    {...methods.register("owner_email")}
                    placeholder="john@acme.com"
                  />
                </FormField>
              </CardContent>
            </Card>
          )}

          {/* Step 3: Settings */}
          {step === 3 && (
            <Card>
              <CardHeader>
                <CardTitle>Initial Settings</CardTitle>
                <CardDescription>Configure limits and access for the tenant</CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="grid grid-cols-2 gap-4">
                  <FormField name="max_users" label="Max Users" required>
                    <Input type="number" {...methods.register("max_users")} />
                  </FormField>
                  <FormField name="max_storage_gb" label="Max Storage (GB)" required>
                    <Input type="number" {...methods.register("max_storage_gb")} />
                  </FormField>
                </div>

                <div className="flex items-center gap-2">
                  <Checkbox
                    id="mfa_required"
                    checked={methods.watch("mfa_required")}
                    onCheckedChange={(checked) =>
                      methods.setValue("mfa_required", !!checked)
                    }
                  />
                  <Label htmlFor="mfa_required" className="cursor-pointer">
                    Require MFA for all users
                  </Label>
                </div>

                <div className="space-y-2">
                  <Label>Enabled Suites</Label>
                  <div className="grid grid-cols-2 gap-2">
                    {AVAILABLE_SUITES.map((suite) => (
                      <div key={suite.value} className="flex items-center gap-2">
                        <Checkbox
                          id={`suite-${suite.value}`}
                          checked={values.enabled_suites.includes(suite.value)}
                          onCheckedChange={(checked) => {
                            const current = methods.getValues("enabled_suites");
                            if (checked) {
                              methods.setValue("enabled_suites", [...current, suite.value]);
                            } else {
                              methods.setValue(
                                "enabled_suites",
                                current.filter((s) => s !== suite.value),
                              );
                            }
                          }}
                        />
                        <Label htmlFor={`suite-${suite.value}`} className="cursor-pointer">
                          {suite.label}
                        </Label>
                      </div>
                    ))}
                  </div>
                </div>
              </CardContent>
            </Card>
          )}

          {/* Step 4: Review */}
          {step === 4 && (
            <Card>
              <CardHeader>
                <CardTitle>Review & Confirm</CardTitle>
                <CardDescription>Verify the details before provisioning</CardDescription>
              </CardHeader>
              <CardContent>
                <dl className="space-y-4 text-sm">
                  <div className="grid grid-cols-2 gap-4">
                    <div>
                      <dt className="text-muted-foreground">Tenant Name</dt>
                      <dd className="font-medium mt-0.5">{values.name}</dd>
                    </div>
                    <div>
                      <dt className="text-muted-foreground">Slug</dt>
                      <dd className="font-mono text-xs mt-0.5">{values.slug}</dd>
                    </div>
                    <div>
                      <dt className="text-muted-foreground">Plan</dt>
                      <dd className="capitalize mt-0.5">{tierLabel(values.subscription_tier)}</dd>
                    </div>
                    <div>
                      <dt className="text-muted-foreground">Owner</dt>
                      <dd className="mt-0.5">{values.owner_name} ({values.owner_email})</dd>
                    </div>
                    <div>
                      <dt className="text-muted-foreground">Max Users</dt>
                      <dd className="mt-0.5">{values.max_users}</dd>
                    </div>
                    <div>
                      <dt className="text-muted-foreground">Max Storage</dt>
                      <dd className="mt-0.5">{values.max_storage_gb} GB</dd>
                    </div>
                    <div>
                      <dt className="text-muted-foreground">MFA Required</dt>
                      <dd className="mt-0.5">{values.mfa_required ? "Yes" : "No"}</dd>
                    </div>
                    <div>
                      <dt className="text-muted-foreground">Enabled Suites</dt>
                      <dd className="mt-0.5 capitalize">
                        {values.enabled_suites.length > 0
                          ? values.enabled_suites.join(", ")
                          : "None"}
                      </dd>
                    </div>
                  </div>
                </dl>
              </CardContent>
            </Card>
          )}

          {/* Navigation */}
          <div className="flex items-center justify-between mt-6">
            <Button
              type="button"
              variant="outline"
              onClick={handleBack}
              disabled={step === 1}
            >
              <ArrowLeft className="mr-2 h-4 w-4" />
              Back
            </Button>

            {step < 4 ? (
              <Button type="button" onClick={handleNext}>
                Next
                <ArrowRight className="ml-2 h-4 w-4" />
              </Button>
            ) : (
              <Button type="submit" disabled={provisionMutation.isPending}>
                {provisionMutation.isPending ? "Provisioning..." : "Provision Tenant"}
              </Button>
            )}
          </div>
        </form>
      </FormProvider>
    </div>
  );
}
