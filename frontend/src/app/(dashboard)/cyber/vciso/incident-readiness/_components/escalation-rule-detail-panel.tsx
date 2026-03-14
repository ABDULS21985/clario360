'use client';

import {
  Zap,
  Target,
  Bell,
  Calendar,
  Clock,
  Hash,
  User,
} from 'lucide-react';
import { DetailPanel } from '@/components/shared/detail-panel';
import { Badge } from '@/components/ui/badge';
import { Separator } from '@/components/ui/separator';
import { formatDate } from '@/lib/format';
import { titleCase } from '@/lib/format';
import type { VCISOEscalationRule } from '@/types/cyber';

interface EscalationRuleDetailPanelProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  rule: VCISOEscalationRule;
}

export function EscalationRuleDetailPanel({
  open,
  onOpenChange,
  rule,
}: EscalationRuleDetailPanelProps) {
  return (
    <DetailPanel
      open={open}
      onOpenChange={onOpenChange}
      title={rule.name}
      description="Escalation Rule Details"
      width="xl"
    >
      <div className="space-y-6">
        {/* Overview */}
        <div className="space-y-3">
          <h3 className="text-sm font-semibold text-muted-foreground uppercase tracking-wider">
            Overview
          </h3>
          <p className="text-sm text-foreground leading-relaxed">
            {rule.description || 'No description provided.'}
          </p>
          <div className="flex items-center gap-2">
            <Badge variant={rule.enabled ? 'default' : 'secondary'}>
              {rule.enabled ? 'Enabled' : 'Disabled'}
            </Badge>
          </div>
        </div>

        <Separator />

        {/* Trigger Configuration */}
        <div className="space-y-3">
          <h3 className="text-sm font-semibold text-muted-foreground uppercase tracking-wider">
            Trigger Configuration
          </h3>
          <div className="space-y-2">
            <div className="flex items-center gap-2 text-sm">
              <Zap className="h-4 w-4 text-muted-foreground" />
              <span className="text-muted-foreground">Trigger Type:</span>
              <Badge variant="outline">{titleCase(rule.trigger_type)}</Badge>
            </div>
            <div className="flex items-start gap-2 text-sm">
              <Hash className="h-4 w-4 text-muted-foreground mt-0.5" />
              <span className="text-muted-foreground">Condition:</span>
              <span className="font-mono text-xs bg-muted px-2 py-1 rounded">
                {rule.trigger_condition}
              </span>
            </div>
          </div>
        </div>

        <Separator />

        {/* Escalation Target */}
        <div className="space-y-3">
          <h3 className="text-sm font-semibold text-muted-foreground uppercase tracking-wider">
            Escalation Target
          </h3>
          <div className="flex items-center gap-2 text-sm">
            <Target className="h-4 w-4 text-muted-foreground" />
            <span className="text-muted-foreground">Target:</span>
            <Badge variant="outline">{titleCase(rule.escalation_target)}</Badge>
          </div>
        </div>

        {/* Target Contacts */}
        {rule.target_contacts.length > 0 && (
          <>
            <Separator />
            <div className="space-y-3">
              <h3 className="text-sm font-semibold text-muted-foreground uppercase tracking-wider">
                Target Contacts
              </h3>
              <div className="flex flex-wrap gap-1.5">
                {rule.target_contacts.map((contact) => (
                  <Badge key={contact} variant="secondary" className="text-xs">
                    <User className="mr-1 h-3 w-3" />
                    {contact}
                  </Badge>
                ))}
              </div>
            </div>
          </>
        )}

        {/* Notification Channels */}
        {rule.notification_channels.length > 0 && (
          <>
            <Separator />
            <div className="space-y-3">
              <h3 className="text-sm font-semibold text-muted-foreground uppercase tracking-wider">
                Notification Channels
              </h3>
              <div className="flex flex-wrap gap-1.5">
                {rule.notification_channels.map((channel) => (
                  <Badge key={channel} variant="outline" className="text-xs capitalize">
                    <Bell className="mr-1 h-3 w-3" />
                    {channel}
                  </Badge>
                ))}
              </div>
            </div>
          </>
        )}

        <Separator />

        {/* Trigger History */}
        <div className="space-y-3">
          <h3 className="text-sm font-semibold text-muted-foreground uppercase tracking-wider">
            Trigger History
          </h3>
          <div className="space-y-2">
            <div className="flex items-center gap-2 text-sm">
              <Hash className="h-4 w-4 text-muted-foreground" />
              <span className="text-muted-foreground">Trigger Count:</span>
              <span className="font-semibold">{rule.trigger_count}</span>
            </div>
            <div className="flex items-center gap-2 text-sm">
              <Clock className="h-4 w-4 text-muted-foreground" />
              <span className="text-muted-foreground">Last Triggered:</span>
              <span className="font-medium">
                {rule.last_triggered_at ? formatDate(rule.last_triggered_at) : 'Never'}
              </span>
            </div>
          </div>
        </div>

        <Separator />

        {/* Timestamps */}
        <div className="space-y-3">
          <h3 className="text-sm font-semibold text-muted-foreground uppercase tracking-wider">
            Timestamps
          </h3>
          <div className="space-y-2">
            <div className="flex items-center gap-2 text-sm">
              <Calendar className="h-4 w-4 text-muted-foreground" />
              <span className="text-muted-foreground">Created:</span>
              <span className="font-medium">{formatDate(rule.created_at)}</span>
            </div>
            <div className="flex items-center gap-2 text-sm">
              <Calendar className="h-4 w-4 text-muted-foreground" />
              <span className="text-muted-foreground">Updated:</span>
              <span className="font-medium">{formatDate(rule.updated_at)}</span>
            </div>
          </div>
        </div>
      </div>
    </DetailPanel>
  );
}
