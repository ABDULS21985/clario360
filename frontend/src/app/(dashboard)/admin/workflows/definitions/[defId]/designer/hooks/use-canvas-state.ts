'use client';

import { useCallback, useReducer, useRef, useEffect } from 'react';
import type { WorkflowStep, WorkflowTransition, WorkflowStepType, WorkflowStepConfig, AssigneeStrategy } from '@/types/models';

// ── State ──

interface CanvasState {
  steps: WorkflowStep[];
  selectedStepId: string | null;
  selectedTransitionId: string | null;
  pan: { x: number; y: number };
  zoom: number;
  connecting: { fromStepId: string; mouseX: number; mouseY: number } | null;
}

// ── Actions ──

type CanvasAction =
  | { type: 'SET_STEPS'; steps: WorkflowStep[] }
  | { type: 'ADD_STEP'; step: WorkflowStep }
  | { type: 'UPDATE_STEP'; stepId: string; updates: Partial<WorkflowStep> }
  | { type: 'REMOVE_STEP'; stepId: string }
  | { type: 'MOVE_STEP'; stepId: string; position: { x: number; y: number } }
  | { type: 'ADD_TRANSITION'; fromStepId: string; transition: WorkflowTransition }
  | { type: 'REMOVE_TRANSITION'; stepId: string; transitionId: string }
  | { type: 'SELECT_STEP'; stepId: string | null }
  | { type: 'SELECT_TRANSITION'; transitionId: string | null }
  | { type: 'SET_PAN'; pan: { x: number; y: number } }
  | { type: 'SET_ZOOM'; zoom: number }
  | { type: 'SET_CONNECTING'; connecting: CanvasState['connecting'] }
  | { type: 'UNDO' }
  | { type: 'REDO' };

function canvasReducer(state: CanvasState, action: CanvasAction): CanvasState {
  switch (action.type) {
    case 'SET_STEPS':
      return { ...state, steps: action.steps };
    case 'ADD_STEP':
      return { ...state, steps: [...state.steps, action.step] };
    case 'UPDATE_STEP':
      return {
        ...state,
        steps: state.steps.map((s) =>
          s.id === action.stepId ? { ...s, ...action.updates } : s,
        ),
      };
    case 'REMOVE_STEP': {
      const filtered = state.steps
        .filter((s) => s.id !== action.stepId)
        .map((s) => ({
          ...s,
          transitions: s.transitions.filter(
            (t) => t.target_step_id !== action.stepId,
          ),
        }));
      return {
        ...state,
        steps: filtered,
        selectedStepId:
          state.selectedStepId === action.stepId ? null : state.selectedStepId,
      };
    }
    case 'MOVE_STEP':
      return {
        ...state,
        steps: state.steps.map((s) =>
          s.id === action.stepId ? { ...s, position: action.position } : s,
        ),
      };
    case 'ADD_TRANSITION':
      return {
        ...state,
        steps: state.steps.map((s) =>
          s.id === action.fromStepId
            ? { ...s, transitions: [...s.transitions, action.transition] }
            : s,
        ),
      };
    case 'REMOVE_TRANSITION':
      return {
        ...state,
        steps: state.steps.map((s) =>
          s.id === action.stepId
            ? {
                ...s,
                transitions: s.transitions.filter(
                  (t) => t.id !== action.transitionId,
                ),
              }
            : s,
        ),
      };
    case 'SELECT_STEP':
      return {
        ...state,
        selectedStepId: action.stepId,
        selectedTransitionId: null,
      };
    case 'SELECT_TRANSITION':
      return {
        ...state,
        selectedTransitionId: action.transitionId,
        selectedStepId: null,
      };
    case 'SET_PAN':
      return { ...state, pan: action.pan };
    case 'SET_ZOOM':
      return { ...state, zoom: Math.max(0.25, Math.min(2, action.zoom)) };
    case 'SET_CONNECTING':
      return { ...state, connecting: action.connecting };
    default:
      return state;
  }
}

// ── Undo/Redo wrapper ──

interface UndoableState {
  current: CanvasState;
  past: CanvasState[];
  future: CanvasState[];
}

const STEP_MODIFYING_ACTIONS = new Set([
  'ADD_STEP',
  'UPDATE_STEP',
  'REMOVE_STEP',
  'MOVE_STEP',
  'ADD_TRANSITION',
  'REMOVE_TRANSITION',
  'SET_STEPS',
]);

function undoableReducer(
  state: UndoableState,
  action: CanvasAction,
): UndoableState {
  if (action.type === 'UNDO') {
    if (state.past.length === 0) return state;
    const previous = state.past[state.past.length - 1];
    return {
      past: state.past.slice(0, -1),
      current: previous,
      future: [state.current, ...state.future],
    };
  }

  if (action.type === 'REDO') {
    if (state.future.length === 0) return state;
    const next = state.future[0];
    return {
      past: [...state.past, state.current],
      current: next,
      future: state.future.slice(1),
    };
  }

  const newCurrent = canvasReducer(state.current, action);

  if (STEP_MODIFYING_ACTIONS.has(action.type)) {
    return {
      past: [...state.past.slice(-49), state.current],
      current: newCurrent,
      future: [],
    };
  }

  return { ...state, current: newCurrent };
}

// ── Hook ──

const initialState: CanvasState = {
  steps: [],
  selectedStepId: null,
  selectedTransitionId: null,
  pan: { x: 0, y: 0 },
  zoom: 1,
  connecting: null,
};

let stepCounter = 0;

function generateStepId(): string {
  stepCounter += 1;
  return `step_${Date.now()}_${stepCounter}`;
}

function generateTransitionId(): string {
  return `trans_${Date.now()}_${Math.random().toString(36).slice(2, 8)}`;
}

function defaultStepConfig(type: WorkflowStepType): WorkflowStepConfig {
  switch (type) {
    case 'approval':
      return { approval_type: 'single', min_approvers: 1 };
    case 'notification':
      return { notification_channels: ['in_app'] };
    case 'delay':
      return { delay_minutes: 60 };
    case 'webhook':
      return { webhook_method: 'POST' };
    default:
      return {};
  }
}

function defaultAssigneeStrategy(): AssigneeStrategy {
  return { type: 'role', role_id: '' };
}

export function useCanvasState(initialSteps?: WorkflowStep[]) {
  const [undoable, dispatch] = useReducer(undoableReducer, {
    past: [],
    current: { ...initialState, steps: initialSteps ?? [] },
    future: [],
  });

  const state = undoable.current;
  const canUndo = undoable.past.length > 0;
  const canRedo = undoable.future.length > 0;

  // Keyboard shortcuts
  const keydownRef = useRef<((e: KeyboardEvent) => void) | null>(null);

  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key === 'z') {
        e.preventDefault();
        if (e.shiftKey) {
          dispatch({ type: 'REDO' });
        } else {
          dispatch({ type: 'UNDO' });
        }
      }
      if (
        state.selectedStepId &&
        (e.key === 'Delete' || e.key === 'Backspace') &&
        !(e.target instanceof HTMLInputElement || e.target instanceof HTMLTextAreaElement)
      ) {
        dispatch({ type: 'REMOVE_STEP', stepId: state.selectedStepId });
      }
    };
    keydownRef.current = handler;
    window.addEventListener('keydown', handler);
    return () => window.removeEventListener('keydown', handler);
  }, [state.selectedStepId]);

  const setSteps = useCallback(
    (steps: WorkflowStep[]) => dispatch({ type: 'SET_STEPS', steps }),
    [],
  );

  const addStep = useCallback(
    (
      type: WorkflowStepType,
      position: { x: number; y: number },
      name?: string,
    ) => {
      const step: WorkflowStep = {
        id: generateStepId(),
        name: name ?? titleCaseType(type),
        type,
        config: defaultStepConfig(type),
        position,
        transitions: [],
        timeout_minutes: null,
        on_timeout: 'fail',
        assignee_strategy: defaultAssigneeStrategy(),
      };
      dispatch({ type: 'ADD_STEP', step });
      return step.id;
    },
    [],
  );

  const updateStep = useCallback(
    (stepId: string, updates: Partial<WorkflowStep>) =>
      dispatch({ type: 'UPDATE_STEP', stepId, updates }),
    [],
  );

  const removeStep = useCallback(
    (stepId: string) => dispatch({ type: 'REMOVE_STEP', stepId }),
    [],
  );

  const moveStep = useCallback(
    (stepId: string, position: { x: number; y: number }) =>
      dispatch({ type: 'MOVE_STEP', stepId, position }),
    [],
  );

  const addTransition = useCallback(
    (fromStepId: string, targetStepId: string, label?: string) => {
      const transition: WorkflowTransition = {
        id: generateTransitionId(),
        target_step_id: targetStepId,
        label: label ?? '',
      };
      dispatch({ type: 'ADD_TRANSITION', fromStepId, transition });
    },
    [],
  );

  const removeTransition = useCallback(
    (stepId: string, transitionId: string) =>
      dispatch({ type: 'REMOVE_TRANSITION', stepId, transitionId }),
    [],
  );

  const selectStep = useCallback(
    (stepId: string | null) => dispatch({ type: 'SELECT_STEP', stepId }),
    [],
  );

  const selectTransition = useCallback(
    (transitionId: string | null) =>
      dispatch({ type: 'SELECT_TRANSITION', transitionId }),
    [],
  );

  const setPan = useCallback(
    (pan: { x: number; y: number }) => dispatch({ type: 'SET_PAN', pan }),
    [],
  );

  const setZoom = useCallback(
    (zoom: number) => dispatch({ type: 'SET_ZOOM', zoom }),
    [],
  );

  const setConnecting = useCallback(
    (connecting: CanvasState['connecting']) =>
      dispatch({ type: 'SET_CONNECTING', connecting }),
    [],
  );

  const undo = useCallback(() => dispatch({ type: 'UNDO' }), []);
  const redo = useCallback(() => dispatch({ type: 'REDO' }), []);

  const fitToScreen = useCallback(
    (containerWidth: number, containerHeight: number) => {
      if (state.steps.length === 0) return;
      const xs = state.steps.map((s) => s.position.x);
      const ys = state.steps.map((s) => s.position.y);
      const minX = Math.min(...xs);
      const maxX = Math.max(...xs) + 200; // node width
      const minY = Math.min(...ys);
      const maxY = Math.max(...ys) + 80; // node height
      const graphW = maxX - minX + 100;
      const graphH = maxY - minY + 100;
      const z = Math.min(containerWidth / graphW, containerHeight / graphH, 1);
      const panX = (containerWidth - graphW * z) / 2 - minX * z + 50;
      const panY = (containerHeight - graphH * z) / 2 - minY * z + 50;
      setZoom(z);
      setPan({ x: panX, y: panY });
    },
    [state.steps, setPan, setZoom],
  );

  const autoLayout = useCallback(() => {
    if (state.steps.length === 0) return;
    const sorted = [...state.steps];
    // Simple topological-ish layout: BFS from nodes with no incoming edges
    const incoming = new Map<string, Set<string>>();
    for (const step of sorted) {
      if (!incoming.has(step.id)) incoming.set(step.id, new Set());
      for (const t of step.transitions) {
        if (!incoming.has(t.target_step_id))
          incoming.set(t.target_step_id, new Set());
        incoming.get(t.target_step_id)!.add(step.id);
      }
    }
    const roots = sorted.filter(
      (s) => (incoming.get(s.id)?.size ?? 0) === 0,
    );
    const visited = new Set<string>();
    const levels: string[][] = [];
    let queue = roots.map((r) => r.id);
    while (queue.length > 0) {
      const level: string[] = [];
      const next: string[] = [];
      for (const id of queue) {
        if (visited.has(id)) continue;
        visited.add(id);
        level.push(id);
        const step = sorted.find((s) => s.id === id);
        if (step) {
          for (const t of step.transitions) {
            if (!visited.has(t.target_step_id)) {
              next.push(t.target_step_id);
            }
          }
        }
      }
      if (level.length > 0) levels.push(level);
      queue = next;
    }
    // Position unvisited nodes
    for (const step of sorted) {
      if (!visited.has(step.id)) {
        levels.push([step.id]);
        visited.add(step.id);
      }
    }
    const xGap = 280;
    const yGap = 120;
    const updates: WorkflowStep[] = sorted.map((s) => ({ ...s }));
    for (let col = 0; col < levels.length; col++) {
      const level = levels[col];
      const totalH = (level.length - 1) * yGap;
      const startY = -totalH / 2;
      for (let row = 0; row < level.length; row++) {
        const step = updates.find((s) => s.id === level[row]);
        if (step) {
          step.position = { x: col * xGap + 50, y: startY + row * yGap + 200 };
        }
      }
    }
    dispatch({ type: 'SET_STEPS', steps: updates });
  }, [state.steps]);

  return {
    steps: state.steps,
    selectedStepId: state.selectedStepId,
    selectedTransitionId: state.selectedTransitionId,
    pan: state.pan,
    zoom: state.zoom,
    connecting: state.connecting,
    canUndo,
    canRedo,
    setSteps,
    addStep,
    updateStep,
    removeStep,
    moveStep,
    addTransition,
    removeTransition,
    selectStep,
    selectTransition,
    setPan,
    setZoom,
    setConnecting,
    undo,
    redo,
    fitToScreen,
    autoLayout,
  };
}

function titleCaseType(type: string): string {
  return type
    .split('_')
    .map((w) => w.charAt(0).toUpperCase() + w.slice(1))
    .join(' ');
}
