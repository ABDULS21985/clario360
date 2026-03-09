'use client';

import {
  Undo2,
  Redo2,
  ZoomIn,
  ZoomOut,
  Maximize,
  LayoutGrid,
  Save,
  Upload,
  Loader2,
} from 'lucide-react';
import { Button } from '@/components/ui/button';
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip';

interface CanvasToolbarProps {
  canUndo: boolean;
  canRedo: boolean;
  zoom: number;
  readOnly: boolean;
  isSaving: boolean;
  isPublishing: boolean;
  isDraft: boolean;
  onUndo: () => void;
  onRedo: () => void;
  onZoomIn: () => void;
  onZoomOut: () => void;
  onFitToScreen: () => void;
  onAutoLayout: () => void;
  onSave: () => void;
  onPublish: () => void;
}

export function CanvasToolbar({
  canUndo,
  canRedo,
  zoom,
  readOnly,
  isSaving,
  isPublishing,
  isDraft,
  onUndo,
  onRedo,
  onZoomIn,
  onZoomOut,
  onFitToScreen,
  onAutoLayout,
  onSave,
  onPublish,
}: CanvasToolbarProps) {
  return (
    <TooltipProvider delayDuration={300}>
      <div className="flex items-center gap-1 border-b px-3 py-1.5 bg-background">
        {!readOnly && (
          <>
            <ToolbarButton
              icon={Undo2}
              label="Undo (Ctrl+Z)"
              onClick={onUndo}
              disabled={!canUndo}
            />
            <ToolbarButton
              icon={Redo2}
              label="Redo (Ctrl+Shift+Z)"
              onClick={onRedo}
              disabled={!canRedo}
            />
            <div className="w-px h-5 bg-border mx-1" />
          </>
        )}

        <ToolbarButton icon={ZoomOut} label="Zoom Out" onClick={onZoomOut} />
        <span className="text-xs text-muted-foreground w-12 text-center tabular-nums">
          {Math.round(zoom * 100)}%
        </span>
        <ToolbarButton icon={ZoomIn} label="Zoom In" onClick={onZoomIn} />
        <ToolbarButton
          icon={Maximize}
          label="Fit to Screen"
          onClick={onFitToScreen}
        />

        {!readOnly && (
          <>
            <ToolbarButton
              icon={LayoutGrid}
              label="Auto Layout"
              onClick={onAutoLayout}
            />
            <div className="flex-1" />
            <Button
              variant="outline"
              size="sm"
              onClick={onSave}
              disabled={isSaving}
              className="h-7 text-xs"
            >
              {isSaving ? (
                <Loader2 className="mr-1 h-3.5 w-3.5 animate-spin" />
              ) : (
                <Save className="mr-1 h-3.5 w-3.5" />
              )}
              Save Draft
            </Button>
            {isDraft && (
              <Button
                size="sm"
                onClick={onPublish}
                disabled={isPublishing}
                className="h-7 text-xs"
              >
                {isPublishing ? (
                  <Loader2 className="mr-1 h-3.5 w-3.5 animate-spin" />
                ) : (
                  <Upload className="mr-1 h-3.5 w-3.5" />
                )}
                Publish
              </Button>
            )}
          </>
        )}
      </div>
    </TooltipProvider>
  );
}

function ToolbarButton({
  icon: Icon,
  label,
  onClick,
  disabled,
}: {
  icon: React.ElementType;
  label: string;
  onClick: () => void;
  disabled?: boolean;
}) {
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <Button
          variant="ghost"
          size="icon"
          className="h-7 w-7"
          onClick={onClick}
          disabled={disabled}
          aria-label={label}
        >
          <Icon className="h-4 w-4" />
        </Button>
      </TooltipTrigger>
      <TooltipContent side="bottom" className="text-xs">
        {label}
      </TooltipContent>
    </Tooltip>
  );
}
