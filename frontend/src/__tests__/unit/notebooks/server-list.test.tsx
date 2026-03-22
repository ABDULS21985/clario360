import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { ServerList } from '@/app/(dashboard)/notebooks/_components/server-list';
import type { NotebookServer } from '@/lib/notebooks';

const runningServer: NotebookServer = {
  id: 'default',
  profile: 'soc-analyst',
  status: 'running',
  url: 'https://notebooks.example.com/user/analyst/lab',
  started_at: new Date(Date.now() - 3600_000).toISOString(),
  last_activity: new Date(Date.now() - 300_000).toISOString(),
  cpu_percent: 25.0,
  memory_mb: 1024,
  memory_limit_mb: 4096,
};

const startingServer: NotebookServer = {
  ...runningServer,
  id: 'default',
  status: 'starting',
  started_at: undefined,
  last_activity: undefined,
};

describe('ServerList', () => {
  it('renders empty state when no servers', () => {
    render(<ServerList servers={[]} busyServerId={null} onStop={() => {}} />);
    expect(screen.getByText(/no active notebook server/i)).toBeInTheDocument();
  });

  it('renders a running server card with correct profile label', () => {
    render(<ServerList servers={[runningServer]} busyServerId={null} onStop={() => {}} />);
    expect(screen.getByText('Soc Analyst')).toBeInTheDocument();
    expect(screen.getByText('running')).toBeInTheDocument();
  });

  it('renders Open JupyterLab link with server URL', () => {
    render(<ServerList servers={[runningServer]} busyServerId={null} onStop={() => {}} />);
    const link = screen.getByRole('link', { name: /open jupyterlab/i });
    expect(link).toHaveAttribute('href', runningServer.url);
    expect(link).toHaveAttribute('target', '_blank');
  });

  it('renders Stop Server button', () => {
    render(<ServerList servers={[runningServer]} busyServerId={null} onStop={() => {}} />);
    expect(screen.getByRole('button', { name: /stop server/i })).toBeInTheDocument();
  });

  it('calls onStop when Stop Server is clicked', async () => {
    const user = userEvent.setup();
    const onStop = vi.fn();
    render(<ServerList servers={[runningServer]} busyServerId={null} onStop={onStop} />);
    await user.click(screen.getByRole('button', { name: /stop server/i }));
    expect(onStop).toHaveBeenCalledWith(runningServer);
  });

  it('disables Stop button when busyServerId matches server id', () => {
    render(<ServerList servers={[runningServer]} busyServerId="default" onStop={() => {}} />);
    expect(screen.getByRole('button', { name: /stop server/i })).toBeDisabled();
  });

  it('does not disable Stop button for non-busy server', () => {
    render(<ServerList servers={[runningServer]} busyServerId="other-id" onStop={() => {}} />);
    expect(screen.getByRole('button', { name: /stop server/i })).not.toBeDisabled();
  });

  it('shows recently for missing started_at', () => {
    render(<ServerList servers={[startingServer]} busyServerId={null} onStop={() => {}} />);
    expect(screen.getByText(/recently/i)).toBeInTheDocument();
  });

  it('shows starting badge', () => {
    render(<ServerList servers={[startingServer]} busyServerId={null} onStop={() => {}} />);
    expect(screen.getByText('starting')).toBeInTheDocument();
  });

  it('renders CPU and memory usage bars', () => {
    render(<ServerList servers={[runningServer]} busyServerId={null} onStop={() => {}} />);
    expect(screen.getByText('CPU')).toBeInTheDocument();
    expect(screen.getByText('Memory')).toBeInTheDocument();
    expect(screen.getByText('1024 / 4096 MB')).toBeInTheDocument();
  });
});
