'use client';

import { Button } from '@/components/ui/button';

interface LineageControlsProps {
  direction: 'LR' | 'TB';
  onDirectionChange: (direction: 'LR' | 'TB') => void;
  onReset: () => void;
}

export function LineageControls({
  direction,
  onDirectionChange,
  onReset,
}: LineageControlsProps) {
  return (
    <div className="flex flex-wrap items-center gap-2">
      <Button type="button" size="sm" variant={direction === 'LR' ? 'default' : 'outline'} onClick={() => onDirectionChange('LR')}>
        Horizontal
      </Button>
      <Button type="button" size="sm" variant={direction === 'TB' ? 'default' : 'outline'} onClick={() => onDirectionChange('TB')}>
        Vertical
      </Button>
      <Button type="button" size="sm" variant="outline" onClick={onReset}>
        Reset
      </Button>
    </div>
  );
}
