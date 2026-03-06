'use client';

import { useEffect, useState } from 'react';
import { cn } from '@/lib/utils';

interface HighlightAnimationProps {
  children: React.ReactNode;
  highlight?: boolean;
  className?: string;
  duration?: number; // ms, default 3000
}

export function HighlightAnimation({
  children,
  highlight = false,
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
  }, [highlight, duration]);

  return (
    <div
      className={cn(
        'transition-all duration-500',
        isHighlighted && 'ring-2 ring-yellow-400 ring-opacity-75',
        className,
      )}
    >
      {children}
    </div>
  );
}
