import type { Metadata } from 'next';
import {
  Activity,
  ArrowUpRight,
  BarChart3,
  CheckCircle2,
  Clock3,
  ShieldCheck,
  Sparkles,
  Workflow,
  type LucideIcon,
} from 'lucide-react';

import { cn } from '@/lib/utils';

export const metadata: Metadata = {
  title: 'Clario 360 — Sign In',
};

const PLATFORM_KPIS: Array<{
  icon: LucideIcon;
  label: string;
  value: string;
  detail: string;
  accent: string;
}> = [
  {
    icon: Clock3,
    label: 'Median triage',
    value: '4m 12s',
    detail: 'from detection to assignment',
    accent: 'from-emerald-400/20 to-emerald-500/5',
  },
  {
    icon: Activity,
    label: 'Live telemetry',
    value: '1.2M',
    detail: 'signals normalized today',
    accent: 'from-cyan-400/20 to-cyan-500/5',
  },
  {
    icon: Workflow,
    label: 'Automation coverage',
    value: '84%',
    detail: 'repeatable tasks orchestrated',
    accent: 'from-amber-300/20 to-amber-400/5',
  },
];

const OPERATIONAL_PULSE = [
  { label: 'Critical', value: '12', tone: 'bg-rose-400/90 text-rose-50' },
  { label: 'High', value: '28', tone: 'bg-orange-400/90 text-orange-950' },
  { label: 'In progress', value: '41', tone: 'bg-emerald-400/90 text-emerald-950' },
  { label: 'Resolved today', value: '96', tone: 'bg-cyan-300/90 text-cyan-950' },
] as const;

const EXECUTION_STREAMS = [
  { label: 'Alert triage automation', value: 82, tone: 'bg-emerald-300' },
  { label: 'Evidence enrichment', value: 68, tone: 'bg-cyan-300' },
  { label: 'Governance workflow readiness', value: 91, tone: 'bg-amber-300' },
] as const;

const COMMAND_FEED = [
  { title: 'UEBA anomaly cluster promoted', detail: '7 correlated identities moved to analyst review', tone: 'bg-emerald-400' },
  { title: 'Data-quality drift stabilized', detail: 'Automated remediation closed three failing models', tone: 'bg-cyan-400' },
  { title: 'Board pack regenerated', detail: 'Executive dashboards refreshed across active suites', tone: 'bg-amber-300' },
] as const;

const ACTIVE_SUITES = ['Cyber', 'Data', 'Acta', 'Lex', 'Visus'] as const;

function PlatformKpiCard({
  icon: Icon,
  label,
  value,
  detail,
  accent,
}: {
  icon: LucideIcon;
  label: string;
  value: string;
  detail: string;
  accent: string;
}) {
  return (
    <div
      className={cn(
        'rounded-[24px] border border-white/10 bg-white/[0.08] p-4 backdrop-blur-sm',
        'shadow-[0_24px_70px_rgba(3,10,18,0.22)]',
        `bg-gradient-to-br ${accent}`,
      )}
    >
      <div className="flex items-start justify-between gap-3">
        <div>
          <p className="text-[11px] uppercase tracking-[0.28em] text-white/60">{label}</p>
          <p className="mt-3 text-3xl font-semibold tracking-tight text-white">{value}</p>
        </div>
        <div className="rounded-2xl border border-white/10 bg-white/10 p-2.5">
          <Icon className="h-5 w-5 text-white/[0.85]" />
        </div>
      </div>
      <p className="mt-3 text-sm leading-6 text-white/70">{detail}</p>
    </div>
  );
}

export default function AuthLayout({ children }: { children: React.ReactNode }) {
  return (
    <div className="relative min-h-screen overflow-hidden bg-[#041416]">
      <div className="absolute inset-0 bg-[radial-gradient(circle_at_top_left,_rgba(15,81,50,0.28),_transparent_32%),radial-gradient(circle_at_bottom_right,_rgba(198,169,98,0.14),_transparent_24%),linear-gradient(135deg,_#041416_0%,_#072125_44%,_#031114_100%)]" />
      <div className="absolute inset-0 opacity-20 [background-image:linear-gradient(rgba(255,255,255,0.06)_1px,transparent_1px),linear-gradient(90deg,rgba(255,255,255,0.06)_1px,transparent_1px)] [background-size:72px_72px]" />

      <div className="relative min-h-screen p-4 sm:p-6">
        <div className="grid min-h-[calc(100vh-2rem)] overflow-hidden rounded-[32px] border border-white/10 bg-white/[0.04] shadow-[0_40px_140px_rgba(2,12,27,0.55)] backdrop-blur-sm lg:grid-cols-[1.18fr_0.82fr]">
          <section className="relative overflow-hidden border-b border-white/10 p-6 text-white sm:p-8 lg:border-b-0 lg:border-r lg:p-10">
            <div className="absolute inset-0 bg-[radial-gradient(circle_at_top,_rgba(255,255,255,0.08),_transparent_34%),linear-gradient(180deg,_rgba(255,255,255,0.04)_0%,_rgba(255,255,255,0.02)_100%)]" />
            <div className="relative flex h-full flex-col">
              <div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
                <div>
                  <div className="inline-flex items-center gap-2 rounded-full border border-white/10 bg-white/[0.08] px-3 py-1 text-[11px] font-medium uppercase tracking-[0.28em] text-white/70">
                    <Sparkles className="h-3.5 w-3.5 text-[#C6A962]" />
                    Unified operations cockpit
                  </div>
                  <div className="mt-5">
                    <h1 className="text-3xl font-semibold tracking-tight sm:text-4xl lg:text-[2.8rem]">
                      Clario 360
                    </h1>
                    <p className="mt-2 text-sm uppercase tracking-[0.34em] text-[#C6A962]">
                      Enterprise command surface
                    </p>
                  </div>
                </div>

                <div className="rounded-[22px] border border-white/10 bg-white/[0.08] px-4 py-3 text-sm backdrop-blur-sm">
                  <p className="text-[11px] uppercase tracking-[0.28em] text-white/[0.55]">
                    Workspace status
                  </p>
                  <div className="mt-2 flex items-center gap-2 font-medium text-white">
                    <span className="h-2.5 w-2.5 rounded-full bg-emerald-400 shadow-[0_0_0_5px_rgba(74,222,128,0.14)]" />
                    Live and synchronized
                  </div>
                </div>
              </div>

              <div className="mt-10 max-w-3xl">
                <h2 className="text-3xl font-semibold leading-tight tracking-tight sm:text-[2.5rem]">
                  Observe threats, workflows, and executive signals from one secure entry point.
                </h2>
                <p className="mt-4 max-w-2xl text-sm leading-7 text-white/[0.68] sm:text-base">
                  The same surface that powers cyber alert handling and portfolio analytics should
                  frame identity too. This sign-in flow now borrows that layered, high-signal
                  dashboard language instead of looking like a detached form.
                </p>
              </div>

              <div className="mt-8 grid gap-3 xl:grid-cols-3">
                {PLATFORM_KPIS.map((item) => (
                  <PlatformKpiCard key={item.label} {...item} />
                ))}
              </div>

              <div className="mt-8 grid gap-4 xl:grid-cols-[1.15fr_0.85fr]">
                <div className="rounded-[28px] border border-white/10 bg-[#071f22]/90 p-5 shadow-[0_30px_70px_rgba(3,10,18,0.22)] backdrop-blur-sm">
                  <div className="flex items-center justify-between gap-3">
                    <div>
                      <p className="text-[11px] uppercase tracking-[0.28em] text-white/[0.55]">
                        Operational pulse
                      </p>
                      <h3 className="mt-2 text-xl font-semibold">Alert pressure and execution rhythm</h3>
                    </div>
                    <div className="rounded-2xl border border-white/10 bg-white/[0.08] p-2.5">
                      <BarChart3 className="h-5 w-5 text-white/80" />
                    </div>
                  </div>

                  <div className="mt-5 flex flex-wrap gap-2">
                    {OPERATIONAL_PULSE.map((item) => (
                      <div
                        key={item.label}
                        className={cn('rounded-full px-3 py-1.5 text-sm font-medium', item.tone)}
                      >
                        <span className="opacity-70">{item.label}</span>
                        <span className="ml-2 font-semibold">{item.value}</span>
                      </div>
                    ))}
                  </div>

                  <div className="mt-6 space-y-4">
                    {EXECUTION_STREAMS.map((stream) => (
                      <div key={stream.label}>
                        <div className="flex items-center justify-between text-sm">
                          <span className="text-white/[0.72]">{stream.label}</span>
                          <span className="font-semibold text-white">{stream.value}%</span>
                        </div>
                        <div className="mt-2 h-2 rounded-full bg-white/10">
                          <div
                            className={cn('h-full rounded-full', stream.tone)}
                            style={{ width: `${stream.value}%` }}
                          />
                        </div>
                      </div>
                    ))}
                  </div>
                </div>

                <div className="rounded-[28px] border border-white/10 bg-white/[0.08] p-5 backdrop-blur-sm">
                  <div className="flex items-center justify-between gap-3">
                    <div>
                      <p className="text-[11px] uppercase tracking-[0.28em] text-white/[0.55]">
                        Command feed
                      </p>
                      <h3 className="mt-2 text-xl font-semibold">Cross-suite movement</h3>
                    </div>
                    <ArrowUpRight className="h-5 w-5 text-white/[0.65]" />
                  </div>

                  <div className="mt-5 space-y-4">
                    {COMMAND_FEED.map((item) => (
                      <div
                        key={item.title}
                        className="rounded-[22px] border border-white/10 bg-black/10 px-4 py-3"
                      >
                        <div className="flex items-start gap-3">
                          <span className={cn('mt-1 h-2.5 w-2.5 rounded-full', item.tone)} />
                          <div>
                            <p className="font-medium text-white">{item.title}</p>
                            <p className="mt-1 text-sm leading-6 text-white/[0.65]">{item.detail}</p>
                          </div>
                        </div>
                      </div>
                    ))}
                  </div>
                </div>
              </div>

              <div className="mt-auto pt-8">
                <div className="flex flex-wrap items-center gap-2">
                  {ACTIVE_SUITES.map((suite) => (
                    <span
                      key={suite}
                      className="rounded-full border border-white/10 bg-white/[0.08] px-3 py-1.5 text-xs font-medium uppercase tracking-[0.22em] text-white/[0.72]"
                    >
                      {suite}
                    </span>
                  ))}
                </div>
                <div className="mt-5 flex items-start gap-3 rounded-[24px] border border-emerald-300/15 bg-emerald-400/[0.08] px-4 py-4 text-sm text-white/[0.74]">
                  <CheckCircle2 className="mt-0.5 h-5 w-5 shrink-0 text-emerald-300" />
                  <p className="leading-6">
                    Identity, telemetry, and workflow context should feel like part of the same
                    product. This shell makes auth look like it belongs to the platform you land in.
                  </p>
                </div>
              </div>
            </div>
          </section>

          <section className="relative flex flex-col bg-[radial-gradient(circle_at_top_right,_rgba(198,169,98,0.12),_transparent_28%),linear-gradient(180deg,_rgba(248,250,247,0.96)_0%,_rgba(255,255,255,0.98)_100%)] p-4 sm:p-6 lg:p-8">
            <div className="flex items-center justify-between gap-3 text-xs uppercase tracking-[0.24em] text-slate-500">
              <div className="inline-flex items-center gap-2 rounded-full border border-slate-200 bg-white/80 px-3 py-1.5 shadow-sm">
                <ShieldCheck className="h-3.5 w-3.5 text-[#0f5132]" />
                Secure access
              </div>
              <div className="hidden items-center gap-2 lg:flex">
                <span className="h-2 w-2 rounded-full bg-emerald-500" />
                Identity controls active
              </div>
            </div>

            <div className="mx-auto flex w-full max-w-[44rem] flex-1 items-center py-6 lg:py-10">
              <div className="w-full rounded-[30px] border border-slate-200/[0.85] bg-white/95 p-6 shadow-[0_28px_90px_rgba(15,23,42,0.12)] backdrop-blur sm:p-8">
                {children}
              </div>
            </div>

            <div className="flex flex-col gap-3 text-xs text-slate-500 sm:flex-row sm:items-center sm:justify-between">
              <p>&copy; {new Date().getFullYear()} Clario 360. All rights reserved.</p>
              <div className="flex flex-wrap items-center gap-2">
                <span className="rounded-full border border-slate-200 bg-white/80 px-3 py-1">
                  Adaptive MFA
                </span>
                <span className="rounded-full border border-slate-200 bg-white/80 px-3 py-1">
                  Session monitoring
                </span>
                <span className="rounded-full border border-slate-200 bg-white/80 px-3 py-1">
                  Audit logging
                </span>
              </div>
            </div>
          </section>
        </div>
      </div>
    </div>
  );
}
