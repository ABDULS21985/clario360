import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it, vi } from 'vitest';
import { TaskDetailForm } from './task-detail-form';
import type { FormField } from '@/types/models';

const formSchema: FormField[] = [
  { name: 'approved', type: 'boolean', label: 'Approved', required: true },
  { name: 'title', type: 'text', label: 'Title', required: true, placeholder: 'Enter title' },
  {
    name: 'summary',
    type: 'textarea',
    label: 'Summary',
    required: false,
    placeholder: 'Add summary',
  },
  {
    name: 'decision',
    type: 'select',
    label: 'Decision',
    required: true,
    options: ['approve', 'reject'],
  },
  { name: 'due_date', type: 'date', label: 'Due Date', required: false },
];

describe('TaskDetailForm', () => {
  it('renders boolean, text, textarea, select, and date fields', () => {
    render(
      <TaskDetailForm
        formSchema={formSchema}
        onSubmit={vi.fn().mockResolvedValue(undefined)}
      />,
    );

    expect(screen.getByRole('radiogroup')).toBeInTheDocument();
    expect(screen.getByPlaceholderText('Enter title')).toBeInTheDocument();
    expect(screen.getByPlaceholderText('Add summary')).toBeInTheDocument();
    expect(screen.getByRole('combobox')).toBeInTheDocument();
    expect(screen.getByLabelText(/due date/i)).toBeInTheDocument();
  });

  it('blocks submit when required fields are empty', async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn().mockResolvedValue(undefined);

    render(<TaskDetailForm formSchema={formSchema} onSubmit={onSubmit} />);

    await user.click(screen.getByRole('button', { name: /submit/i }));

    await waitFor(() => {
      expect(onSubmit).not.toHaveBeenCalled();
    });
  });

  it('accepts optional fields left empty', async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn().mockResolvedValue(undefined);

    render(
      <TaskDetailForm
        formSchema={formSchema}
        initialValues={{ decision: 'approve' }}
        onSubmit={onSubmit}
      />,
    );

    await user.click(screen.getByLabelText(/yes/i));
    await user.type(screen.getByPlaceholderText('Enter title'), 'Task title');
    await user.click(screen.getByRole('button', { name: /submit/i }));

    await waitFor(() => {
      expect(onSubmit).toHaveBeenCalledTimes(1);
      expect(onSubmit.mock.calls[0]?.[0]).toEqual({
        approved: true,
        title: 'Task title',
        summary: undefined,
        decision: 'approve',
        due_date: undefined,
      });
    });
  });

  it('uses provided initial values', () => {
    render(
      <TaskDetailForm
        formSchema={formSchema}
        initialValues={{ title: 'Existing title' }}
        onSubmit={vi.fn().mockResolvedValue(undefined)}
      />,
    );

    expect(screen.getByDisplayValue('Existing title')).toBeInTheDocument();
  });

  it('disables inputs in read only mode', () => {
    render(
      <TaskDetailForm
        formSchema={formSchema}
        readOnly
        onSubmit={vi.fn().mockResolvedValue(undefined)}
      />,
    );

    expect(screen.getByPlaceholderText('Enter title')).toBeDisabled();
    expect(screen.getByPlaceholderText('Add summary')).toBeDisabled();
    expect(screen.getByRole('combobox')).toHaveAttribute('data-disabled');
  });
});
