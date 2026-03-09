import * as React from 'react';
import { cva, type VariantProps } from 'class-variance-authority';
import { cn } from '@/lib/utils';

const badgeVariants = cva(
  'inline-flex items-center rounded-full border px-2.5 py-1 text-[11px] font-semibold uppercase tracking-[0.18em] transition-colors',
  {
    variants: {
      variant: {
        default: 'border-primary/15 bg-primary/10 text-primary',
        secondary: 'border-transparent bg-secondary text-secondary-foreground',
        destructive: 'border-red-200 bg-red-50 text-red-700',
        warning: 'border-amber-200 bg-amber-50 text-amber-800',
        outline: 'border-border/80 bg-white/70 text-foreground',
        success: 'border-emerald-200 bg-emerald-50 text-emerald-800',
      },
    },
    defaultVariants: { variant: 'default' },
  },
);

export interface BadgeProps
  extends React.HTMLAttributes<HTMLSpanElement>,
    VariantProps<typeof badgeVariants> {}

function Badge({ className, variant, ...props }: BadgeProps) {
  return <span className={cn(badgeVariants({ variant }), className)} {...props} />;
}

export { Badge, badgeVariants };
