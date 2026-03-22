import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { ResourceUsage } from '@/app/(dashboard)/notebooks/_components/resource-usage';

describe('ResourceUsage', () => {
  it('renders CPU and memory bars with correct labels', () => {
    render(<ResourceUsage cpuPercent={45} memoryMB={1024} memoryLimitMB={4096} />);
    expect(screen.getByText('CPU')).toBeInTheDocument();
    expect(screen.getByText('45%')).toBeInTheDocument();
    expect(screen.getByText('Memory')).toBeInTheDocument();
    expect(screen.getByText('1024 / 4096 MB')).toBeInTheDocument();
  });

  it('shows unknown memory limit as question mark', () => {
    render(<ResourceUsage cpuPercent={10} memoryMB={512} memoryLimitMB={0} />);
    expect(screen.getByText('512 / ? MB')).toBeInTheDocument();
  });

  it('rounds CPU percent to integer', () => {
    render(<ResourceUsage cpuPercent={33.7} memoryMB={100} memoryLimitMB={1000} />);
    expect(screen.getByText('34%')).toBeInTheDocument();
  });
});
