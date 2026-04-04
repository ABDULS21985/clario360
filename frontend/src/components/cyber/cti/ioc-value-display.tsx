'use client';

import { useMemo, useState } from 'react';
import { Copy, Expand, Globe, Mail, ShieldAlert } from 'lucide-react';
import { toast } from 'sonner';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { countryCodeToFlag } from '@/lib/cti-utils';
import { cn } from '@/lib/utils';

interface IOCValueDisplayProps {
  type?: string | null;
  value?: string | null;
  originCountryCode?: string | null;
  className?: string;
  showCopy?: boolean;
  copyable?: boolean;
}

function isHash(type?: string | null): boolean {
  return type === 'hash_sha256' || type === 'hash_md5';
}

function trunc(value: string, expanded: boolean): string {
  if (expanded || value.length <= 28) {
    return value;
  }
  return `${value.slice(0, 12)}…${value.slice(-12)}`;
}

export function IOCValueDisplay({
  type,
  value,
  originCountryCode,
  className,
  showCopy,
  copyable,
}: IOCValueDisplayProps) {
  const [expanded, setExpanded] = useState(false);
  const allowCopy = showCopy ?? copyable ?? true;

  const tone = useMemo(() => {
    switch (type) {
      case 'domain':
      case 'url':
      case 'email':
        return {
          icon: Globe,
          chip: 'text-rose-300 border-rose-500/30 bg-rose-500/10',
          body: 'text-rose-200',
        };
      case 'ip':
        return {
          icon: Globe,
          chip: 'text-sky-300 border-sky-500/30 bg-sky-500/10',
          body: 'text-sky-100',
        };
      case 'cve':
        return {
          icon: ShieldAlert,
          chip: 'text-amber-300 border-amber-500/30 bg-amber-500/10',
          body: 'text-amber-100',
        };
      default:
        return {
          icon: ShieldAlert,
          chip: 'text-emerald-300 border-emerald-500/30 bg-emerald-500/10',
          body: 'text-foreground',
        };
    }
  }, [type]);

  if (!value) {
    return <span className="text-sm text-muted-foreground">—</span>;
  }

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(value);
      toast.success('IOC copied to clipboard');
    } catch {
      toast.error('Unable to copy IOC');
    }
  };

  const Icon = tone.icon;
  const renderedValue = isHash(type) ? trunc(value, expanded) : value;
  const emailDomain = type === 'email' && value.includes('@') ? value.split('@')[1] : null;

  return (
    <div className={cn('flex items-start gap-3 rounded-2xl border bg-muted/20 p-3', className)}>
      <div className="mt-0.5 rounded-full bg-slate-900/70 p-2 text-slate-100">
        <Icon className="h-4 w-4" />
      </div>
      <div className="min-w-0 flex-1 space-y-1">
        {type && (
          <div className="flex flex-wrap items-center gap-2">
            <Badge variant="outline" className={cn('w-fit text-[10px] uppercase tracking-[0.2em]', tone.chip)}>
              {type.replaceAll('_', ' ')}
            </Badge>
            {type === 'ip' && originCountryCode && (
              <span className="text-xs text-muted-foreground">
                {countryCodeToFlag(originCountryCode)} {originCountryCode.toUpperCase()}
              </span>
            )}
          </div>
        )}
        {type === 'cve' ? (
          <a
            href={`https://nvd.nist.gov/vuln/detail/${encodeURIComponent(value)}`}
            target="_blank"
            rel="noreferrer"
            className={cn('inline-flex items-center gap-1 break-all font-mono text-xs hover:underline', tone.body)}
          >
            {value}
          </a>
        ) : type === 'email' && emailDomain ? (
          <p className={cn('break-all font-mono text-xs', tone.body)}>
            {value.replace(emailDomain, '')}
            <span className="text-amber-200">@{emailDomain}</span>
          </p>
        ) : (
          <p className={cn('break-all font-mono text-xs', tone.body)}>{renderedValue}</p>
        )}
      </div>
      <div className="flex shrink-0 items-center gap-1">
        {isHash(type) && value.length > 28 && (
          <Button
            type="button"
            size="icon"
            variant="ghost"
            className="h-8 w-8"
            onClick={() => setExpanded((current) => !current)}
            aria-label={expanded ? 'Collapse IOC value' : 'Expand IOC value'}
          >
            <Expand className="h-4 w-4" />
          </Button>
        )}
        {allowCopy && (
          <Button
            type="button"
            size="icon"
            variant="ghost"
            className="h-8 w-8"
            onClick={() => void handleCopy()}
            aria-label="Copy IOC value"
          >
            <Copy className="h-4 w-4" />
          </Button>
        )}
      </div>
    </div>
  );
}
