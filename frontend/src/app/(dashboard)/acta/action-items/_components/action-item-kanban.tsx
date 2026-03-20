'use client';

import { DndContext, PointerSensor, closestCenter, useDraggable, useDroppable, useSensor, useSensors, type DragEndEvent } from '@dnd-kit/core';
import { CSS } from '@dnd-kit/utilities';
import { AlertTriangle } from 'lucide-react';
import { Badge } from '@/components/ui/badge';
import { cn } from '@/lib/utils';
import { daysOverdue } from '@/lib/enterprise';
import type { ActaActionItem, ActaActionItemStatus } from '@/types/suites';

interface ActionItemKanbanProps {
  items: ActaActionItem[];
  onMove: (item: ActaActionItem, nextStatus: ActaActionItemStatus) => void;
}

const COLUMNS: Array<{ key: ActaActionItemStatus; label: string }> = [
  { key: 'pending', label: 'Pending' },
  { key: 'in_progress', label: 'In Progress' },
  { key: 'completed', label: 'Completed' },
  { key: 'overdue', label: 'Overdue' },
];

export function ActionItemKanban({ items, onMove }: ActionItemKanbanProps) {
  const sensors = useSensors(useSensor(PointerSensor, { activationConstraint: { distance: 4 } }));

  const handleDragEnd = (event: DragEndEvent) => {
    const itemId = String(event.active.id);
    const nextStatus = event.over?.id ? String(event.over.id) as ActaActionItemStatus : null;
    if (!nextStatus) {
      return;
    }
    const item = items.find((candidate) => candidate.id === itemId);
    if (!item || item.status === nextStatus) {
      return;
    }
    onMove(item, nextStatus);
  };

  return (
    <DndContext sensors={sensors} collisionDetection={closestCenter} onDragEnd={handleDragEnd}>
      <div className="grid grid-cols-1 gap-4 xl:grid-cols-4">
        {COLUMNS.map((column) => (
          <KanbanColumn
            key={column.key}
            id={column.key}
            label={column.label}
            items={items.filter((item) => item.status === column.key)}
          />
        ))}
      </div>
    </DndContext>
  );
}

function KanbanColumn({
  id,
  label,
  items,
}: {
  id: ActaActionItemStatus;
  label: string;
  items: ActaActionItem[];
}) {
  const { isOver, setNodeRef } = useDroppable({ id });

  return (
    <div
      ref={setNodeRef}
      className={cn(
        'rounded-xl border bg-card p-3',
        isOver && 'border-primary bg-accent/20',
      )}
    >
      <div className="mb-3 flex items-center justify-between">
        <div className="font-medium">{label}</div>
        <Badge variant="outline">{items.length}</Badge>
      </div>
      <div className="space-y-3">
        {items.map((item) => (
          <DraggableCard key={item.id} item={item} />
        ))}
      </div>
    </div>
  );
}

function DraggableCard({ item }: { item: ActaActionItem }) {
  const { attributes, listeners, setNodeRef, transform, isDragging } = useDraggable({ id: item.id });
  const overdueDays = item.status === 'overdue' ? daysOverdue(item.due_date) : 0;

  return (
    <div
      ref={setNodeRef}
      style={{ transform: CSS.Translate.toString(transform) }}
      {...listeners}
      {...attributes}
      className={cn(
        'cursor-grab rounded-xl border bg-background px-3 py-3 shadow-sm transition',
        isDragging && 'opacity-60',
      )}
    >
      <div className="flex items-start justify-between gap-2">
        <p className="font-medium">{item.title}</p>
        <Badge variant="outline" className="capitalize">
          {item.priority}
        </Badge>
      </div>
      <p className="mt-1 text-xs text-muted-foreground">{item.assignee_name}</p>
      <div className="mt-3 flex flex-wrap gap-2 text-xs text-muted-foreground">
        <span>{item.committee_id.slice(0, 8)}…</span>
        <span>Due {item.due_date}</span>
        {item.status === 'overdue' ? (
          <span className="inline-flex items-center gap-1 text-destructive">
            <AlertTriangle className="h-3.5 w-3.5" />
            {overdueDays} day{overdueDays === 1 ? '' : 's'} overdue
          </span>
        ) : null}
      </div>
    </div>
  );
}
