import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MFACodeInput } from './mfa-code-input';

function setup(onComplete = vi.fn()) {
  const user = userEvent.setup();
  const utils = render(<MFACodeInput onComplete={onComplete} />);
  const inputs = screen.getAllByRole('textbox');
  return { user, inputs, onComplete, ...utils };
}

describe('MFACodeInput', () => {
  it('test_renders6Inputs: 6 input elements rendered', () => {
    setup();
    const inputs = screen.getAllByRole('textbox');
    expect(inputs).toHaveLength(6);
  });

  it('test_autoAdvance: type "1" in first input → focus moves to second', async () => {
    const { user, inputs } = setup();
    await user.click(inputs[0]);
    await user.keyboard('1');
    // After typing '1', focus should be on second input
    expect(document.activeElement).toBe(inputs[1]);
  });

  it('test_rejectsLetters: type "a" → input remains empty', async () => {
    const { user, inputs } = setup();
    await user.click(inputs[0]);
    await user.keyboard('a');
    expect(inputs[0]).toHaveValue('');
  });

  it('test_backspace: focus on input 3, backspace → input 2 focused', async () => {
    const { user, inputs } = setup();
    // Fill first 3 inputs
    await user.click(inputs[0]);
    await user.keyboard('123');
    // Now on input 3 (index 3), press backspace
    await user.click(inputs[3]);
    await user.keyboard('{Backspace}');
    // Since index 3 is empty after click, backspace moves to index 2
    expect(document.activeElement).toBe(inputs[2]);
  });

  it('test_paste6Digits: paste "123456" → all inputs filled, onComplete called', async () => {
    const onComplete = vi.fn();
    render(<MFACodeInput onComplete={onComplete} />);
    const firstInput = screen.getAllByRole('textbox')[0];

    // Simulate paste event
    fireEvent.paste(firstInput, {
      clipboardData: {
        getData: () => '123456',
      },
    });

    expect(onComplete).toHaveBeenCalledWith('123456');
  });
});
