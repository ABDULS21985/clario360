import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { LaunchButton } from '@/app/(dashboard)/notebooks/_components/launch-button';

describe('LaunchButton', () => {
  it('renders with "Launch Notebook" label', () => {
    render(<LaunchButton onClick={() => {}} />);
    expect(screen.getByRole('button', { name: /launch notebook/i })).toBeInTheDocument();
  });

  it('calls onClick when clicked', async () => {
    const user = userEvent.setup();
    const onClick = vi.fn();
    render(<LaunchButton onClick={onClick} />);
    await user.click(screen.getByRole('button', { name: /launch notebook/i }));
    expect(onClick).toHaveBeenCalledOnce();
  });

  it('is disabled when disabled prop is true', () => {
    render(<LaunchButton disabled onClick={() => {}} />);
    expect(screen.getByRole('button', { name: /launch notebook/i })).toBeDisabled();
  });

  it('does not fire onClick when disabled', async () => {
    const user = userEvent.setup();
    const onClick = vi.fn();
    render(<LaunchButton disabled onClick={onClick} />);
    await user.click(screen.getByRole('button', { name: /launch notebook/i }));
    expect(onClick).not.toHaveBeenCalled();
  });
});
