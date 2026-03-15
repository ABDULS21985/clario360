"use client";

import { useEffect, useMemo, useState } from "react";
import { AlertCircle, ExternalLink, KeyRound, Plus, Sparkles, Trash2 } from "lucide-react";
import { toast } from "sonner";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Separator } from "@/components/ui/separator";
import { Switch } from "@/components/ui/switch";
import { Textarea } from "@/components/ui/textarea";
import { Checkbox } from "@/components/ui/checkbox";
import { apiPost, apiPut } from "@/lib/api";
import type { ApiResponse } from "@/types/api";
import type { IntegrationProviderStatus, IntegrationRecord, IntegrationType } from "@/types/integration";
import {
  buildIntegrationPayload,
  emptyFilterState,
  EVENT_TYPE_OPTIONS,
  type EventFilterFormState,
  formStateFromIntegration,
  getDefaultFormState,
  prepareOAuthInstall,
  prettyType,
  SEVERITY_OPTIONS,
  SUITE_OPTIONS,
  type IntegrationFormState,
} from "./integration-utils";

interface IntegrationFormDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSaved: (integration: IntegrationRecord) => void;
  providers: IntegrationProviderStatus[];
  integration?: IntegrationRecord | null;
  initialType?: IntegrationType | null;
}

export function IntegrationFormDialog({
  open,
  onOpenChange,
  onSaved,
  providers,
  integration,
  initialType,
}: IntegrationFormDialogProps) {
  const [state, setState] = useState<IntegrationFormState>(getDefaultFormState(initialType ?? "webhook"));
  const [submitting, setSubmitting] = useState(false);
  const [oauthLoading, setOAuthLoading] = useState(false);
  const provider = useMemo(
    () => providers.find((item) => item.type === state.type) ?? null,
    [providers, state.type],
  );

  useEffect(() => {
    if (!open) {
      return;
    }
    if (integration) {
      setState(formStateFromIntegration(integration));
      return;
    }
    setState(getDefaultFormState(initialType ?? "webhook"));
  }, [open, integration, initialType]);

  const updateConfig = <K extends keyof IntegrationFormState["config"]>(
    key: K,
    value: IntegrationFormState["config"][K],
  ) => {
    setState((current) => ({
      ...current,
      config: {
        ...current.config,
        [key]: value,
      },
    }));
  };

  const handleSubmit = async () => {
    setSubmitting(true);
    try {
      const payload = buildIntegrationPayload(state, integration ?? undefined);
      if (integration) {
        const response = await apiPut<ApiResponse<IntegrationRecord>>(`/api/v1/integrations/${integration.id}`, payload);
        toast.success(`${prettyType(integration.type)} integration updated`);
        onSaved(response.data);
      } else {
        const response = await apiPost<ApiResponse<IntegrationRecord>>("/api/v1/integrations", payload);
        toast.success(`${prettyType(state.type)} integration created`);
        onSaved(response.data);
      }
      onOpenChange(false);
    } catch (error) {
      const message = error instanceof Error ? error.message : "Unable to save integration";
      toast.error(message);
    } finally {
      setSubmitting(false);
    }
  };

  const handleOAuthStart = async () => {
    if (!provider || (provider.type !== "slack" && provider.type !== "jira")) {
      toast.error("OAuth is unavailable for this provider");
      return;
    }
    setOAuthLoading(true);
    try {
      const url = await prepareOAuthInstall(provider.type, {
        name: state.name,
        project_key: provider.type === "jira" ? state.config.project_key : undefined,
      });
      window.location.assign(url);
    } catch (error) {
      const message = error instanceof Error ? error.message : "Unable to start OAuth setup";
      toast.error(message);
    } finally {
      setOAuthLoading(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[90vh] max-w-4xl overflow-y-auto">
        <DialogHeader>
          <DialogTitle>{integration ? `Edit ${integration.name}` : "Configure Integration"}</DialogTitle>
          <DialogDescription>
            {integration
              ? "Update connection settings, delivery filters, and synchronization behavior."
              : "Create a new outbound or bidirectional integration for your tenant."}
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-6">
          {!integration ? (
            <section className="space-y-3">
              <Label>Provider</Label>
              <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
                {providers.map((item) => {
                  const selected = item.type === state.type;
                  return (
                    <button
                      key={item.type}
                      type="button"
                      className={`rounded-lg border p-4 text-left transition ${
                        selected ? "border-primary bg-primary/5" : "border-border hover:border-primary/40"
                      }`}
                      onClick={() => setState(getDefaultFormState(item.type))}
                    >
                      <div className="flex items-start justify-between gap-3">
                        <div>
                          <div className="font-medium">{item.name}</div>
                          <div className="mt-1 text-sm text-muted-foreground">{item.description}</div>
                        </div>
                        <span className="rounded-full border px-2 py-0.5 text-[11px] uppercase tracking-wide text-muted-foreground">
                          {item.setup_mode}
                        </span>
                      </div>
                    </button>
                  );
                })}
              </div>
            </section>
          ) : null}

          {provider && (provider.type === "slack" || provider.type === "jira") ? (
            <Alert>
              <Sparkles className="h-4 w-4" />
              <AlertTitle>OAuth-assisted setup is available</AlertTitle>
              <AlertDescription className="space-y-3">
                <p>
                  {provider.configured
                    ? `Use ${provider.name} OAuth to prefill tenant credentials, then finish the remaining ${provider.type === "slack" ? "channel" : "project"} fields in Clario 360.`
                    : `${provider.name} OAuth is not configured in this local runtime. You can still use advanced/manual setup below.`}
                </p>
                <div className="flex flex-wrap gap-2">
                  <Button
                    type="button"
                    variant="outline"
                    disabled={!provider.configured || !provider.oauth_enabled || oauthLoading}
                    onClick={() => void handleOAuthStart()}
                  >
                    <ExternalLink className="mr-2 h-4 w-4" />
                    Connect via OAuth
                  </Button>
                  {!provider.configured && provider.missing_config?.length ? (
                    <span className="text-xs text-muted-foreground">
                      Missing runtime config: {provider.missing_config.join(", ")}
                    </span>
                  ) : null}
                </div>
              </AlertDescription>
            </Alert>
          ) : null}

          <section className="grid gap-4 md:grid-cols-2">
            <div className="space-y-2">
              <Label htmlFor="integration-name">Name</Label>
              <Input
                id="integration-name"
                value={state.name}
                onChange={(event) => setState((current) => ({ ...current, name: event.target.value }))}
                placeholder={`${prettyType(state.type)} production integration`}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="integration-type">Type</Label>
              <Select
                value={state.type}
                onValueChange={(value) => {
                  if (integration) {
                    return;
                  }
                  setState(getDefaultFormState(value as IntegrationType));
                }}
                disabled={Boolean(integration)}
              >
                <SelectTrigger id="integration-type">
                  <SelectValue placeholder="Select an integration type" />
                </SelectTrigger>
                <SelectContent>
                  {providers.map((item) => (
                    <SelectItem key={item.type} value={item.type}>
                      {item.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-2 md:col-span-2">
              <Label htmlFor="integration-description">Description</Label>
              <Textarea
                id="integration-description"
                value={state.description}
                onChange={(event) => setState((current) => ({ ...current, description: event.target.value }))}
                placeholder="Where this integration is used, what it notifies, and who owns it."
              />
            </div>
          </section>

          <Separator />
          <ConnectionFields
            type={state.type}
            state={state}
            updateConfig={updateConfig}
            editing={Boolean(integration)}
          />

          <Separator />
          <EventFilterEditor state={state} onChange={setState} />
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)} disabled={submitting}>
            Cancel
          </Button>
          <Button onClick={() => void handleSubmit()} disabled={submitting || !state.name.trim()}>
            {submitting ? "Saving..." : integration ? "Save Changes" : "Create Integration"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function ConnectionFields({
  type,
  state,
  updateConfig,
  editing,
}: {
  type: IntegrationType;
  state: IntegrationFormState;
  updateConfig: <K extends keyof IntegrationFormState["config"]>(
    key: K,
    value: IntegrationFormState["config"][K],
  ) => void;
  editing: boolean;
}) {
  const secretHint = editing ? "Leave blank to keep the stored secret." : "Required for a working integration.";

  switch (type) {
    case "slack":
      return (
        <section className="space-y-4">
          <SectionTitle title="Slack Connection" description="OAuth can populate the workspace credentials automatically. Manual fields remain available for advanced setup." />
          <div className="grid gap-4 md:grid-cols-2">
            <Field label="Bot token" hint={secretHint}>
              <Input value={state.config.bot_token} onChange={(event) => updateConfig("bot_token", event.target.value)} placeholder="xoxb-..." />
            </Field>
            <Field label="Signing secret" hint={secretHint}>
              <Input value={state.config.signing_secret} onChange={(event) => updateConfig("signing_secret", event.target.value)} type="password" placeholder="Set via OAuth or enter manually" />
            </Field>
            <Field label="Workspace ID">
              <Input value={state.config.team_id} onChange={(event) => updateConfig("team_id", event.target.value)} placeholder="T12345678" />
            </Field>
            <Field label="Workspace name">
              <Input value={state.config.team_name} onChange={(event) => updateConfig("team_name", event.target.value)} placeholder="Security Operations" />
            </Field>
            <Field label="Channel ID">
              <Input value={state.config.channel_id} onChange={(event) => updateConfig("channel_id", event.target.value)} placeholder="C12345678" />
            </Field>
            <Field label="Incoming webhook URL" className="md:col-span-2">
              <Input
                value={state.config.incoming_webhook_url}
                onChange={(event) => updateConfig("incoming_webhook_url", event.target.value)}
                placeholder="https://hooks.slack.com/services/..."
              />
            </Field>
          </div>
          <div className="grid gap-4 md:grid-cols-2">
            <ToggleField
              label="Thread updates per alert"
              checked={state.config.thread_per_alert}
              onCheckedChange={(checked) => updateConfig("thread_per_alert", checked)}
            />
            <ToggleField
              label="Include explanation content"
              checked={state.config.include_explanation}
              onCheckedChange={(checked) => updateConfig("include_explanation", checked)}
            />
          </div>
        </section>
      );
    case "teams":
      return (
        <section className="space-y-4">
          <SectionTitle title="Teams Connection" description="Configure the Bot Framework app credentials and the target conversation details." />
          <div className="grid gap-4 md:grid-cols-2">
            <Field label="Bot app ID">
              <Input value={state.config.bot_app_id} onChange={(event) => updateConfig("bot_app_id", event.target.value)} />
            </Field>
            <Field label="Bot password" hint={secretHint}>
              <Input value={state.config.bot_password} onChange={(event) => updateConfig("bot_password", event.target.value)} type="password" />
            </Field>
            <Field label="Service URL">
              <Input value={state.config.service_url} onChange={(event) => updateConfig("service_url", event.target.value)} placeholder="https://smba.trafficmanager.net/emea/" />
            </Field>
            <Field label="Conversation ID">
              <Input value={state.config.conversation_id} onChange={(event) => updateConfig("conversation_id", event.target.value)} />
            </Field>
            <Field label="Tenant ID" className="md:col-span-2">
              <Input value={state.config.tenant_id} onChange={(event) => updateConfig("tenant_id", event.target.value)} />
            </Field>
          </div>
        </section>
      );
    case "jira":
      return (
        <section className="space-y-4">
          <SectionTitle title="Jira Connection" description="Project and token fields can be completed after OAuth or entered manually for advanced setup." />
          <div className="grid gap-4 md:grid-cols-2">
            <Field label="Base URL">
              <Input value={state.config.base_url} onChange={(event) => updateConfig("base_url", event.target.value)} placeholder="https://company.atlassian.net" />
            </Field>
            <Field label="Cloud ID">
              <Input value={state.config.cloud_id} onChange={(event) => updateConfig("cloud_id", event.target.value)} />
            </Field>
            <Field label="Project key">
              <Input value={state.config.project_key} onChange={(event) => updateConfig("project_key", event.target.value)} placeholder="SEC" />
            </Field>
            <Field label="Issue type ID">
              <Input value={state.config.issue_type_id} onChange={(event) => updateConfig("issue_type_id", event.target.value)} />
            </Field>
            <Field label="Auth token" hint={secretHint}>
              <Input value={state.config.auth_token} onChange={(event) => updateConfig("auth_token", event.target.value)} type="password" />
            </Field>
            <Field label="Refresh token" hint={secretHint}>
              <Input value={state.config.refresh_token} onChange={(event) => updateConfig("refresh_token", event.target.value)} type="password" />
            </Field>
            <Field label="Webhook secret" hint="Used to verify inbound Jira webhooks." className="md:col-span-2">
              <Input value={state.config.webhook_secret} onChange={(event) => updateConfig("webhook_secret", event.target.value)} type="password" />
            </Field>
            <Field label="Priority mapping JSON">
              <Textarea value={state.config.priority_mapping} onChange={(event) => updateConfig("priority_mapping", event.target.value)} className="min-h-[140px] font-mono text-xs" />
            </Field>
            <Field label="Status mapping JSON">
              <Textarea value={state.config.status_mapping} onChange={(event) => updateConfig("status_mapping", event.target.value)} className="min-h-[140px] font-mono text-xs" />
            </Field>
            <Field label="Custom fields JSON" className="md:col-span-2">
              <Textarea value={state.config.custom_fields} onChange={(event) => updateConfig("custom_fields", event.target.value)} className="min-h-[160px] font-mono text-xs" />
            </Field>
          </div>
        </section>
      );
    case "servicenow":
      return (
        <section className="space-y-4">
          <SectionTitle title="ServiceNow Connection" description="Choose the auth mode and provide the corresponding credentials, assignment defaults, and inbound webhook secret." />
          <div className="grid gap-4 md:grid-cols-2">
            <Field label="Instance URL">
              <Input value={state.config.instance_url} onChange={(event) => updateConfig("instance_url", event.target.value)} placeholder="https://company.service-now.com" />
            </Field>
            <Field label="Auth type">
              <Select value={state.config.auth_type} onValueChange={(value) => updateConfig("auth_type", value)}>
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="basic">Basic</SelectItem>
                  <SelectItem value="oauth">OAuth</SelectItem>
                </SelectContent>
              </Select>
            </Field>
            {state.config.auth_type === "basic" ? (
              <>
                <Field label="Username">
                  <Input value={state.config.username} onChange={(event) => updateConfig("username", event.target.value)} />
                </Field>
                <Field label="Password" hint={secretHint}>
                  <Input value={state.config.password} onChange={(event) => updateConfig("password", event.target.value)} type="password" />
                </Field>
              </>
            ) : (
              <Field label="OAuth token" hint={secretHint} className="md:col-span-2">
                <Input value={state.config.oauth_token} onChange={(event) => updateConfig("oauth_token", event.target.value)} type="password" />
              </Field>
            )}
            <Field label="Assignment group">
              <Input value={state.config.assignment_group} onChange={(event) => updateConfig("assignment_group", event.target.value)} />
            </Field>
            <Field label="Caller ID">
              <Input value={state.config.caller_id} onChange={(event) => updateConfig("caller_id", event.target.value)} />
            </Field>
            <Field label="Category">
              <Input value={state.config.category} onChange={(event) => updateConfig("category", event.target.value)} placeholder="Security" />
            </Field>
            <Field label="Subcategory">
              <Input value={state.config.subcategory} onChange={(event) => updateConfig("subcategory", event.target.value)} placeholder="Threat Detection" />
            </Field>
            <Field label="Webhook secret" hint="Shared secret for outbound ServiceNow business-rule callbacks." className="md:col-span-2">
              <Input value={state.config.webhook_secret} onChange={(event) => updateConfig("webhook_secret", event.target.value)} type="password" />
            </Field>
            <Field label="Status mapping JSON">
              <Textarea value={state.config.status_mapping} onChange={(event) => updateConfig("status_mapping", event.target.value)} className="min-h-[140px] font-mono text-xs" />
            </Field>
            <Field label="Custom fields JSON">
              <Textarea value={state.config.custom_fields} onChange={(event) => updateConfig("custom_fields", event.target.value)} className="min-h-[140px] font-mono text-xs" />
            </Field>
          </div>
        </section>
      );
    case "webhook":
      return (
        <section className="space-y-4">
          <SectionTitle title="Webhook Connection" description="Configure the outbound receiver, request method, signed secret, and any static headers." />
          <div className="grid gap-4 md:grid-cols-2">
            <Field label="URL" className="md:col-span-2">
              <Input value={state.config.url} onChange={(event) => updateConfig("url", event.target.value)} placeholder="https://ops.example.com/hooks/clario" />
            </Field>
            <Field label="Method">
              <Select value={state.config.method} onValueChange={(value) => updateConfig("method", value)}>
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="POST">POST</SelectItem>
                  <SelectItem value="PUT">PUT</SelectItem>
                </SelectContent>
              </Select>
            </Field>
            <Field label="Content type">
              <Input value={state.config.content_type} onChange={(event) => updateConfig("content_type", event.target.value)} placeholder="application/json" />
            </Field>
            <Field label="Shared secret" hint={secretHint}>
              <Input value={state.config.secret} onChange={(event) => updateConfig("secret", event.target.value)} type="password" />
            </Field>
            <Field label="Headers JSON" className="md:col-span-2">
              <Textarea value={state.config.headers} onChange={(event) => updateConfig("headers", event.target.value)} className="min-h-[160px] font-mono text-xs" />
            </Field>
          </div>
        </section>
      );
  }
}

function EventFilterEditor({
  state,
  onChange,
}: {
  state: IntegrationFormState;
  onChange: React.Dispatch<React.SetStateAction<IntegrationFormState>>;
}) {
  const updateFilter = (index: number, patch: Partial<EventFilterFormState>) => {
    onChange((current) => ({
      ...current,
      filters: current.filters.map((f, i) => (i === index ? { ...f, ...patch } : f)),
    }));
  };

  const addFilter = () => {
    onChange((current) => ({ ...current, filters: [...current.filters, emptyFilterState()] }));
  };

  const removeFilter = (index: number) => {
    onChange((current) => ({
      ...current,
      filters: current.filters.length > 1 ? current.filters.filter((_, i) => i !== index) : [emptyFilterState()],
    }));
  };

  return (
    <section className="space-y-4">
      <SectionTitle title="Event Filters" description="Leave everything empty to deliver every event. Multiple filter rules are evaluated with OR logic — an event matching any rule is delivered." />

      {state.filters.map((filter, index) => (
        <div key={index} className="space-y-4 rounded-lg border p-4">
          <div className="flex items-center justify-between">
            <div className="text-sm font-semibold">
              {state.filters.length > 1 ? `Filter rule ${index + 1}` : "Filter rule"}
            </div>
            {state.filters.length > 1 ? (
              <Button type="button" variant="ghost" size="sm" onClick={() => removeFilter(index)}>
                <Trash2 className="mr-1 h-3.5 w-3.5" />
                Remove
              </Button>
            ) : null}
          </div>

          <FilterGroup
            title="Event types"
            options={EVENT_TYPE_OPTIONS}
            selected={filter.eventTypes}
            onToggle={(value) =>
              updateFilter(index, { eventTypes: toggleArrayValue(filter.eventTypes, value) })
            }
          />

          <div className="grid gap-4 md:grid-cols-2">
            <FilterGroup
              title="Severities"
              options={SEVERITY_OPTIONS}
              selected={filter.severities}
              onToggle={(value) =>
                updateFilter(index, { severities: toggleArrayValue(filter.severities, value) })
              }
            />
            <FilterGroup
              title="Suites"
              options={SUITE_OPTIONS}
              selected={filter.suites}
              onToggle={(value) =>
                updateFilter(index, { suites: toggleArrayValue(filter.suites, value) })
              }
            />
          </div>

          <Field label="Minimum confidence">
            <Input
              type="number"
              min={0}
              max={1}
              step="0.01"
              value={filter.minConfidence}
              onChange={(event) => updateFilter(index, { minConfidence: event.target.value })}
              placeholder="0.70"
            />
          </Field>
        </div>
      ))}

      <Button type="button" variant="outline" size="sm" onClick={addFilter}>
        <Plus className="mr-1 h-3.5 w-3.5" />
        Add filter rule
      </Button>
    </section>
  );
}

function FilterGroup({
  title,
  options,
  selected,
  onToggle,
}: {
  title: string;
  options: readonly string[];
  selected: string[];
  onToggle: (value: string) => void;
}) {
  return (
    <div className="space-y-3 rounded-lg border p-4">
      <div className="font-medium">{title}</div>
      <div className="grid gap-2 sm:grid-cols-2">
        {options.map((option) => (
          <label key={option} className="flex items-center gap-2 text-sm">
            <Checkbox checked={selected.includes(option)} onCheckedChange={() => onToggle(option)} />
            <span>{option}</span>
          </label>
        ))}
      </div>
    </div>
  );
}

function Field({
  label,
  hint,
  className,
  children,
}: {
  label: string;
  hint?: string;
  className?: string;
  children: React.ReactNode;
}) {
  return (
    <div className={`space-y-2 ${className ?? ""}`}>
      <Label>{label}</Label>
      {children}
      {hint ? <p className="text-xs text-muted-foreground">{hint}</p> : null}
    </div>
  );
}

function ToggleField({
  label,
  checked,
  onCheckedChange,
}: {
  label: string;
  checked: boolean;
  onCheckedChange: (checked: boolean) => void;
}) {
  return (
    <div className="flex items-center justify-between rounded-lg border p-4">
      <div className="space-y-1">
        <div className="font-medium">{label}</div>
        <div className="text-sm text-muted-foreground">Stored as part of the integration configuration.</div>
      </div>
      <Switch checked={checked} onCheckedChange={onCheckedChange} />
    </div>
  );
}

function SectionTitle({ title, description }: { title: string; description: string }) {
  return (
    <div className="space-y-1">
      <div className="flex items-center gap-2">
        <KeyRound className="h-4 w-4 text-muted-foreground" />
        <h3 className="font-semibold">{title}</h3>
      </div>
      <p className="text-sm text-muted-foreground">{description}</p>
    </div>
  );
}

function toggleArrayValue(values: string[], value: string) {
  return values.includes(value) ? values.filter((item) => item !== value) : [...values, value];
}
