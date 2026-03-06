import '@testing-library/jest-dom';
import { cleanup } from '@testing-library/react';
import { afterEach, vi } from 'vitest';

// Polyfill ResizeObserver for jsdom (required by Radix UI components)
global.ResizeObserver = class ResizeObserver {
  observe() {}
  unobserve() {}
  disconnect() {}
};

// Polyfill PointerEvent for Radix UI
if (!global.PointerEvent) {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  (global as any).PointerEvent = MouseEvent;
}

afterEach(() => {
  cleanup();
});

// Mock next/navigation
vi.mock('next/navigation', () => ({
  useRouter: () => ({
    push: vi.fn(),
    replace: vi.fn(),
    back: vi.fn(),
    prefetch: vi.fn(),
  }),
  usePathname: () => '/',
  useSearchParams: () => ({
    get: (_key: string) => null,
  }),
  redirect: vi.fn(),
}));

// Mock next/font/google
vi.mock('next/font/google', () => ({
  Inter: () => ({ className: 'mock-font' }),
}));

// Suppress console.error for expected React errors in tests
const originalError = console.error.bind(console);
console.error = (...args: unknown[]) => {
  if (
    typeof args[0] === 'string' &&
    (args[0].includes('Warning:') ||
      args[0].includes('Error: Not implemented') ||
      args[0].includes('Consider adding an error boundary'))
  ) {
    return;
  }
  originalError(...args);
};
