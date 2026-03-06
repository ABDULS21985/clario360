'use client';

import { create } from 'zustand';

interface CommandPaletteState {
  open: boolean;
  query: string;
  setOpen: (open: boolean) => void;
  toggle: () => void;
  setQuery: (query: string) => void;
  close: () => void;
}

export const useCommandPaletteStore = create<CommandPaletteState>((set) => ({
  open: false,
  query: '',

  setOpen: (open) => set({ open, query: open ? '' : '' }),
  toggle: () => set((s) => ({ open: !s.open, query: '' })),
  setQuery: (query) => set({ query }),
  close: () => set({ open: false, query: '' }),
}));
