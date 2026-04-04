import { afterEach, describe, expect, it, vi } from 'vitest';

const {
  apiClientDeleteMock,
  apiDeleteMock,
  apiGetMock,
  apiPostMock,
  apiPutMock,
  apiUploadMock,
} = vi.hoisted(() => ({
  apiClientDeleteMock: vi.fn(),
  apiDeleteMock: vi.fn(),
  apiGetMock: vi.fn(),
  apiPostMock: vi.fn(),
  apiPutMock: vi.fn(),
  apiUploadMock: vi.fn(),
}));

vi.mock('@/lib/api', () => ({
  default: {
    delete: apiClientDeleteMock,
  },
  apiDelete: apiDeleteMock,
  apiGet: apiGetMock,
  apiPost: apiPostMock,
  apiPut: apiPutMock,
  apiUpload: apiUploadMock,
}));

const { enterpriseApi } = await import('./api');

describe('enterpriseApi.acta payload normalization', () => {
  afterEach(() => {
    vi.clearAllMocks();
  });

  it('normalizes committee date fields before create', async () => {
    apiPostMock.mockResolvedValue({ data: { id: 'committee-1' } });

    await enterpriseApi.acta.createCommittee({
      name: 'Board Audit Committee',
      established_date: '2026-03-15',
      dissolution_date: '',
      charter: '',
      vice_chair_user_id: '',
      secretary_user_id: null,
      vice_chair_name: '',
      vice_chair_email: '',
      secretary_name: '',
      secretary_email: '',
    });

    expect(apiPostMock).toHaveBeenCalledWith(
      '/api/v1/acta/committees',
      expect.objectContaining({
        established_date: '2026-03-15T12:00:00.000Z',
        dissolution_date: null,
        charter: null,
        vice_chair_user_id: null,
        secretary_user_id: null,
        vice_chair_name: null,
        vice_chair_email: null,
        secretary_name: null,
        secretary_email: null,
      }),
    );
  });

  it('normalizes meeting schedule fields and optional empty strings', async () => {
    apiPostMock.mockResolvedValue({ data: { id: 'meeting-1' } });

    await enterpriseApi.acta.createMeeting({
      committee_id: 'committee-1',
      title: 'Quarterly Board Meeting',
      description: 'Quarterly review',
      scheduled_at: '2026-03-15T09:30',
      scheduled_end_at: '',
      location: '',
      virtual_link: '   ',
      virtual_platform: '',
    });

    expect(apiPostMock).toHaveBeenCalledWith(
      '/api/v1/acta/meetings',
      expect.objectContaining({
        scheduled_at: new Date('2026-03-15T09:30').toISOString(),
        scheduled_end_at: null,
        location: null,
        virtual_link: null,
        virtual_platform: null,
      }),
    );
  });

  it('normalizes action-item due dates for create and extend requests', async () => {
    apiPostMock.mockResolvedValue({ data: { id: 'action-1' } });

    await enterpriseApi.acta.createActionItem({
      meeting_id: 'meeting-1',
      committee_id: 'committee-1',
      title: 'Prepare board pack',
      description: 'Compile materials',
      assigned_to: 'user-1',
      assignee_name: 'Amina Analyst',
      due_date: '2026-03-21',
      agenda_item_id: '',
    });

    expect(apiPostMock).toHaveBeenNthCalledWith(
      1,
      '/api/v1/acta/action-items',
      expect.objectContaining({
        due_date: '2026-03-21T12:00:00.000Z',
        agenda_item_id: null,
      }),
    );

    await enterpriseApi.acta.extendActionItem('action-1', {
      new_due_date: '2026-03-28',
      reason: 'Waiting on external evidence',
    });

    expect(apiPostMock).toHaveBeenNthCalledWith(
      2,
      '/api/v1/acta/action-items/action-1/extend',
      expect.objectContaining({
        new_due_date: '2026-03-28T12:00:00.000Z',
      }),
    );
  });
});
