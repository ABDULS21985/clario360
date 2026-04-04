import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { describe, expect, it, vi, beforeEach } from 'vitest';

import { MessageDiagnostics } from './message-diagnostics';
import type { VCISOConversationMessage, VCISOLLMAuditResponse } from '@/types/cyber';

const apiGetMock = vi.fn();

vi.mock('@/lib/api', () => ({
  apiGet: (...args: unknown[]) => apiGetMock(...args),
}));

describe('MessageDiagnostics', () => {
  beforeEach(() => {
    apiGetMock.mockReset();
  });

  it('renders engine and routing metadata for assistant messages', () => {
    render(<MessageDiagnostics message={makeMessage()} />);

    expect(screen.getByText('LLM')).toBeInTheDocument();
    expect(screen.getByText('Grounding: passed')).toBeInTheDocument();
    expect(screen.getByText('420 tokens')).toBeInTheDocument();
    expect(screen.getByText('2 reasoning steps')).toBeInTheDocument();
    expect(screen.getByText(/Route: explicit llm preference/i)).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /view trace/i })).toBeInTheDocument();
  });

  it('loads and displays the audit trace on demand', async () => {
    const audit: VCISOLLMAuditResponse = {
      message_id: 'msg-1',
      provider: 'openai',
      model: 'gpt-4o',
      prompt_tokens: 200,
      completion_tokens: 220,
      total_tokens: 420,
      tool_calls: [
        {
          name: 'threat_forecast',
          arguments: { horizon_days: 7 },
          result_summary: 'Forecasted a moderate increase in alert volume.',
          success: true,
          latency_ms: 182,
          called_at: '2026-03-13T08:15:00Z',
        },
      ],
      reasoning_trace: [
        {
          step: 1,
          action: 'classify_request',
          detail: 'Detected a forecasting request with explicit executive framing.',
          tool_names: ['threat_forecast'],
        },
      ],
      grounding_result: 'passed',
      engine_used: 'llm',
      routing_reason: 'explicit_llm_preference',
      created_at: '2026-03-13T08:15:00Z',
    };

    apiGetMock.mockResolvedValue(audit);

    render(<MessageDiagnostics message={makeMessage()} />);
    fireEvent.click(screen.getByRole('button', { name: /view trace/i }));

    await waitFor(() => {
      expect(apiGetMock).toHaveBeenCalledWith('/api/v1/cyber/vciso/llm/audit/msg-1');
    });

    expect(await screen.findByText('Reasoning Trace')).toBeInTheDocument();
    expect(screen.getByText('openai')).toBeInTheDocument();
    expect(screen.getByText('gpt-4o')).toBeInTheDocument();
    expect(screen.getByText(/Detected a forecasting request/i)).toBeInTheDocument();
    expect(screen.getAllByText('threat_forecast').length).toBeGreaterThan(0);
  });
});

function makeMessage(): VCISOConversationMessage {
  return {
    id: 'msg-1',
    role: 'assistant',
    content: 'This is an LLM-routed response.',
    response_type: 'text',
    actions: [],
    created_at: '2026-03-13T08:15:00Z',
    engine: 'llm',
    meta: {
      engine: 'llm',
      grounding: 'passed',
      tokens_used: 420,
      routing_reason: 'explicit_llm_preference',
      reasoning_steps: 2,
    },
  };
}
