import { describe, expect, it } from 'vitest';
import { buildDynamicZodSchema, formatSLAStatus } from './workflow-utils';
import type { FormField, HumanTask } from '@/types/models';

const formFields: FormField[] = [
  { name: 'notes', type: 'text', label: 'Notes', required: true },
  { name: 'approved', type: 'boolean', label: 'Approved', required: false },
  {
    name: 'decision',
    type: 'select',
    label: 'Decision',
    required: true,
    options: ['approve', 'reject'],
  },
];

const baseTask: HumanTask = {
  id: 'task-1',
  name: 'Review Task',
  description: 'Review a pending item',
  instance_id: 'instance-1',
  definition_name: 'Approval Workflow',
  workflow_name: 'Approval Workflow',
  step_id: 'review',
  status: 'pending',
  priority: 1,
  form_schema: [],
  form_data: null,
  sla_deadline: null,
  sla_breached: false,
  claimed_by: null,
  claimed_by_name: null,
  assignee_role: 'reviewer',
  assignee_id: null,
  metadata: {},
  created_at: new Date().toISOString(),
  updated_at: new Date().toISOString(),
};

describe('workflow-utils', () => {
  it('buildDynamicZodSchema enforces required text fields', () => {
    const schema = buildDynamicZodSchema(formFields);
    const result = schema.safeParse({
      notes: '',
      approved: undefined,
      decision: 'approve',
    });

    expect(result.success).toBe(false);
  });

  it('buildDynamicZodSchema accepts optional booleans', () => {
    const schema = buildDynamicZodSchema(formFields);
    const result = schema.safeParse({
      notes: 'Looks good',
      decision: 'approve',
    });

    expect(result.success).toBe(true);
  });

  it('buildDynamicZodSchema enforces required select fields', () => {
    const schema = buildDynamicZodSchema(formFields);
    const result = schema.safeParse({
      notes: 'Looks good',
      decision: '',
    });

    expect(result.success).toBe(false);
  });

  it('formatSLAStatus handles overdue tasks', () => {
    const task = {
      ...baseTask,
      sla_deadline: new Date(Date.now() - 2 * 60 * 60 * 1000).toISOString(),
      sla_breached: true,
    };

    expect(formatSLAStatus(task)).toEqual({
      text: 'Overdue by 2h',
      color: 'text-red-600',
      urgent: true,
    });
  });

  it('formatSLAStatus handles soon deadlines', () => {
    const task = {
      ...baseTask,
      sla_deadline: new Date(Date.now() + 2 * 60 * 60 * 1000).toISOString(),
      sla_breached: false,
    };

    expect(formatSLAStatus(task)).toEqual({
      text: '2h left',
      color: 'text-orange-600',
      urgent: true,
    });
  });

  it('formatSLAStatus handles no deadline', () => {
    expect(formatSLAStatus(baseTask)).toEqual({
      text: 'No deadline',
      color: 'text-muted-foreground',
      urgent: false,
    });
  });
});
