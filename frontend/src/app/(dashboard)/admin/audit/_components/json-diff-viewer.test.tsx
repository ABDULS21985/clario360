import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { JsonDiffViewer } from './json-diff-viewer';

describe('JsonDiffViewer', () => {
  it('test_showsAddedFields: new field → shown as added (+ prefix)', () => {
    render(<JsonDiffViewer oldValue={{}} newValue={{ name: 'John' }} />);
    expect(screen.getByText('+')).toBeInTheDocument();
    expect(screen.getByText('name:')).toBeInTheDocument();
  });

  it('test_showsRemovedFields: removed field → shown with - prefix', () => {
    render(<JsonDiffViewer oldValue={{ name: 'John' }} newValue={{}} />);
    expect(screen.getByText('-')).toBeInTheDocument();
    expect(screen.getByText('name:')).toBeInTheDocument();
  });

  it('test_showsChangedFields: changed field → both old and new values visible', () => {
    render(
      <JsonDiffViewer oldValue={{ name: 'John' }} newValue={{ name: 'Jane' }} />
    );
    const row = screen.getByText('name:');
    expect(row).toBeInTheDocument();
    // The changed row should show ~ prefix
    expect(screen.getByText('~')).toBeInTheDocument();
  });

  it('test_showsUnchangedFields: same field → shown (space prefix)', () => {
    render(
      <JsonDiffViewer oldValue={{ age: 30 }} newValue={{ age: 30 }} />
    );
    expect(screen.getByText('age:')).toBeInTheDocument();
    const prefixes = screen.getAllByText(' ');
    expect(prefixes.length).toBeGreaterThan(0);
  });

  it('test_handlesNullOld: old=null → all new fields shown as added', () => {
    render(<JsonDiffViewer oldValue={null} newValue={{ id: '123', name: 'Alice' }} />);
    const plusCells = screen.getAllByText('+');
    expect(plusCells.length).toBe(2);
  });

  it('test_handlesNullBoth: both null → no data message', () => {
    render(<JsonDiffViewer oldValue={null} newValue={null} />);
    expect(screen.getByText('No change data available.')).toBeInTheDocument();
  });

  it('test_handlesUndefinedBoth: both undefined → no data message', () => {
    render(<JsonDiffViewer oldValue={undefined} newValue={undefined} />);
    expect(screen.getByText('No change data available.')).toBeInTheDocument();
  });
});
