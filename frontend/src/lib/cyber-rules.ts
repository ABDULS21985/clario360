import { parse, stringify } from 'yaml';

import type {
  AnomalyRuleContent,
  CorrelationRuleContent,
  CyberSeverity,
  DetectionRule,
  DetectionRuleType,
  MITRETechniqueCoverage,
  RuleCondition,
  RuleTemplate,
  SigmaRuleContent,
  ThresholdRuleContent,
} from '@/types/cyber';
import type { FetchParams } from '@/types/table';
import { slugToTitle } from '@/lib/utils';

export const DETECTION_RULE_TYPE_OPTIONS: Array<{ label: string; value: DetectionRuleType }> = [
  { label: 'Sigma', value: 'sigma' },
  { label: 'Threshold', value: 'threshold' },
  { label: 'Correlation', value: 'correlation' },
  { label: 'Anomaly', value: 'anomaly' },
];

export const RULE_SEVERITY_OPTIONS: Array<{ label: string; value: CyberSeverity }> = [
  { label: 'Critical', value: 'critical' },
  { label: 'High', value: 'high' },
  { label: 'Medium', value: 'medium' },
  { label: 'Low', value: 'low' },
  { label: 'Info', value: 'info' },
];

export const RULE_FIELD_OPTIONS = [
  { label: 'Event Type', value: 'type' },
  { label: 'Source', value: 'source' },
  { label: 'Severity', value: 'severity' },
  { label: 'Source IP', value: 'source_ip' },
  { label: 'Destination IP', value: 'dest_ip' },
  { label: 'Destination Port', value: 'dest_port' },
  { label: 'Protocol', value: 'protocol' },
  { label: 'Username', value: 'username' },
  { label: 'Process', value: 'process' },
  { label: 'Parent Process', value: 'parent_process' },
  { label: 'Command Line', value: 'command_line' },
  { label: 'File Path', value: 'file_path' },
  { label: 'File Hash', value: 'file_hash' },
  { label: 'Asset ID', value: 'asset_id' },
  { label: 'Raw Event Name', value: 'raw.event_name' },
  { label: 'Raw Action', value: 'raw.action' },
  { label: 'Raw Status', value: 'raw.status' },
  { label: 'Bytes Transferred', value: 'raw.bytes_transferred' },
  { label: 'DNS Query', value: 'raw.dns_query' },
  { label: 'HTTP Path', value: 'raw.http_path' },
  { label: 'Registry Key', value: 'raw.registry_key' },
  { label: 'User Agent', value: 'raw.user_agent' },
] as const;

export const RULE_OPERATOR_OPTIONS = [
  { label: 'Equals', value: 'eq' },
  { label: 'In List', value: 'in' },
  { label: 'Contains', value: 'contains' },
  { label: 'Starts With', value: 'startswith' },
  { label: 'Ends With', value: 'endswith' },
  { label: 'Regex', value: 're' },
  { label: 'Greater Than', value: 'gt' },
  { label: 'Greater Than Or Equal', value: 'gte' },
  { label: 'Less Than', value: 'lt' },
  { label: 'Less Than Or Equal', value: 'lte' },
  { label: 'CIDR Match', value: 'cidr' },
  { label: 'Exists', value: 'exists' },
  { label: 'Contains All', value: 'all' },
  { label: 'Base64 Contains', value: 'base64' },
] as const;

const OPERATOR_TO_SUFFIX: Record<string, string> = {
  eq: '',
  exact: '',
  in: '|in',
  '|in': '|in',
  contains: '|contains',
  '|contains': '|contains',
  startswith: '|startswith',
  '|startswith': '|startswith',
  endswith: '|endswith',
  '|endswith': '|endswith',
  re: '|re',
  '|re': '|re',
  gt: '|gt',
  '|gt': '|gt',
  gte: '|gte',
  '|gte': '|gte',
  lt: '|lt',
  '|lt': '|lt',
  lte: '|lte',
  '|lte': '|lte',
  cidr: '|cidr',
  '|cidr': '|cidr',
  exists: '|exists',
  '|exists': '|exists',
  all: '|all',
  '|all': '|all',
  base64: '|base64',
  '|base64': '|base64',
};

const SUFFIX_TO_OPERATOR: Record<string, string> = {
  '': 'eq',
  eq: 'eq',
  in: 'in',
  contains: 'contains',
  startswith: 'startswith',
  endswith: 'endswith',
  re: 're',
  gt: 'gt',
  gte: 'gte',
  lt: 'lt',
  lte: 'lte',
  cidr: 'cidr',
  exists: 'exists',
  all: 'all',
  base64: 'base64',
};

export function getRuleTypeLabel(value: string): string {
  return DETECTION_RULE_TYPE_OPTIONS.find((option) => option.value === value)?.label ?? slugToTitle(value);
}

export function getRuleTypeColor(value: string): string {
  const palette: Record<string, string> = {
    sigma: 'bg-sky-100 text-sky-800',
    threshold: 'bg-emerald-100 text-emerald-800',
    correlation: 'bg-amber-100 text-amber-900',
    anomaly: 'bg-orange-100 text-orange-800',
  };
  return palette[value] ?? 'bg-slate-100 text-slate-800';
}

export function normalizeRule(rule: DetectionRule): DetectionRule {
  const type = (rule.rule_type ?? rule.type ?? 'sigma') as DetectionRuleType;
  const truePositiveCount = rule.true_positive_count ?? rule.tp_count ?? 0;
  const falsePositiveCount = rule.false_positive_count ?? rule.fp_count ?? 0;
  const totalFeedback = truePositiveCount + falsePositiveCount;
  const falsePositiveRate = rule.false_positive_rate ?? (totalFeedback > 0 ? falsePositiveCount / totalFeedback : 0);
  const truePositiveRate = rule.true_positive_rate ?? (totalFeedback > 0 ? truePositiveCount / totalFeedback : 0);

  return {
    ...rule,
    rule_type: type,
    type,
    true_positive_count: truePositiveCount,
    false_positive_count: falsePositiveCount,
    false_positive_rate: falsePositiveRate,
    true_positive_rate: truePositiveRate,
    last_triggered_at: rule.last_triggered_at ?? rule.last_triggered,
    last_triggered: rule.last_triggered ?? rule.last_triggered_at,
    tp_count: truePositiveCount,
    fp_count: falsePositiveCount,
    rule_content: normalizeRuleContent(type, rule.rule_content),
  };
}

export function normalizeRuleTemplate(template: RuleTemplate): RuleTemplate {
  const type = (template.rule_type ?? template.type ?? 'sigma') as DetectionRuleType;
  return {
    ...template,
    rule_type: type,
    type,
  };
}

export function normalizeRuleList(items: DetectionRule[]): DetectionRule[] {
  return items.map(normalizeRule);
}

export function buildRuleQueryParams(params: FetchParams): Record<string, unknown> {
  return {
    page: params.page,
    per_page: params.per_page,
    sort: params.sort,
    order: params.order,
    search: params.search,
    ...(params.filters ?? {}),
  };
}

export function parseSigmaYamlText(text: string): Record<string, unknown> {
  const parsed = parse(text) as Record<string, unknown> | null;
  if (!parsed || typeof parsed !== 'object') {
    throw new Error('Sigma YAML must define an object');
  }
  return parsed;
}

export function stringifySigmaContent(content: unknown): string {
  return stringify(content ?? { title: 'New Sigma Rule', detection: { selection_1: {}, condition: 'selection_1' } });
}

export function normalizeRuleContent(
  ruleType: DetectionRuleType,
  content: unknown,
): SigmaRuleContent | ThresholdRuleContent | CorrelationRuleContent | AnomalyRuleContent | Record<string, unknown> {
  if (!content || typeof content !== 'object') {
    return defaultContentByType(ruleType);
  }
  if (ruleType === 'sigma') {
    return sigmaRuleFromBackend(content as Record<string, unknown>);
  }
  if (ruleType === 'threshold') {
    return thresholdRuleFromBackend(content as Record<string, unknown>);
  }
  if (ruleType === 'correlation') {
    return correlationRuleFromBackend(content as Record<string, unknown>);
  }
  if (ruleType === 'anomaly') {
    return anomalyRuleFromBackend(content as Record<string, unknown>);
  }
  return content as Record<string, unknown>;
}

export function serializeRuleContent(
  ruleType: DetectionRuleType,
  content: SigmaRuleContent | ThresholdRuleContent | CorrelationRuleContent | AnomalyRuleContent,
): Record<string, unknown> {
  if (ruleType === 'sigma') {
    const sigmaContent = content as SigmaRuleContent;
    const detection: Record<string, unknown> = {};
    sigmaContent.selections.forEach((selection) => {
      detection[selection.name] = buildSelectionObject(selection.conditions);
    });
    (sigmaContent.filters ?? []).forEach((selection) => {
      detection[selection.name] = buildSelectionObject(selection.conditions);
    });
    detection.condition = sigmaContent.condition;

    return {
      title: 'Custom Sigma Rule',
      detection,
      timeframe: sigmaContent.timeframe,
      threshold: sigmaContent.threshold,
    };
  }

  if (ruleType === 'threshold') {
    const thresholdContent = content as ThresholdRuleContent;
    const metric =
      thresholdContent.metric === 'count'
        ? 'count'
        : `${thresholdContent.metric}(${thresholdContent.metric_field ?? thresholdContent.group_by ?? 'source_ip'})`;

    return {
      field: thresholdContent.group_by ?? 'source_ip',
      condition: buildSelectionObject(thresholdContent.filter_conditions),
      threshold: thresholdContent.threshold,
      window: thresholdContent.window,
      metric,
    };
  }

  if (ruleType === 'correlation') {
    const correlationContent = content as CorrelationRuleContent;
    return {
      events: correlationContent.event_types.map((eventType) => ({
        name: eventType.name,
        condition: buildSelectionObject(eventType.conditions),
      })),
      sequence: correlationContent.sequence,
      group_by: correlationContent.group_by ?? 'source_ip',
      window: correlationContent.window,
      min_failed_count: correlationContent.min_failed_count ?? 0,
    };
  }

  return { ...(content as AnomalyRuleContent) };
}

export function buildSelectionObject(conditions: RuleCondition[]): Record<string, unknown> {
  return conditions.reduce<Record<string, unknown>>((acc, condition) => {
    const field = condition.field.trim();
    if (!field) {
      return acc;
    }
    const suffix = OPERATOR_TO_SUFFIX[condition.operator] ?? '';
    acc[`${field}${suffix}`] = parseConditionValue(condition.operator, condition.value);
    return acc;
  }, {});
}

export function parseSelectionObject(selection: Record<string, unknown>): RuleCondition[] {
  return Object.entries(selection).map(([rawKey, rawValue]) => {
    const [field, rawOperator = ''] = rawKey.split('|');
    return {
      field,
      operator: SUFFIX_TO_OPERATOR[rawOperator] ?? 'eq',
      value: stringifyConditionValue(rawValue),
    };
  });
}

export function sigmaRuleFromBackend(content: Record<string, unknown>): SigmaRuleContent {
  if ('selections' in content) {
    return content as unknown as SigmaRuleContent;
  }

  const detection = (content.detection as Record<string, unknown> | undefined) ?? {};
  const selections: SigmaRuleContent['selections'] = [];
  const filters: SigmaRuleContent['filters'] = [];

  Object.entries(detection).forEach(([name, value]) => {
    if (name === 'condition' || !value || typeof value !== 'object') {
      return;
    }
    const entry = {
      name,
      conditions: parseSelectionObject(value as Record<string, unknown>),
    };
    if (name.toLowerCase().includes('filter')) {
      filters.push(entry);
    } else {
      selections.push(entry);
    }
  });

  return {
    selections: selections.length > 0 ? selections : defaultSigmaContent().selections,
    filters,
    condition: typeof detection.condition === 'string' ? detection.condition : 'selection_1',
    timeframe: typeof content.timeframe === 'string' ? content.timeframe : undefined,
    threshold: typeof content.threshold === 'number' ? content.threshold : undefined,
  };
}

export function thresholdRuleFromBackend(content: Record<string, unknown>): ThresholdRuleContent {
  if ('filter_conditions' in content) {
    return content as unknown as ThresholdRuleContent;
  }
  const metricRaw = typeof content.metric === 'string' ? content.metric : 'count';
  const metricMatch = metricRaw.match(/^(sum|distinct)\((.+)\)$/);
  return {
    filter_conditions: parseSelectionObject((content.condition as Record<string, unknown>) ?? {}),
    group_by: typeof content.field === 'string' ? content.field : 'source_ip',
    metric: metricMatch ? (metricMatch[1] as ThresholdRuleContent['metric']) : 'count',
    metric_field: metricMatch ? metricMatch[2] : undefined,
    threshold: typeof content.threshold === 'number' ? content.threshold : 5,
    window: typeof content.window === 'string' ? content.window : '5m',
  };
}

export function correlationRuleFromBackend(content: Record<string, unknown>): CorrelationRuleContent {
  if ('event_types' in content) {
    return content as unknown as CorrelationRuleContent;
  }
  const events = Array.isArray(content.events) ? content.events : [];
  return {
    event_types: events.map((eventType) => ({
      name: typeof eventType?.name === 'string' ? eventType.name : 'event_1',
      conditions: parseSelectionObject((eventType?.condition as Record<string, unknown>) ?? {}),
    })),
    sequence: Array.isArray(content.sequence) ? (content.sequence as string[]) : [],
    group_by: typeof content.group_by === 'string' ? content.group_by : 'source_ip',
    window: typeof content.window === 'string' ? content.window : '10m',
    min_failed_count: typeof content.min_failed_count === 'number' ? content.min_failed_count : undefined,
  };
}

export function anomalyRuleFromBackend(content: Record<string, unknown>): AnomalyRuleContent {
  if ('metric' in content && 'window' in content && 'z_score_threshold' in content) {
    return {
      metric: typeof content.metric === 'string' ? content.metric : 'event_count',
      group_by: typeof content.group_by === 'string' ? content.group_by : undefined,
      window: typeof content.window === 'string' ? content.window : '1h',
      z_score_threshold: typeof content.z_score_threshold === 'number' ? content.z_score_threshold : 3,
      min_baseline_samples: typeof content.min_baseline_samples === 'number' ? content.min_baseline_samples : 100,
      direction:
        content.direction === 'below' || content.direction === 'both' ? content.direction : 'above',
    };
  }
  return defaultAnomalyContent();
}

export function defaultSigmaContent(): SigmaRuleContent {
  return {
    selections: [{ name: 'selection_1', conditions: [{ field: 'type', operator: 'eq', value: '' }] }],
    filters: [],
    condition: 'selection_1',
  };
}

export function defaultThresholdContent(): ThresholdRuleContent {
  return {
    filter_conditions: [{ field: 'type', operator: 'eq', value: '' }],
    group_by: 'source_ip',
    metric: 'count',
    threshold: 5,
    window: '5m',
  };
}

export function defaultCorrelationContent(): CorrelationRuleContent {
  return {
    event_types: [
      { name: 'event_a', conditions: [{ field: 'type', operator: 'eq', value: '' }] },
      { name: 'event_b', conditions: [{ field: 'type', operator: 'eq', value: '' }] },
    ],
    sequence: ['event_a', 'event_b'],
    group_by: 'source_ip',
    window: '10m',
  };
}

export function defaultAnomalyContent(): AnomalyRuleContent {
  return {
    metric: 'event_count',
    group_by: 'source_ip',
    window: '1h',
    z_score_threshold: 3,
    min_baseline_samples: 100,
    direction: 'above',
  };
}

export function defaultContentByType(ruleType: DetectionRuleType) {
  if (ruleType === 'threshold') {
    return defaultThresholdContent();
  }
  if (ruleType === 'correlation') {
    return defaultCorrelationContent();
  }
  if (ruleType === 'anomaly') {
    return defaultAnomalyContent();
  }
  return defaultSigmaContent();
}

export function getTechniqueStateLabel(technique: MITRETechniqueCoverage): string {
  return {
    covered: 'Covered',
    noisy: 'Covered, noisy',
    gap: 'Gap',
    idle: 'Not covered',
  }[technique.coverage_state];
}

function parseConditionValue(operator: string, value: string): unknown {
  if (operator === 'in' || operator === 'all' || operator === '|in') {
    return value
      .split(',')
      .map((item) => item.trim())
      .filter(Boolean)
      .map(parseScalarValue);
  }
  if (operator === 'exists' || operator === '|exists') {
    return value.trim().toLowerCase() === 'true';
  }
  return parseScalarValue(value);
}

function parseScalarValue(value: string): unknown {
  const trimmed = value.trim();
  if (trimmed === '') {
    return '';
  }
  if (trimmed === 'true') {
    return true;
  }
  if (trimmed === 'false') {
    return false;
  }
  if (!Number.isNaN(Number(trimmed)) && trimmed !== '') {
    return Number(trimmed);
  }
  return trimmed;
}

function stringifyConditionValue(value: unknown): string {
  if (Array.isArray(value)) {
    return value.join(', ');
  }
  if (typeof value === 'boolean') {
    return value ? 'true' : 'false';
  }
  if (value === null || value === undefined) {
    return '';
  }
  return String(value);
}
