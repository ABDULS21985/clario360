'use client';

import { create } from 'zustand';
import { devtools } from 'zustand/middleware';
import {
  fetchCategories,
  fetchExecutiveDashboard,
  fetchGlobalThreatMap,
  fetchSectorThreatOverview,
  fetchSeverityLevels,
  fetchSectors,
} from '@/lib/cti-api';
import type {
  CTICampaign,
  CTIExecutiveDashboardResponse,
  CTIExecutiveSnapshot,
  CTIGeoThreatHotspot,
  CTIIndustrySector,
  CTIPeriod,
  CTISectorThreatSummary,
  CTISeverityLevel,
  CTIThreatCategory,
  CTIThreatEvent,
} from '@/types/cti';

interface MapViewport {
  zoom: number;
  center: [number, number];
}

interface CTIState {
  severityLevels: CTISeverityLevel[];
  categories: CTIThreatCategory[];
  sectors: CTIIndustrySector[];
  referenceDataLoaded: boolean;

  dashboardPeriod: Extract<CTIPeriod, '24h' | '7d' | '30d'>;
  executiveSnapshot: CTIExecutiveSnapshot | null;
  geoHotspots: CTIGeoThreatHotspot[];
  sectorSummaries: CTISectorThreatSummary[];
  topCampaigns: CTICampaign[];
  criticalBrands: CTIExecutiveDashboardResponse['critical_brands'];
  recentEvents: CTIThreatEvent[];
  isLoadingDashboard: boolean;

  selectedHotspot: CTIGeoThreatHotspot | null;
  mapViewport: MapViewport;

  liveEvents: CTIThreatEvent[];
  liveEventCount: number;

  loadReferenceData: () => Promise<void>;
  setDashboardPeriod: (period: Extract<CTIPeriod, '24h' | '7d' | '30d'>) => void;
  loadDashboard: () => Promise<void>;
  loadThreatMap: (period?: Extract<CTIPeriod, '24h' | '7d' | '30d'>) => Promise<void>;
  refreshExecutiveSnapshot: () => Promise<void>;
  setSelectedHotspot: (hotspot: CTIGeoThreatHotspot | null) => void;
  setMapViewport: (viewport: Partial<MapViewport>) => void;
  pushLiveEvent: (event: CTIThreatEvent) => void;
}

function dedupeEvents(events: CTIThreatEvent[], limit = 50): CTIThreatEvent[] {
  const seen = new Set<string>();
  const next: CTIThreatEvent[] = [];

  for (const event of events) {
    if (seen.has(event.id)) {
      continue;
    }
    seen.add(event.id);
    next.push(event);
    if (next.length >= limit) {
      break;
    }
  }

  return next;
}

function attachSectorLabel(
  snapshot: CTIExecutiveSnapshot | null,
  sectors: CTIIndustrySector[],
): CTIExecutiveSnapshot | null {
  if (!snapshot) {
    return null;
  }

  if (!snapshot.top_targeted_sector_id) {
    return {
      ...snapshot,
      top_targeted_sector_label: null,
    };
  }

  const sector = sectors.find((item) => item.id === snapshot.top_targeted_sector_id);
  return {
    ...snapshot,
    top_targeted_sector_label: sector?.label ?? null,
  };
}

export const useCTIStore = create<CTIState>()(
  devtools(
    (set, get) => ({
      severityLevels: [],
      categories: [],
      sectors: [],
      referenceDataLoaded: false,

      dashboardPeriod: '24h',
      executiveSnapshot: null,
      geoHotspots: [],
      sectorSummaries: [],
      topCampaigns: [],
      criticalBrands: [],
      recentEvents: [],
      isLoadingDashboard: false,

      selectedHotspot: null,
      mapViewport: { zoom: 2, center: [20, 0] },

      liveEvents: [],
      liveEventCount: 0,

      loadReferenceData: async () => {
        if (get().referenceDataLoaded) {
          return;
        }

        const [severityLevels, categories, sectors] = await Promise.all([
          fetchSeverityLevels(),
          fetchCategories(),
          fetchSectors(),
        ]);

        set((state) => ({
          severityLevels,
          categories,
          sectors,
          referenceDataLoaded: true,
          executiveSnapshot: attachSectorLabel(state.executiveSnapshot, sectors),
        }));
      },

      setDashboardPeriod: (period) => {
        set({ dashboardPeriod: period });
        void get().loadDashboard();
      },

      loadDashboard: async () => {
        set({ isLoadingDashboard: true });
        await get().loadReferenceData();
        const period = get().dashboardPeriod;
        const [mapData, sectorData, execData] = await Promise.all([
          fetchGlobalThreatMap(period),
          fetchSectorThreatOverview(period),
          fetchExecutiveDashboard(),
        ]);

        const sectors = get().sectors;
        set((state) => ({
          geoHotspots: mapData.hotspots,
          sectorSummaries: sectorData.sectors,
          executiveSnapshot: attachSectorLabel(execData.snapshot, sectors),
          topCampaigns: execData.top_campaigns,
          criticalBrands: execData.critical_brands,
          recentEvents: execData.recent_events,
          liveEvents:
            state.liveEvents.length > 0
              ? dedupeEvents([...state.liveEvents, ...execData.recent_events], 50)
              : dedupeEvents(execData.recent_events, 50),
          isLoadingDashboard: false,
        }));
      },

      loadThreatMap: async (period) => {
        const requestedPeriod = period ?? get().dashboardPeriod;
        const mapData = await fetchGlobalThreatMap(requestedPeriod);
        set({
          dashboardPeriod: requestedPeriod,
          geoHotspots: mapData.hotspots,
        });
      },

      refreshExecutiveSnapshot: async () => {
        await get().loadReferenceData();
        const data = await fetchExecutiveDashboard();
        const sectors = get().sectors;
        set((state) => ({
          executiveSnapshot: attachSectorLabel(data.snapshot, sectors),
          topCampaigns: data.top_campaigns,
          criticalBrands: data.critical_brands,
          recentEvents: data.recent_events,
          liveEvents: dedupeEvents([...state.liveEvents, ...data.recent_events], 50),
        }));
      },

      setSelectedHotspot: (hotspot) => set({ selectedHotspot: hotspot }),

      setMapViewport: (viewport) =>
        set((state) => ({
          mapViewport: {
            ...state.mapViewport,
            ...viewport,
          },
        })),

      pushLiveEvent: (event) =>
        set((state) => {
          const liveEvents = dedupeEvents([event, ...state.liveEvents], 50);
          const recentEvents = dedupeEvents([event, ...state.recentEvents], 12);
          return {
            liveEvents,
            recentEvents,
            liveEventCount: state.liveEventCount + 1,
          };
        }),
    }),
    { name: 'cti-store' },
  ),
);
