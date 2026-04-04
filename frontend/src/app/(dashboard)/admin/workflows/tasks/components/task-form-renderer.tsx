'use client';

import { useForm, Controller } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import { Checkbox } from '@/components/ui/checkbox';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { buildDynamicZodSchema } from '@/lib/workflow-utils';
import type { FormField } from '@/types/models';

interface TaskFormRendererProps {
  fields: FormField[];
  initialData?: Record<string, unknown> | null;
  readOnly: boolean;
  onSubmit: (data: Record<string, unknown>) => void;
  formRef?: React.Ref<HTMLFormElement>;
}

export function TaskFormRenderer({
  fields,
  initialData,
  readOnly,
  onSubmit,
  formRef,
}: TaskFormRendererProps) {
  const schema = buildDynamicZodSchema(fields);
  const defaults: Record<string, unknown> = {};
  for (const f of fields) {
    defaults[f.name] =
      initialData?.[f.name] ?? f.default ?? getFieldDefault(f.type);
  }

  const {
    control,
    handleSubmit,
    formState: { errors },
  } = useForm({
    resolver: zodResolver(schema),
    defaultValues: defaults as Record<string, string | number | boolean | undefined>,
  });

  return (
    <form
      ref={formRef}
      onSubmit={handleSubmit((data) => onSubmit(data))}
      className="space-y-3"
    >
      {fields.map((field) => (
        <Controller
          key={field.name}
          control={control}
          name={field.name}
          render={({ field: rhf }) => (
            <div className="space-y-1">
              <Label htmlFor={`field-${field.name}`} className="text-xs">
                {field.label}
                {field.required && (
                  <span className="text-red-500 ml-0.5">*</span>
                )}
              </Label>
              {field.description && (
                <p className="text-[10px] text-muted-foreground">
                  {field.description}
                </p>
              )}

              {field.type === 'boolean' ? (
                <div className="flex items-center gap-2">
                  <Checkbox
                    id={`field-${field.name}`}
                    checked={Boolean(rhf.value)}
                    onCheckedChange={rhf.onChange}
                    disabled={readOnly}
                  />
                  <label
                    htmlFor={`field-${field.name}`}
                    className="text-sm"
                  >
                    {field.label}
                  </label>
                </div>
              ) : field.type === 'select' && field.options ? (
                <Select
                  value={rhf.value as string}
                  onValueChange={rhf.onChange}
                  disabled={readOnly}
                >
                  <SelectTrigger
                    id={`field-${field.name}`}
                    className="h-8 text-sm"
                  >
                    <SelectValue
                      placeholder={field.placeholder ?? 'Select...'}
                    />
                  </SelectTrigger>
                  <SelectContent>
                    {field.options.map((opt) => (
                      <SelectItem key={opt} value={opt}>
                        {opt}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              ) : field.type === 'textarea' ? (
                <Textarea
                  id={`field-${field.name}`}
                  value={(rhf.value as string) ?? ''}
                  onChange={rhf.onChange}
                  placeholder={field.placeholder}
                  disabled={readOnly}
                  rows={3}
                  className="text-sm"
                />
              ) : field.type === 'number' ? (
                <Input
                  id={`field-${field.name}`}
                  type="number"
                  value={rhf.value !== undefined ? String(rhf.value) : ''}
                  onChange={(e) =>
                    rhf.onChange(
                      e.target.value ? parseFloat(e.target.value) : undefined,
                    )
                  }
                  placeholder={field.placeholder}
                  disabled={readOnly}
                  className="h-8 text-sm"
                />
              ) : field.type === 'date' ? (
                <Input
                  id={`field-${field.name}`}
                  type="date"
                  value={(rhf.value as string) ?? ''}
                  onChange={rhf.onChange}
                  disabled={readOnly}
                  className="h-8 text-sm"
                />
              ) : (
                <Input
                  id={`field-${field.name}`}
                  value={(rhf.value as string) ?? ''}
                  onChange={rhf.onChange}
                  placeholder={field.placeholder}
                  disabled={readOnly}
                  className="h-8 text-sm"
                />
              )}

              {errors[field.name] && (
                <p className="text-[10px] text-red-500">
                  {errors[field.name]?.message as string}
                </p>
              )}
            </div>
          )}
        />
      ))}
    </form>
  );
}

function getFieldDefault(type: string): unknown {
  switch (type) {
    case 'boolean':
      return false;
    case 'number':
      return undefined;
    default:
      return '';
  }
}
