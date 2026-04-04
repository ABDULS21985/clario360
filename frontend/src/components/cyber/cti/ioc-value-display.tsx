'use client';

import { Copy, ShieldAlert } from 'lucide-react';
import { toast } from 'sonner';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { cn } from '@/lib/utils';

interface IOCValueDisplayProps {
  type?: string | null;
  value?: string | null;
  className?: string;
  copyable?: boolean;
}

export function IOCValueDisplay({
  type,
  value,
  className,
  copyable = true,
}: IOCValueDisplayProps) {
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

  return (
    <div className={cn('flex items-start gap-3 rounded-2xl border bg-muted/20 p-3', className)}>
      <div className="mt-0.5 rounded-full bg-emerald-50 p-2 text-emerald-700">
        <ShieldAlert className="h-4 w-4" />
      </div>
      <div className="min-w-0 flex-1 space-y-1">
        {type && (
          <Badge variant="outline" className="w-fit text-[10px] uppercase tracking-[0.2em]">
            {type.replaceAll('_', ' ')}
          </Badge>
        )}
        <p className="break-all font-mono text-xs text-foreground">{value}</p>
      </div>
      {copyable && (
        <Button
          type="button"
          size="icon"
          variant="ghost"
          className="h-8 w-8 shrink-0"
          onClick={() => void handleCopy()}
          aria-label="Copy IOC value"
        >
          <Copy className="h-4 w-4" />
        </Button>
      )}
    </div>
  );
}