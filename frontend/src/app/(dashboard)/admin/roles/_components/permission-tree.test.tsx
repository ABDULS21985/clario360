import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { PermissionTree } from './permission-tree';

describe('PermissionTree', () => {
  it('test_rendersTree: renders top-level nodes', () => {
    const onChange = vi.fn();
    render(<PermissionTree value={[]} onChange={onChange} />);
    expect(screen.getByText('Cybersecurity')).toBeInTheDocument();
    expect(screen.getByText('Data Intelligence')).toBeInTheDocument();
    expect(screen.getByText('Full Access')).toBeInTheDocument();
  });

  it('test_searchFilters: search by "pipeline" filters tree', () => {
    const onChange = vi.fn();
    render(<PermissionTree value={[]} onChange={onChange} />);
    const searchInput = screen.getByPlaceholderText('Search permissions...');
    fireEvent.change(searchInput, { target: { value: 'pipeline' } });
    expect(screen.getByText('Pipelines')).toBeInTheDocument();
    // Cybersecurity should be hidden when searching for "pipeline"
    expect(screen.queryByText('Cybersecurity')).toBeNull();
  });

  it('test_selectFullAccess_showsWarning: selecting * shows warning', () => {
    const onChange = vi.fn();
    const { rerender } = render(<PermissionTree value={[]} onChange={onChange} />);
    // Call onChange and rerender with * selected
    rerender(<PermissionTree value={['*']} onChange={onChange} />);
    expect(
      screen.getByText(/Full Access grants unrestricted access/i)
    ).toBeInTheDocument();
  });

  it('test_selectCount: shows selected count', () => {
    const onChange = vi.fn();
    render(<PermissionTree value={['cyber:read', 'cyber:write']} onChange={onChange} />);
    expect(screen.getByText(/2 permissions selected/i)).toBeInTheDocument();
  });

  it('test_noSelection_showsNone: empty shows no permissions', () => {
    const onChange = vi.fn();
    render(<PermissionTree value={[]} onChange={onChange} />);
    expect(screen.getByText('No permissions selected')).toBeInTheDocument();
  });

  it('test_permissionSlugs: permission slugs displayed', () => {
    const onChange = vi.fn();
    render(<PermissionTree value={[]} onChange={onChange} />);
    expect(screen.getByText('cyber:*')).toBeInTheDocument();
  });
});
