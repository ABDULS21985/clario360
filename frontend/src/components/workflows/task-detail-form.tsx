'use client';

import { useMemo } from 'react';
import { Form } from '@/components/ui/form';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import { Label } from '@/components/ui/label';
import { RadioGroup, RadioGroupItem } from '@/components/ui/radio-group';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Controller } from 'react-hook-form';
import { buildDynamicZodSchema } from '@/lib/workflow-utils';
import { useTaskForm } from '@/hooks/use-task-form';
import type { FormField as TaskFormField } from '@/types/models';

interface TaskDetailFormProps {
  formSchema: TaskFormField[];
  initialValues?: Record<string, unknown>;
  readOnly?: boolean;
  onSubmit: (data: Record<string, unknown>) => Promise<void>;
  onDraftSave?: (data: Record<string, unknown>) => void;
  isSubmitting?: boolean;
}

export function TaskDetailForm({
  formSchema,
  initialValues,
  readOnly = false,
  onSubmit,
  onDraftSave,
  isSubmitting = false,
}: TaskDetailFormProps) {
  const form = useTaskForm({ formSchema, initialValues, onDraftSave, readOnly });

  // Validate we have the schema (memoized so it doesn't re-run on every render)
  const zodSchema = useMemo(() => buildDynamicZodSchema(formSchema), [formSchema]);
  void zodSchema;

  const handleSubmit = form.handleSubmit(async (data) => {
    await onSubmit(data);
  });

  if (formSchema.length === 0) {
    return (
      <p className="text-sm text-muted-foreground">
        This task has no form fields to fill in.
      </p>
    );
  }

  return (
    <Form {...form}>
      <form onSubmit={handleSubmit} className="space-y-6">
        {formSchema.map((field) => (
          <div key={field.name} className="space-y-1.5">
            <Label htmlFor={`field-${field.name}`}>
              {field.label}
              {field.required && <span className="ml-1 text-destructive">*</span>}
            </Label>

            <Controller
              control={form.control}
              name={field.name}
              render={({ field: rhfField, fieldState }) => {
                const errorMsg = fieldState.error?.message;

                if (field.type === 'boolean') {
                  return (
                    <div>
                      <RadioGroup
                        value={rhfField.value === true ? 'true' : rhfField.value === false ? 'false' : ''}
                        onValueChange={(v) => rhfField.onChange(v === 'true')}
                        disabled={readOnly}
                        className="flex gap-4"
                      >
                        <div className="flex items-center gap-2">
                          <RadioGroupItem value="true" id={`${field.name}-yes`} />
                          <Label htmlFor={`${field.name}-yes`} className="font-normal cursor-pointer">Yes</Label>
                        </div>
                        <div className="flex items-center gap-2">
                          <RadioGroupItem value="false" id={`${field.name}-no`} />
                          <Label htmlFor={`${field.name}-no`} className="font-normal cursor-pointer">No</Label>
                        </div>
                      </RadioGroup>
                      {errorMsg && <p className="text-xs text-destructive mt-1">{errorMsg}</p>}
                    </div>
                  );
                }

                if (field.type === 'textarea') {
                  return (
                    <div>
                      <Textarea
                        id={`field-${field.name}`}
                        placeholder={field.placeholder}
                        className="min-h-[100px]"
                        disabled={readOnly}
                        {...rhfField}
                        value={(rhfField.value as string) ?? ''}
                      />
                      {errorMsg && <p className="text-xs text-destructive mt-1">{errorMsg}</p>}
                    </div>
                  );
                }

                if (field.type === 'select') {
                  return (
                    <div>
                      <Select
                        value={(rhfField.value as string) ?? ''}
                        onValueChange={rhfField.onChange}
                        disabled={readOnly || !field.options?.length}
                      >
                        <SelectTrigger id={`field-${field.name}`}>
                          <SelectValue placeholder={field.placeholder ?? 'Select...'} />
                        </SelectTrigger>
                        <SelectContent>
                          {!field.options?.length ? (
                            <SelectItem value="__none__" disabled>
                              No options available
                            </SelectItem>
                          ) : (
                            field.options.map((opt) => (
                              <SelectItem key={opt} value={opt}>
                                {opt}
                              </SelectItem>
                            ))
                          )}
                        </SelectContent>
                      </Select>
                      {errorMsg && <p className="text-xs text-destructive mt-1">{errorMsg}</p>}
                    </div>
                  );
                }

                if (field.type === 'number') {
                  return (
                    <div>
                      <Input
                        id={`field-${field.name}`}
                        type="number"
                        step="any"
                        placeholder={field.placeholder}
                        disabled={readOnly}
                        {...rhfField}
                        value={(rhfField.value as number | '') ?? ''}
                        onChange={(e) =>
                          rhfField.onChange(e.target.value === '' ? undefined : Number(e.target.value))
                        }
                      />
                      {errorMsg && <p className="text-xs text-destructive mt-1">{errorMsg}</p>}
                    </div>
                  );
                }

                if (field.type === 'date') {
                  return (
                    <div>
                      <Input
                        id={`field-${field.name}`}
                        type="date"
                        disabled={readOnly}
                        {...rhfField}
                        value={(rhfField.value as string) ?? ''}
                      />
                      {errorMsg && <p className="text-xs text-destructive mt-1">{errorMsg}</p>}
                    </div>
                  );
                }

                // Default: text
                return (
                  <div>
                    <Input
                      id={`field-${field.name}`}
                      type="text"
                      placeholder={field.placeholder}
                      disabled={readOnly}
                      {...rhfField}
                      value={(rhfField.value as string) ?? ''}
                    />
                    {errorMsg && <p className="text-xs text-destructive mt-1">{errorMsg}</p>}
                  </div>
                );
              }}
            />

            {field.description && (
              <p className="text-xs text-muted-foreground">{field.description}</p>
            )}
          </div>
        ))}

        {!readOnly && (
          <div className="flex justify-end">
            <Button type="submit" disabled={isSubmitting}>
              {isSubmitting ? 'Submitting...' : 'Submit'}
            </Button>
          </div>
        )}
      </form>
    </Form>
  );
}
