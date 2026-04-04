import type { PaginationMeta } from '@/types/api';

export type CTIPeriod = '24h' | '7d' | '30d' | '90d';
export type CTISeverityCode = 'critical' | 'high' | 'medium' | 'low' | 'informational';
export type CTIEventType =
  | 'indicator_sighting'
  | 'attack_attempt'
  | 'vulnerability_exploit'
  | 'malware_detection'
  | 'anomaly'
  | 'policy_violation';
export type CTICampaignStatus = 'active' | 'monitoring' | 'dormant' | 'resolved' | 'archived';
export type CTIThreatActorType = 'state_sponsored' | 'cybercriminal' | 'hacktivist' | 'insider' | 'unknown';
export type CTISophisticationLevel = 'advanced' | 'intermediate' | 'basic';
export type CTIActorMotivation = 'espionage' | 'financial_gain' | 'disruption' | 'ideological' | 'unknown';
export type CTIRiskLevel = 'critical' | 'high' | 'medium' | 'low';
export type CTITakedownStatus =
  | 'detected'
  | 'reported'
  | 'takedown_requested'
  | 'taken_down'
  | 'monitoring'
  | 'false_positive';
export type CTITrendDirection = 'increasing' | 'stable' | 'decreasing';

// Reference data

export interface CTISeverityLevel {
  id: string;
  tenant_id: string;
  code: CTISeverityCode;
  label: string;
  color_hex: string;
  sort_order: number;
  created_at: string;
  updated_at: string;
  created_by?: string | null;
  updated_by?: string | null;
}

export interface CTIThreatCategory {
  id: string;
  tenant_id: string;
  code: string;
  label: string;
  description: string | null;
  mitre_tactic_ids: string[];
  created_at: string;
  updated_at: string;
  created_by?: string | null;
  updated_by?: string | null;
}

export interface CTIGeographicRegion {
  id: string;
  tenant_id: string;
  code: string;
  label: string;
  parent_region_id: string | null;
  latitude: number | null;
  longitude: number | null;
  iso_country_code: string | null;
  created_at: string;
  updated_at: string;
  created_by?: string | null;
  updated_by?: string | null;
}

export interface CTIIndustrySector {
  id: string;
  tenant_id: string;
  code: string;
  label: string;
  description: string | null;
  naics_code: string | null;
  created_at: string;
  updated_at: string;
  created_by?: string | null;
  updated_by?: string | null;
}

export interface CTIDataSource {
  id: string;
  tenant_id: string;
  name: string;
  source_type: string;
  url?: string | null;
  api_endpoint?: string | null;
  api_key_vault_path?: string | null;
  reliability_score: number;
  is_active: boolean;
  last_polled_at: string | null;
  poll_interval_seconds?: number | null;
  created_at: string;
  updated_at: string;
  created_by?: string | null;
  updated_by?: string | null;
}

// Threat events

export interface CTIThreatEvent {
  id: string;
  tenant_id: string;
  event_type: CTIEventType;
  title: string;
  description: string | null;
  severity_id: string | null;
  severity_code: string;
  severity_label: string;
  category_id: string | null;
  category_code?: string | null;
  category_label: string | null;
  source_id: string | null;
  source_name: string | null;
  source_reference?: string | null;
  confidence_score: number;
  origin_latitude: number | null;
  origin_longitude: number | null;
  origin_country_code: string | null;
  origin_city: string | null;
  origin_region_id?: string | null;
  target_sector_id: string | null;
  sector_label: string | null;
  target_sector_label?: string | null;
  target_org_name: string | null;
  target_country_code: string | null;
  ioc_type: string | null;
  ioc_value: string | null;
  mitre_technique_ids: string[];
  raw_payload?: Record<string, unknown> | null;
  is_false_positive: boolean;
  resolved_at: string | null;
  resolved_by?: string | null;
  first_seen_at: string;
  last_seen_at: string;
  created_at: string;
  updated_at?: string;
  created_by?: string | null;
  updated_by?: string | null;
  tags: string[];
}

export type CTIThreatEventResponse = CTIThreatEvent;

export interface CreateThreatEventRequest {
  event_type: CTIEventType;
  title: string;
  description?: string;
  severity_code: CTISeverityCode;
  category_code?: string;
  source_name?: string;
  source_reference?: string;
  confidence_score: number;
  origin_country_code?: string;
  origin_city?: string;
  origin_latitude?: number;
  origin_longitude?: number;
  target_sector_code?: string;
  target_org_name?: string;
  target_country_code?: string;
  ioc_type?: string;
  ioc_value?: string;
  mitre_technique_ids?: string[];
  raw_payload?: Record<string, unknown>;
  tags?: string[];
  first_seen_at?: string;
}

export interface UpdateThreatEventRequest {
  title?: string;
  description?: string;
  severity_code?: CTISeverityCode;
  category_code?: string;
  confidence_score?: number;
  origin_country_code?: string;
  origin_city?: string;
  target_sector_code?: string;
  target_country_code?: string;
  ioc_type?: string;
  ioc_value?: string;
  mitre_technique_ids?: string[];
  tags?: string[];
}

export interface CTIThreatEventFilters {
  search?: string;
  severity?: string | string[];
  category?: string | string[];
  event_type?: string | string[];
  origin_country?: string | string[];
  target_country?: string | string[];
  target_sector?: string | string[];
  ioc_type?: string;
  ioc_value?: string;
  is_false_positive?: boolean;
  min_confidence?: number;
  max_confidence?: number;
  first_seen_from?: string;
  first_seen_to?: string;
  source_id?: string;
  date_from?: string;
  date_to?: string;
  sort?: string;
  order?: 'asc' | 'desc';
  sort_by?: string;
  sort_dir?: 'asc' | 'desc';
  page?: number;
  per_page?: number;
}

export interface CTIEventTimelineItem {
  id: string;
  label: string;
  timestamp: string;
  tone?: 'default' | 'warning' | 'success' | 'destructive';
  description?: string;
}

// Campaigns

export interface CTICampaign {
  id: string;
  tenant_id: string;
  campaign_code: string;
  name: string;
  description: string | null;
  status: CTICampaignStatus;
  severity_id?: string | null;
  severity_code: string;
  severity_label: string;
  primary_actor_id: string | null;
  actor_name: string | null;
  target_sectors: string[];
  target_regions: string[];
  target_description: string | null;
  mitre_technique_ids: string[];
  ttps_summary: string | null;
  ioc_count: number;
  event_count: number;
  first_seen_at: string;
  last_seen_at: string | null;
  resolved_at?: string | null;
  resolved_by?: string | null;
  external_references?: Record<string, unknown> | null;
  created_at: string;
  updated_at: string;
  created_by?: string | null;
  updated_by?: string | null;
}

export interface CTICampaignIOC {
  id: string;
  tenant_id: string;
  campaign_id: string;
  ioc_type: string;
  ioc_value: string;
  confidence_score: number;
  first_seen_at: string;
  last_seen_at: string;
  is_active: boolean;
  source_id?: string | null;
  source_name?: string | null;
  created_at?: string;
  updated_at?: string;
}

export interface CTICampaignDetail extends CTICampaign {
  recent_iocs?: CTICampaignIOC[];
}

export interface CTICampaignFilters {
  search?: string;
  status?: string | string[];
  severity?: string | string[];
  actor_id?: string;
  first_seen_from?: string;
  first_seen_to?: string;
  sort?: string;
  order?: 'asc' | 'desc';
  sort_by?: string;
  sort_dir?: 'asc' | 'desc';
  page?: number;
  per_page?: number;
}

export interface CreateCampaignRequest {
  campaign_code: string;
  name: string;
  description?: string;
  status: CTICampaignStatus;
  severity_code: CTISeverityCode;
  primary_actor_id?: string;
  target_sectors?: string[];
  target_regions?: string[];
  target_description?: string;
  mitre_technique_ids?: string[];
  ttps_summary?: string;
  first_seen_at: string;
}

export interface UpdateCampaignRequest {
  name?: string;
  description?: string;
  severity_code?: CTISeverityCode;
  primary_actor_id?: string;
  target_sectors?: string[];
  target_regions?: string[];
  target_description?: string;
  mitre_technique_ids?: string[];
  ttps_summary?: string;
}

export interface CreateCampaignIOCRequest {
  ioc_type: string;
  ioc_value: string;
  confidence_score: number;
  source_name?: string;
}

// Threat actors

export interface CTIThreatActor {
  id: string;
  tenant_id: string;
  name: string;
  aliases: string[];
  actor_type: CTIThreatActorType;
  origin_country_code: string | null;
  origin_region_id?: string | null;
  sophistication_level: CTISophisticationLevel;
  primary_motivation: CTIActorMotivation;
  description: string | null;
  first_observed_at: string | null;
  last_activity_at: string | null;
  mitre_group_id: string | null;
  external_references?: Record<string, unknown> | null;
  is_active: boolean;
  risk_score: number;
  created_at: string;
  updated_at?: string;
  created_by?: string | null;
  updated_by?: string | null;
}

export interface CTIThreatActorFilters {
  search?: string;
  actor_type?: string | string[];
  sophistication?: string | string[];
  is_active?: boolean;
  sort?: string;
  order?: 'asc' | 'desc';
  page?: number;
  per_page?: number;
}

export interface CreateThreatActorRequest {
  name: string;
  aliases?: string[];
  actor_type: CTIThreatActorType;
  origin_country_code?: string;
  sophistication_level: CTISophisticationLevel;
  primary_motivation: CTIActorMotivation;
  description?: string;
  mitre_group_id?: string;
  external_references?: Record<string, unknown>;
  risk_score: number;
}

export interface UpdateThreatActorRequest {
  name?: string;
  aliases?: string[];
  actor_type?: CTIThreatActorType;
  origin_country_code?: string;
  sophistication_level?: CTISophisticationLevel;
  primary_motivation?: CTIActorMotivation;
  description?: string;
  mitre_group_id?: string;
  risk_score?: number;
  is_active?: boolean;
}

// Brand abuse

export interface CreateMonitoredBrandRequest {
  brand_name: string;
  domain_pattern?: string;
  keywords?: string[];
}

export interface CTIMonitoredBrand {
  id: string;
  tenant_id: string;
  brand_name: string;
  domain_pattern: string | null;
  keywords: string[];
  is_active: boolean;
  created_at: string;
  updated_at: string;
  created_by?: string | null;
  updated_by?: string | null;
}

export interface UpdateMonitoredBrandRequest {
  brand_name?: string;
  domain_pattern?: string;
  keywords?: string[];
  is_active?: boolean;
}

export interface CTIBrandAbuseIncident {
  id: string;
  tenant_id: string;
  brand_id: string;
  brand_name: string;
  malicious_domain: string;
  abuse_type: string;
  risk_level: CTIRiskLevel;
  region_id?: string | null;
  region_label: string | null;
  detection_count: number;
  source_id?: string | null;
  whois_registrant?: string | null;
  whois_created_date?: string | null;
  ssl_issuer?: string | null;
  hosting_ip: string | null;
  hosting_asn?: string | null;
  screenshot_file_id?: string | null;
  takedown_status: CTITakedownStatus;
  takedown_requested_at?: string | null;
  taken_down_at?: string | null;
  first_detected_at: string;
  last_detected_at: string;
  created_at?: string;
  updated_at?: string;
}

export interface CTIBrandAbuseFilters {
  brand_id?: string;
  risk_level?: string | string[];
  abuse_type?: string | string[];
  takedown_status?: string | string[];
  sort?: string;
  order?: 'asc' | 'desc';
  sort_by?: string;
  sort_dir?: 'asc' | 'desc';
  page?: number;
  per_page?: number;
}

export interface CreateBrandAbuseIncidentRequest {
  brand_id: string;
  malicious_domain: string;
  abuse_type: string;
  risk_level: CTIRiskLevel;
  source_name?: string;
  whois_registrant?: string;
  whois_created_date?: string;
  ssl_issuer?: string;
  hosting_ip?: string;
  hosting_asn?: string;
}

export interface UpdateBrandAbuseIncidentRequest {
  brand_id?: string;
  malicious_domain?: string;
  abuse_type?: string;
  risk_level?: CTIRiskLevel;
  source_name?: string;
  whois_registrant?: string;
  whois_created_date?: string;
  ssl_issuer?: string;
  hosting_ip?: string;
  hosting_asn?: string;
  region_id?: string;
  detection_count?: number;
}

// Dashboard / aggregation

export interface CTIGeoThreatHotspot {
  id: string;
  tenant_id: string;
  country_code: string;
  city: string;
  latitude: number | null;
  longitude: number | null;
  region_id?: string | null;
  severity_critical_count: number;
  severity_high_count: number;
  severity_medium_count: number;
  severity_low_count: number;
  total_count: number;
  top_category_id?: string | null;
  top_threat_type: string | null;
  period_start: string;
  period_end: string;
  computed_at: string;
}

export interface CTIGlobalThreatMapResponse {
  hotspots: CTIGeoThreatHotspot[];
  total_events: number;
  period: string;
}

export interface CTISectorThreatSummary {
  id: string;
  tenant_id: string;
  sector_id: string;
  sector_code?: string | null;
  sector_label: string;
  severity_critical_count: number;
  severity_high_count: number;
  severity_medium_count: number;
  severity_low_count: number;
  total_count: number;
  period_start: string;
  period_end: string;
  computed_at?: string;
}

export interface CTISectorThreatResponse {
  sectors: CTISectorThreatSummary[];
  period: string;
}

export interface CTIExecutiveSnapshot {
  id?: string;
  tenant_id: string;
  total_events_24h: number;
  total_events_7d: number;
  total_events_30d: number;
  active_campaigns_count: number;
  critical_campaigns_count: number;
  total_iocs: number;
  brand_abuse_critical_count: number;
  brand_abuse_total_count: number;
  top_targeted_sector_id: string | null;
  top_targeted_sector_label?: string | null;
  top_threat_origin_country: string | null;
  mean_time_to_detect_hours: number | null;
  mean_time_to_respond_hours: number | null;
  risk_score_overall: number;
  trend_direction: CTITrendDirection;
  trend_percentage: number;
  computed_at: string;
}

export interface CTIExecutiveDashboardResponse {
  snapshot: CTIExecutiveSnapshot;
  top_campaigns: CTICampaign[];
  critical_brands: CTIBrandAbuseIncident[];
  top_sectors: CTISectorThreatSummary[];
  recent_events: CTIThreatEvent[];
}

// Pagination / websocket

export interface CTIPaginatedResponse<T> {
  data: T[];
  meta: PaginationMeta;
}

export interface CTIWebSocketMessage<T = unknown> {
  type: string;
  data: T;
  timestamp: string;
}

export const CTI_SEVERITY_COLORS: Record<string, string> = {
  critical: '#FF3B5C',
  high: '#FF8C42',
  medium: '#FFD93D',
  low: '#4ADE80',
  informational: '#94A3B8',
};

export const CTI_SEVERITY_ORDER = ['critical', 'high', 'medium', 'low', 'informational'] as const;

export const CTI_STATUS_COLORS: Record<string, string> = {
  active: '#FF3B5C',
  monitoring: '#FFD93D',
  dormant: '#6B7280',
  resolved: '#4ADE80',
  archived: '#475569',
  detected: '#F97316',
  reported: '#F59E0B',
  takedown_requested: '#0EA5E9',
  taken_down: '#22C55E',
  false_positive: '#64748B',
};

export const CTI_CAMPAIGN_STATUS_LABELS: Record<string, string> = {
  active: 'Active',
  monitoring: 'Monitoring',
  dormant: 'Dormant',
  resolved: 'Resolved',
  archived: 'Archived',
};

export const CTI_TAKEDOWN_STATUS_LABELS: Record<string, string> = {
  detected: 'Detected',
  reported: 'Reported',
  takedown_requested: 'Takedown Requested',
  taken_down: 'Taken Down',
  monitoring: 'Monitoring',
  false_positive: 'False Positive',
};

export const CTI_EVENT_TYPE_LABELS: Record<string, string> = {
  indicator_sighting: 'Indicator Sighting',
  attack_attempt: 'Attack Attempt',
  vulnerability_exploit: 'Vulnerability Exploit',
  malware_detection: 'Malware Detection',
  anomaly: 'Anomaly',
  policy_violation: 'Policy Violation',
};

export const CTI_ACTOR_TYPE_LABELS: Record<string, string> = {
  state_sponsored: 'State Sponsored',
  cybercriminal: 'Cybercriminal',
  hacktivist: 'Hacktivist',
  insider: 'Insider',
  unknown: 'Unknown',
};

export const CTI_SOPHISTICATION_LABELS: Record<string, string> = {
  advanced: 'Advanced',
  intermediate: 'Intermediate',
  basic: 'Basic',
};

export const CTI_MOTIVATION_LABELS: Record<string, string> = {
  espionage: 'Espionage',
  financial_gain: 'Financial Gain',
  disruption: 'Disruption',
  ideological: 'Ideological',
  unknown: 'Unknown',
};

export const CTI_RISK_LEVEL_LABELS: Record<string, string> = {
  critical: 'Critical',
  high: 'High',
  medium: 'Medium',
  low: 'Low',
};

export const CTI_PERIOD_OPTIONS: CTIPeriod[] = ['24h', '7d', '30d', '90d'];
