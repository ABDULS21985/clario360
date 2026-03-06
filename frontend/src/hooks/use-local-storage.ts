import { useState, useCallback } from "react";

export function useLocalStorage<T>(key: string, initialValue: T): [T, (value: T | ((prev: T) => T)) => void, () => void] {
  const [storedValue, setStoredValue] = useState<T>(() => {
    if (typeof window === "undefined") return initialValue;
    try {
      const item = window.localStorage.getItem(key);
      return item ? (JSON.parse(item) as T) : initialValue;
    } catch {
      return initialValue;
    }
  });

  const setValue = useCallback((value: T | ((prev: T) => T)) => {
    setStoredValue((prev) => {
      const next = typeof value === "function" ? (value as (prev: T) => T)(prev) : value;
      try {
        if (typeof window !== "undefined") {
          window.localStorage.setItem(key, JSON.stringify(next));
        }
      } catch { /* ignore */ }
      return next;
    });
  }, [key]);

  const remove = useCallback(() => {
    try {
      if (typeof window !== "undefined") {
        window.localStorage.removeItem(key);
      }
      setStoredValue(initialValue);
    } catch { /* ignore */ }
  }, [key, initialValue]);

  return [storedValue, setValue, remove];
}
