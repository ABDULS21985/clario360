import { describe, expect, it } from 'vitest';
import { fireEvent, render, screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { useState } from 'react';
import { TransformList } from '@/app/(dashboard)/data/pipelines/_components/transform-builder/transform-list';
import type { PipelineTransformDraft } from '@/app/(dashboard)/data/pipelines/_components/pipeline-wizard-types';

function TransformListHarness({
  initialTransforms = [],
}: {
  initialTransforms?: PipelineTransformDraft[];
}) {
  const [transforms, setTransforms] = useState<PipelineTransformDraft[]>(initialTransforms);
  return (
    <TransformList
      transforms={transforms}
      availableColumns={['old', 'new', 'email', 'updated_at']}
      previewBeforeRows={[]}
      previewAfterRows={[]}
      previewError={null}
      onChange={setTransforms}
      onPreview={() => undefined}
    />
  );
}

describe('TransformList', () => {
  it('test_addTransform: adds a rename transform card', async () => {
    const user = userEvent.setup();
    render(<TransformListHarness />);

    await user.click(screen.getByRole('button', { name: /add transformation/i }));
    await user.click(screen.getByText('Rename'));

    expect(screen.getByText('Rename column')).toBeInTheDocument();
  });

  it('test_removeTransform: removes a transform card', async () => {
    const user = userEvent.setup();
    render(
      <TransformListHarness
        initialTransforms={[
          { id: 't-1', type: 'rename', config: { from: 'old', to: 'new' } },
        ]}
      />,
    );

    await user.click(screen.getByRole('button', { name: /remove transform/i }));

    expect(screen.queryByText("Rename 'old' → 'new'")).not.toBeInTheDocument();
  });

  it('test_reorderTransforms: reorders transforms by drag and drop', () => {
    render(
      <TransformListHarness
        initialTransforms={[
          { id: 't-1', type: 'rename', config: { from: 'old', to: 'new_1' } },
          { id: 't-2', type: 'rename', config: { from: 'email', to: 'email_new' } },
          { id: 't-3', type: 'rename', config: { from: 'updated_at', to: 'updated_time' } },
        ]}
      />,
    );

    const thirdHandle = screen.getByRole('button', { name: /drag transform 3/i });
    const firstCard = screen.getByText("Rename 'old' → 'new_1'").closest('div.rounded-xl.border.bg-card');

    expect(firstCard).not.toBeNull();
    fireEvent.dragStart(thirdHandle);
    fireEvent.dragOver(firstCard!);
    fireEvent.drop(firstCard!);

    const cards = screen.getAllByText(/Rename '/i);
    expect(cards[0]).toHaveTextContent("Rename 'updated_at' → 'updated_time'");
  });

  it("test_transformSummary: shows the rename summary text", () => {
    render(
      <TransformListHarness
        initialTransforms={[
          { id: 't-1', type: 'rename', config: { from: 'old', to: 'new' } },
        ]}
      />,
    );

    expect(screen.getByText("Rename 'old' → 'new'")).toBeInTheDocument();
  });
});

