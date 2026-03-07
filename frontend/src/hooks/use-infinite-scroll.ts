'use client';

import { useState, useCallback, useRef, useEffect } from 'react';
import type { PaginatedResponse } from '@/types/api';

interface UseInfiniteScrollOptions {
  initialPageSize?: number;
  maxPages?: number;
}

interface UseInfiniteScrollResult<T> {
  items: T[];
  isLoading: boolean;
  isLoadingMore: boolean;
  hasMore: boolean;
  limitReached: boolean;
  error: Error | null;
  loadMore: () => void;
  onLoadMore: () => void;
  mutate: () => Promise<void>;
  sentinelRef: (el: HTMLDivElement | null) => void;
  reset: () => void;
  updateItems: (updater: (items: T[]) => T[]) => void;
}

export function useInfiniteScroll<T>(
  fetchFn: (page: number) => Promise<PaginatedResponse<T>>,
  options: UseInfiniteScrollOptions = {},
): UseInfiniteScrollResult<T> {
  const { maxPages = 5 } = options;

  const [items, setItems] = useState<T[]>([]);
  const [page, setPage] = useState(1);
  const [isLoading, setIsLoading] = useState(true);
  const [isLoadingMore, setIsLoadingMore] = useState(false);
  const [hasMore, setHasMore] = useState(false);
  const [limitReached, setLimitReached] = useState(false);
  const [error, setError] = useState<Error | null>(null);
  const observerRef = useRef<IntersectionObserver | null>(null);
  const pageRef = useRef(1);
  const loadedPagesRef = useRef(1);
  const loadingRef = useRef(false);

  const loadPage = useCallback(
    async (pageNum: number, append: boolean) => {
      if (loadingRef.current) return;
      loadingRef.current = true;
      if (append) setIsLoadingMore(true);
      else setIsLoading(true);

      try {
        const resp = await fetchFn(pageNum);
        setItems((prev) => (append ? [...prev, ...resp.data] : resp.data));
        loadedPagesRef.current = pageNum;
        const hasNextPage = pageNum < resp.meta.total_pages && pageNum < maxPages;
        setHasMore(hasNextPage);
        setLimitReached(pageNum >= maxPages && pageNum < resp.meta.total_pages);
        pageRef.current = pageNum;
        setError(null);
      } catch (err) {
        setError(err instanceof Error ? err : new Error('Failed to load'));
      } finally {
        loadingRef.current = false;
        if (append) setIsLoadingMore(false);
        else setIsLoading(false);
      }
    },
    [fetchFn, maxPages],
  );

  // Initial load
  useEffect(() => {
    setItems([]);
    setPage(1);
    pageRef.current = 1;
    loadPage(1, false);
  }, [loadPage]);

  const onLoadMore = useCallback(() => {
    if (!hasMore || isLoadingMore || isLoading) return;
    const nextPage = pageRef.current + 1;
    setPage(nextPage);
    loadPage(nextPage, true);
  }, [hasMore, isLoadingMore, isLoading, loadPage]);

  // IntersectionObserver sentinel
  const sentinelRef = useCallback(
    (el: HTMLDivElement | null) => {
      if (observerRef.current) {
        observerRef.current.disconnect();
        observerRef.current = null;
      }
      if (!el) return;
      if (typeof IntersectionObserver === 'undefined') return;

      observerRef.current = new IntersectionObserver(
        (entries) => {
          if (entries[0]?.isIntersecting) {
            onLoadMore();
          }
        },
        { threshold: 0.1 },
      );
      observerRef.current.observe(el);
    },
    [onLoadMore],
  );

  const reset = useCallback(() => {
    setItems([]);
    setPage(1);
    pageRef.current = 1;
    loadedPagesRef.current = 1;
    loadPage(1, false);
  }, [loadPage]);

  const mutate = useCallback(async () => {
    const pagesToRefetch = Math.min(loadedPagesRef.current, maxPages);
    setIsLoading(true);
    try {
      const responses = await Promise.all(
        Array.from({ length: pagesToRefetch }, (_, index) => fetchFn(index + 1)),
      );
      setItems(responses.flatMap((response) => response.data));
      const lastResponse = responses[responses.length - 1];
      if (lastResponse) {
        setHasMore(pagesToRefetch < lastResponse.meta.total_pages && pagesToRefetch < maxPages);
        setLimitReached(pagesToRefetch >= maxPages && pagesToRefetch < lastResponse.meta.total_pages);
      }
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err : new Error('Failed to load'));
    } finally {
      setIsLoading(false);
    }
  }, [fetchFn, maxPages]);

  const updateItems = useCallback((updater: (items: T[]) => T[]) => {
    setItems((current) => updater(current));
  }, []);

  void page; // used to track current page externally if needed

  return {
    items,
    isLoading,
    isLoadingMore,
    hasMore,
    limitReached,
    error,
    loadMore: onLoadMore,
    onLoadMore,
    mutate,
    sentinelRef,
    reset,
    updateItems,
  };
}
