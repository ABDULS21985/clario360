import { describe, expect, it } from 'vitest';
import { render, screen } from '@testing-library/react';
import { QuorumIndicator } from './quorum-indicator';
import type { ActaAttendee } from '@/types/suites';

function attendee(id: string, status: ActaAttendee['status']): ActaAttendee {
  return {
    id,
    tenant_id: 'tenant-1',
    meeting_id: 'meeting-1',
    user_id: `user-${id}`,
    user_name: `User ${id}`,
    user_email: `user${id}@example.com`,
    member_role: 'member',
    status,
    created_at: '2026-03-08T10:00:00Z',
    updated_at: '2026-03-08T10:00:00Z',
  };
}

describe('QuorumIndicator', () => {
  it('test_quorumMet: 7 present of 10, required 6 → green "Quorum Met"', () => {
    const attendance = [
      ...Array.from({ length: 7 }, (_, index) => attendee(String(index), 'present')),
      ...Array.from({ length: 3 }, (_, index) => attendee(`a${index}`, 'absent')),
    ];

    render(<QuorumIndicator attendance={attendance} quorumRequired={6} />);

    expect(screen.getByText('Quorum Met')).toBeInTheDocument();
    expect(screen.getByText('7 of 10 members counted for quorum')).toBeInTheDocument();
  });

  it('test_quorumNotMet: 4 present of 10, required 6 → red "Quorum Not Met"', () => {
    const attendance = [
      ...Array.from({ length: 4 }, (_, index) => attendee(String(index), 'present')),
      ...Array.from({ length: 6 }, (_, index) => attendee(`a${index}`, 'absent')),
    ];

    render(<QuorumIndicator attendance={attendance} quorumRequired={6} />);

    expect(screen.getByText('Quorum Not Met')).toBeInTheDocument();
    expect(screen.getByText('4 of 10 members counted for quorum')).toBeInTheDocument();
  });

  it('test_proxyCountedAsPresent: 5 present + 2 proxy → 7 counted', () => {
    const attendance = [
      ...Array.from({ length: 5 }, (_, index) => attendee(String(index), 'present')),
      attendee('proxy-1', 'proxy'),
      attendee('proxy-2', 'proxy'),
      attendee('absent-1', 'absent'),
      attendee('absent-2', 'absent'),
      attendee('absent-3', 'absent'),
    ];

    render(<QuorumIndicator attendance={attendance} quorumRequired={6} />);

    expect(screen.getByText('7 of 10 members counted for quorum')).toBeInTheDocument();
    expect(screen.getByText('2 proxy')).toBeInTheDocument();
  });

  it('test_progressBar: 70% attendance → bar at 70%', () => {
    const attendance = [
      ...Array.from({ length: 7 }, (_, index) => attendee(String(index), 'present')),
      ...Array.from({ length: 3 }, (_, index) => attendee(`a${index}`, 'absent')),
    ];

    const { container } = render(<QuorumIndicator attendance={attendance} quorumRequired={6} />);
    const indicator = container.querySelector('[data-state]');

    expect(indicator).toBeTruthy();
    expect(container.innerHTML).toContain('translateX(-30%)');
  });
});
