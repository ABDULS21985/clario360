import { fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

import { MessageInput } from './message-input';

describe('MessageInput', () => {
  it('renders the selected engine preference and sends on button click', () => {
    const onChange = vi.fn();
    const onPreferredEngineChange = vi.fn();
    const onSend = vi.fn();

    render(
      <MessageInput
        value="Show critical alerts"
        preferredEngine="llm"
        onChange={onChange}
        onPreferredEngineChange={onPreferredEngineChange}
        onSend={onSend}
      />,
    );

    expect(screen.getByText('Force LLM')).toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: /send/i }));
    expect(onSend).toHaveBeenCalledTimes(1);
  });
});
