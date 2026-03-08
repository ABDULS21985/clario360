import type { Metadata } from 'next';

export const metadata: Metadata = {
  title: 'Setup — Clario 360',
};

const WIZARD_STEPS = [
  { number: 1, label: 'Organization' },
  { number: 2, label: 'Branding' },
  { number: 3, label: 'Team' },
  { number: 4, label: 'Suites' },
  { number: 5, label: 'Ready' },
] as const;

export default function OnboardingLayout({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex min-h-screen flex-col bg-[radial-gradient(circle_at_top,_rgba(15,81,50,0.09),_transparent_38%),linear-gradient(180deg,_#f8faf7_0%,_#f3f7f4_48%,_#eef4f0_100%)]">
      <header className="border-b border-[#0f5132]/10 bg-white/85 px-6 py-6 backdrop-blur">
        <div className="mx-auto flex w-full max-w-5xl flex-col items-center gap-5">
          <div className="flex items-center gap-3">
            <div className="flex h-10 w-10 items-center justify-center rounded-2xl bg-[#0f5132] text-sm font-bold text-white shadow-sm">
              C
            </div>
            <div className="text-center">
              <p className="text-base font-semibold text-slate-900">Clario 360</p>
              <p className="text-xs uppercase tracking-[0.25em] text-slate-500">Tenant Onboarding</p>
            </div>
          </div>

          <nav className="hidden items-center gap-0 sm:flex" aria-label="Onboarding steps">
            {WIZARD_STEPS.map((step, idx) => (
              <div key={step.number} className="flex items-center">
                <div className="flex flex-col items-center">
                  <div className="flex h-8 w-8 items-center justify-center rounded-full border border-[#0f5132]/20 bg-white text-xs font-semibold text-slate-600">
                    {step.number}
                  </div>
                  <span className="mt-1 text-[10px] uppercase tracking-[0.2em] text-slate-500">
                    {step.label}
                  </span>
                </div>
                {idx < WIZARD_STEPS.length - 1 && (
                  <div className="mx-3 h-px w-10 bg-[#0f5132]/15" />
                )}
              </div>
            ))}
          </nav>
        </div>
      </header>

      <main className="flex flex-1 flex-col items-center px-4 py-10">
        {children}
      </main>

      <footer className="pb-8 text-center text-sm text-slate-500">
        Need help? <a href="mailto:support@clario360.com" className="font-medium text-[#0f5132] hover:underline">support@clario360.com</a>
      </footer>
    </div>
  );
}
