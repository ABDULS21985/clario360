import { apiDelete, apiGet, apiPatch, apiPost, apiPut } from './api';
import { API_ENDPOINTS } from './constants';
import type {
  CTIBrandAbuseFilters,
  CTIBrandAbuseIncident,
  CTICampaign,
  CTICampaignDetail,
  CTICampaignFilters,
  CTICampaignIOC,
  CTIExecutiveDashboardResponse,
  CTIGeographicRegion,
  CTIGlobalThreatMapResponse,
  CTIIndustrySector,
  CTIMonitoredBrand,
  CTIPaginatedResponse,
  CTISeverityLevel,
  CTISectorThreatResponse,
  CTIThreatActor,
  CTIThreatActorFilters,
  CTIThreatCategory,
  CTIThreatEvent,
  CTIThreatEventFilters,
  CTIThreatEventResponse,
  CTIDataSource,
  CreateBrandAbuseIncidentRequest,
  CreateMonitoredBrandRequest,
  CreateCampaignIOCRequest,
  CreateCampaignRequest,
  CreateThreatActorRequest,
  CreateThreatEventRequest,
  UpdateBrandAbuseIncidentRequest,
  UpdateCampaignRequest,
  UpdateMonitoredBrandRequest,
  UpdateThreatActorRequest,
  UpdateThreatEventRequest,
} from '@/types/cti';
import type { FetchParams } from '@/types/table';

function stripEmpty(params: Record<string, unknown>): Record<string, unknown> {
  return Object.fromEntries(
    Object.entries(params).filter(([, value]) => {
      if (value === undefined || value === null || value === '') {
        return false;
      }
      if (Array.isArray(value)) {
        return value.length > 0;
      }
      return true;
    }),
  );
}

function normalizeThreatEvent(event: CTIThreatEvent): CTIThreatEvent {
  return {
    ...event,
    category_label: event.category_label || null,
    category_code: event.category_code || null,
    source_name: event.source_name || null,
    sector_label: event.sector_label || null,
    target_sector_label: event.target_sector_label ?? event.sector_label ?? null,
    tags: event.tags ?? [],
  };
}

function normalizeCampaign(campaign: CTICampaignDetail | CTICampaign): CTICampaignDetail {
  return {
    ...campaign,
    actor_name: campaign.actor_name || null,
    target_sectors: campaign.target_sectors ?? [],
    target_regions: campaign.target_regions ?? [],
    recent_iocs: 'recent_iocs' in campaign && Array.isArray(campaign.recent_iocs) ? campaign.recent_iocs : undefined,
  };
}

function normalizeBrandAbuse(incident: CTIBrandAbuseIncident): CTIBrandAbuseIncident {
  return {
    ...incident,
    region_label: incident.region_label || null,
  };
}

function normalizeThreatActor(actor: CTIThreatActor): CTIThreatActor {
  return {
    ...actor,
    aliases: actor.aliases ?? [],
  };
}

function normalizeMonitoredBrand(brand: CTIMonitoredBrand): CTIMonitoredBrand {
  return {
    ...brand,
    domain_pattern: brand.domain_pattern ?? null,
    keywords: brand.keywords ?? [],
  };
}

function normalizeExecutiveResponse(data: CTIExecutiveDashboardResponse): CTIExecutiveDashboardResponse {
  return {
    snapshot: {
      ...data.snapshot,
      top_targeted_sector_label: data.snapshot.top_targeted_sector_label ?? null,
      mean_time_to_detect_hours: data.snapshot.mean_time_to_detect_hours ?? 0,
      mean_time_to_respond_hours: data.snapshot.mean_time_to_respond_hours ?? 0,
    },
    top_campaigns: data.top_campaigns.map(normalizeCampaign),
    critical_brands: data.critical_brands.map(normalizeBrandAbuse),
    top_sectors: data.top_sectors,
    recent_events: data.recent_events.map(normalizeThreatEvent),
  };
}

function buildThreatEventQuery(filters: CTIThreatEventFilters): Record<string, unknown> {
  return stripEmpty({
    page: filters.page,
    per_page: filters.per_page,
    sort: filters.sort ?? filters.sort_by,
    order: filters.order ?? filters.sort_dir,
    search: filters.search,
    severity: filters.severity,
    category: filters.category,
    event_type: filters.event_type,
    origin_country: filters.origin_country,
    target_country: filters.target_country,
    target_sector: filters.target_sector,
    ioc_type: filters.ioc_type,
    ioc_value: filters.ioc_value,
    is_false_positive:
      typeof filters.is_false_positive === 'boolean' ? String(filters.is_false_positive) : undefined,
    min_confidence: filters.min_confidence,
    max_confidence: filters.max_confidence,
    first_seen_from: filters.first_seen_from ?? filters.date_from,
    first_seen_to: filters.first_seen_to ?? filters.date_to,
    source_id: filters.source_id,
  });
}

function buildCampaignQuery(filters: CTICampaignFilters): Record<string, unknown> {
  return stripEmpty({
    page: filters.page,
    per_page: filters.per_page,
    sort: filters.sort ?? filters.sort_by,
    order: filters.order ?? filters.sort_dir,
    search: filters.search,
    status: filters.status,
    severity: filters.severity,
    actor_id: filters.actor_id,
    first_seen_from: filters.first_seen_from,
    first_seen_to: filters.first_seen_to,
  });
}

function buildThreatActorQuery(filters: CTIThreatActorFilters = {}): Record<string, unknown> {
  return stripEmpty({
    page: filters.page,
    per_page: filters.per_page,
    sort: filters.sort,
    order: filters.order,
    search: filters.search,
    actor_type: filters.actor_type,
    sophistication: filters.sophistication,
    is_active: typeof filters.is_active === 'boolean' ? String(filters.is_active) : undefined,
  });
}

function buildBrandAbuseQuery(filters: CTIBrandAbuseFilters): Record<string, unknown> {
  return stripEmpty({
    page: filters.page,
    per_page: filters.per_page,
    sort: filters.sort ?? filters.sort_by,
    order: filters.order ?? filters.sort_dir,
    brand_id: filters.brand_id,
    risk_level: filters.risk_level,
    abuse_type: filters.abuse_type,
    takedown_status: filters.takedown_status,
  });
}

export function flattenThreatEventFetchParams(params: FetchParams): CTIThreatEventFilters {
  const filters = params.filters ?? {};
  const firstSeen = typeof filters.first_seen === 'string' ? filters.first_seen.split(',') : [];
  const confidence = typeof filters.confidence === 'string' ? filters.confidence.split(',') : [];

  return {
    page: params.page,
    per_page: params.per_page,
    sort: params.sort,
    order: params.order,
    search: params.search,
    severity: filters.severity,
    category: filters.category,
    event_type: filters.event_type,
    origin_country: filters.origin_country,
    target_country: filters.target_country,
    target_sector: filters.target_sector,
    ioc_type: typeof filters.ioc_type === 'string' ? filters.ioc_type : undefined,
    ioc_value: typeof filters.ioc_value === 'string' ? filters.ioc_value : undefined,
    is_false_positive:
      typeof filters.is_false_positive === 'string'
        ? filters.is_false_positive === 'true'
        : undefined,
    min_confidence: confidence[0] ? Number(confidence[0]) : undefined,
    max_confidence: confidence[1] ? Number(confidence[1]) : undefined,
    first_seen_from: firstSeen[0] || undefined,
    first_seen_to: firstSeen[1] || undefined,
    source_id: typeof filters.source_id === 'string' ? filters.source_id : undefined,
  };
}

export function flattenCampaignFetchParams(params: FetchParams): CTICampaignFilters {
  const filters = params.filters ?? {};
  const firstSeen = typeof filters.first_seen === 'string' ? filters.first_seen.split(',') : [];

  return {
    page: params.page,
    per_page: params.per_page,
    sort: params.sort,
    order: params.order,
    search: params.search,
    status: filters.status,
    severity: filters.severity,
    actor_id: typeof filters.actor_id === 'string' ? filters.actor_id : undefined,
    first_seen_from: firstSeen[0] || undefined,
    first_seen_to: firstSeen[1] || undefined,
  };
}

export function flattenThreatActorFetchParams(params: FetchParams): CTIThreatActorFilters {
  const filters = params.filters ?? {};
  return {
    page: params.page,
    per_page: params.per_page,
    sort: params.sort,
    order: params.order,
    search: params.search,
    actor_type: filters.actor_type,
    sophistication: filters.sophistication,
    is_active: typeof filters.is_active === 'string' ? filters.is_active === 'true' : undefined,
  };
}

export function flattenBrandAbuseFetchParams(params: FetchParams): CTIBrandAbuseFilters {
  const filters = params.filters ?? {};
  return {
    page: params.page,
    per_page: params.per_page,
    sort: params.sort,
    order: params.order,
    brand_id: typeof filters.brand_id === 'string' ? filters.brand_id : undefined,
    risk_level: filters.risk_level,
    abuse_type: filters.abuse_type,
    takedown_status: filters.takedown_status,
  };
}

// Reference data

export async function fetchSeverityLevels(): Promise<CTISeverityLevel[]> {
  const response = await apiGet<{ data: CTISeverityLevel[] }>(API_ENDPOINTS.CTI_SEVERITY_LEVELS);
  return response.data;
}

export async function fetchCategories(): Promise<CTIThreatCategory[]> {
  const response = await apiGet<{ data: CTIThreatCategory[] }>(API_ENDPOINTS.CTI_CATEGORIES);
  return response.data;
}

export async function fetchRegions(parentId?: string): Promise<CTIGeographicRegion[]> {
  const response = await apiGet<{ data: CTIGeographicRegion[] }>(
    API_ENDPOINTS.CTI_REGIONS,
    stripEmpty({ parent_id: parentId }),
  );
  return response.data;
}

export async function fetchSectors(): Promise<CTIIndustrySector[]> {
  const response = await apiGet<{ data: CTIIndustrySector[] }>(API_ENDPOINTS.CTI_SECTORS);
  return response.data;
}

export async function fetchDataSources(): Promise<CTIDataSource[]> {
  const response = await apiGet<{ data: CTIDataSource[] }>(API_ENDPOINTS.CTI_DATA_SOURCES);
  return response.data;
}

// Threat events

export async function fetchThreatEvents(filters: CTIThreatEventFilters): Promise<CTIPaginatedResponse<CTIThreatEvent>> {
  const response = await apiGet<CTIPaginatedResponse<CTIThreatEvent>>(
    API_ENDPOINTS.CTI_EVENTS,
    buildThreatEventQuery(filters),
  );
  return {
    ...response,
    data: response.data.map(normalizeThreatEvent),
  };
}

export async function fetchThreatEvent(id: string): Promise<CTIThreatEventResponse> {
  const response = await apiGet<{ data: CTIThreatEventResponse }>(API_ENDPOINTS.CTI_EVENT_DETAIL(id));
  return normalizeThreatEvent(response.data);
}

export async function createThreatEvent(data: CreateThreatEventRequest): Promise<CTIThreatEventResponse> {
  const response = await apiPost<{ data: CTIThreatEventResponse }>(API_ENDPOINTS.CTI_EVENTS, data);
  return normalizeThreatEvent(response.data);
}

export async function updateThreatEvent(
  id: string,
  data: UpdateThreatEventRequest,
): Promise<CTIThreatEventResponse> {
  const response = await apiPut<{ data: CTIThreatEventResponse }>(API_ENDPOINTS.CTI_EVENT_DETAIL(id), data);
  return normalizeThreatEvent(response.data);
}

export async function deleteThreatEvent(id: string): Promise<void> {
  await apiDelete<void>(API_ENDPOINTS.CTI_EVENT_DETAIL(id));
}

export async function markEventFalsePositive(id: string): Promise<void> {
  await apiPost(API_ENDPOINTS.CTI_EVENT_FALSE_POSITIVE(id));
}

export async function resolveEvent(id: string): Promise<void> {
  await apiPost(API_ENDPOINTS.CTI_EVENT_RESOLVE(id));
}

export async function fetchEventTags(id: string): Promise<string[]> {
  const response = await apiGet<{ data: string[] }>(API_ENDPOINTS.CTI_EVENT_TAGS(id));
  return response.data;
}

export async function addEventTags(id: string, tags: string[]): Promise<void> {
  await apiPost(API_ENDPOINTS.CTI_EVENT_TAGS(id), { tags });
}

export async function removeEventTag(id: string, tag: string): Promise<void> {
  await apiDelete<void>(`${API_ENDPOINTS.CTI_EVENT_TAGS(id)}/${encodeURIComponent(tag)}`);
}

// Campaigns

export async function fetchCampaigns(filters: CTICampaignFilters): Promise<CTIPaginatedResponse<CTICampaign>> {
  const response = await apiGet<CTIPaginatedResponse<CTICampaign>>(API_ENDPOINTS.CTI_CAMPAIGNS, buildCampaignQuery(filters));
  return {
    ...response,
    data: response.data.map(normalizeCampaign),
  };
}

export async function fetchCampaign(id: string): Promise<CTICampaignDetail> {
  const response = await apiGet<{ data: CTICampaignDetail }>(API_ENDPOINTS.CTI_CAMPAIGN_DETAIL(id));
  return normalizeCampaign(response.data);
}

export async function createCampaign(data: CreateCampaignRequest): Promise<CTICampaignDetail> {
  const response = await apiPost<{ data: CTICampaignDetail }>(API_ENDPOINTS.CTI_CAMPAIGNS, data);
  return normalizeCampaign(response.data);
}

export async function updateCampaign(id: string, data: UpdateCampaignRequest): Promise<CTICampaignDetail> {
  const response = await apiPut<{ data: CTICampaignDetail }>(API_ENDPOINTS.CTI_CAMPAIGN_DETAIL(id), data);
  return normalizeCampaign(response.data);
}

export async function deleteCampaign(id: string): Promise<void> {
  await apiDelete<void>(API_ENDPOINTS.CTI_CAMPAIGN_DETAIL(id));
}

export async function updateCampaignStatus(id: string, status: string): Promise<void> {
  await apiPatch(API_ENDPOINTS.CTI_CAMPAIGN_STATUS(id), { status });
}

export async function fetchCampaignEvents(
  id: string,
  page = 1,
  perPage = 25,
): Promise<CTIPaginatedResponse<CTIThreatEvent>> {
  const response = await apiGet<CTIPaginatedResponse<CTIThreatEvent>>(API_ENDPOINTS.CTI_CAMPAIGN_EVENTS(id), {
    page,
    per_page: perPage,
  });
  return {
    ...response,
    data: response.data.map(normalizeThreatEvent),
  };
}

export async function fetchCampaignIOCs(
  id: string,
  page = 1,
  perPage = 25,
): Promise<CTIPaginatedResponse<CTICampaignIOC>> {
  return apiGet<CTIPaginatedResponse<CTICampaignIOC>>(API_ENDPOINTS.CTI_CAMPAIGN_IOCS(id), {
    page,
    per_page: perPage,
  });
}

export async function createCampaignIOC(id: string, data: CreateCampaignIOCRequest): Promise<CTICampaignIOC> {
  const response = await apiPost<{ data: CTICampaignIOC }>(API_ENDPOINTS.CTI_CAMPAIGN_IOCS(id), data);
  return response.data;
}

export async function deleteCampaignIOC(campaignId: string, iocId: string): Promise<void> {
  await apiDelete<void>(`${API_ENDPOINTS.CTI_CAMPAIGN_IOCS(campaignId)}/${iocId}`);
}

export async function linkEventToCampaign(campaignId: string, eventId: string): Promise<void> {
  await apiPost(`${API_ENDPOINTS.CTI_CAMPAIGN_EVENTS(campaignId)}/${eventId}`);
}

export async function unlinkEventFromCampaign(campaignId: string, eventId: string): Promise<void> {
  await apiDelete<void>(`${API_ENDPOINTS.CTI_CAMPAIGN_EVENTS(campaignId)}/${eventId}`);
}

export async function fetchRelatedCampaignsForEvent(eventId: string): Promise<CTICampaignDetail[]> {
  const campaigns = await fetchCampaigns({ page: 1, per_page: 100, sort: 'first_seen_at', order: 'desc' });
  const results = await Promise.all(
    campaigns.data.map(async (campaign) => {
      const events = await fetchCampaignEvents(campaign.id, 1, 200);
      return events.data.some((event) => event.id === eventId) ? campaign : null;
    }),
  );
  return results.filter((item): item is CTICampaignDetail => item !== null);
}

// Threat actors

export async function fetchThreatActors(
  filters: CTIThreatActorFilters = {},
): Promise<CTIPaginatedResponse<CTIThreatActor>> {
  const response = await apiGet<CTIPaginatedResponse<CTIThreatActor>>(
    API_ENDPOINTS.CTI_ACTORS,
    buildThreatActorQuery(filters),
  );
  return {
    ...response,
    data: response.data.map(normalizeThreatActor),
  };
}

export async function fetchThreatActor(id: string): Promise<CTIThreatActor> {
  const response = await apiGet<{ data: CTIThreatActor }>(API_ENDPOINTS.CTI_ACTOR_DETAIL(id));
  return normalizeThreatActor(response.data);
}

export async function createThreatActor(data: CreateThreatActorRequest): Promise<CTIThreatActor> {
  const response = await apiPost<{ data: CTIThreatActor }>(API_ENDPOINTS.CTI_ACTORS, data);
  return normalizeThreatActor(response.data);
}

export async function updateThreatActor(
  id: string,
  data: UpdateThreatActorRequest,
): Promise<CTIThreatActor> {
  const response = await apiPut<{ data: CTIThreatActor }>(API_ENDPOINTS.CTI_ACTOR_DETAIL(id), data);
  return normalizeThreatActor(response.data);
}

export async function deleteThreatActor(id: string): Promise<void> {
  await apiDelete<void>(API_ENDPOINTS.CTI_ACTOR_DETAIL(id));
}

// Brand abuse

export async function fetchMonitoredBrands(): Promise<CTIMonitoredBrand[]> {
  const response = await apiGet<{ data: CTIMonitoredBrand[] }>(API_ENDPOINTS.CTI_BRANDS);
  return response.data.map(normalizeMonitoredBrand);
}

export async function createMonitoredBrand(
  data: CreateMonitoredBrandRequest,
): Promise<CTIMonitoredBrand> {
  const response = await apiPost<{ data: CTIMonitoredBrand }>(API_ENDPOINTS.CTI_BRANDS, data);
  return normalizeMonitoredBrand(response.data);
}

export async function updateMonitoredBrand(
  id: string,
  data: UpdateMonitoredBrandRequest,
): Promise<void> {
  await apiPut<{ data: { updated: boolean } }>(`${API_ENDPOINTS.CTI_BRANDS}/${id}`, data);
}

export async function deleteMonitoredBrand(id: string): Promise<void> {
  await apiDelete<void>(`${API_ENDPOINTS.CTI_BRANDS}/${id}`);
}

export async function fetchBrandAbuseIncidents(
  filters: CTIBrandAbuseFilters,
): Promise<CTIPaginatedResponse<CTIBrandAbuseIncident>> {
  const response = await apiGet<CTIPaginatedResponse<CTIBrandAbuseIncident>>(
    API_ENDPOINTS.CTI_BRAND_ABUSE,
    buildBrandAbuseQuery(filters),
  );
  return {
    ...response,
    data: response.data.map(normalizeBrandAbuse),
  };
}

export async function fetchBrandAbuseIncident(id: string): Promise<CTIBrandAbuseIncident> {
  const response = await apiGet<{ data: CTIBrandAbuseIncident }>(API_ENDPOINTS.CTI_BRAND_ABUSE_DETAIL(id));
  return normalizeBrandAbuse(response.data);
}

export async function createBrandAbuseIncident(
  data: CreateBrandAbuseIncidentRequest,
): Promise<CTIBrandAbuseIncident> {
  const response = await apiPost<{ data: CTIBrandAbuseIncident }>(API_ENDPOINTS.CTI_BRAND_ABUSE, data);
  return normalizeBrandAbuse(response.data);
}

export async function updateBrandAbuseIncident(
  id: string,
  data: UpdateBrandAbuseIncidentRequest,
): Promise<void> {
  await apiPut<{ data: { updated: boolean } }>(API_ENDPOINTS.CTI_BRAND_ABUSE_DETAIL(id), data);
}

export async function updateTakedownStatus(id: string, status: string): Promise<void> {
  await apiPatch(API_ENDPOINTS.CTI_BRAND_ABUSE_TAKEDOWN(id), { status });
}

// Dashboard

export async function fetchGlobalThreatMap(period: string): Promise<CTIGlobalThreatMapResponse> {
  const response = await apiGet<{ data: CTIGlobalThreatMapResponse }>(API_ENDPOINTS.CTI_DASHBOARD_THREAT_MAP, { period });
  return response.data;
}

export async function fetchSectorThreatOverview(period: string): Promise<CTISectorThreatResponse> {
  const response = await apiGet<{ data: CTISectorThreatResponse }>(API_ENDPOINTS.CTI_DASHBOARD_SECTORS, { period });
  return response.data;
}

export async function fetchExecutiveDashboard(): Promise<CTIExecutiveDashboardResponse> {
  const response = await apiGet<{ data: CTIExecutiveDashboardResponse }>(API_ENDPOINTS.CTI_DASHBOARD_EXECUTIVE);
  return normalizeExecutiveResponse(response.data);
}

export async function refreshAggregations(scope: 'tenant' | 'all' = 'tenant'): Promise<void> {
  const url = scope === 'all'
    ? `${API_ENDPOINTS.CTI_ADMIN_REFRESH}?scope=all`
    : API_ENDPOINTS.CTI_ADMIN_REFRESH;
  await apiPost(url);
}
