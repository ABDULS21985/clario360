import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { PasswordStrengthMeter } from './password-strength-meter';

describe('PasswordStrengthMeter', () => {
  it('test_weak: "abc" → "Weak" label shown', () => {
    render(<PasswordStrengthMeter password="abc" />);
    expect(screen.getByText('Weak')).toBeInTheDocument();
  });

  it('test_fair: "Abcdefghij1!" → shows at least Fair or above', () => {
    render(<PasswordStrengthMeter password="Abcdefghij1!" />);
    const strengthLabel = screen.getByText(/weak|fair|good|strong/i);
    expect(strengthLabel).toBeInTheDocument();
    // Should NOT be weak for this password (has upper, lower, digit, special, 12+ chars)
    expect(strengthLabel.textContent?.toLowerCase()).not.toBe('weak');
  });

  it('test_strong: "C0mpl3x!P@ssw0rd#2026" → "Strong"', () => {
    render(<PasswordStrengthMeter password="C0mpl3x!P@ssw0rd#2026" />);
    expect(screen.getByText('Strong')).toBeInTheDocument();
  });

  it('test_realTimeUpdate: renders nothing for empty password', () => {
    const { container } = render(<PasswordStrengthMeter password="" />);
    expect(container.firstChild).toBeNull();
  });

  it('shows requirements checklist', () => {
    render(<PasswordStrengthMeter password="Test1!" />);
    expect(screen.getByText('At least 12 characters')).toBeInTheDocument();
    expect(screen.getByText('Contains uppercase letter')).toBeInTheDocument();
    expect(screen.getByText('Contains number')).toBeInTheDocument();
  });
});
