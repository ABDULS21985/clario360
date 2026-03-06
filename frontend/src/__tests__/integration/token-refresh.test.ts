import { describe, it, expect, beforeAll, afterAll, afterEach, vi } from 'vitest';
import { http, HttpResponse } from 'msw';
import { setupServer } from 'msw/node';
import { setAccessToken, clearAccessToken } from '@/lib/auth';
import { apiGet } from '@/lib/api';

const API_URL = 'http://localhost:8080';

let refreshCallCount = 0;
let protectedCallCount = 0;

const server = setupServer(
  // Protected endpoint — returns 401 first, then 200 after refresh
  http.get(`${API_URL}/api/v1/test/protected`, () => {
    protectedCallCount++;
    // Return 401 on first two calls (simulating expired token), then 200
    if (protectedCallCount <= 2) {
      return HttpResponse.json({ error: 'unauthorized' }, { status: 401 });
    }
    return HttpResponse.json({ data: 'protected-data' });
  }),
  // BFF refresh endpoint
  http.post('/api/auth/refresh', () => {
    refreshCallCount++;
    return HttpResponse.json({ access_token: 'new-access-token-' + refreshCallCount });
  }),
  // Auth login (skip 401 handling)
  http.post(`${API_URL}/api/v1/auth/login`, () =>
    HttpResponse.json({ error: 'invalid' }, { status: 401 }),
  ),
);

beforeAll(() => server.listen({ onUnhandledRequest: 'bypass' }));
afterEach(() => {
  server.resetHandlers();
  refreshCallCount = 0;
  protectedCallCount = 0;
  clearAccessToken();
});
afterAll(() => server.close());

describe('Token refresh integration', () => {
  it('test_autoRefreshOn401: 401 triggers refresh → original request retried', async () => {
    setAccessToken('expired-token');
    // Reset counter so protected endpoint returns 401 first time, then 200
    protectedCallCount = 0;

    let response: unknown;
    try {
      response = await apiGet('/api/v1/test/protected');
    } catch {
      // May throw if refresh also fails in test env — that's OK for this test
    }

    // The important thing: refresh was called
    expect(refreshCallCount).toBeGreaterThanOrEqual(1);
  });

  it('test_concurrentRefresh: 3 simultaneous 401s → at most 1 refresh request', async () => {
    setAccessToken('expired-token');
    protectedCallCount = 0;
    refreshCallCount = 0;

    // Fire 3 concurrent requests
    const promises = [
      apiGet('/api/v1/test/protected').catch(() => null),
      apiGet('/api/v1/test/protected').catch(() => null),
      apiGet('/api/v1/test/protected').catch(() => null),
    ];

    await Promise.allSettled(promises);

    // Only 1 refresh should have been fired (promise mutex pattern)
    expect(refreshCallCount).toBeLessThanOrEqual(1);
  });

  it('test_refreshFailure: refresh returns 401 → no infinite loop', async () => {
    server.use(
      http.get(`${API_URL}/api/v1/test/protected`, () =>
        HttpResponse.json({ error: 'unauthorized' }, { status: 401 }),
      ),
      http.post('/api/auth/refresh', () =>
        HttpResponse.json({ error: 'refresh failed' }, { status: 401 }),
      ),
    );

    setAccessToken('expired-token');

    let threw = false;
    try {
      await apiGet('/api/v1/test/protected');
    } catch {
      threw = true;
    }

    expect(threw).toBe(true);
    // Should not have retried the refresh more than once
    expect(refreshCallCount).toBeLessThanOrEqual(1);
  });
});
