'use client';

import { useEffect, useMemo, useRef } from 'react';
import { useForm, useWatch } from 'react-hook-form';
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
  const zodSchema = useMemo(() => buildDynamicZodSchema(formSchema), [formSchema]);

  const defaultValues = useMemo(() => {
    const values: Record<string, unknown> = {};
    for (const field of formSchema) {
      if (field.default !== undefined) {
        values[field.name] = field.default;
      }
    }
    if (initialValues) {
      Object.assign(values, initialValues);
    }
    return values;
  }, [formSchema, initialValues]);

  const form = useForm<Record<string, unknown>>({
    resolver: zodResolver(zodSchema),
    defaultValues,
    disabled: readOnly,
  });

  const autoSaveTimerRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const latestValuesRef = useRef<Record<string, unknown>>(defaultValues);
  const isDirtyRef = useRef(false);
  const watchedValues = useWatch({ control: form.control });

  useEffect(() => {
    latestValuesRef.current = (watchedValues as Record<string, unknown> | undefined) ?? {};
    isDirtyRef.current = form.formState.isDirty;
  }, [form.formState.isDirty, watchedValues]);

  // Auto-save draft every 30 seconds if form is dirty
  useEffect(() => {
    if (!onDraftSave || readOnly) return;

    const saveDraftIfDirty = () => {
      if (isDirtyRef.current) {
        onDraftSave(latestValuesRef.current);
      }
    };

    autoSaveTimerRef.current = setInterval(() => {
      saveDraftIfDirty();
    }, 30000);

    // Also save on page visibility change
    const handleVisibilityChange = () => {
      if (document.hidden) {
        saveDraftIfDirty();
      }
    };
    document.addEventListener('visibilitychange', handleVisibilityChange);

    return () => {
      if (autoSaveTimerRef.current) clearInterval(autoSaveTimerRef.current);
      document.removeEventListener('visibilitychange', handleVisibilityChange);
    };
  }, [onDraftSave, readOnly]);

  useEffect(() => {
    latestValuesRef.current = defaultValues;
    isDirtyRef.current = false;
    form.reset(defaultValues);
  }, [defaultValues, form]);

  return form;
}
