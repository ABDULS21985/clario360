"use client";

import { apiGet, apiPost } from "@/lib/api";
import type { ApiResponse, PaginatedResponse } from "@/types/api";
import type {
  ExternalTicketLinkRecord,
  IntegrationDeliveryRecord,
  IntegrationEventFilter,
  IntegrationProviderStatus,
  IntegrationRecord,
  IntegrationStatus,
  IntegrationType,
} from "@/types/integration";

export const EVENT_TYPE_OPTIONS = [
  "alert.created",
  "alert.escalated",
  "alert.resolved",
  "remediation.started",
  "pipeline.failed",
  "quality.failed",
  "contradiction.detected",
  "meeting.scheduled",
  "action_item.overdue",
  "contract.expiring",
] as const;

export const SUITE_OPTIONS = ["cyber", "data", "acta", "lex", "visus", "platform"] as const;
export const SEVERITY_OPTIONS = ["critical", "high", "medium", "low", "info"] as const;

export interface DeliveryFilters {
  page: number;
  per_page: number;
  status?: string;
  event_type?: string;
}

export interface EventFilterFormState {
  eventTypes: string[];
  severities: string[];
  suites: string[];
  minConfidence: string;
}

export function emptyFilterState(): EventFilterFormState {
  return { eventTypes: [], severities: [], suites: [], minConfidence: "" };
}

export interface IntegrationFormState {
  type: IntegrationType;
  name: string;
  description: string;
  filters: EventFilterFormState[];
  config: {
    bot_token: string;
    signing_secret: string;
    channel_id: string;
    team_id: string;
    team_name: string;
    incoming_webhook_url: string;
    thread_per_alert: boolean;
    include_explanation: boolean;
    bot_app_id: string;
    bot_password: string;
    service_url: string;
    conversation_id: string;
    tenant_id: string;
    base_url: string;
    cloud_id: string;
    project_key: string;
    issue_type_id: string;
    auth_token: string;
    refresh_token: string;
    webhook_secret: string;
    priority_mapping: string;
    status_mapping: string;
    custom_fields: string;
    instance_url: string;
    auth_type: string;
    username: string;
    password: string;
    oauth_token: string;
    assignment_group: string;
    caller_id: string;
    category: string;
    subcategory: string;
    url: string;
    method: string;
    headers: string;
    secret: string;
    content_type: string;
  };
}

export async function fetchProviders(): Promise<IntegrationProviderStatus[]> {
  const response = await apiGet<ApiResponse<IntegrationProviderStatus[]>>("/api/v1/integrations/providers");
  return response.data ?? [];
}

export async function fetchIntegrations(): Promise<IntegrationRecord[]> {
  const response = await apiGet<PaginatedResponse<IntegrationRecord>>("/api/v1/integrations", {
    page: 1,
    per_page: 100,
    sort: "updated_at",
    order: "desc",
  });
  return response.data ?? [];
}

export async function fetchIntegration(id: string): Promise<IntegrationRecord> {
  const response = await apiGet<ApiResponse<IntegrationRecord>>(`/api/v1/integrations/${id}`);
  return response.data;
}

export async function fetchDeliveries(id: string, filters: DeliveryFilters): Promise<PaginatedResponse<IntegrationDeliveryRecord>> {
  return apiGet<PaginatedResponse<IntegrationDeliveryRecord>>(`/api/v1/integrations/${id}/deliveries`, filters);
}

export async function fetchTicketLinks(integrationID: string): Promise<ExternalTicketLinkRecord[]> {
  const response = await apiGet<ApiResponse<ExternalTicketLinkRecord[]>>("/api/v1/integrations/ticket-links", {
    integration_id: integrationID,
  });
  return response.data ?? [];
}

export async function fetchTicketLink(id: string): Promise<ExternalTicketLinkRecord> {
  const response = await apiGet<ApiResponse<ExternalTicketLinkRecord>>(`/api/v1/integrations/ticket-links/${id}`);
  return response.data;
}

export async function prepareOAuthInstall(
  providerType: "slack" | "jira",
  options?: { name?: string; project_key?: string },
): Promise<string> {
  const response = await apiPost<ApiResponse<{ url: string }>>(`/api/v1/integrations/${providerType}/oauth/session`, options ?? {});
  return response.data.url;
}

export function statusBadgeVariant(status: IntegrationStatus): "default" | "secondary" | "destructive" | "outline" {
  switch (status) {
    case "active":
      return "default";
    case "error":
      return "destructive";
    case "setup_pending":
      return "secondary";
    default:
      return "outline";
  }
}

export function prettyType(type: IntegrationType): string {
  switch (type) {
    case "servicenow":
      return "ServiceNow";
    case "jira":
      return "Jira";
    default:
      return type.charAt(0).toUpperCase() + type.slice(1);
  }
}

export function getDefaultFormState(type: IntegrationType = "webhook"): IntegrationFormState {
  return {
    type,
    name: "",
    description: "",
    filters: [emptyFilterState()],
    config: {
      bot_token: "",
      signing_secret: "",
      channel_id: "",
      team_id: "",
      team_name: "",
      incoming_webhook_url: "",
      thread_per_alert: true,
      include_explanation: true,
      bot_app_id: "",
      bot_password: "",
      service_url: "",
      conversation_id: "",
      tenant_id: "",
      base_url: "",
      cloud_id: "",
      project_key: "",
      issue_type_id: "",
      auth_token: "",
      refresh_token: "",
      webhook_secret: "",
      priority_mapping: "",
      status_mapping: "",
      custom_fields: "",
      instance_url: "",
      auth_type: "basic",
      username: "",
      password: "",
      oauth_token: "",
      assignment_group: "",
      caller_id: "",
      category: "",
      subcategory: "",
      url: "",
      method: "POST",
      headers: "{\n  \n}",
      secret: "",
      content_type: "application/json",
    },
  };
}

export function formStateFromIntegration(integration: IntegrationRecord): IntegrationFormState {
  const state = getDefaultFormState(integration.type);
  const rawFilters = integration.event_filters ?? [];
  const filters: EventFilterFormState[] =
    rawFilters.length > 0
      ? rawFilters.map((f) => ({
          eventTypes: f.event_types ?? [],
          severities: f.severities ?? [],
          suites: f.suites ?? [],
          minConfidence: f.min_confidence ? String(f.min_confidence) : "",
        }))
      : [emptyFilterState()];

  return {
    ...state,
    type: integration.type,
    name: integration.name,
    description: integration.description ?? "",
    filters,
    config: {
      ...state.config,
      bot_token: safeSecretPrefill(integration.config?.bot_token),
      signing_secret: safeSecretPrefill(integration.config?.signing_secret),
      channel_id: safeString(integration.config?.channel_id),
      team_id: safeString(integration.config?.team_id),
      team_name: safeString(integration.config?.team_name),
      incoming_webhook_url: safeString(integration.config?.incoming_webhook_url),
      thread_per_alert: safeBoolean(integration.config?.thread_per_alert, true),
      include_explanation: safeBoolean(integration.config?.include_explanation, true),
      bot_app_id: safeString(integration.config?.bot_app_id),
      bot_password: safeSecretPrefill(integration.config?.bot_password),
      service_url: safeString(integration.config?.service_url),
      conversation_id: safeString(integration.config?.conversation_id),
      tenant_id: safeString(integration.config?.tenant_id),
      base_url: safeString(integration.config?.base_url),
      cloud_id: safeString(integration.config?.cloud_id),
      project_key: safeString(integration.config?.project_key),
      issue_type_id: safeString(integration.config?.issue_type_id),
      auth_token: safeSecretPrefill(integration.config?.auth_token),
      refresh_token: safeSecretPrefill(integration.config?.refresh_token),
      webhook_secret: safeSecretPrefill(integration.config?.webhook_secret),
      priority_mapping: safeJSONString(integration.config?.priority_mapping),
      status_mapping: safeJSONString(integration.config?.status_mapping),
      custom_fields: safeJSONString(integration.config?.custom_fields),
      instance_url: safeString(integration.config?.instance_url),
      auth_type: safeString(integration.config?.auth_type) || "basic",
      username: safeString(integration.config?.username),
      password: safeSecretPrefill(integration.config?.password),
      oauth_token: safeSecretPrefill(integration.config?.oauth_token),
      assignment_group: safeString(integration.config?.assignment_group),
      caller_id: safeString(integration.config?.caller_id),
      category: safeString(integration.config?.category),
      subcategory: safeString(integration.config?.subcategory),
      url: safeString(integration.config?.url),
      method: safeString(integration.config?.method) || "POST",
      headers: safeJSONString(integration.config?.headers),
      secret: safeSecretPrefill(integration.config?.secret),
      content_type: safeString(integration.config?.content_type) || "application/json",
    },
  };
}

export function buildIntegrationPayload(state: IntegrationFormState, existing?: IntegrationRecord) {
  const config: Record<string, unknown> = {};
  const assign = (key: keyof IntegrationFormState["config"], value: string | boolean, secret = false) => {
    if (typeof value === "boolean") {
      config[key] = value;
      return;
    }
    const trimmed = value.trim();
    if (!trimmed) {
      return;
    }
    if (secret && isMaskedSecret(trimmed)) {
      return;
    }
    config[key] = trimmed;
  };

  switch (state.type) {
    case "slack":
      assign("bot_token", state.config.bot_token, true);
      assign("signing_secret", state.config.signing_secret, true);
      assign("channel_id", state.config.channel_id);
      assign("team_id", state.config.team_id);
      assign("team_name", state.config.team_name);
      assign("incoming_webhook_url", state.config.incoming_webhook_url);
      config.thread_per_alert = state.config.thread_per_alert;
      config.include_explanation = state.config.include_explanation;
      break;
    case "teams":
      assign("bot_app_id", state.config.bot_app_id);
      assign("bot_password", state.config.bot_password, true);
      assign("service_url", state.config.service_url);
      assign("conversation_id", state.config.conversation_id);
      assign("tenant_id", state.config.tenant_id);
      break;
    case "jira":
      assign("base_url", state.config.base_url);
      assign("cloud_id", state.config.cloud_id);
      assign("project_key", state.config.project_key);
      assign("issue_type_id", state.config.issue_type_id);
      assign("auth_token", state.config.auth_token, true);
      assign("refresh_token", state.config.refresh_token, true);
      assign("webhook_secret", state.config.webhook_secret, true);
      assignParsedJSON(config, "priority_mapping", state.config.priority_mapping);
      assignParsedJSON(config, "status_mapping", state.config.status_mapping);
      assignParsedJSON(config, "custom_fields", state.config.custom_fields);
      break;
    case "servicenow":
      assign("instance_url", state.config.instance_url);
      assign("auth_type", state.config.auth_type);
      assign("username", state.config.username);
      assign("password", state.config.password, true);
      assign("oauth_token", state.config.oauth_token, true);
      assign("assignment_group", state.config.assignment_group);
      assign("caller_id", state.config.caller_id);
      assign("category", state.config.category);
      assign("subcategory", state.config.subcategory);
      assign("webhook_secret", state.config.webhook_secret, true);
      assignParsedJSON(config, "status_mapping", state.config.status_mapping);
      assignParsedJSON(config, "custom_fields", state.config.custom_fields);
      break;
    case "webhook":
      assign("url", state.config.url);
      assign("method", state.config.method || "POST");
      assign("secret", state.config.secret, true);
      assign("content_type", state.config.content_type || "application/json");
      assignParsedJSON(config, "headers", state.config.headers);
      break;
  }

  const event_filters: IntegrationEventFilter[] = state.filters
    .map((f) => {
      const filter: IntegrationEventFilter = {};
      if (f.eventTypes.length > 0) filter.event_types = f.eventTypes;
      if (f.severities.length > 0) filter.severities = f.severities;
      if (f.suites.length > 0) filter.suites = f.suites;
      if (f.minConfidence.trim()) {
        const parsed = Number(f.minConfidence);
        if (!Number.isNaN(parsed) && parsed > 0) filter.min_confidence = parsed;
      }
      return filter;
    })
    .filter((f) => Object.keys(f).length > 0);

  return {
    type: existing?.type ?? state.type,
    name: state.name.trim(),
    description: state.description.trim(),
    config,
    event_filters,
  };
}

export function getSetupPendingFields(integration: IntegrationRecord): string[] {
  const missing: string[] = [];
  switch (integration.type) {
    case "slack":
      if (!safeString(integration.config?.channel_id) && !safeString(integration.config?.incoming_webhook_url)) {
        missing.push("channel_id or incoming_webhook_url");
      }
      break;
    case "jira":
      if (!safeString(integration.config?.project_key)) {
        missing.push("project_key");
      }
      break;
    case "teams":
      for (const key of ["bot_app_id", "bot_password", "service_url", "conversation_id"] as const) {
        if (!safeString(integration.config?.[key])) {
          missing.push(key);
        }
      }
      break;
    case "servicenow":
      if (!safeString(integration.config?.instance_url)) {
        missing.push("instance_url");
      }
      break;
    case "webhook":
      if (!safeString(integration.config?.url)) {
        missing.push("url");
      }
      break;
  }
  return missing;
}

export function summarizeIntegrationConfig(integration: IntegrationRecord): Array<{ label: string; value: string }> {
  const config = integration.config ?? {};
  switch (integration.type) {
    case "slack":
      return compactSummary([
        ["Workspace", safeString(config.team_name) || safeString(config.team_id)],
        ["Channel", safeString(config.channel_id)],
        ["Incoming webhook", safeString(config.incoming_webhook_url)],
      ]);
    case "teams":
      return compactSummary([
        ["Bot app", safeString(config.bot_app_id)],
        ["Conversation", safeString(config.conversation_id)],
        ["Service URL", safeString(config.service_url)],
      ]);
    case "jira":
      return compactSummary([
        ["Base URL", safeString(config.base_url)],
        ["Project", safeString(config.project_key)],
        ["Issue type", safeString(config.issue_type_id)],
      ]);
    case "servicenow":
      return compactSummary([
        ["Instance", safeString(config.instance_url)],
        ["Auth", safeString(config.auth_type)],
        ["Assignment group", safeString(config.assignment_group)],
      ]);
    case "webhook":
      return compactSummary([
        ["URL", safeString(config.url)],
        ["Method", safeString(config.method)],
        ["Content type", safeString(config.content_type)],
      ]);
  }
}

function compactSummary(entries: Array<[string, string]>): Array<{ label: string; value: string }> {
  return entries
    .filter(([, value]) => value.trim() !== "")
    .map(([label, value]) => ({ label, value }));
}

function safeString(value: unknown): string {
  return typeof value === "string" ? value : "";
}

function safeSecretPrefill(value: unknown): string {
  const stringValue = safeString(value);
  return isMaskedSecret(stringValue) ? "" : stringValue;
}

function safeBoolean(value: unknown, fallback = false): boolean {
  return typeof value === "boolean" ? value : fallback;
}

function safeJSONString(value: unknown): string {
  if (!value) {
    return "{\n  \n}";
  }
  if (typeof value === "string") {
    return value;
  }
  try {
    return JSON.stringify(value, null, 2);
  } catch {
    return "{\n  \n}";
  }
}

function assignParsedJSON(target: Record<string, unknown>, key: string, raw: string) {
  const trimmed = raw.trim();
  if (!trimmed) {
    return;
  }
  try {
    const parsed = JSON.parse(trimmed) as unknown;
    target[key] = parsed;
  } catch {
    const label = key.replace(/_/g, " ");
    throw new Error(`Invalid JSON in "${label}". Please check the syntax and try again.`);
  }
}

function isMaskedSecret(value: string): boolean {
  return value.includes("*");
}
