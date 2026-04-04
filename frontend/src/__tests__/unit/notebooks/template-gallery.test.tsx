import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { TemplateGallery } from '@/app/(dashboard)/notebooks/_components/template-gallery';
import type { NotebookServer, NotebookTemplate } from '@/lib/notebooks';

const templates: NotebookTemplate[] = [
  {
    id: '01_threat_detection_quickstart',
    title: 'Threat Detection Quickstart',
    description: 'Pull recent alerts and visualize trends.',
    difficulty: 'beginner',
    tags: ['security', 'alerts'],
    filename: '01_threat_detection_quickstart.ipynb',
  },
  {
    id: '08_spark_large_scale_analysis',
    title: 'Spark Large-Scale Analysis',
    description: 'Correlate 30 days of telemetry.',
    difficulty: 'advanced',
    tags: ['spark', 'big-data'],
    filename: '08_spark_large_scale_analysis.ipynb',
  },
];

const runningServer: NotebookServer = {
  id: 'default',
  profile: 'soc-analyst',
  status: 'running',
  url: 'https://notebooks.example.com/user/analyst/lab',
  cpu_percent: 10,
  memory_mb: 512,
  memory_limit_mb: 4096,
};

describe('TemplateGallery', () => {
  it('renders all templates', () => {
    render(
      <TemplateGallery
        templates={templates}
        activeServer={null}
        busyTemplateId={null}
        onOpenTemplate={() => {}}
      />,
    );
    expect(screen.getByText('Threat Detection Quickstart')).toBeInTheDocument();
    expect(screen.getByText('Spark Large-Scale Analysis')).toBeInTheDocument();
  });

  it('shows difficulty badge', () => {
    render(
      <TemplateGallery
        templates={templates}
        activeServer={null}
        busyTemplateId={null}
        onOpenTemplate={() => {}}
      />,
    );
    expect(screen.getByText('beginner')).toBeInTheDocument();
    expect(screen.getByText('advanced')).toBeInTheDocument();
  });

  it('renders tag badges', () => {
    render(
      <TemplateGallery
        templates={templates}
        activeServer={null}
        busyTemplateId={null}
        onOpenTemplate={() => {}}
      />,
    );
    expect(screen.getByText('security')).toBeInTheDocument();
    expect(screen.getByText('spark')).toBeInTheDocument();
  });

  it('disables Open Template button when no active server', () => {
    render(
      <TemplateGallery
        templates={[templates[0]]}
        activeServer={null}
        busyTemplateId={null}
        onOpenTemplate={() => {}}
      />,
    );
    expect(screen.getByRole('button', { name: /launch a server first/i })).toBeDisabled();
  });

  it('enables Open Template button when server is active', () => {
    render(
      <TemplateGallery
        templates={[templates[0]]}
        activeServer={runningServer}
        busyTemplateId={null}
        onOpenTemplate={() => {}}
      />,
    );
    expect(screen.getByRole('button', { name: /open template/i })).not.toBeDisabled();
  });

  it('disables busy template button while copying', () => {
    render(
      <TemplateGallery
        templates={templates}
        activeServer={runningServer}
        busyTemplateId="01_threat_detection_quickstart"
        onOpenTemplate={() => {}}
      />,
    );
    const buttons = screen.getAllByRole('button', { name: /open template/i });
    // first template is busy
    expect(buttons[0]).toBeDisabled();
    // second template is not busy
    expect(buttons[1]).not.toBeDisabled();
  });

  it('calls onOpenTemplate with correct template when clicked', async () => {
    const user = userEvent.setup();
    const onOpenTemplate = vi.fn();
    render(
      <TemplateGallery
        templates={[templates[0]]}
        activeServer={runningServer}
        busyTemplateId={null}
        onOpenTemplate={onOpenTemplate}
      />,
    );
    await user.click(screen.getByRole('button', { name: /open template/i }));
    expect(onOpenTemplate).toHaveBeenCalledWith(templates[0]);
  });

  it('renders empty state gracefully', () => {
    const { container } = render(
      <TemplateGallery
        templates={[]}
        activeServer={null}
        busyTemplateId={null}
        onOpenTemplate={() => {}}
      />,
    );
    expect(container.querySelectorAll('[role="article"], .card')).toHaveLength(0);
  });
});
