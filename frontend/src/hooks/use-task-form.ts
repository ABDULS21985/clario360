'use client';

import { useEffect, useRef } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { buildDynamicZodSchema } from '@/lib/workflow-utils';
import type { FormField } from '@/types/models';

interface UseTaskFormOptions {
  formSchema: FormField[];
  initialValues?: Record<string, unknown>;
  onDraftSave?: (data: Record<string, unknown>) => void;
  readOnly?: boolean;
}

export function useTaskForm({
  formSchema,
  initialValues,
  onDraftSave,
  readOnly = false,
}: UseTaskFormOptions) {
  const zodSchema = buildDynamicZodSchema(formSchema);

  // Build default values from field.default merged with initialValues
  const defaultValues: Record<string, unknown> = {};
  for (const field of formSchema) {
    if (field.default !== undefined) {
      defaultValues[field.name] = field.default;
    }
  }
  if (initialValues) {
    Object.assign(defaultValues, initialValues);
  }

  const form = useForm<Record<string, unknown>>({
    resolver: zodResolver(zodSchema),
    defaultValues,
    disabled: readOnly,
  });

  const autoSaveTimerRef = useRef<ReturnType<typeof setInterval> | null>(null);

  // Auto-save draft every 30 seconds if form is dirty
  useEffect(() => {
    if (!onDraftSave || readOnly) return;

    autoSaveTimerRef.current = setInterval(() => {
      if (form.formState.isDirty) {
        onDraftSave(form.getValues());
      }
    }, 30000);

    // Also save on page visibility change
    const handleVisibilityChange = () => {
      if (document.hidden && form.formState.isDirty) {
        onDraftSave(form.getValues());
      }
    };
    document.addEventListener('visibilitychange', handleVisibilityChange);

    return () => {
      if (autoSaveTimerRef.current) clearInterval(autoSaveTimerRef.current);
      document.removeEventListener('visibilitychange', handleVisibilityChange);
    };
  }, [form, onDraftSave, readOnly]);

  return form;
}
