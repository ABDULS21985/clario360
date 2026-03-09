'use client';

import { useCallback, useRef, useState } from 'react';
import { StepNode } from './step-node';
import { ConnectionLine, TempConnectionLine } from './connection-line';
import { StepPalette } from './step-palette';
import { PropertiesPanel } from './properties-panel';
import { CanvasToolbar } from './canvas-toolbar';
import { useCanvasState } from '../hooks/use-canvas-state';
import type {
  WorkflowStep,
  WorkflowStepType,
  WorkflowDefinition,
  StepExecution,
} from '@/types/models';

const NODE_WIDTH = 200;
const NODE_HEIGHT = 80;

interface WorkflowCanvasProps {
  definition: WorkflowDefinition;
  readOnly: boolean;
  isSaving: boolean;
  isPublishing: boolean;
  stepStatuses?: Record<string, string>;
  onSave: (steps: WorkflowStep[]) => void;
  onPublish: () => void;
}

export function WorkflowCanvas({
  definition,
  readOnly,
  isSaving,
  isPublishing,
  stepStatuses,
  onSave,
  onPublish,
}: WorkflowCanvasProps) {
  const canvas = useCanvasState(definition.steps);
  const containerRef = useRef<HTMLDivElement>(null);
  const svgRef = useRef<SVGSVGElement>(null);

  // Drag state
  const [dragging, setDragging] = useState<{
    stepId: string;
    startX: number;
    startY: number;
    origX: number;
    origY: number;
  } | null>(null);

  // Pan state
  const [panning, setPanning] = useState<{
    startX: number;
    startY: number;
    origPanX: number;
    origPanY: number;
  } | null>(null);

  // Connection temp line
  const [connectMouse, setConnectMouse] = useState<{ x: number; y: number } | null>(null);

  const selectedStep = canvas.steps.find((s) => s.id === canvas.selectedStepId) ?? null;

  // ── Handlers ──

  const handleDragStart = useCallback(
    (stepId: string, e: React.MouseEvent) => {
      if (readOnly) return;
      const step = canvas.steps.find((s) => s.id === stepId);
      if (!step) return;
      setDragging({
        stepId,
        startX: e.clientX,
        startY: e.clientY,
        origX: step.position.x,
        origY: step.position.y,
      });
    },
    [readOnly, canvas.steps],
  );

  const handleConnectStart = useCallback(
    (stepId: string, e: React.MouseEvent) => {
      if (readOnly) return;
      const step = canvas.steps.find((s) => s.id === stepId);
      if (!step) return;
      canvas.setConnecting({ fromStepId: stepId, mouseX: e.clientX, mouseY: e.clientY });
      setConnectMouse({ x: e.clientX, y: e.clientY });
    },
    [readOnly, canvas],
  );

  const handleConnectEnd = useCallback(
    (targetStepId: string) => {
      if (!canvas.connecting) return;
      if (canvas.connecting.fromStepId === targetStepId) return;
      // Prevent duplicate transitions
      const fromStep = canvas.steps.find((s) => s.id === canvas.connecting!.fromStepId);
      if (fromStep?.transitions.some((t) => t.target_step_id === targetStepId)) {
        canvas.setConnecting(null);
        setConnectMouse(null);
        return;
      }
      canvas.addTransition(canvas.connecting.fromStepId, targetStepId);
      canvas.setConnecting(null);
      setConnectMouse(null);
    },
    [canvas],
  );

  const handleCanvasMouseDown = useCallback(
    (e: React.MouseEvent) => {
      // Only pan on background click (left button)
      if (e.button !== 0) return;
      if (e.target !== containerRef.current && e.target !== svgRef.current) return;
      canvas.selectStep(null);
      setPanning({
        startX: e.clientX,
        startY: e.clientY,
        origPanX: canvas.pan.x,
        origPanY: canvas.pan.y,
      });
    },
    [canvas],
  );

  const handleCanvasMouseMove = useCallback(
    (e: React.MouseEvent) => {
      if (dragging) {
        const dx = (e.clientX - dragging.startX) / canvas.zoom;
        const dy = (e.clientY - dragging.startY) / canvas.zoom;
        canvas.moveStep(dragging.stepId, {
          x: Math.round(dragging.origX + dx),
          y: Math.round(dragging.origY + dy),
        });
      } else if (panning) {
        canvas.setPan({
          x: panning.origPanX + (e.clientX - panning.startX),
          y: panning.origPanY + (e.clientY - panning.startY),
        });
      } else if (canvas.connecting) {
        setConnectMouse({ x: e.clientX, y: e.clientY });
      }
    },
    [dragging, panning, canvas],
  );

  const handleCanvasMouseUp = useCallback(() => {
    setDragging(null);
    setPanning(null);
    if (canvas.connecting) {
      canvas.setConnecting(null);
      setConnectMouse(null);
    }
  }, [canvas]);

  const handleWheel = useCallback(
    (e: React.WheelEvent) => {
      if (e.ctrlKey || e.metaKey) {
        e.preventDefault();
        const delta = e.deltaY > 0 ? -0.1 : 0.1;
        canvas.setZoom(canvas.zoom + delta);
      } else {
        canvas.setPan({
          x: canvas.pan.x - e.deltaX,
          y: canvas.pan.y - e.deltaY,
        });
      }
    },
    [canvas],
  );

  const handleDrop = useCallback(
    (e: React.DragEvent) => {
      e.preventDefault();
      const stepType = e.dataTransfer.getData('step-type') as WorkflowStepType;
      if (!stepType) return;
      const rect = containerRef.current?.getBoundingClientRect();
      if (!rect) return;
      const x = (e.clientX - rect.left - canvas.pan.x) / canvas.zoom;
      const y = (e.clientY - rect.top - canvas.pan.y) / canvas.zoom;
      canvas.addStep(stepType, { x: Math.round(x), y: Math.round(y) });
    },
    [canvas],
  );

  const handleAddFromPalette = useCallback(
    (type: WorkflowStepType) => {
      // Place new step at a reasonable position
      const maxX = canvas.steps.length > 0
        ? Math.max(...canvas.steps.map((s) => s.position.x))
        : 0;
      canvas.addStep(type, { x: maxX + 280, y: 200 });
    },
    [canvas],
  );

  const handleFitToScreen = useCallback(() => {
    const el = containerRef.current;
    if (!el) return;
    canvas.fitToScreen(el.clientWidth, el.clientHeight);
  }, [canvas]);

  // Get temp connection line coords in canvas space
  const getTempLineCoords = () => {
    if (!canvas.connecting || !connectMouse || !containerRef.current) return null;
    const fromStep = canvas.steps.find((s) => s.id === canvas.connecting!.fromStepId);
    if (!fromStep) return null;
    const rect = containerRef.current.getBoundingClientRect();
    return {
      fromX: fromStep.position.x + NODE_WIDTH / 2,
      fromY: fromStep.position.y + NODE_HEIGHT + 8,
      toX: (connectMouse.x - rect.left - canvas.pan.x) / canvas.zoom,
      toY: (connectMouse.y - rect.top - canvas.pan.y) / canvas.zoom,
    };
  };

  const tempLine = getTempLineCoords();

  return (
    <div className="flex h-full">
      {/* Left palette */}
      {!readOnly && <StepPalette onAddStep={handleAddFromPalette} />}

      {/* Main canvas area */}
      <div className="flex-1 flex flex-col min-w-0">
        <CanvasToolbar
          canUndo={canvas.canUndo}
          canRedo={canvas.canRedo}
          zoom={canvas.zoom}
          readOnly={readOnly}
          isSaving={isSaving}
          isPublishing={isPublishing}
          isDraft={definition.status === 'draft'}
          onUndo={canvas.undo}
          onRedo={canvas.redo}
          onZoomIn={() => canvas.setZoom(canvas.zoom + 0.1)}
          onZoomOut={() => canvas.setZoom(canvas.zoom - 0.1)}
          onFitToScreen={handleFitToScreen}
          onAutoLayout={canvas.autoLayout}
          onSave={() => onSave(canvas.steps)}
          onPublish={onPublish}
        />

        <div
          ref={containerRef}
          className="flex-1 relative overflow-hidden bg-[radial-gradient(#e5e7eb_1px,transparent_1px)] [background-size:20px_20px] cursor-grab active:cursor-grabbing"
          onMouseDown={handleCanvasMouseDown}
          onMouseMove={handleCanvasMouseMove}
          onMouseUp={handleCanvasMouseUp}
          onMouseLeave={handleCanvasMouseUp}
          onWheel={handleWheel}
          onDragOver={(e) => e.preventDefault()}
          onDrop={handleDrop}
          role="application"
          aria-label="Workflow canvas"
          tabIndex={-1}
        >
          <div
            style={{
              transform: `translate(${canvas.pan.x}px, ${canvas.pan.y}px) scale(${canvas.zoom})`,
              transformOrigin: '0 0',
            }}
            className="absolute inset-0"
          >
            {/* SVG connections layer */}
            <svg
              ref={svgRef}
              className="absolute inset-0 w-full h-full pointer-events-none"
              style={{ overflow: 'visible' }}
            >
              <defs>
                <marker
                  id="arrowhead"
                  markerWidth="10"
                  markerHeight="7"
                  refX="9"
                  refY="3.5"
                  orient="auto"
                >
                  <polygon
                    points="0 0, 10 3.5, 0 7"
                    fill="#94a3b8"
                  />
                </marker>
              </defs>
              {canvas.steps.flatMap((step) =>
                step.transitions.map((transition) => {
                  const targetStep = canvas.steps.find(
                    (s) => s.id === transition.target_step_id,
                  );
                  if (!targetStep) return null;
                  return (
                    <ConnectionLine
                      key={transition.id}
                      fromStep={step}
                      toStep={targetStep}
                      transition={transition}
                      selected={canvas.selectedTransitionId === transition.id}
                      onSelect={canvas.selectTransition}
                    />
                  );
                }),
              )}
              {tempLine && (
                <TempConnectionLine
                  fromX={tempLine.fromX}
                  fromY={tempLine.fromY}
                  toX={tempLine.toX}
                  toY={tempLine.toY}
                />
              )}
            </svg>

            {/* Step nodes layer */}
            {canvas.steps.map((step) => (
              <StepNode
                key={step.id}
                step={step}
                selected={canvas.selectedStepId === step.id}
                readOnly={readOnly}
                zoom={canvas.zoom}
                stepStatus={stepStatuses?.[step.id]}
                onSelect={canvas.selectStep}
                onDragStart={handleDragStart}
                onConnectStart={handleConnectStart}
                onConnectEnd={handleConnectEnd}
              />
            ))}
          </div>

          {/* Empty state */}
          {canvas.steps.length === 0 && (
            <div className="absolute inset-0 flex items-center justify-center pointer-events-none">
              <div className="text-center text-muted-foreground">
                <p className="text-sm font-medium">No steps yet</p>
                <p className="text-xs mt-1">
                  {readOnly
                    ? 'This workflow has no steps defined.'
                    : 'Drag steps from the palette or click to add.'}
                </p>
              </div>
            </div>
          )}
        </div>
      </div>

      {/* Right properties panel */}
      {selectedStep && (
        <PropertiesPanel
          step={selectedStep}
          onUpdate={(updates) => canvas.updateStep(selectedStep.id, updates)}
          onRemove={() => canvas.removeStep(selectedStep.id)}
          onClose={() => canvas.selectStep(null)}
          readOnly={readOnly}
        />
      )}
    </div>
  );
}
