import type { Metadata } from 'next';
import { CheckCircle2 } from 'lucide-react';

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
    <div className="flex min-h-screen flex-col bg-gray-50 dark:bg-gray-950">
      {/* Top bar */}
      <header className="flex h-16 items-center border-b border-gray-200 bg-white px-6 shadow-sm dark:border-gray-800 dark:bg-gray-900">
        <div className="flex items-center gap-3">
          <div className="flex h-8 w-8 items-center justify-center rounded-md bg-[#1B5E20]">
            <span className="text-xs font-bold text-white">C</span>
          </div>
          <span className="text-base font-semibold text-gray-900 dark:text-gray-100">Clario 360</span>
        </div>

        {/* Step indicator */}
        <nav className="mx-auto hidden items-center gap-0 sm:flex" aria-label="Onboarding steps">
          {WIZARD_STEPS.map((step, idx) => (
            <div key={step.number} className="flex items-center">
              <div className="flex flex-col items-center">
                <div className="flex h-7 w-7 items-center justify-center rounded-full border-2 border-gray-300 bg-white text-xs font-medium text-gray-500 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-400">
                  {step.number}
                </div>
                <span className="mt-1 hidden text-[10px] text-gray-500 dark:text-gray-400 lg:block">
                  {step.label}
                </span>
              </div>
              {idx < WIZARD_STEPS.length - 1 && (
                <div className="mx-2 h-px w-8 bg-gray-300 dark:bg-gray-600" />
              )}
            </div>
          ))}
        </nav>

        <div className="ml-auto" />
      </header>

      {/* Content */}
      <main className="flex flex-1 flex-col items-center px-4 py-10">
        {children}
      </main>
    </div>
  );
}
