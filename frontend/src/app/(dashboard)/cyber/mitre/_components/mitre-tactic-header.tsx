'use client';

import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip';

interface MitreTacticHeaderProps {
  id: string;
  name: string;
  shortName?: string;
  covered: number;
  total: number;
}

function abbreviate(name: string): string {
  const MAP: Record<string, string> = {
    Reconnaissance: 'Recon',
    'Resource Development': 'Rsrc Dev',
    'Initial Access': 'Init Accs',
    Execution: 'Execution',
    Persistence: 'Persist.',
    'Privilege Escalation': 'Priv Esc',
    'Defense Evasion': 'Def Evas',
    'Credential Access': 'Cred Accs',
    Discovery: 'Discovery',
    'Lateral Movement': 'Lat Move',
    Collection: 'Collect.',
    'Command and Control': 'C2',
    Exfiltration: 'Exfil',
    Impact: 'Impact',
  };
  return MAP[name] ?? name.slice(0, 8);
}

export function MitreTacticHeader({ id, name, shortName, covered, total }: MitreTacticHeaderProps) {
  const pct = total > 0 ? Math.round((covered / total) * 100) : 0;
  const display = shortName ?? abbreviate(name);
  const barColor =
    pct >= 80 ? 'bg-green-500' : pct >= 50 ? 'bg-yellow-400' : pct > 0 ? 'bg-orange-400' : 'bg-red-300';

  return (
    <TooltipProvider>
      <Tooltip>
        <TooltipTrigger asChild>
          <div className="flex flex-col gap-1 p-2">
            <span className="block truncate text-xs font-bold text-foreground" title={name}>
              {display}
            </span>
            <span className="block font-mono text-[10px] text-muted-foreground">{id}</span>
            <div className="flex items-center gap-1.5">
              <div className="h-1 flex-1 overflow-hidden rounded-full bg-muted">
                <div className={`h-full rounded-full transition-all ${barColor}`} style={{ width: `${pct}%` }} />
              </div>
              <span className="shrink-0 text-[10px] tabular-nums text-muted-foreground">
                {covered}/{total}
              </span>
            </div>
          </div>
        </TooltipTrigger>
        <TooltipContent>
          <p className="font-semibold">{name}</p>
          <p className="text-xs text-muted-foreground">
            {covered} of {total} techniques covered ({pct}%)
          </p>
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
}
