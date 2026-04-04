import { formatDistanceToNowStrict } from 'date-fns';
import {
  CTI_ACTOR_TYPE_LABELS,
  CTI_CAMPAIGN_STATUS_LABELS,
  CTI_MOTIVATION_LABELS,
  CTI_RISK_LEVEL_LABELS,
  CTI_SOPHISTICATION_LABELS,
  CTI_TAKEDOWN_STATUS_LABELS,
  type CTIActorMotivation,
  type CTICampaignStatus,
  type CTIRiskLevel,
  type CTISophisticationLevel,
  type CTITakedownStatus,
  type CTIThreatActorType,
} from '@/types/cti';

export const CTI_CAMPAIGN_STATUS_OPTIONS: Array<{ label: string; value: CTICampaignStatus }> = [
  { label: 'Active', value: 'active' },
  { label: 'Monitoring', value: 'monitoring' },
  { label: 'Dormant', value: 'dormant' },
  { label: 'Resolved', value: 'resolved' },
  { label: 'Archived', value: 'archived' },
];

export const CTI_THREAT_ACTOR_OPTIONS: Array<{ label: string; value: CTIThreatActorType }> = [
  { label: CTI_ACTOR_TYPE_LABELS.state_sponsored, value: 'state_sponsored' },
  { label: CTI_ACTOR_TYPE_LABELS.cybercriminal, value: 'cybercriminal' },
  { label: CTI_ACTOR_TYPE_LABELS.hacktivist, value: 'hacktivist' },
  { label: CTI_ACTOR_TYPE_LABELS.insider, value: 'insider' },
  { label: CTI_ACTOR_TYPE_LABELS.unknown, value: 'unknown' },
];

export const CTI_ACTOR_MOTIVATION_OPTIONS: Array<{ label: string; value: CTIActorMotivation }> = [
  { label: CTI_MOTIVATION_LABELS.espionage, value: 'espionage' },
  { label: CTI_MOTIVATION_LABELS.financial_gain, value: 'financial_gain' },
  { label: CTI_MOTIVATION_LABELS.disruption, value: 'disruption' },
  { label: CTI_MOTIVATION_LABELS.ideological, value: 'ideological' },
  { label: CTI_MOTIVATION_LABELS.unknown, value: 'unknown' },
];

export const CTI_SOPHISTICATION_OPTIONS: Array<{ label: string; value: CTISophisticationLevel }> = [
  { label: CTI_SOPHISTICATION_LABELS.advanced, value: 'advanced' },
  { label: CTI_SOPHISTICATION_LABELS.intermediate, value: 'intermediate' },
  { label: CTI_SOPHISTICATION_LABELS.basic, value: 'basic' },
];

export const CTI_RISK_LEVEL_OPTIONS: Array<{ label: string; value: CTIRiskLevel }> = [
  { label: CTI_RISK_LEVEL_LABELS.critical, value: 'critical' },
  { label: CTI_RISK_LEVEL_LABELS.high, value: 'high' },
  { label: CTI_RISK_LEVEL_LABELS.medium, value: 'medium' },
  { label: CTI_RISK_LEVEL_LABELS.low, value: 'low' },
];

export const CTI_TAKEDOWN_STATUS_OPTIONS: Array<{ label: string; value: CTITakedownStatus }> = [
  { label: CTI_TAKEDOWN_STATUS_LABELS.detected, value: 'detected' },
  { label: CTI_TAKEDOWN_STATUS_LABELS.reported, value: 'reported' },
  { label: CTI_TAKEDOWN_STATUS_LABELS.takedown_requested, value: 'takedown_requested' },
  { label: CTI_TAKEDOWN_STATUS_LABELS.taken_down, value: 'taken_down' },
  { label: CTI_TAKEDOWN_STATUS_LABELS.monitoring, value: 'monitoring' },
  { label: CTI_TAKEDOWN_STATUS_LABELS.false_positive, value: 'false_positive' },
];

export const CTI_BRAND_ABUSE_TYPE_OPTIONS = [
  { label: 'Typosquatting', value: 'typosquatting' },
  { label: 'Phishing Kit', value: 'phishing_kit' },
  { label: 'Brand Impersonation', value: 'brand_impersonation' },
  { label: 'Social Impersonation', value: 'social_impersonation' },
  { label: 'Fake Mobile App', value: 'fake_mobile_app' },
  { label: 'Credential Harvesting', value: 'credential_harvesting' },
];

export const CTI_TAKEDOWN_WORKFLOW: CTITakedownStatus[] = [
  'detected',
  'reported',
  'takedown_requested',
  'taken_down',
];

export function parseTagInput(value: string): string[] {
  return Array.from(
    new Set(
      value
        .split(',')
        .map((item) => item.trim())
        .filter(Boolean),
    ),
  );
}

export function buildTagInputValue(values?: string[] | null): string {
  return values?.join(', ') ?? '';
}

export function formatConfidenceScore(score?: number | null): string {
  if (score === null || score === undefined || Number.isNaN(score)) {
    return '—';
  }

  return `${Math.round(score * 100)}%`;
}

export function buildMitreTechniqueHref(techniqueId: string): string {
  return `https://attack.mitre.org/techniques/${techniqueId.replace('.', '/')}/`;
}

export function formatCountryCode(countryCode?: string | null): string {
  return countryCode ? countryCode.toUpperCase() : '—';
}

export function countryCodeToFlag(code?: string | null): string {
  if (!code || code.length !== 2) {
    return '🌐';
  }

  return [...code.toUpperCase()]
    .map((char) => String.fromCodePoint(0x1f1e6 - 65 + char.charCodeAt(0)))
    .join('');
}

export function formatRelativeTime(dateStr?: string | null): string {
  if (!dateStr) {
    return '—';
  }

  const date = new Date(dateStr);
  if (Number.isNaN(date.getTime())) {
    return '—';
  }

  return `${formatDistanceToNowStrict(date, { addSuffix: true })}`;
}

export function formatNumber(value?: number | null): string {
  if (value === null || value === undefined || Number.isNaN(value)) {
    return '0';
  }

  return value.toLocaleString();
}

export function severityToColor(severity?: string | null): string {
  if (!severity) {
    return '#6B7280';
  }

  return {
    critical: '#FF3B5C',
    high: '#FF8C42',
    medium: '#FFD93D',
    low: '#4ADE80',
    informational: '#94A3B8',
  }[severity] ?? '#6B7280';
}

export function labelFromMap<T extends string>(
  value: T | null | undefined,
  labels: Record<string, string>,
): string {
  if (!value) {
    return '—';
  }

  return labels[value] ?? value;
}

export function isHighPriorityRisk(riskLevel?: string | null): boolean {
  return riskLevel === 'critical' || riskLevel === 'high';
}

export function isActiveCampaign(status?: string | null): boolean {
  return status === 'active' || status === 'monitoring';
}

export function getCampaignStatusLabel(status?: string | null): string {
  return labelFromMap(status, CTI_CAMPAIGN_STATUS_LABELS);
}
