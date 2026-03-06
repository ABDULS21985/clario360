'use client';

import React from 'react';
import { Check } from 'lucide-react';
import { cn } from '@/lib/utils';

interface PasswordRequirement {
  label: string;
  test: (password: string) => boolean;
}

const REQUIREMENTS: PasswordRequirement[] = [
  { label: 'At least 12 characters', test: (p) => p.length >= 12 },
  { label: 'Contains uppercase letter', test: (p) => /[A-Z]/.test(p) },
  { label: 'Contains lowercase letter', test: (p) => /[a-z]/.test(p) },
  { label: 'Contains number', test: (p) => /[0-9]/.test(p) },
  { label: 'Contains special character', test: (p) => /[^a-zA-Z0-9]/.test(p) },
];

type StrengthLevel = 'weak' | 'fair' | 'good' | 'strong';

interface StrengthResult {
  level: StrengthLevel;
  score: number;
  metCount: number;
}

function calculateStrength(password: string): StrengthResult {
  const metCount = REQUIREMENTS.filter((r) => r.test(password)).length;
  let level: StrengthLevel;

  if (metCount <= 1) {
    level = 'weak';
  } else if (metCount <= 3) {
    level = 'fair';
  } else if (metCount === 4) {
    level = 'good';
  } else {
    // All 5 met — strong only if also ≥ 16 chars
    level = password.length >= 16 ? 'strong' : 'good';
  }

  return { level, score: metCount, metCount };
}

const STRENGTH_CONFIG: Record<
  StrengthLevel,
  { label: string; color: string; segments: number }
> = {
  weak: { label: 'Weak', color: 'bg-red-500', segments: 1 },
  fair: { label: 'Fair', color: 'bg-orange-400', segments: 2 },
  good: { label: 'Good', color: 'bg-yellow-400', segments: 3 },
  strong: { label: 'Strong', color: 'bg-green-500', segments: 4 },
};

interface PasswordStrengthMeterProps {
  password: string;
  className?: string;
}

export function PasswordStrengthMeter({ password, className }: PasswordStrengthMeterProps) {
  const { level, metCount } = calculateStrength(password);
  const config = STRENGTH_CONFIG[level];

  if (!password) return null;

  return (
    <div className={cn('space-y-3', className)} aria-live="polite">
      {/* Strength bar */}
      <div className="space-y-1">
        <div className="flex gap-1" role="progressbar" aria-label={`Password strength: ${config.label}`} aria-valuenow={config.segments} aria-valuemin={0} aria-valuemax={4}>
          {[1, 2, 3, 4].map((seg) => (
            <div
              key={seg}
              className={cn(
                'h-1.5 flex-1 rounded-full transition-all duration-300',
                seg <= config.segments ? config.color : 'bg-muted',
              )}
            />
          ))}
        </div>
        <p className={cn('text-xs font-medium', {
          'text-red-500': level === 'weak',
          'text-orange-500': level === 'fair',
          'text-yellow-600': level === 'good',
          'text-green-600': level === 'strong',
        })}>
          {config.label}
        </p>
      </div>

      {/* Requirements checklist */}
      <ul className="space-y-1" aria-label="Password requirements">
        {REQUIREMENTS.map((req) => {
          const met = req.test(password);
          return (
            <li key={req.label} className="flex items-center gap-2 text-xs">
              <span
                className={cn(
                  'flex h-4 w-4 flex-shrink-0 items-center justify-center rounded-full transition-colors',
                  met ? 'bg-green-500 text-white' : 'bg-muted text-muted-foreground',
                )}
                aria-hidden="true"
              >
                <Check className="h-2.5 w-2.5" />
              </span>
              <span className={cn(met ? 'text-green-700' : 'text-muted-foreground')}>
                {req.label}
              </span>
            </li>
          );
        })}
      </ul>
    </div>
  );
}
