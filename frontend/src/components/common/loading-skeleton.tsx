import { cn } from '@/lib/utils';

interface SkeletonProps {
  variant?: 'card' | 'table-row' | 'list-item' | 'text' | 'avatar' | 'chart';
  count?: number;
  className?: string;
}

function SkeletonBase({ className }: { className?: string }) {
  return <div className={cn('animate-pulse rounded-md bg-muted', className)} />;
}

function CardSkeleton() {
  return (
    <div className="rounded-lg border bg-card p-6 space-y-3">
      <SkeletonBase className="h-4 w-1/3" />
      <SkeletonBase className="h-8 w-1/2" />
      <SkeletonBase className="h-3 w-2/3" />
    </div>
  );
}

function TableRowSkeleton() {
  return (
    <div className="flex items-center gap-4 border-b px-4 py-3">
      <SkeletonBase className="h-4 w-16" />
      <SkeletonBase className="h-4 flex-1" />
      <SkeletonBase className="h-4 w-20" />
      <SkeletonBase className="h-4 w-24" />
    </div>
  );
}

function ListItemSkeleton() {
  return (
    <div className="flex items-center gap-3 py-3">
      <SkeletonBase className="h-8 w-8 rounded-full" />
      <div className="flex-1 space-y-1.5">
        <SkeletonBase className="h-3.5 w-3/4" />
        <SkeletonBase className="h-3 w-1/2" />
      </div>
    </div>
  );
}

function TextSkeleton() {
  return (
    <div className="space-y-2">
      <SkeletonBase className="h-4 w-full" />
      <SkeletonBase className="h-4 w-5/6" />
      <SkeletonBase className="h-4 w-4/6" />
    </div>
  );
}

function AvatarSkeleton() {
  return <SkeletonBase className="h-8 w-8 rounded-full" />;
}

function ChartSkeleton() {
  return (
    <div className="rounded-lg border bg-card p-6">
      <SkeletonBase className="h-4 w-1/4 mb-4" />
      <SkeletonBase className="h-40 w-full" />
    </div>
  );
}

export function LoadingSkeleton({ variant = 'card', count = 1, className }: SkeletonProps) {
  const items = Array.from({ length: count });

  const renderItem = (idx: number) => {
    switch (variant) {
      case 'table-row':
        return <TableRowSkeleton key={idx} />;
      case 'list-item':
        return <ListItemSkeleton key={idx} />;
      case 'text':
        return <TextSkeleton key={idx} />;
      case 'avatar':
        return <AvatarSkeleton key={idx} />;
      case 'chart':
        return <ChartSkeleton key={idx} />;
      default:
        return <CardSkeleton key={idx} />;
    }
  };

  return (
    <div className={cn('space-y-3', className)}>
      {items.map((_, idx) => renderItem(idx))}
    </div>
  );
}
