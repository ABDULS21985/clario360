import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { ProfileSelector } from '@/app/(dashboard)/notebooks/_components/profile-selector';
import type { NotebookProfile } from '@/lib/notebooks';

const profiles: NotebookProfile[] = [
  {
    slug: 'soc-analyst',
    display_name: 'SOC Analyst',
    description: 'Security analysis, incident investigation.',
    cpu: '2 CPU',
    memory: '4 GB',
    storage: '5 GiB',
    spark_enabled: false,
    default: true,
  },
  {
    slug: 'data-scientist',
    display_name: 'Data Scientist',
    description: 'Model development and evaluation.',
    cpu: '4 CPU',
    memory: '8 GB',
    storage: '20 GiB',
    spark_enabled: false,
    default: false,
  },
];

describe('ProfileSelector', () => {
  it('renders nothing when closed', () => {
    render(
      <ProfileSelector
        open={false}
        onOpenChange={() => {}}
        profiles={profiles}
        busy={false}
        onSelect={() => {}}
      />,
    );
    expect(screen.queryByText('Launch Notebook Workspace')).not.toBeInTheDocument();
  });

  it('renders dialog title when open', () => {
    render(
      <ProfileSelector
        open={true}
        onOpenChange={() => {}}
        profiles={profiles}
        busy={false}
        onSelect={() => {}}
      />,
    );
    expect(screen.getByText('Launch Notebook Workspace')).toBeInTheDocument();
  });

  it('renders all profile cards', () => {
    render(
      <ProfileSelector
        open={true}
        onOpenChange={() => {}}
        profiles={profiles}
        busy={false}
        onSelect={() => {}}
      />,
    );
    expect(screen.getByText('SOC Analyst')).toBeInTheDocument();
    expect(screen.getByText('Data Scientist')).toBeInTheDocument();
  });

  it('shows resource specs for each profile', () => {
    render(
      <ProfileSelector
        open={true}
        onOpenChange={() => {}}
        profiles={profiles}
        busy={false}
        onSelect={() => {}}
      />,
    );
    expect(screen.getByText('CPU: 2 CPU')).toBeInTheDocument();
    expect(screen.getByText('Memory: 4 GB')).toBeInTheDocument();
    expect(screen.getByText('Storage: 5 GiB')).toBeInTheDocument();
  });

  it('calls onSelect with correct profile on launch click', async () => {
    const user = userEvent.setup();
    const onSelect = vi.fn();
    render(
      <ProfileSelector
        open={true}
        onOpenChange={() => {}}
        profiles={[profiles[0]]}
        busy={false}
        onSelect={onSelect}
      />,
    );
    await user.click(screen.getByRole('button', { name: /launch soc analyst/i }));
    expect(onSelect).toHaveBeenCalledWith(profiles[0]);
  });

  it('disables all launch buttons when busy', () => {
    render(
      <ProfileSelector
        open={true}
        onOpenChange={() => {}}
        profiles={profiles}
        busy={true}
        onSelect={() => {}}
      />,
    );
    const launchButtons = screen.getAllByRole('button', { name: /launch/i });
    launchButtons.forEach((btn) => expect(btn).toBeDisabled());
  });
});
