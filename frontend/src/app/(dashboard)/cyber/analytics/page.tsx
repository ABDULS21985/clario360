'use client';

import { PageHeader } from '@/components/common/page-header';
import { PermissionRedirect } from '@/components/common/permission-redirect';

import { ThreatLandscape } from './_components/threat-landscape';
import { ThreatForecast } from './_components/threat-forecast';
import { AlertVolumeForecast } from './_components/alert-volume-forecast';
import { TechniqueTrends } from './_components/technique-trends';
import { CampaignDetection } from './_components/campaign-detection';

export default function CyberAnalyticsPage() {
  return (
    <PermissionRedirect permission="cyber:read">
      <div className="space-y-8">
        <PageHeader
          title="Threat Analytics"
          description="Predictive intelligence, campaign detection, and attack technique trend analysis powered by ML models."
        />

        {/* Section 1: Threat Landscape Overview */}
        <ThreatLandscape />

        {/* Section 2: Threat Forecast */}
        <ThreatForecast />

        {/* Section 3: Alert Volume Forecast */}
        <AlertVolumeForecast />

        {/* Section 4: Technique Trends */}
        <TechniqueTrends />

        {/* Section 5: Campaign Detection */}
        <CampaignDetection />
      </div>
    </PermissionRedirect>
  );
}
