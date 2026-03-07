'use client';

import { Button } from '@/components/ui/button';

interface LineageControlsProps {
  direction: 'LR' | 'TB';
  onDirectionChange: (direction: 'LR' | 'TB') => void;
  onFit: () => void;
  onReset: () => void;
  onZoomIn: () => void;
  onZoomOut: () => void;
  onFullscreen: () => void;
}

export function LineageControls({
  direction,
  onDirectionChange,
  onFit,
  onReset,
  onZoomIn,
  onZoomOut,
  onFullscreen,
}: LineageControlsProps) {
  return (
    <div className="flex flex-wrap items-center gap-2">
      <Button type="button" size="sm" variant={direction === 'LR' ? 'default' : 'outline'} onClick={() => onDirectionChange('LR')}>
        Horizontal
      </Button>
      <Button type="button" size="sm" variant={direction === 'TB' ? 'default' : 'outline'} onClick={() => onDirectionChange('TB')}>
        Vertical
      </Button>
      <Button type="button" size="sm" variant="outline" onClick={onFit}>
        Fit to screen
      </Button>
      <Button type="button" size="sm" variant="outline" onClick={onZoomIn}>
        Zoom +
      </Button>
      <Button type="button" size="sm" variant="outline" onClick={onZoomOut}>
        Zoom -
      </Button>
      <Button type="button" size="sm" variant="outline" onClick={onReset}>
        Reset
      </Button>
      <Button type="button" size="sm" variant="outline" onClick={onFullscreen}>
        Full screen
      </Button>
    </div>
  );
}
