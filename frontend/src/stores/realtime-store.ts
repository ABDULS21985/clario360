'use client';

import { create } from 'zustand';

interface RealtimeState {
  // topic → Set of react-query key prefixes (as JSON strings)
  subscriptions: Map<string, Set<string>>;
  register: (topic: string, queryKey: string) => void;
  unregister: (topic: string, queryKey: string) => void;
  getKeysForTopic: (topic: string) => string[];
}

export const useRealtimeStore = create<RealtimeState>()((set, get) => ({
  subscriptions: new Map(),

  register: (topic, queryKey) => {
    set((state) => {
      const updated = new Map(state.subscriptions);
      if (!updated.has(topic)) {
        updated.set(topic, new Set());
      }
      updated.get(topic)!.add(queryKey);
      return { subscriptions: updated };
    });
  },

  unregister: (topic, queryKey) => {
    set((state) => {
      const updated = new Map(state.subscriptions);
      updated.get(topic)?.delete(queryKey);
      return { subscriptions: updated };
    });
  },

  getKeysForTopic: (topic) => {
    return Array.from(get().subscriptions.get(topic) ?? []);
  },
}));
