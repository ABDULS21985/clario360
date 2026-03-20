'use client';

import { PlayCircle } from 'lucide-react';
import { Button } from '@/components/ui/button';

interface LaunchButtonProps {
  disabled?: boolean;
  onClick: () => void;
}

export function LaunchButton({ disabled, onClick }: LaunchButtonProps) {
  return (
    <Button onClick={onClick} disabled={disabled}>
      <PlayCircle className="mr-2 h-4 w-4" />
      Launch Notebook
    </Button>
  );
}
