import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { RemediationLifecycleBadge } from '@/app/(dashboard)/cyber/remediation/_components/remediation-lifecycle-badge';

describe('RemediationLifecycleBadge', () => {
  it('renders draft status', () => {
    render(<RemediationLifecycleBadge status="draft" />);
    expect(screen.getByText('Draft')).toBeInTheDocument();
  });

  it('renders pending_approval status', () => {
    render(<RemediationLifecycleBadge status="pending_approval" />);
    expect(screen.getByText('Pending Approval')).toBeInTheDocument();
  });

  it('renders approved status', () => {
    render(<RemediationLifecycleBadge status="approved" />);
    expect(screen.getByText('Approved')).toBeInTheDocument();
  });

  it('renders verified status with green color', () => {
    render(<RemediationLifecycleBadge status="verified" />);
    const badge = screen.getByText('Verified');
    expect(badge).toBeInTheDocument();
    expect(badge.className).toContain('green');
  });

  it('renders rollback_failed status with red color', () => {
    render(<RemediationLifecycleBadge status="rollback_failed" />);
    const badge = screen.getByText('Rollback Failed');
    expect(badge).toBeInTheDocument();
    expect(badge.className).toContain('red');
  });

  it('applies custom className', () => {
    render(<RemediationLifecycleBadge status="closed" className="test-class" />);
    const badge = screen.getByText('Closed');
    expect(badge.className).toContain('test-class');
  });

  it('renders executing status with animate-pulse', () => {
    render(<RemediationLifecycleBadge status="executing" />);
    const badge = screen.getByText('Executing…');
    expect(badge.className).toContain('animate-pulse');
  });
});
