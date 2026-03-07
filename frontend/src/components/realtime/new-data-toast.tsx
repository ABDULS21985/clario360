'use client';

import { showInfo } from '@/lib/toast';

interface NewDataToastOptions {
  title: string;
  description?: string;
}

export function showNewDataToast({ title, description }: NewDataToastOptions): void {
  showInfo(title, description);
}
