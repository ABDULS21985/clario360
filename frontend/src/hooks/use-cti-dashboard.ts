'use client';

import { useEffect } from 'react';
import { useCTIStore } from '@/stores/cti-store';

export function useCTIDashboard() {
  const {
    loadReferenceData,
    loadDashboard,
    loadThreatMap,
    dashboardPeriod,
    setDashboardPeriod,
    executiveSnapshot,
    geoHotspots,
    sectorSummaries,
    topCampaigns,
    criticalBrands,
    recentEvents,
    isLoadingDashboard,
  } = useCTIStore();

  useEffect(() => {
    void loadReferenceData();
    void loadDashboard();

    const interval = setInterval(() => {
      void loadDashboard();
    }, 60_000);

    return () => clearInterval(interval);
  }, [loadDashboard, loadReferenceData]);

  return {
    period: dashboardPeriod,
    setPeriod: setDashboardPeriod,
    snapshot: executiveSnapshot,
    hotspots: geoHotspots,
    sectors: sectorSummaries,
    topCampaigns,
    criticalBrands,
    recentEvents,
    reloadThreatMap: loadThreatMap,
    isLoading: isLoadingDashboard && !executiveSnapshot,
  };
}
