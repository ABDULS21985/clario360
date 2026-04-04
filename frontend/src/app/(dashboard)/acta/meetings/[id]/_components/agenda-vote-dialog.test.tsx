import { describe, expect, it, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { AgendaVoteDialog } from './agenda-vote-dialog';
import type { ActaAgendaItem } from '@/types/suites';

function buildItem(): ActaAgendaItem {
  return {
    id: 'agenda-1',
    tenant_id: 'tenant-1',
    meeting_id: 'meeting-1',
    title: 'Approve policy',
    description: 'Policy approval',
    item_number: '3.1',
    presenter_name: 'Sarah Ahmed',
    duration_minutes: 15,
    order_index: 1,
    status: 'pending',
    notes: '',
    requires_vote: true,
    vote_type: 'majority',
    votes_for: 0,
    votes_against: 0,
    votes_abstained: 0,
    vote_result: null,
    vote_notes: '',
    attachment_ids: [],
    category: 'decision',
    confidential: false,
    created_at: '2026-03-08T10:00:00Z',
    updated_at: '2026-03-08T10:00:00Z',
  };
}

async function changeVoteType(user: ReturnType<typeof userEvent.setup>, label: string) {
  await user.click(screen.getByRole('combobox'));
  await user.click(screen.getByRole('option', { name: label }));
}

async function setNumberField(index: number, value: string) {
  const input = screen.getAllByRole('spinbutton')[index];
  await userEvent.clear(input);
  await userEvent.type(input, value);
}

describe('AgendaVoteDialog', () => {
  it('test_unanimousResult: 0 against + 0 abstained → "Approved" shown', async () => {
    const user = userEvent.setup();
    render(<AgendaVoteDialog open onOpenChange={() => {}} item={buildItem()} presentCount={10} onSubmit={() => {}} />);

    await changeVoteType(user, 'Unanimous');
    await setNumberField(0, '10');
    await setNumberField(1, '0');
    await setNumberField(2, '0');

    expect(screen.getByText(/Approved/)).toBeInTheDocument();
  });

  it('test_majorityApproved: 7 for, 3 against → "Approved" shown', async () => {
    render(<AgendaVoteDialog open onOpenChange={() => {}} item={buildItem()} presentCount={10} onSubmit={() => {}} />);

    await setNumberField(0, '7');
    await setNumberField(1, '3');

    expect(screen.getByText(/Approved/)).toBeInTheDocument();
  });

  it('test_majorityRejected: 4 for, 6 against → "Rejected" shown', async () => {
    render(<AgendaVoteDialog open onOpenChange={() => {}} item={buildItem()} presentCount={10} onSubmit={() => {}} />);

    await setNumberField(0, '4');
    await setNumberField(1, '6');

    expect(screen.getByText(/Rejected/)).toBeInTheDocument();
  });

  it('test_twoThirdsApproved: 8 for, 2 against → "Approved" shown', async () => {
    const user = userEvent.setup();
    render(<AgendaVoteDialog open onOpenChange={() => {}} item={buildItem()} presentCount={10} onSubmit={() => {}} />);

    await changeVoteType(user, 'Two-thirds Majority');
    await setNumberField(0, '8');
    await setNumberField(1, '2');

    expect(screen.getByText(/Approved/)).toBeInTheDocument();
  });

  it('test_twoThirdsFailed: 6 for, 4 against → "Rejected" shown', async () => {
    const user = userEvent.setup();
    render(<AgendaVoteDialog open onOpenChange={() => {}} item={buildItem()} presentCount={10} onSubmit={() => {}} />);

    await changeVoteType(user, 'Two-thirds Majority');
    await setNumberField(0, '6');
    await setNumberField(1, '4');

    expect(screen.getByText(/Rejected/)).toBeInTheDocument();
  });

  it('test_tied: 5 for, 5 against → "Tied" shown', async () => {
    render(<AgendaVoteDialog open onOpenChange={() => {}} item={buildItem()} presentCount={10} onSubmit={() => {}} />);

    await setNumberField(0, '5');
    await setNumberField(1, '5');

    expect(screen.getByText(/Tied/)).toBeInTheDocument();
  });

  it('test_totalExceedsPresent: 12 for (10 present) → validation error', async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn();
    render(<AgendaVoteDialog open onOpenChange={() => {}} item={buildItem()} presentCount={10} onSubmit={onSubmit} />);

    await setNumberField(0, '12');
    const notes = screen.getAllByRole('textbox').at(-1);
    if (!notes) {
      throw new Error('Notes field not found');
    }
    await user.type(notes, 'Motion carried with 12 votes.');
    await user.click(screen.getByRole('button', { name: 'Record vote' }));

    expect(await screen.findByText('Vote total cannot exceed 10 present attendees.')).toBeInTheDocument();
    expect(onSubmit).not.toHaveBeenCalled();
  });
});
