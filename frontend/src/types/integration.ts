export type IntegrationType = 'slack' | 'teams' | 'jira' | 'servicenow' | 'webhook';
export type IntegrationStatus = 'active' | 'inactive' | 'error' | 'setup_pending';

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

