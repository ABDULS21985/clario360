import type {
  CreateIndicatorInput,
  IndicatorType,
  ThreatStatus,
  ThreatType,
} from '@/types/cyber';
import { slugToTitle } from '@/lib/utils';

export const THREAT_TYPE_OPTIONS: Array<{ label: string; value: ThreatType }> = [
  { label: 'Malware', value: 'malware' },
  { label: 'Phishing', value: 'phishing' },
  { label: 'APT', value: 'apt' },
  { label: 'Ransomware', value: 'ransomware' },
  { label: 'DDoS', value: 'ddos' },
  { label: 'Insider Threat', value: 'insider_threat' },
  { label: 'Supply Chain', value: 'supply_chain' },
  { label: 'Zero Day', value: 'zero_day' },
  { label: 'Brute Force', value: 'brute_force' },
  { label: 'Other', value: 'other' },
];

export const THREAT_STATUS_OPTIONS: Array<{ label: string; value: ThreatStatus }> = [
  { label: 'Active', value: 'active' },
  { label: 'Contained', value: 'contained' },
  { label: 'Eradicated', value: 'eradicated' },
  { label: 'Monitoring', value: 'monitoring' },
  { label: 'Closed', value: 'closed' },
];

export const INDICATOR_TYPE_OPTIONS: Array<{ label: string; value: IndicatorType }> = [
  { label: 'IP Address', value: 'ip' },
  { label: 'Domain', value: 'domain' },
  { label: 'URL', value: 'url' },
  { label: 'Email', value: 'email' },
  { label: 'File Hash (MD5)', value: 'file_hash_md5' },
  { label: 'File Hash (SHA1)', value: 'file_hash_sha1' },
  { label: 'File Hash (SHA256)', value: 'file_hash_sha256' },
  { label: 'Certificate', value: 'certificate' },
  { label: 'Registry Key', value: 'registry_key' },
  { label: 'User Agent', value: 'user_agent' },
  { label: 'CIDR', value: 'cidr' },
];

export const THREAT_STATUS_TRANSITIONS: Record<ThreatStatus, ThreatStatus[]> = {
  active: ['contained', 'monitoring'],
  contained: ['eradicated', 'active'],
  monitoring: ['closed', 'active'],
  eradicated: ['closed'],
  closed: [],
};

export const SEVERITY_COLORS: Record<string, string> = {
  critical: '#DC2626',
  high: '#EA580C',
  medium: '#CA8A04',
  low: '#2563EB',
  info: '#6B7280',
};

export function getThreatTypeLabel(value: string): string {
  return THREAT_TYPE_OPTIONS.find((option) => option.value === value)?.label ?? slugToTitle(value);
}

export function getIndicatorTypeLabel(value: string): string {
  return INDICATOR_TYPE_OPTIONS.find((option) => option.value === value)?.label ?? slugToTitle(value);
}

export function emptyIndicator(): CreateIndicatorInput {
  return {
    type: 'ip',
    value: '',
    severity: 'medium',
    confidence: 75,
    source: 'manual',
    description: '',
    tags: [],
  };
}
