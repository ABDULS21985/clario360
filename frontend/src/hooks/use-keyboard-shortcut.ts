import { useEffect, useCallback } from "react";

interface ShortcutOptions {
  meta?: boolean;
  ctrl?: boolean;
  shift?: boolean;
  enabled?: boolean;
}

export function useKeyboardShortcut(
  key: string,
  callback: () => void,
  options: ShortcutOptions = {}
) {
  const { meta = false, ctrl = false, shift = false, enabled = true } = options;

  const handleKeyDown = useCallback((e: KeyboardEvent) => {
    if (!enabled) return;
    const target = e.target as HTMLElement;
    const isInputFocused = ["INPUT", "TEXTAREA", "SELECT"].includes(target.tagName) || target.isContentEditable;
    if (isInputFocused) return;
    if (meta && !e.metaKey) return;
    if (ctrl && !e.ctrlKey) return;
    if (shift && !e.shiftKey) return;
    if (e.key.toLowerCase() !== key.toLowerCase()) return;
    e.preventDefault();
    callback();
  }, [key, callback, meta, ctrl, shift, enabled]);

  useEffect(() => {
    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [handleKeyDown]);
}
