import '@testing-library/jest-dom';
import { cleanup } from '@testing-library/react';
import { afterEach, vi } from 'vitest';

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
    get: (key: string) => null,
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
    (args[0].includes('Warning:') || args[0].includes('Error: Not implemented'))
  ) {
    return;
  }
  originalError(...args);
};
