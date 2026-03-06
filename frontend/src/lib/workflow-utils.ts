import {
  UserCheck,
  Zap,
  GitBranch,
  GitMerge,
  Clock,
  CheckCircle,
} from 'lucide-react';
import type { LucideIcon } from 'lucide-react';
import { z } from 'zod';
import type { FormField, HumanTask } from '@/types/models';
import { differenceInHours, parseISO } from 'date-fns';

export function formatStepType(type: string): string {
  const map: Record<string, string> = {
    human_task: 'Human Task',
    service_task: 'Automated',
    condition: 'Condition',
    parallel_gateway: 'Parallel',
    timer: 'Timer',
    end: 'End',
  };
  return map[type] ?? type;
}

export function getStepIcon(type: string): LucideIcon {
  const map: Record<string, LucideIcon> = {
    human_task: UserCheck,
    service_task: Zap,
    condition: GitBranch,
    parallel_gateway: GitMerge,
    timer: Clock,
    end: CheckCircle,
  };
  return map[type] ?? CheckCircle;
}

export function getStepStatusColor(status: string): string {
  const map: Record<string, string> = {
    completed: 'text-green-600',
    running: 'text-blue-600',
    failed: 'text-red-600',
    pending: 'text-gray-400',
    skipped: 'text-gray-300',
    cancelled: 'text-gray-400',
  };
  return map[status] ?? 'text-gray-400';
}

export function buildDynamicZodSchema(
  fields: FormField[],
): z.ZodObject<Record<string, z.ZodTypeAny>> {
  const shape: Record<string, z.ZodTypeAny> = {};

  for (const field of fields) {
    let schema: z.ZodTypeAny;

    switch (field.type) {
      case 'boolean':
        schema = field.required
          ? z.boolean({ required_error: 'Please select an option' })
          : z.boolean().optional();
        break;
      case 'number':
        schema = field.required
          ? z.number({ required_error: `${field.label} is required` })
          : z.number().optional();
        break;
      case 'date':
        schema = field.required
          ? z.string().min(1, `${field.label} is required`)
          : z.string().optional();
        break;
      case 'select':
        schema = field.required
          ? z.string().min(1, 'Please select an option')
          : z.string().optional();
        break;
      default:
        // text, textarea
        schema = field.required
          ? z.string().min(1, `${field.label} is required`)
          : z.string().optional();
    }

    shape[field.name] = schema;
  }

  return z.object(shape);
}

export function formatSLAStatus(task: HumanTask): {
  text: string;
  color: string;
  urgent: boolean;
} {
  if (!task.sla_deadline) {
    return { text: 'No deadline', color: 'text-muted-foreground', urgent: false };
  }

  if (task.sla_breached) {
    const deadline = parseISO(task.sla_deadline);
    const hoursOverdue = Math.abs(differenceInHours(new Date(), deadline));
    const text =
      hoursOverdue < 1
        ? 'Overdue by <1h'
        : `Overdue by ${hoursOverdue}h`;
    return { text, color: 'text-red-600', urgent: true };
  }

  const deadline = parseISO(task.sla_deadline);
  const hoursLeft = differenceInHours(deadline, new Date());

  if (hoursLeft <= 4) {
    return {
      text: hoursLeft < 1 ? '<1h left' : `${hoursLeft}h left`,
      color: 'text-orange-600',
      urgent: true,
    };
  }

  const daysLeft = Math.floor(hoursLeft / 24);
  if (daysLeft === 0) {
    return { text: `${hoursLeft}h left`, color: 'text-foreground', urgent: false };
  }
  return { text: `${daysLeft}d left`, color: 'text-foreground', urgent: false };
}

export const PRIORITY_LABELS: Record<number, string> = {
  2: 'Critical',
  1: 'High',
  0: 'Normal',
};

export const PRIORITY_COLORS: Record<number, string> = {
  2: 'bg-red-500',
  1: 'bg-orange-400',
  0: 'bg-blue-400',
};
