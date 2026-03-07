import type { Metadata } from 'next';
import { cookies } from 'next/headers';
import { COOKIES } from '@/lib/constants';
import type { HumanTask, WorkflowInstance } from '@/types/models';

const API_BASE_URL =
  process.env.NEXT_PUBLIC_API_URL ?? process.env.API_URL ?? 'http://localhost:8080';

async function fetchWithAccessToken<T>(path: string): Promise<T | null> {
  const accessToken = cookies().get(COOKIES.ACCESS)?.value;
  if (!accessToken) {
    return null;
  }

  try {
    const response = await fetch(`${API_BASE_URL}${path}`, {
      headers: {
        Authorization: `Bearer ${accessToken}`,
      },
      cache: 'no-store',
    });

    if (!response.ok) {
      return null;
    }

    return (await response.json()) as T;
  } catch {
    return null;
  }
}

export async function getTaskPageMetadata(taskId: string): Promise<Metadata> {
  const task = await fetchWithAccessToken<Pick<HumanTask, 'name'>>(
    `/api/v1/workflows/tasks/${taskId}`,
  );

  return {
    title: task?.name ? `${task.name} | My Tasks` : 'Task Detail',
  };
}

export async function getWorkflowInstancePageMetadata(
  instanceId: string,
): Promise<Metadata> {
  const instance = await fetchWithAccessToken<Pick<WorkflowInstance, 'definition_name'>>(
    `/api/v1/workflows/instances/${instanceId}`,
  );

  return {
    title: instance?.definition_name
      ? `${instance.definition_name} | Workflows`
      : 'Workflow Detail',
  };
}
