import type { PaginationMeta } from '@/types/api';
import type {
  CTIBrandAbuseIncident,
  CTICampaign,
  CTIExecutiveSnapshot,
  CTIGeoThreatHotspot,
  CTIGlobalThreatMapResponse,
  CTISectorThreatResponse,
  CTISectorThreatSummary,
  CTIThreatEvent,
} from '@/types/cti';

export type CTICampaignSummary = CTICampaign;
export type CTIBrandAbuseSummary = CTIBrandAbuseIncident;
export type CTISectorSummary = CTISectorThreatSummary;
export type CTIThreatEventSummary = CTIThreatEvent;

export interface VisusCTIOverview {
  snapshot: CTIExecutiveSnapshot;
  top_campaigns: CTICampaignSummary[];
  critical_brands: CTIBrandAbuseSummary[];
  top_sectors: CTISectorSummary[];
  recent_events: CTIThreatEventSummary[];
}

export interface VisusCTIRiskScoreResponse {
  risk_score: number;
  trend_direction: string;
  trend_percentage: number;
  total_events_24h: number;
  mttd_hours: number;
  mttr_hours: number;
  computed_at: string;
}

export interface VisusCTICampaignListResponse {
  data: CTICampaignSummary[];
  meta: PaginationMeta;
}

export interface VisusCTIBrandAbuseListResponse {
  data: CTIBrandAbuseSummary[];
  meta: PaginationMeta;
}

export type VisusCTIThreatMapResponse = CTIGlobalThreatMapResponse;
export type VisusCTIGeoThreatHotspot = CTIGeoThreatHotspot;
export type VisusCTISectorResponse = CTISectorThreatResponse;
