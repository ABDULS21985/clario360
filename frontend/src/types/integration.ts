export type IntegrationType = 'slack' | 'teams' | 'jira' | 'servicenow' | 'webhook';
export type IntegrationStatus = 'active' | 'inactive' | 'error' | 'setup_pending';
export type IntegrationSetupMode = 'oauth' | 'manual';
export type DeliveryStatus = 'pending' | 'delivered' | 'failed' | 'retrying';
export type SyncDirection = 'outbound' | 'inbound' | 'bidirectional';

export interface IntegrationEventFilter {
  event_types?: string[];
  severities?: string[];
  suites?: string[];
  min_confidence?: number;
}

export interface IntegrationRecord {
  id: string;
  tenant_id: string;
  type: IntegrationType;
  name: string;
  description: string;
  status: IntegrationStatus;
  error_message?: string | null;
  error_count: number;
  last_error_at?: string | null;
  event_filters: IntegrationEventFilter[];
  last_used_at?: string | null;
  delivery_count: number;
  created_by: string;
  created_at: string;
  updated_at: string;
  config?: Record<string, unknown>;
}

export interface IntegrationProviderStatus {
  type: IntegrationType;
  name: string;
  description: string;
  setup_mode: IntegrationSetupMode;
  configured: boolean;
  oauth_enabled: boolean;
  oauth_start_url?: string;
  missing_config?: string[];
  supports_inbound: boolean;
  supports_outbound: boolean;
}

export interface IntegrationDeliveryRecord {
  id: string;
  tenant_id: string;
  integration_id: string;
  event_type: string;
  event_id: string;
  event_data?: unknown;
  status: DeliveryStatus;
  attempts: number;
  max_attempts: number;
  response_code?: number | null;
  response_body?: string | null;
  last_error?: string | null;
  error_category?: string | null;
  next_retry_at?: string | null;
  latency_ms?: number | null;
  delivered_at?: string | null;
  created_at: string;
}

export interface ExternalTicketLinkRecord {
  id: string;
  tenant_id: string;
  integration_id: string;
  entity_type: string;
  entity_id: string;
  external_system: 'jira' | 'servicenow';
  external_id: string;
  external_key: string;
  external_url: string;
  external_status?: string | null;
  external_priority?: string | null;
  sync_direction: SyncDirection;
  last_synced_at?: string | null;
  last_sync_direction?: string | null;
  sync_error?: string | null;
  created_at: string;
  updated_at: string;
}
