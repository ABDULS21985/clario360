'use client';

import { forwardRef, useImperativeHandle } from 'react';
import { Controller } from 'react-hook-form';
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
import { useTaskForm } from '@/hooks/use-task-form';
import type { FormField as TaskFormField } from '@/types/models';

export interface TaskDetailFormHandle {
  submit: () => Promise<void>;
  saveDraft: () => void;
  getValues: () => Record<string, unknown>;
  isDirty: () => boolean;
}

interface TaskDetailFormProps {
  formSchema: TaskFormField[];
  initialValues?: Record<string, unknown>;
  readOnly?: boolean;
  onSubmit: (data: Record<string, unknown>) => Promise<void>;
  onDraftSave?: (data: Record<string, unknown>) => void;
  isSubmitting?: boolean;
  formId?: string;
  showSubmitButton?: boolean;
}

export const TaskDetailForm = forwardRef<TaskDetailFormHandle, TaskDetailFormProps>(
  function TaskDetailForm(
    {
      formSchema,
      initialValues,
      readOnly = false,
      onSubmit,
      onDraftSave,
      isSubmitting = false,
      formId = 'task-detail-form',
      showSubmitButton = true,
    },
    ref,
  ) {
    const form = useTaskForm({ formSchema, initialValues, onDraftSave, readOnly });

    useImperativeHandle(
      ref,
      () => ({
        submit: async () => {
          await form.handleSubmit(onSubmit)();
        },
        saveDraft: () => {
          if (onDraftSave) {
            onDraftSave(form.getValues());
          }
        },
        getValues: () => form.getValues(),
        isDirty: () => form.formState.isDirty,
      }),
      [form, onDraftSave, onSubmit],
    );

    if (formSchema.length === 0) {
      return (
        <p className="text-sm text-muted-foreground">
          This task has no form fields to fill in.
        </p>
      );
    }

    return (
      <Form {...form}>
        <form
          id={formId}
          onSubmit={form.handleSubmit(onSubmit)}
          className="space-y-6"
        >
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
                          value={
                            rhfField.value === true
                              ? 'true'
                              : rhfField.value === false
                                ? 'false'
                                : ''
                          }
                          onValueChange={(value) => rhfField.onChange(value === 'true')}
                          disabled={readOnly}
                          className="flex gap-4"
                        >
                          <div className="flex items-center gap-2">
                            <RadioGroupItem value="true" id={`${field.name}-yes`} />
                            <Label htmlFor={`${field.name}-yes`} className="cursor-pointer font-normal">
                              Yes
                            </Label>
                          </div>
                          <div className="flex items-center gap-2">
                            <RadioGroupItem value="false" id={`${field.name}-no`} />
                            <Label htmlFor={`${field.name}-no`} className="cursor-pointer font-normal">
                              No
                            </Label>
                          </div>
                        </RadioGroup>
                        {errorMsg && <p className="mt-1 text-xs text-destructive">{errorMsg}</p>}
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
                        {errorMsg && <p className="mt-1 text-xs text-destructive">{errorMsg}</p>}
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
                              field.options.map((option) => (
                                <SelectItem key={option} value={option}>
                                  {option}
                                </SelectItem>
                              ))
                            )}
                          </SelectContent>
                        </Select>
                        {errorMsg && <p className="mt-1 text-xs text-destructive">{errorMsg}</p>}
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
                          onChange={(event) => {
                            rhfField.onChange(
                              event.target.value === '' ? undefined : Number(event.target.value),
                            );
                          }}
                        />
                        {errorMsg && <p className="mt-1 text-xs text-destructive">{errorMsg}</p>}
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
                        {errorMsg && <p className="mt-1 text-xs text-destructive">{errorMsg}</p>}
                      </div>
                    );
                  }

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
                      {errorMsg && <p className="mt-1 text-xs text-destructive">{errorMsg}</p>}
                    </div>
                  );
                }}
              />

              {field.description && (
                <p className="text-xs text-muted-foreground">{field.description}</p>
              )}
            </div>
          ))}

          {!readOnly && showSubmitButton && (
            <div className="flex justify-end">
              <Button type="submit" disabled={isSubmitting}>
                {isSubmitting ? 'Submitting...' : 'Submit'}
              </Button>
            </div>
          )}
        </form>
      </Form>
    );
  },
);
