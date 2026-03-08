'use client';

import type { UseFormReturn } from 'react-hook-form';
import { FormField } from '@/components/shared/forms/form-field';
import { Input } from '@/components/ui/input';
import type { DagsterConnectionValues } from '@/lib/data-suite/forms';

interface DagsterFormProps {
  form: UseFormReturn<DagsterConnectionValues>;
}

export function DagsterForm({ form }: DagsterFormProps) {
  return (
    <div className="grid gap-4 md:grid-cols-2">
      <FormField name="graphql_url" label="GraphQL URL" required>
        <Input {...form.register('graphql_url')} placeholder="http://dagster-webserver:3000/graphql" />
      </FormField>
      <FormField name="workspace" label="Workspace">
        <Input {...form.register('workspace')} placeholder="default" />
      </FormField>
      <FormField name="api_token" label="API Token">
        <Input type="password" autoComplete="new-password" {...form.register('api_token')} />
      </FormField>
      <FormField name="timeout_seconds" label="Timeout (seconds)" required>
        <Input type="number" {...form.register('timeout_seconds', { valueAsNumber: true })} />
      </FormField>
    </div>
  );
}
