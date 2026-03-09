import api, { apiDelete, apiGet, apiPost, apiPut, apiUpload } from '@/lib/api';
import { fetchSuiteData, fetchSuitePaginated } from '@/lib/suite-api';
import type { PaginatedResponse } from '@/types/api';
import type {
  FileAccessLogEntry,
  FilePresignedDownload,
  FileQuarantineEntry,
  FileRecord,
  FileStorageStat,
} from '@/types/models';
import type { FetchParams } from '@/types/table';
import type {
  AICreateVersionPayload,
  AIDashboardData,
  AIDriftReport,
  AIExplanation,
  AILifecycleHistoryEntry,
  AIModelVersion,
  AIRegisterModelPayload,
  AIRegisteredModel,
  AIModelWithVersions,
  AIPerformancePoint,
  AIPredictionLog,
  AIPredictionStats,
  AIShadowComparison,
  AIShadowDivergence,
  AIUpdateModelPayload,
  AIValidationPreview,
  AIValidationResult,
} from '@/types/ai-governance';
import type {
  ActaActionItem,
  ActaActionItemStats,
  ActaCalendarDay,
  ActaCommittee,
  ActaComplianceCheck,
  ActaComplianceReport,
  ActaDashboard,
  ActaMeeting,
  ActaMeetingAttachment,
  ActaMeetingMinutes,
  ActaMeetingSummary,
  ActaAgendaItem,
  ActaAttendee,
  FileUploadRecord,
  JsonObject,
  LexClause,
  LexComplianceAlert,
  LexComplianceDashboard,
  LexComplianceRule,
  LexComplianceRunResult,
  LexComplianceScore,
  LexContractDetail,
  LexContractRecord,
  LexContractRiskAnalysis,
  LexContractSummary,
  LexContractVersion,
  LexDashboard,
  LexDocument,
  LexDocumentVersion,
  LexExpiringContractSummary,
  LexWorkflowSummary,
  UserDirectoryEntry,
  VisusDashboard,
  VisusExecutiveAlert,
  VisusExecutiveSummary,
  VisusKPIDefinition,
  VisusKPIGetResponse,
  VisusKPISnapshot,
  VisusReportDefinition,
  VisusReportSnapshot,
  VisusWidget,
  VisusWidgetData,
  VisusWidgetTypeDefinition,
} from '@/types/suites';

export const enterpriseApi = {
  users: {
    list: async (params: FetchParams): Promise<PaginatedResponse<UserDirectoryEntry>> => {
      const response = await apiGet<PaginatedResponse<UserDirectoryEntry>>('/api/v1/users', {
        page: params.page,
        per_page: params.per_page,
        sort: params.sort,
        order: params.order,
        search: params.search,
        ...params.filters,
      });
      return response;
    },
  },
  files: {
    list: (params?: {
      page?: number;
      per_page?: number;
      suite?: string;
      entity_type?: string;
      entity_id?: string;
      uploaded_by?: string;
      tag?: string;
    }): Promise<PaginatedResponse<FileRecord>> =>
      apiGet<PaginatedResponse<FileRecord>>('/api/v1/files', params),
    get: (id: string): Promise<FileRecord> => apiGet<FileRecord>(`/api/v1/files/${id}`),
    upload: (
      file: File,
      fields: Record<string, string>,
      onProgress?: (progress: number) => void,
    ): Promise<FileUploadRecord> => apiUpload<FileUploadRecord>('/api/v1/files/upload', file, fields, onProgress),
    delete: (id: string): Promise<{ status: string }> => apiDelete<{ status: string }>(`/api/v1/files/${id}`),
    versions: (id: string): Promise<FileRecord[]> =>
      apiGet<{ versions: FileRecord[] }>(`/api/v1/files/${id}/versions`).then((res) => res.versions),
    accessLog: (
      id: string,
      params?: { page?: number; per_page?: number },
    ): Promise<PaginatedResponse<FileAccessLogEntry>> =>
      apiGet<PaginatedResponse<FileAccessLogEntry>>(`/api/v1/files/${id}/access-log`, params),
    stats: (): Promise<FileStorageStat[]> =>
      apiGet<{ storage_stats: FileStorageStat[] }>('/api/v1/files/stats').then((res) => res.storage_stats),
    quarantine: (
      params?: { page?: number; per_page?: number },
    ): Promise<PaginatedResponse<FileQuarantineEntry>> =>
      apiGet<PaginatedResponse<FileQuarantineEntry>>('/api/v1/files/quarantine', params),
    resolveQuarantine: (
      id: string,
      action: 'deleted' | 'restored' | 'false_positive',
    ): Promise<{ quarantine_id: string; action: string; resolved_by: string; status: string }> =>
      apiPost<{ quarantine_id: string; action: string; resolved_by: string; status: string }>(
        `/api/v1/files/quarantine/${id}/resolve`,
        { action },
      ),
    rescan: (id: string): Promise<{ file_id: string; status: string }> =>
      apiPost<{ file_id: string; status: string }>(`/api/v1/files/${id}/rescan`),
    getPresignedDownload: (id: string): Promise<FilePresignedDownload> =>
      apiGet<FilePresignedDownload>(`/api/v1/files/${id}/presigned`),
    download: async (id: string): Promise<Blob> => {
      const response = await api.get<Blob>(`/api/v1/files/${id}/download`, { responseType: 'blob' });
      return response.data;
    },
  },
  acta: {
    getDashboard: (): Promise<ActaDashboard> => fetchSuiteData('/api/v1/acta/dashboard'),
    listCommittees: (params: FetchParams) => fetchSuitePaginated<ActaCommittee>('/api/v1/acta/committees', params),
    getCommittee: (id: string) => fetchSuiteData<ActaCommittee>(`/api/v1/acta/committees/${id}`),
    createCommittee: (payload: unknown) => apiPost<{ data: ActaCommittee }>('/api/v1/acta/committees', payload).then((res) => res.data),
    updateCommittee: (id: string, payload: unknown) => apiPut<{ data: ActaCommittee }>(`/api/v1/acta/committees/${id}`, payload).then((res) => res.data),
    deleteCommittee: (id: string) => apiDelete<void>(`/api/v1/acta/committees/${id}`),
    addCommitteeMember: (id: string, payload: unknown) => apiPost<{ data: ActaCommittee }>(`/api/v1/acta/committees/${id}/members`, payload).then((res) => res.data),
    updateCommitteeMember: (id: string, userId: string, payload: unknown) =>
      apiPut<{ data: ActaCommittee }>(`/api/v1/acta/committees/${id}/members/${userId}`, payload).then((res) => res.data),
    removeCommitteeMember: (id: string, userId: string) => apiDelete<void>(`/api/v1/acta/committees/${id}/members/${userId}`),
    listMeetings: (params: FetchParams) => fetchSuitePaginated<ActaMeeting>('/api/v1/acta/meetings', params),
    getMeeting: (id: string) => fetchSuiteData<ActaMeeting>(`/api/v1/acta/meetings/${id}`),
    createMeeting: (payload: unknown) => apiPost<{ data: ActaMeeting }>('/api/v1/acta/meetings', payload).then((res) => res.data),
    updateMeeting: (id: string, payload: unknown) => apiPut<{ data: ActaMeeting }>(`/api/v1/acta/meetings/${id}`, payload).then((res) => res.data),
    cancelMeeting: (id: string, payload: unknown) =>
      api.delete<{ data: ActaMeeting }>(`/api/v1/acta/meetings/${id}`, { data: payload }).then((res) => res.data.data),
    startMeeting: (id: string) => apiPost<{ data: ActaMeeting }>(`/api/v1/acta/meetings/${id}/start`).then((res) => res.data),
    endMeeting: (id: string) => apiPost<{ data: ActaMeeting }>(`/api/v1/acta/meetings/${id}/end`).then((res) => res.data),
    postponeMeeting: (id: string, payload: unknown) => apiPost<{ data: ActaMeeting }>(`/api/v1/acta/meetings/${id}/postpone`, payload).then((res) => res.data),
    getUpcomingMeetings: (): Promise<ActaMeetingSummary[]> => fetchSuiteData('/api/v1/acta/meetings/upcoming'),
    getCalendar: (month: string): Promise<ActaCalendarDay[]> => fetchSuiteData('/api/v1/acta/meetings/calendar', { month }),
    getAttendance: (meetingId: string): Promise<ActaAttendee[]> => fetchSuiteData(`/api/v1/acta/meetings/${meetingId}/attendance`),
    recordAttendance: (meetingId: string, payload: unknown) =>
      apiPost<{ data: ActaAttendee[] }>(`/api/v1/acta/meetings/${meetingId}/attendance`, payload).then((res) => res.data),
    bulkRecordAttendance: (meetingId: string, payload: unknown) =>
      apiPost<{ data: ActaAttendee[] }>(`/api/v1/acta/meetings/${meetingId}/attendance/bulk`, payload).then((res) => res.data),
    listAgenda: (meetingId: string): Promise<ActaAgendaItem[]> => fetchSuiteData(`/api/v1/acta/meetings/${meetingId}/agenda`),
    createAgendaItem: (meetingId: string, payload: unknown) =>
      apiPost<{ data: ActaAgendaItem }>(`/api/v1/acta/meetings/${meetingId}/agenda`, payload).then((res) => res.data),
    updateAgendaItem: (meetingId: string, itemId: string, payload: unknown) =>
      apiPut<{ data: ActaAgendaItem }>(`/api/v1/acta/meetings/${meetingId}/agenda/${itemId}`, payload).then((res) => res.data),
    deleteAgendaItem: (meetingId: string, itemId: string) => apiDelete<void>(`/api/v1/acta/meetings/${meetingId}/agenda/${itemId}`),
    reorderAgenda: (meetingId: string, itemIds: string[]) =>
      apiPut<{ data: ActaAgendaItem[] }>(`/api/v1/acta/meetings/${meetingId}/agenda/reorder`, { item_ids: itemIds }).then((res) => res.data),
    updateAgendaNotes: (meetingId: string, itemId: string, notes: string) =>
      apiPut<{ data: ActaAgendaItem }>(`/api/v1/acta/meetings/${meetingId}/agenda/${itemId}/notes`, { notes }).then((res) => res.data),
    voteAgendaItem: (meetingId: string, itemId: string, payload: unknown) =>
      apiPost<{ data: ActaAgendaItem }>(`/api/v1/acta/meetings/${meetingId}/agenda/${itemId}/vote`, payload).then((res) => res.data),
    createMinutes: (meetingId: string, content: string) =>
      apiPost<{ data: ActaMeetingMinutes }>(`/api/v1/acta/meetings/${meetingId}/minutes`, { content }).then((res) => res.data),
    getMinutes: (meetingId: string): Promise<ActaMeetingMinutes> => fetchSuiteData(`/api/v1/acta/meetings/${meetingId}/minutes`),
    listMinutesVersions: (meetingId: string): Promise<ActaMeetingMinutes[]> => fetchSuiteData(`/api/v1/acta/meetings/${meetingId}/minutes/versions`),
    generateMinutes: (meetingId: string) =>
      apiPost<{ data: ActaMeetingMinutes }>(`/api/v1/acta/meetings/${meetingId}/minutes/generate`).then((res) => res.data),
    updateMinutes: (meetingId: string, content: string) =>
      apiPut<{ data: ActaMeetingMinutes }>(`/api/v1/acta/meetings/${meetingId}/minutes`, { content }).then((res) => res.data),
    submitMinutes: (meetingId: string) =>
      apiPost<{ data: ActaMeetingMinutes }>(`/api/v1/acta/meetings/${meetingId}/minutes/submit`).then((res) => res.data),
    requestMinutesRevision: (meetingId: string, notes: string) =>
      apiPost<{ data: ActaMeetingMinutes }>(`/api/v1/acta/meetings/${meetingId}/minutes/request-revision`, { notes }).then((res) => res.data),
    approveMinutes: (meetingId: string) =>
      apiPost<{ data: ActaMeetingMinutes }>(`/api/v1/acta/meetings/${meetingId}/minutes/approve`).then((res) => res.data),
    publishMinutes: (meetingId: string) =>
      apiPost<{ data: ActaMeetingMinutes }>(`/api/v1/acta/meetings/${meetingId}/minutes/publish`).then((res) => res.data),
    listActionItems: (params: FetchParams) => fetchSuitePaginated<ActaActionItem>('/api/v1/acta/action-items', params),
    getActionItem: (id: string) => fetchSuiteData<ActaActionItem>(`/api/v1/acta/action-items/${id}`),
    createActionItem: (payload: unknown) => apiPost<{ data: ActaActionItem }>('/api/v1/acta/action-items', payload).then((res) => res.data),
    updateActionItem: (id: string, payload: unknown) => apiPut<{ data: ActaActionItem }>(`/api/v1/acta/action-items/${id}`, payload).then((res) => res.data),
    updateActionItemStatus: (id: string, payload: unknown) =>
      apiPut<{ data: ActaActionItem }>(`/api/v1/acta/action-items/${id}/status`, payload).then((res) => res.data),
    extendActionItem: (id: string, payload: unknown) =>
      apiPost<{ data: ActaActionItem }>(`/api/v1/acta/action-items/${id}/extend`, payload).then((res) => res.data),
    listOverdueActionItems: (): Promise<ActaActionItem[]> => fetchSuiteData('/api/v1/acta/action-items/overdue'),
    listMyActionItems: (): Promise<ActaActionItem[]> => fetchSuiteData('/api/v1/acta/action-items/my'),
    getActionItemStats: (): Promise<ActaActionItemStats> => fetchSuiteData('/api/v1/acta/action-items/stats'),
    runCompliance: (): Promise<ActaComplianceReport> => fetchSuiteData('/api/v1/acta/compliance/run'),
    listComplianceResults: (params: FetchParams) => fetchSuitePaginated<ActaComplianceCheck>('/api/v1/acta/compliance/results', params),
    getComplianceReport: (): Promise<ActaComplianceReport> => fetchSuiteData('/api/v1/acta/compliance/report'),
    getComplianceScore: (): Promise<{ score: number }> => fetchSuiteData('/api/v1/acta/compliance/score'),
    listAttachments: (meetingId: string): Promise<ActaMeetingAttachment[]> => fetchSuiteData(`/api/v1/acta/meetings/${meetingId}/attachments`),
    addAttachmentReference: (meetingId: string, payload: unknown) =>
      apiPost<{ data: ActaMeetingAttachment[] }>(`/api/v1/acta/meetings/${meetingId}/attachments`, payload).then((res) => res.data),
    deleteAttachment: (meetingId: string, fileId: string) => apiDelete<void>(`/api/v1/acta/meetings/${meetingId}/attachments/${fileId}`),
  },
  lex: {
    getDashboard: (): Promise<LexDashboard> => fetchSuiteData('/api/v1/lex/dashboard'),
    listContracts: (params: FetchParams) => fetchSuitePaginated<LexContractSummary>('/api/v1/lex/contracts', params),
    searchContracts: async (query: string, params: FetchParams): Promise<PaginatedResponse<LexContractSummary>> => {
      const response = await apiGet<{ data: LexContractSummary[]; meta: PaginatedResponse<LexContractSummary>['meta'] }>(
        '/api/v1/lex/contracts/search',
        {
          q: query,
          page: params.page,
          per_page: params.per_page,
        },
      );
      return {
        data: response.data,
        meta: response.meta,
      };
    },
    getContract: (id: string): Promise<LexContractDetail> => fetchSuiteData(`/api/v1/lex/contracts/${id}`),
    getContractAnalysis: (id: string): Promise<LexContractRiskAnalysis> => fetchSuiteData(`/api/v1/lex/contracts/${id}/analysis`),
    analyzeContract: (id: string) => apiPost<{ data: LexContractRiskAnalysis }>(`/api/v1/lex/contracts/${id}/analyze`).then((res) => res.data),
    createContract: (payload: unknown) => apiPost<{ data: LexContractRecord }>('/api/v1/lex/contracts', payload).then((res) => res.data),
    updateContract: (id: string, payload: unknown) => apiPut<{ data: LexContractRecord }>(`/api/v1/lex/contracts/${id}`, payload).then((res) => res.data),
    updateContractStatus: (id: string, payload: unknown) =>
      apiPut<{ data: LexContractRecord }>(`/api/v1/lex/contracts/${id}/status`, payload).then((res) => res.data),
    listContractVersions: (id: string): Promise<LexContractVersion[]> => fetchSuiteData(`/api/v1/lex/contracts/${id}/versions`),
    renewContract: (id: string, payload: unknown) => apiPost<{ data: LexContractRecord }>(`/api/v1/lex/contracts/${id}/renew`, payload).then((res) => res.data),
    startContractReview: (id: string, payload: unknown) =>
      apiPost<{ data: LexWorkflowSummary }>(`/api/v1/lex/contracts/${id}/review`, payload).then((res) => res.data),
    listWorkflows: (params: FetchParams) => fetchSuitePaginated<LexWorkflowSummary>('/api/v1/lex/workflows', params),
    listContractClauses: (id: string): Promise<LexClause[]> => fetchSuiteData(`/api/v1/lex/contracts/${id}/clauses`),
    getClause: (contractId: string, clauseId: string): Promise<LexClause> =>
      fetchSuiteData(`/api/v1/lex/contracts/${contractId}/clauses/${clauseId}`),
    listClauseRiskSummary: (contractId: string): Promise<LexClause[]> =>
      fetchSuiteData(`/api/v1/lex/contracts/${contractId}/clauses/risks`),
    updateClauseReview: (contractId: string, clauseId: string, payload: unknown) =>
      apiPut<{ data: LexClause }>(`/api/v1/lex/contracts/${contractId}/clauses/${clauseId}/review`, payload).then((res) => res.data),
    listDocuments: (params: FetchParams) => fetchSuitePaginated<LexDocument>('/api/v1/lex/documents', params),
    createDocument: (payload: unknown) => apiPost<{ data: LexDocument }>('/api/v1/lex/documents', payload).then((res) => res.data),
    getDocument: (id: string): Promise<LexDocument> => fetchSuiteData(`/api/v1/lex/documents/${id}`),
    updateDocument: (id: string, payload: unknown) => apiPut<{ data: LexDocument }>(`/api/v1/lex/documents/${id}`, payload).then((res) => res.data),
    listDocumentVersions: (id: string): Promise<LexDocumentVersion[]> => fetchSuiteData(`/api/v1/lex/documents/${id}/versions`),
    uploadDocumentVersion: (id: string, payload: unknown) =>
      apiPost<{ data: LexDocumentVersion[] }>(`/api/v1/lex/documents/${id}/upload`, payload).then((res) => res.data),
    listComplianceRules: (params: FetchParams) => fetchSuitePaginated<LexComplianceRule>('/api/v1/lex/compliance/rules', params),
    createComplianceRule: (payload: unknown) => apiPost<{ data: LexComplianceRule }>('/api/v1/lex/compliance/rules', payload).then((res) => res.data),
    updateComplianceRule: (id: string, payload: unknown) => apiPut<{ data: LexComplianceRule }>(`/api/v1/lex/compliance/rules/${id}`, payload).then((res) => res.data),
    deleteComplianceRule: (id: string) => apiDelete<void>(`/api/v1/lex/compliance/rules/${id}`),
    runCompliance: (payload: unknown) => apiPost<{ data: LexComplianceRunResult }>('/api/v1/lex/compliance/run', payload).then((res) => res.data),
    listComplianceAlerts: (params: FetchParams) => fetchSuitePaginated<LexComplianceAlert>('/api/v1/lex/compliance/alerts', params),
    updateComplianceAlertStatus: (id: string, payload: unknown) =>
      apiPut<{ data: LexComplianceAlert }>(`/api/v1/lex/compliance/alerts/${id}/status`, payload).then((res) => res.data),
    getComplianceDashboard: (): Promise<LexComplianceDashboard> => fetchSuiteData('/api/v1/lex/compliance/dashboard'),
    getComplianceScore: (): Promise<LexComplianceScore> => fetchSuiteData('/api/v1/lex/compliance/score'),
    getExpiringContracts: (days?: number): Promise<LexExpiringContractSummary[]> =>
      fetchSuiteData('/api/v1/lex/contracts/expiring', days ? { days } : undefined),
  },
  visus: {
    listDashboards: (params: FetchParams) => fetchSuitePaginated<VisusDashboard>('/api/v1/visus/dashboards', params),
    getDashboard: (id: string): Promise<VisusDashboard> => fetchSuiteData(`/api/v1/visus/dashboards/${id}`),
    createDashboard: (payload: unknown) => apiPost<{ data: VisusDashboard }>('/api/v1/visus/dashboards', payload).then((res) => res.data),
    updateDashboard: (id: string, payload: unknown) => apiPut<{ data: VisusDashboard }>(`/api/v1/visus/dashboards/${id}`, payload).then((res) => res.data),
    deleteDashboard: (id: string) => apiDelete<void>(`/api/v1/visus/dashboards/${id}`),
    duplicateDashboard: (id: string) => apiPost<{ data: VisusDashboard }>(`/api/v1/visus/dashboards/${id}/duplicate`).then((res) => res.data),
    shareDashboard: (id: string, payload: unknown) => apiPut<{ data: VisusDashboard }>(`/api/v1/visus/dashboards/${id}/share`, payload).then((res) => res.data),
    listWidgets: (dashboardId: string): Promise<VisusWidget[]> => fetchSuiteData(`/api/v1/visus/dashboards/${dashboardId}/widgets`),
    createWidget: (dashboardId: string, payload: unknown) =>
      apiPost<{ data: VisusWidget }>(`/api/v1/visus/dashboards/${dashboardId}/widgets`, payload).then((res) => res.data),
    updateWidget: (dashboardId: string, widgetId: string, payload: unknown) =>
      apiPut<{ data: VisusWidget }>(`/api/v1/visus/dashboards/${dashboardId}/widgets/${widgetId}`, payload).then((res) => res.data),
    deleteWidget: (dashboardId: string, widgetId: string) =>
      apiDelete<void>(`/api/v1/visus/dashboards/${dashboardId}/widgets/${widgetId}`),
    updateWidgetLayout: (dashboardId: string, positions: Array<{ widget_id: string; x: number; y: number; w: number; h: number }>) =>
      apiPut<{ data: { updated: number } }>(`/api/v1/visus/dashboards/${dashboardId}/widgets/layout`, { positions }).then((res) => res.data),
    getWidgetData: (dashboardId: string, widgetId: string): Promise<VisusWidgetData> =>
      fetchSuiteData(`/api/v1/visus/dashboards/${dashboardId}/widgets/${widgetId}/data`),
    listWidgetTypes: (): Promise<VisusWidgetTypeDefinition[]> => fetchSuiteData('/api/v1/visus/widgets/types'),
    listKpis: (params: FetchParams) => fetchSuitePaginated<VisusKPIDefinition>('/api/v1/visus/kpis', params),
    getKpi: (id: string): Promise<VisusKPIGetResponse> => fetchSuiteData(`/api/v1/visus/kpis/${id}`),
    getKpiHistory: (id: string, params?: { start?: string; end?: string; per_page?: number }): Promise<VisusKPISnapshot[]> =>
      fetchSuiteData(`/api/v1/visus/kpis/${id}/history`, params),
    createKpi: (payload: unknown) => apiPost<{ data: VisusKPIDefinition }>('/api/v1/visus/kpis', payload).then((res) => res.data),
    updateKpi: (id: string, payload: unknown) => apiPut<{ data: VisusKPIDefinition }>(`/api/v1/visus/kpis/${id}`, payload).then((res) => res.data),
    deleteKpi: (id: string) => apiDelete<void>(`/api/v1/visus/kpis/${id}`),
    triggerKpiSnapshot: () => apiPost<{ data: { status: string } }>('/api/v1/visus/kpis/snapshot').then((res) => res.data),
    listReports: (params: FetchParams) => fetchSuitePaginated<VisusReportDefinition>('/api/v1/visus/reports', params),
    getReport: (id: string): Promise<VisusReportDefinition> => fetchSuiteData(`/api/v1/visus/reports/${id}`),
    createReport: (payload: unknown) => apiPost<{ data: VisusReportDefinition }>('/api/v1/visus/reports', payload).then((res) => res.data),
    updateReport: (id: string, payload: unknown) => apiPut<{ data: VisusReportDefinition }>(`/api/v1/visus/reports/${id}`, payload).then((res) => res.data),
    deleteReport: (id: string) => apiDelete<void>(`/api/v1/visus/reports/${id}`),
    generateReport: (id: string) => apiPost<{ data: VisusReportSnapshot }>(`/api/v1/visus/reports/${id}/generate`).then((res) => res.data),
    listReportSnapshots: (id: string): Promise<VisusReportSnapshot[]> => fetchSuiteData(`/api/v1/visus/reports/${id}/snapshots`),
    getLatestReportSnapshot: (id: string): Promise<VisusReportSnapshot> => fetchSuiteData(`/api/v1/visus/reports/${id}/snapshots/latest`),
    getReportSnapshot: (id: string, snapshotId: string): Promise<VisusReportSnapshot> =>
      fetchSuiteData(`/api/v1/visus/reports/${id}/snapshots/${snapshotId}`),
    listAlerts: (params: FetchParams) => fetchSuitePaginated<VisusExecutiveAlert>('/api/v1/visus/alerts', params),
    getAlert: (id: string): Promise<VisusExecutiveAlert> => fetchSuiteData(`/api/v1/visus/alerts/${id}`),
    updateAlertStatus: (id: string, payload: unknown) =>
      apiPut<{ data: VisusExecutiveAlert }>(`/api/v1/visus/alerts/${id}/status`, payload).then((res) => res.data),
    getAlertCount: (): Promise<{ count: number }> => fetchSuiteData('/api/v1/visus/alerts/count'),
    getAlertStats: (): Promise<JsonObject> => fetchSuiteData('/api/v1/visus/alerts/stats'),
    getExecutiveView: (): Promise<VisusExecutiveSummary> => fetchSuiteData('/api/v1/visus/executive'),
  },
  ai: {
    getDashboard: (): Promise<AIDashboardData> => fetchSuiteData('/api/v1/ai/dashboard'),
    listModels: (params: FetchParams) => fetchSuitePaginated<AIModelWithVersions>('/api/v1/ai/models', params),
    createModel: (payload: AIRegisterModelPayload): Promise<AIRegisteredModel> =>
      apiPost<{ data: AIRegisteredModel }>('/api/v1/ai/models', payload).then((res) => res.data),
    getModel: (id: string): Promise<AIModelWithVersions> => fetchSuiteData(`/api/v1/ai/models/${id}`),
    updateModel: (id: string, payload: AIUpdateModelPayload): Promise<AIRegisteredModel> =>
      apiPut<{ data: AIRegisteredModel }>(`/api/v1/ai/models/${id}`, payload).then((res) => res.data),
    createVersion: (id: string, payload: AICreateVersionPayload): Promise<AIModelVersion> =>
      apiPost<{ data: AIModelVersion }>(`/api/v1/ai/models/${id}/versions`, payload).then((res) => res.data),
    listVersions: (id: string): Promise<AIModelVersion[]> => fetchSuiteData(`/api/v1/ai/models/${id}/versions`),
    getVersion: (id: string, versionId: string): Promise<AIModelVersion> => fetchSuiteData(`/api/v1/ai/models/${id}/versions/${versionId}`),
    promote: (id: string, versionId: string, payload?: { approved_by?: string; override?: boolean }) =>
      apiPost<{ data: AIModelVersion }>(`/api/v1/ai/models/${id}/versions/${versionId}/promote`, payload ?? {}).then((res) => res.data),
    retire: (id: string, versionId: string, payload: { reason: string }) =>
      apiPost<{ data: AIModelVersion }>(`/api/v1/ai/models/${id}/versions/${versionId}/retire`, payload).then((res) => res.data),
    failVersion: (id: string, versionId: string, payload: { reason: string }) =>
      apiPost<{ data: AIModelVersion }>(`/api/v1/ai/models/${id}/versions/${versionId}/fail`, payload).then((res) => res.data),
    rollback: (id: string, payload: { reason: string }) =>
      apiPost<{ data: AIModelVersion }>(`/api/v1/ai/models/${id}/rollback`, payload).then((res) => res.data),
    lifecycleHistory: (id: string): Promise<AILifecycleHistoryEntry[]> => fetchSuiteData(`/api/v1/ai/models/${id}/lifecycle-history`),
    startShadow: (id: string, payload: { version_id: string }) =>
      apiPost<{ data: AIModelVersion }>(`/api/v1/ai/models/${id}/shadow/start`, payload).then((res) => res.data),
    stopShadow: (id: string, payload: { version_id: string; reason: string }) =>
      apiPost<{ data: AIModelVersion }>(`/api/v1/ai/models/${id}/shadow/stop`, payload).then((res) => res.data),
    latestComparison: (id: string): Promise<AIShadowComparison> => fetchSuiteData(`/api/v1/ai/models/${id}/shadow/comparison`),
    comparisonHistory: (id: string, limit = 24): Promise<AIShadowComparison[]> =>
      fetchSuiteData(`/api/v1/ai/models/${id}/shadow/comparison/history`, { limit }),
    divergences: (id: string, params: FetchParams) =>
      fetchSuitePaginated<AIShadowDivergence>(`/api/v1/ai/models/${id}/shadow/divergences`, params),
    listPredictions: (params: FetchParams) => fetchSuitePaginated<AIPredictionLog>('/api/v1/ai/predictions', params),
    getPrediction: (id: string): Promise<AIPredictionLog> => fetchSuiteData(`/api/v1/ai/predictions/${id}`),
    submitFeedback: (id: string, payload: { correct: boolean; notes: string; corrected_output?: unknown }) =>
      apiPost<{ data: { message: string } }>(`/api/v1/ai/predictions/${id}/feedback`, payload).then((res) => res.data),
    predictionStats: (): Promise<AIPredictionStats[]> => fetchSuiteData('/api/v1/ai/predictions/stats'),
    getExplanation: (predictionId: string): Promise<AIExplanation> => fetchSuiteData(`/api/v1/ai/explanations/${predictionId}`),
    searchExplanations: (query: string, limit = 20): Promise<AIPredictionLog[]> =>
      fetchSuiteData('/api/v1/ai/explanations/search', { q: query, limit }),
    latestDrift: (id: string): Promise<AIDriftReport> => fetchSuiteData(`/api/v1/ai/models/${id}/drift`),
    driftHistory: (id: string, limit = 24): Promise<AIDriftReport[]> =>
      fetchSuiteData(`/api/v1/ai/models/${id}/drift/history`, { limit }),
    performance: (id: string, period = '30d'): Promise<AIPerformancePoint[]> =>
      fetchSuiteData(`/api/v1/ai/models/${id}/performance`, { period }),
    previewValidation: (id: string, versionId: string, payload: unknown): Promise<AIValidationPreview> =>
      apiPost<{ data: AIValidationPreview }>(`/api/v1/ai/models/${id}/versions/${versionId}/validation/preview`, payload).then((res) => res.data),
    validate: (id: string, versionId: string, payload: unknown): Promise<AIValidationResult> =>
      apiPost<{ data: AIValidationResult }>(`/api/v1/ai/models/${id}/versions/${versionId}/validate`, payload).then((res) => res.data),
    latestValidation: (id: string, versionId: string): Promise<AIValidationResult> =>
      fetchSuiteData(`/api/v1/ai/models/${id}/versions/${versionId}/validation`),
    validationHistory: (id: string, versionId: string, limit = 10): Promise<AIValidationResult[]> =>
      fetchSuiteData(`/api/v1/ai/models/${id}/versions/${versionId}/validation/history`, { limit }),
  },
};

export type EnterpriseApi = typeof enterpriseApi;
