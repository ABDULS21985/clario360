import { differenceInCalendarDays, parseISO } from 'date-fns';
import type {
  ActaAttendee,
  ActaAgendaItem,
  ActaCommittee,
  ActaCommitteeMember,
  ActaMeeting,
  ActaMeetingMinutes,
  JsonValue,
  LexClause,
  UserDirectoryEntry,
  VisusDashboard,
  VisusWidget,
  VisusWidgetPosition,
} from '@/types/suites';

export function userDisplayName(user: Pick<UserDirectoryEntry, 'first_name' | 'last_name' | 'email'>): string {
  const fullName = `${user.first_name} ${user.last_name}`.trim();
  return fullName || user.email;
}

export function committeeMemberName(member: Pick<ActaCommitteeMember, 'user_name' | 'user_email'>): string {
  return member.user_name || member.user_email;
}

export function attendeeCounts(attendance: ActaAttendee[]): {
  present: number;
  proxy: number;
  absent: number;
  excused: number;
  total: number;
  countedForQuorum: number;
} {
  const summary = {
    present: 0,
    proxy: 0,
    absent: 0,
    excused: 0,
    total: attendance.length,
    countedForQuorum: 0,
  };

  for (const attendee of attendance) {
    if (attendee.status === 'present') {
      summary.present += 1;
      summary.countedForQuorum += 1;
    } else if (attendee.status === 'proxy') {
      summary.proxy += 1;
      summary.countedForQuorum += 1;
    } else if (attendee.status === 'absent') {
      summary.absent += 1;
    } else if (attendee.status === 'excused') {
      summary.excused += 1;
    }
  }

  return summary;
}

export function quorumProgress(attendance: ActaAttendee[], quorumRequired: number): {
  counted: number;
  percent: number;
  met: boolean;
} {
  const counted = attendeeCounts(attendance).countedForQuorum;
  const total = Math.max(attendance.length, 1);
  return {
    counted,
    percent: Math.round((counted / total) * 100),
    met: counted >= quorumRequired,
  };
}

export function isCommitteeChair(committee: ActaCommittee | null | undefined, userId: string | null | undefined): boolean {
  return Boolean(committee && userId && committee.chair_user_id === userId);
}

export function isMeetingSecretary(committee: ActaCommittee | null | undefined, userId: string | null | undefined): boolean {
  return Boolean(committee && userId && committee.secretary_user_id === userId);
}

export function canApproveMinutes(
  minutes: ActaMeetingMinutes | null | undefined,
  committee: ActaCommittee | null | undefined,
  userId: string | null | undefined,
): boolean {
  return Boolean(minutes && minutes.status === 'review' && isCommitteeChair(committee, userId));
}

export function calculateVoteOutcome(
  voteType: 'unanimous' | 'majority' | 'two_thirds' | 'roll_call',
  votesFor: number,
  votesAgainst: number,
  votesAbstained: number,
): {
  label: string;
  result: 'approved' | 'rejected' | 'tied' | 'deferred' | null;
  valid: boolean;
} {
  const votingBase = votesFor + votesAgainst;
  if (votesFor === votesAgainst && votingBase > 0) {
    return { label: 'Tied', result: 'tied', valid: true };
  }

  switch (voteType) {
    case 'unanimous':
      if (votesAgainst === 0 && votesAbstained === 0) {
        return { label: 'Approved', result: 'approved', valid: true };
      }
      return { label: 'Not unanimous', result: 'rejected', valid: false };
    case 'majority':
      if (votesFor > votesAgainst) {
        return { label: 'Approved', result: 'approved', valid: true };
      }
      return { label: 'Rejected', result: 'rejected', valid: true };
    case 'two_thirds':
      if (votingBase > 0 && votesFor / votingBase >= 2 / 3) {
        return { label: 'Approved', result: 'approved', valid: true };
      }
      return { label: 'Rejected', result: 'rejected', valid: true };
    case 'roll_call':
      return { label: 'Recorded', result: null, valid: true };
    default:
      return { label: 'Recorded', result: null, valid: true };
  }
}

export function agendaVoteSummary(item: ActaAgendaItem): string | null {
  if (!item.requires_vote || !item.vote_type) {
    return null;
  }
  const votesFor = item.votes_for ?? 0;
  const votesAgainst = item.votes_against ?? 0;
  const abstained = item.votes_abstained ?? 0;
  return `${item.vote_type.replace(/_/g, ' ')} • For ${votesFor}, Against ${votesAgainst}, Abstained ${abstained}`;
}

export function daysUntil(dateValue: string): number {
  return differenceInCalendarDays(parseISO(dateValue), new Date());
}

export function daysOverdue(dateValue: string): number {
  return Math.max(0, differenceInCalendarDays(new Date(), parseISO(dateValue)));
}

export function actaMonthString(date: Date): string {
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, '0');
  return `${year}-${month}`;
}

export function scoreTone(score: number): 'success' | 'warning' | 'destructive' {
  if (score >= 90) {
    return 'success';
  }
  if (score >= 70) {
    return 'warning';
  }
  return 'destructive';
}

export function clauseHighlightSegments(content: string, keywords: string[]): Array<{ text: string; highlighted: boolean }> {
  if (!content || keywords.length === 0) {
    return [{ text: content, highlighted: false }];
  }

  const normalizedKeywords = keywords
    .map((keyword) => keyword.trim())
    .filter((keyword) => keyword.length > 0)
    .sort((left, right) => right.length - left.length);

  const pattern = new RegExp(`(${normalizedKeywords.map(escapeRegExp).join('|')})`, 'gi');
  const parts = content.split(pattern);
  return parts
    .filter((part) => part.length > 0)
    .map((part) => ({
      text: part,
      highlighted: normalizedKeywords.some((keyword) => keyword.toLowerCase() === part.toLowerCase()),
    }));
}

export function nextWidgetPosition(dashboard: VisusDashboard | null | undefined, width = 4, height = 3): VisusWidgetPosition {
  const widgets = dashboard?.widgets ?? [];
  if (widgets.length === 0) {
    return { x: 0, y: 0, w: width, h: height };
  }

  const ordered = [...widgets].sort((left, right) => {
    if (left.position.y === right.position.y) {
      return left.position.x - right.position.x;
    }
    return left.position.y - right.position.y;
  });

  let y = 0;
  while (y < 500) {
    for (let x = 0; x <= 12 - width; x += 1) {
      const candidate = { x, y, w: width, h: height };
      const overlaps = ordered.some((widget) => boxesOverlap(candidate, widget.position));
      if (!overlaps) {
        return candidate;
      }
    }
    y += 1;
  }

  return {
    x: 0,
    y: (ordered[ordered.length - 1]?.position.y ?? 0) + height,
    w: width,
    h: height,
  };
}

export function widgetTitleFromConfig(type: VisusWidget['type'], config: Record<string, JsonValue>): string {
  if (type === 'text') {
    return 'Executive Note';
  }
  if ('title' in config && typeof config.title === 'string' && config.title.trim().length > 0) {
    return config.title.trim();
  }
  if ('metric' in config && typeof config.metric === 'string') {
    return config.metric;
  }
  if ('kpi_name' in config && typeof config.kpi_name === 'string') {
    return config.kpi_name;
  }
  return type.replace(/_/g, ' ').replace(/\b\w/g, (match) => match.toUpperCase());
}

export function sortWidgetsByLayout(widgets: VisusWidget[]): VisusWidget[] {
  return [...widgets].sort((left, right) => {
    if (left.position.y === right.position.y) {
      return left.position.x - right.position.x;
    }
    return left.position.y - right.position.y;
  });
}

export function safeJsonPreview(value: unknown): string {
  try {
    return JSON.stringify(value, null, 2);
  } catch {
    return '{}';
  }
}

export function contractDaysRemaining(expiryDate?: string | null): number | null {
  if (!expiryDate) {
    return null;
  }
  return daysUntil(expiryDate);
}

export function clauseRiskSummary(clause: LexClause): string {
  const section = clause.section_reference ? `Section ${clause.section_reference}` : 'Unnumbered section';
  return `${section} • ${clause.risk_level.replace(/_/g, ' ')} risk`;
}

function boxesOverlap(a: VisusWidgetPosition, b: VisusWidgetPosition): boolean {
  return a.x < b.x + b.w && a.x + a.w > b.x && a.y < b.y + b.h && a.y + a.h > b.y;
}

function escapeRegExp(value: string): string {
  return value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
}
