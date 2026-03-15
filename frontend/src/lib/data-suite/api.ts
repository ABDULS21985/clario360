import { apiDelete, apiGet, apiPatch, apiPost, apiPut, apiUpload } from '@/lib/api';
import type { FetchParams } from '@/types/table';
import type { PaginatedResponse } from '@/types/api';
import type {
  AggregateSourceStats,
  AnalyticsAuditLog,
  AnalyticsQuery,
  ConnectionTestResult,
  Contradiction,
  ContradictionScan,
  ContradictionStats,
  DarkDataAsset,
  DarkDataScan,
  DarkDataStatsSummary,
  DataEnvelope,
  DataModel,
  DataPaginatedEnvelope,
  DataSource,
  DataSuiteDashboard,
  DiscoveredSchema,
  ImpactAnalysis,
  JsonValue,
  LineageGraph,
  LineageStatsSummary,
  ModelLineage,
  ModelValidationResult,
  Pipeline,
  PipelineRun,
  PipelineRunLog,
  PipelineStats,
  QueryExplain,
  QueryResult,
  QualityDashboard,
  QualityResult,
  QualityRule,
  QualityScore,
  QualityTrendPoint,
  SavedQuery,
  SourceStats,
  SyncHistory,
  UploadedFile,
} from '@/lib/data-suite/types';

export const DATA_SUITE_ENDPOINTS = {
  dashboard: '/api/v1/data/dashboard',
  sources: '/api/v1/data/sources',
  sourceTestConfig: '/api/v1/data/sources/test-config',
  sourceStats: '/api/v1/data/sources/stats',
  sourceById: (id: string) => `/api/v1/data/sources/${id}`,
  sourceTest: (id: string) => `/api/v1/data/sources/${id}/test`,
  sourceDiscover: (id: string) => `/api/v1/data/sources/${id}/discover`,
  sourceSchema: (id: string) => `/api/v1/data/sources/${id}/schema`,
  sourceSync: (id: string) => `/api/v1/data/sources/${id}/sync`,
  sourceSyncHistory: (id: string) => `/api/v1/data/sources/${id}/sync-history`,
  sourceDetailsStats: (id: string) => `/api/v1/data/sources/${id}/stats`,
  models: '/api/v1/data/models',
  modelById: (id: string) => `/api/v1/data/models/${id}`,
  modelDerive: '/api/v1/data/models/derive',
  modelValidate: (id: string) => `/api/v1/data/models/${id}/validate`,
  modelVersions: (id: string) => `/api/v1/data/models/${id}/versions`,
  modelLineage: (id: string) => `/api/v1/data/models/${id}/lineage`,
  pipelines: '/api/v1/data/pipelines',
  pipelineStats: '/api/v1/data/pipelines/stats',
  pipelineActiveRuns: '/api/v1/data/pipelines/active',
  pipelineById: (id: string) => `/api/v1/data/pipelines/${id}`,
  pipelineRun: (id: string) => `/api/v1/data/pipelines/${id}/run`,
  pipelinePause: (id: string) => `/api/v1/data/pipelines/${id}/pause`,
  pipelineResume: (id: string) => `/api/v1/data/pipelines/${id}/resume`,
  pipelineRuns: (id: string) => `/api/v1/data/pipelines/${id}/runs`,
  pipelineRunById: (id: string, runId: string) => `/api/v1/data/pipelines/${id}/runs/${runId}`,
  pipelineRunLogs: (id: string, runId: string) => `/api/v1/data/pipelines/${id}/runs/${runId}/logs`,
  qualityDashboard: '/api/v1/data/quality/dashboard',
  qualityScore: '/api/v1/data/quality/score',
  qualityTrend: '/api/v1/data/quality/score/trend',
  qualityRules: '/api/v1/data/quality/rules',
  qualityRuleById: (id: string) => `/api/v1/data/quality/rules/${id}`,
  qualityRuleRun: (id: string) => `/api/v1/data/quality/rules/${id}/run`,
  qualityResults: '/api/v1/data/quality/results',
  qualityResultById: (id: string) => `/api/v1/data/quality/results/${id}`,
  contradictions: '/api/v1/data/contradictions',
  contradictionStats: '/api/v1/data/contradictions/stats',
  contradictionDashboard: '/api/v1/data/contradictions/dashboard',
  contradictionById: (id: string) => `/api/v1/data/contradictions/${id}`,
  contradictionStatus: (id: string) => `/api/v1/data/contradictions/${id}/status`,
  contradictionResolve: (id: string) => `/api/v1/data/contradictions/${id}/resolve`,
  contradictionScan: '/api/v1/data/contradictions/scan',
  contradictionScans: '/api/v1/data/contradictions/scans',
  contradictionScanById: (id: string) => `/api/v1/data/contradictions/scans/${id}`,
  lineageGraph: '/api/v1/data/lineage/graph',
  lineageEntityGraph: (type: string, id: string) => `/api/v1/data/lineage/graph/${type}/${id}`,
  lineageImpact: (type: string, id: string) => `/api/v1/data/lineage/impact/${type}/${id}`,
  lineageSearch: '/api/v1/data/lineage/search',
  lineageStats: '/api/v1/data/lineage/stats',
  darkData: '/api/v1/data/dark-data',
  darkDataStats: '/api/v1/data/dark-data/stats',
  darkDataDashboard: '/api/v1/data/dark-data/dashboard',
  darkDataById: (id: string) => `/api/v1/data/dark-data/${id}`,
  darkDataStatus: (id: string) => `/api/v1/data/dark-data/${id}/status`,
  darkDataGovern: (id: string) => `/api/v1/data/dark-data/${id}/govern`,
  darkDataScan: '/api/v1/data/dark-data/scan',
  darkDataScans: '/api/v1/data/dark-data/scans',
  darkDataScanById: (id: string) => `/api/v1/data/dark-data/scans/${id}`,
  analyticsQuery: '/api/v1/data/analytics/query',
  analyticsExplain: '/api/v1/data/analytics/explain',
  analyticsExplore: (modelId: string) => `/api/v1/data/analytics/explore/${modelId}`,
  analyticsSaved: '/api/v1/data/analytics/saved',
  analyticsSavedById: (id: string) => `/api/v1/data/analytics/saved/${id}`,
  analyticsSavedRun: (id: string) => `/api/v1/data/analytics/saved/${id}/run`,
  analyticsAudit: '/api/v1/data/analytics/audit',
  filesUpload: '/api/v1/files/upload',
} as const;

export function buildDataSuiteQueryParams(
  params: FetchParams,
  extra?: Record<string, JsonValue | undefined>,
): Record<string, string | number | boolean> {
  const query: Record<string, string | number | boolean> = {
    page: params.page,
    per_page: params.per_page,
  };

  if (params.sort) {
    query.sort = params.sort;
  }
  if (params.order) {
    query.order = params.order;
  }
  if (params.search) {
    query.search = params.search;
  }

  for (const [key, value] of Object.entries(params.filters ?? {})) {
    if (value === undefined || value === '') {
      continue;
    }
    query[key] = Array.isArray(value) ? value.join(',') : value;
  }

  for (const [key, value] of Object.entries(extra ?? {})) {
    if (value === undefined || value === null || value === '') {
      continue;
    }
    if (Array.isArray(value)) {
      query[key] = value.join(',');
      continue;
    }
    if (typeof value === 'string' || typeof value === 'number' || typeof value === 'boolean') {
      query[key] = value;
      continue;
    }
    query[key] = JSON.stringify(value);
  }

  return query;
}

export async function fetchDataSuite<T>(url: string, params?: Record<string, unknown>): Promise<T> {
  const envelope = await apiGet<DataEnvelope<T>>(url, params);
  return envelope.data;
}

export async function fetchDataSuitePaginated<T>(
  url: string,
  params: FetchParams,
  extra?: Record<string, JsonValue | undefined>,
): Promise<PaginatedResponse<T>> {
  const envelope = await apiGet<DataPaginatedEnvelope<T>>(url, buildDataSuiteQueryParams(params, extra));
  return {
    data: envelope.data,
    meta: envelope.meta,
  };
}

export const dataSuiteApi = {
  fetchDashboard: () => fetchDataSuite<DataSuiteDashboard>(DATA_SUITE_ENDPOINTS.dashboard),
  listSources: (params: FetchParams) => fetchDataSuitePaginated<DataSource>(DATA_SUITE_ENDPOINTS.sources, params),
  getSource: (id: string) => fetchDataSuite<DataSource>(DATA_SUITE_ENDPOINTS.sourceById(id)),
  createSource: (payload: unknown) => apiPost<DataEnvelope<DataSource>>(DATA_SUITE_ENDPOINTS.sources, payload).then((res) => res.data),
  updateSource: (id: string, payload: unknown) => apiPut<DataEnvelope<DataSource>>(DATA_SUITE_ENDPOINTS.sourceById(id), payload).then((res) => res.data),
  deleteSource: (id: string) => apiDelete<void>(DATA_SUITE_ENDPOINTS.sourceById(id)),
  testSourceConfig: (payload: unknown) => apiPost<DataEnvelope<ConnectionTestResult>>(DATA_SUITE_ENDPOINTS.sourceTestConfig, payload).then((res) => res.data),
  testSource: (id: string) => apiPost<DataEnvelope<ConnectionTestResult>>(DATA_SUITE_ENDPOINTS.sourceTest(id)).then((res) => res.data),
  discoverSource: (id: string) => apiPost<DataEnvelope<DiscoveredSchema>>(DATA_SUITE_ENDPOINTS.sourceDiscover(id), {}).then((res) => res.data),
  getSourceSchema: (id: string) => fetchDataSuite<DiscoveredSchema>(DATA_SUITE_ENDPOINTS.sourceSchema(id)),
  syncSource: (id: string, syncType: 'full' | 'incremental' | 'schema_only') =>
    apiPost<DataEnvelope<SyncHistory>>(DATA_SUITE_ENDPOINTS.sourceSync(id), { sync_type: syncType }).then((res) => res.data),
  listSourceSyncHistory: (id: string, limit = 50) =>
    fetchDataSuite<SyncHistory[]>(DATA_SUITE_ENDPOINTS.sourceSyncHistory(id), { limit }),
  getSourceStats: (id: string) => fetchDataSuite<SourceStats>(DATA_SUITE_ENDPOINTS.sourceDetailsStats(id)),
  getSourceAggregateStats: () => fetchDataSuite<AggregateSourceStats>(DATA_SUITE_ENDPOINTS.sourceStats),
  listModels: (params: FetchParams) => fetchDataSuitePaginated<DataModel>(DATA_SUITE_ENDPOINTS.models, params),
  getModel: (id: string) => fetchDataSuite<DataModel>(DATA_SUITE_ENDPOINTS.modelById(id)),
  deriveModel: (payload: unknown) => apiPost<DataEnvelope<DataModel>>(DATA_SUITE_ENDPOINTS.modelDerive, payload).then((res) => res.data),
  validateModel: (id: string) => apiPost<DataEnvelope<ModelValidationResult>>(DATA_SUITE_ENDPOINTS.modelValidate(id), {}).then((res) => res.data),
  getModelVersions: (id: string) => fetchDataSuite<DataModel[]>(DATA_SUITE_ENDPOINTS.modelVersions(id)),
  getModelLineage: (id: string) => fetchDataSuite<ModelLineage>(DATA_SUITE_ENDPOINTS.modelLineage(id)),
  listPipelines: (params: FetchParams) => fetchDataSuitePaginated<Pipeline>(DATA_SUITE_ENDPOINTS.pipelines, params),
  getPipeline: (id: string) => fetchDataSuite<Pipeline>(DATA_SUITE_ENDPOINTS.pipelineById(id)),
  createPipeline: (payload: unknown) => apiPost<DataEnvelope<Pipeline>>(DATA_SUITE_ENDPOINTS.pipelines, payload).then((res) => res.data),
  updatePipeline: (id: string, payload: unknown) => apiPut<DataEnvelope<Pipeline>>(DATA_SUITE_ENDPOINTS.pipelineById(id), payload).then((res) => res.data),
  deletePipeline: (id: string) => apiDelete<void>(DATA_SUITE_ENDPOINTS.pipelineById(id)),
  runPipeline: (id: string) => apiPost<DataEnvelope<PipelineRun>>(DATA_SUITE_ENDPOINTS.pipelineRun(id), {}).then((res) => res.data),
  pausePipeline: (id: string) => apiPost<DataEnvelope<{ id: string; status: string }>>(DATA_SUITE_ENDPOINTS.pipelinePause(id), {}).then((res) => res.data),
  resumePipeline: (id: string) => apiPost<DataEnvelope<{ id: string; status: string }>>(DATA_SUITE_ENDPOINTS.pipelineResume(id), {}).then((res) => res.data),
  listPipelineRuns: (id: string, params: FetchParams) => fetchDataSuitePaginated<PipelineRun>(DATA_SUITE_ENDPOINTS.pipelineRuns(id), params),
  getPipelineRun: (id: string, runId: string) => fetchDataSuite<PipelineRun>(DATA_SUITE_ENDPOINTS.pipelineRunById(id, runId)),
  getPipelineRunLogs: (id: string, runId: string) => fetchDataSuite<PipelineRunLog[]>(DATA_SUITE_ENDPOINTS.pipelineRunLogs(id, runId)),
  getPipelineStats: () => fetchDataSuite<PipelineStats>(DATA_SUITE_ENDPOINTS.pipelineStats),
  getActivePipelineRuns: () => fetchDataSuite<PipelineRun[]>(DATA_SUITE_ENDPOINTS.pipelineActiveRuns),
  getQualityDashboard: () => fetchDataSuite<QualityDashboard>(DATA_SUITE_ENDPOINTS.qualityDashboard),
  getQualityScore: () => fetchDataSuite<QualityScore>(DATA_SUITE_ENDPOINTS.qualityScore),
  getQualityTrend: (days = 30) => fetchDataSuite<QualityTrendPoint[]>(DATA_SUITE_ENDPOINTS.qualityTrend, { days }),
  listQualityRules: (params: FetchParams) => fetchDataSuitePaginated<QualityRule>(DATA_SUITE_ENDPOINTS.qualityRules, params),
  createQualityRule: (payload: unknown) => apiPost<DataEnvelope<QualityRule>>(DATA_SUITE_ENDPOINTS.qualityRules, payload).then((res) => res.data),
  updateQualityRule: (id: string, payload: unknown) => apiPut<DataEnvelope<QualityRule>>(DATA_SUITE_ENDPOINTS.qualityRuleById(id), payload).then((res) => res.data),
  deleteQualityRule: (id: string) => apiDelete<void>(DATA_SUITE_ENDPOINTS.qualityRuleById(id)),
  runQualityRule: (id: string) => apiPost<DataEnvelope<QualityResult>>(DATA_SUITE_ENDPOINTS.qualityRuleRun(id), {}).then((res) => res.data),
  listQualityResults: (params: FetchParams) => fetchDataSuitePaginated<QualityResult>(DATA_SUITE_ENDPOINTS.qualityResults, params),
  getQualityResult: (id: string) => fetchDataSuite<QualityResult>(DATA_SUITE_ENDPOINTS.qualityResultById(id)),
  listContradictions: (params: FetchParams) => fetchDataSuitePaginated<Contradiction>(DATA_SUITE_ENDPOINTS.contradictions, params),
  getContradiction: (id: string) => fetchDataSuite<Contradiction>(DATA_SUITE_ENDPOINTS.contradictionById(id)),
  updateContradictionStatus: (id: string, status: string) =>
    apiPut<DataEnvelope<{ id: string; status: string }>>(DATA_SUITE_ENDPOINTS.contradictionStatus(id), { status }).then((res) => res.data),
  resolveContradiction: (id: string, payload: unknown) =>
    apiPost<DataEnvelope<{ id: string; resolution_action: string }>>(DATA_SUITE_ENDPOINTS.contradictionResolve(id), payload).then((res) => res.data),
  scanContradictions: () => apiPost<DataEnvelope<ContradictionScan>>(DATA_SUITE_ENDPOINTS.contradictionScan, {}).then((res) => res.data),
  listContradictionScans: (params: FetchParams) =>
    fetchDataSuitePaginated<ContradictionScan>(DATA_SUITE_ENDPOINTS.contradictionScans, params),
  getContradictionScan: (id: string) => fetchDataSuite<ContradictionScan>(DATA_SUITE_ENDPOINTS.contradictionScanById(id)),
  getContradictionStats: () => fetchDataSuite<ContradictionStats>(DATA_SUITE_ENDPOINTS.contradictionStats),
  getLineageGraph: () => fetchDataSuite<LineageGraph>(DATA_SUITE_ENDPOINTS.lineageGraph),
  getEntityLineageGraph: (type: string, id: string) => fetchDataSuite<LineageGraph>(DATA_SUITE_ENDPOINTS.lineageEntityGraph(type, id)),
  getLineageImpact: (type: string, id: string) => fetchDataSuite<ImpactAnalysis>(DATA_SUITE_ENDPOINTS.lineageImpact(type, id)),
  getLineageStats: () => fetchDataSuite<LineageStatsSummary>(DATA_SUITE_ENDPOINTS.lineageStats),
  searchLineage: (query: string, type?: string, limit = 20) =>
    fetchDataSuite<LineageGraph>(DATA_SUITE_ENDPOINTS.lineageSearch, { query, type, limit }),
  listDarkDataAssets: (params: FetchParams) => fetchDataSuitePaginated<DarkDataAsset>(DATA_SUITE_ENDPOINTS.darkData, params),
  getDarkDataAsset: (id: string) => fetchDataSuite<DarkDataAsset>(DATA_SUITE_ENDPOINTS.darkDataById(id)),
  updateDarkDataStatus: (id: string, payload: unknown) => apiPut<DataEnvelope<DarkDataAsset>>(DATA_SUITE_ENDPOINTS.darkDataStatus(id), payload).then((res) => res.data),
  governDarkData: (id: string, payload: unknown) => apiPost<DataEnvelope<DataModel>>(DATA_SUITE_ENDPOINTS.darkDataGovern(id), payload).then((res) => res.data),
  scanDarkData: () => apiPost<DataEnvelope<DarkDataScan>>(DATA_SUITE_ENDPOINTS.darkDataScan, {}).then((res) => res.data),
  listDarkDataScans: (params: FetchParams) => fetchDataSuitePaginated<DarkDataScan>(DATA_SUITE_ENDPOINTS.darkDataScans, params),
  getDarkDataScan: (id: string) => fetchDataSuite<DarkDataScan>(DATA_SUITE_ENDPOINTS.darkDataScanById(id)),
  getDarkDataStats: () => fetchDataSuite<DarkDataStatsSummary>(DATA_SUITE_ENDPOINTS.darkDataStats),
  executeAnalyticsQuery: (payload: { model_id: string; query: AnalyticsQuery }) =>
    apiPost<DataEnvelope<QueryResult>>(DATA_SUITE_ENDPOINTS.analyticsQuery, payload).then((res) => res.data),
  explainAnalyticsQuery: (payload: { model_id: string; query: AnalyticsQuery }) =>
    apiPost<DataEnvelope<QueryExplain>>(DATA_SUITE_ENDPOINTS.analyticsExplain, payload).then((res) => res.data),
  exploreModel: (modelId: string, query: AnalyticsQuery) =>
    apiPost<DataEnvelope<QueryResult>>(DATA_SUITE_ENDPOINTS.analyticsExplore(modelId), { query }).then((res) => res.data),
  listSavedQueries: (params: FetchParams) => fetchDataSuitePaginated<SavedQuery>(DATA_SUITE_ENDPOINTS.analyticsSaved, params),
  createSavedQuery: (payload: unknown) => apiPost<DataEnvelope<SavedQuery>>(DATA_SUITE_ENDPOINTS.analyticsSaved, payload).then((res) => res.data),
  updateSavedQuery: (id: string, payload: unknown) => apiPut<DataEnvelope<SavedQuery>>(DATA_SUITE_ENDPOINTS.analyticsSavedById(id), payload).then((res) => res.data),
  deleteSavedQuery: (id: string) => apiDelete<void>(DATA_SUITE_ENDPOINTS.analyticsSavedById(id)),
  runSavedQuery: (id: string) => apiPost<DataEnvelope<QueryResult>>(DATA_SUITE_ENDPOINTS.analyticsSavedRun(id), {}).then((res) => res.data),
  listAnalyticsAudit: (params: FetchParams) => fetchDataSuitePaginated<AnalyticsAuditLog>(DATA_SUITE_ENDPOINTS.analyticsAudit, params),
  uploadDataFile: (file: File, onProgress?: (progress: number) => void) =>
    apiUpload<UploadedFile>(
      DATA_SUITE_ENDPOINTS.filesUpload,
      file,
      {
        suite: 'data',
        entity_type: 'source',
        lifecycle_policy: 'standard',
        encrypt: 'true',
      },
      onProgress,
    ),
};
