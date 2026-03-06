'use client';

import React, { useRef, useCallback, KeyboardEvent, ClipboardEvent } from 'react';
import { cn } from '@/lib/utils';

interface MFACodeInputProps {
  onComplete: (code: string) => void;
  disabled?: boolean;
  error?: boolean;
  className?: string;
}

export function MFACodeInput({
  onComplete,
  disabled = false,
  error = false,
  className,
}: MFACodeInputProps) {
  const inputsRef = useRef<Array<HTMLInputElement | null>>([null, null, null, null, null, null]);
  const valuesRef = useRef<string[]>(['', '', '', '', '', '']);

  const getCurrentCode = () => valuesRef.current.join('');

  const focusInput = (index: number) => {
    const el = inputsRef.current[index];
    if (el) {
      el.focus();
      el.select();
    }
  };

  const checkComplete = useCallback(() => {
    const code = getCurrentCode();
    if (code.length === 6) {
      onComplete(code);
    }
  }, [onComplete]);

  const handleChange = (index: number, value: string) => {
    // Only allow single digits
    const digit = value.replace(/\D/g, '').slice(-1);
    valuesRef.current[index] = digit;

    // Update the visible input value
    const el = inputsRef.current[index];
    if (el) el.value = digit;

    if (digit && index < 5) {
      focusInput(index + 1);
    }

    checkComplete();
  };

  const handleKeyDown = (index: number, e: KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Backspace') {
      e.preventDefault();
      if (valuesRef.current[index]) {
        // Clear current
        valuesRef.current[index] = '';
        const el = inputsRef.current[index];
        if (el) el.value = '';
      } else if (index > 0) {
        // Move to previous and clear
        valuesRef.current[index - 1] = '';
        const prevEl = inputsRef.current[index - 1];
        if (prevEl) prevEl.value = '';
        focusInput(index - 1);
      }
    } else if (e.key === 'ArrowLeft' && index > 0) {
      e.preventDefault();
      focusInput(index - 1);
    } else if (e.key === 'ArrowRight' && index < 5) {
      e.preventDefault();
      focusInput(index + 1);
    }
  };

  const handlePaste = (e: ClipboardEvent<HTMLInputElement>) => {
    e.preventDefault();
    const pasted = e.clipboardData
      .getData('text')
      .replace(/\D/g, '')
      .slice(0, 6);

    if (!pasted) return;

    for (let i = 0; i < 6; i++) {
      const digit = pasted[i] ?? '';
      valuesRef.current[i] = digit;
      const el = inputsRef.current[i];
      if (el) el.value = digit;
    }

    // Focus last filled or end
    const focusIdx = Math.min(pasted.length, 5);
    focusInput(focusIdx);
    checkComplete();
  };

  return (
    <div
      className={cn('flex gap-2', className)}
      role="group"
      aria-label="Verification code"
    >
      {[0, 1, 2, 3, 4, 5].map((index) => (
        <React.Fragment key={index}>
          {index === 3 && (
            <span className="flex items-center text-muted-foreground" aria-hidden="true">
              –
            </span>
          )}
          <input
            ref={(el) => {
              inputsRef.current[index] = el;
            }}
            type="text"
            inputMode="numeric"
            maxLength={1}
            disabled={disabled}
            aria-label={`Digit ${index + 1} of 6`}
            aria-describedby={error ? 'mfa-error' : undefined}
            autoComplete={index === 0 ? 'one-time-code' : 'off'}
            className={cn(
              'h-12 w-10 rounded-md border text-center text-lg font-semibold transition-colors',
              'focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2',
              'disabled:cursor-not-allowed disabled:opacity-50',
              error
                ? 'border-destructive bg-destructive/5 text-destructive focus:ring-destructive'
                : 'border-input bg-background',
            )}
            onChange={(e) => handleChange(index, e.target.value)}
            onKeyDown={(e) => handleKeyDown(index, e)}
            onPaste={handlePaste}
            onFocus={(e) => e.target.select()}
          />
        </React.Fragment>
      ))}
    </div>
  );
}
