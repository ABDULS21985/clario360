'use client';

import { create } from 'zustand';

interface RealtimeTopicEvent {
  count: number;
  payload: unknown;
  timestamp: string;
}

interface RealtimeQueryEvent extends RealtimeTopicEvent {
  topic: string;
}

interface RealtimeState {
  subscriptions: Record<string, string[]>;
  queryEvents: Record<string, RealtimeQueryEvent>;
  topicEvents: Record<string, RealtimeTopicEvent>;
  register: (topic: string, queryKey: string) => void;
  unregister: (topic: string, queryKey: string) => void;
  getKeysForTopic: (topic: string) => string[];
  publish: (topic: string, payload: unknown, timestamp: string) => void;
}

export const useRealtimeStore = create<RealtimeState>()((set, get) => ({
  subscriptions: {},
  queryEvents: {},
  topicEvents: {},

  register: (topic, queryKey) => {
    set((state) => {
      const existing = state.subscriptions[topic] ?? [];
      if (existing.includes(queryKey)) {
        return state;
      }
      return {
        subscriptions: {
          ...state.subscriptions,
          [topic]: [...existing, queryKey],
        },
      };
    });
  },

  unregister: (topic, queryKey) => {
    set((state) => {
      const existing = state.subscriptions[topic] ?? [];
      const next = existing.filter((key) => key !== queryKey);

      // Clean up stale queryEvents entry if this queryKey has no remaining subscriptions
      const isKeyStillUsed = Object.entries(state.subscriptions).some(
        ([t, keys]) => t !== topic && keys.includes(queryKey),
      );
      const nextQueryEvents = isKeyStillUsed
        ? state.queryEvents
        : (() => {
            const copy = { ...state.queryEvents };
            delete copy[queryKey];
            return copy;
          })();

      // Remove empty topic subscription arrays
      const nextSubs = { ...state.subscriptions, [topic]: next };
      if (next.length === 0) delete nextSubs[topic];

      return {
        subscriptions: nextSubs,
        queryEvents: nextQueryEvents,
      };
    });
  },

  getKeysForTopic: (topic) => {
    return get().subscriptions[topic] ?? [];
  },

  publish: (topic, payload, timestamp) => {
    const keys = get().getKeysForTopic(topic);
    set((state) => {
      const topicEvent = state.topicEvents[topic];
      const nextTopicEvents = {
        ...state.topicEvents,
        [topic]: {
          count: (topicEvent?.count ?? 0) + 1,
          payload,
          timestamp,
        },
      };

      const nextQueryEvents = { ...state.queryEvents };
      for (const key of keys) {
        const event = nextQueryEvents[key];
        nextQueryEvents[key] = {
          count: (event?.count ?? 0) + 1,
          payload,
          timestamp,
          topic,
        };
      }

      return {
        topicEvents: nextTopicEvents,
        queryEvents: nextQueryEvents,
      };
    });
  },
}));
