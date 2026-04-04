'use client';

import { useEffect, useState } from 'react';
import { cn } from '@/lib/utils';

interface HighlightAnimationProps {
  children: React.ReactNode;
  highlight?: boolean;
  highlightKey?: string | number | null;
  className?: string;
  duration?: number; // ms, default 3000
}

export function HighlightAnimation({
  children,
  highlight = false,
  highlightKey,
  className,
  duration = 3000,
}: HighlightAnimationProps) {
  const [isHighlighted, setIsHighlighted] = useState(false);

  useEffect(() => {
    if (highlight) {
      setIsHighlighted(true);
      const timer = setTimeout(() => setIsHighlighted(false), duration);
      return () => clearTimeout(timer);
    }
  }, [duration, highlight, highlightKey]);

  return (
    <div
      className={cn(
        'h-full transition-all duration-500',
        isHighlighted && 'ring-2 ring-yellow-400 ring-opacity-75',
        className,
      )}
    >
      {children}
    </div>
  );
}
