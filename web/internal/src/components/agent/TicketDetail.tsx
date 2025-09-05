import { useEffect, useState } from 'react';
import { useParams } from 'react-router-dom';
import { useQuery, useMutation } from '@tanstack/react-query';
import { Typography, Select, List, Form, Input, Button, Upload, message, Tag, Collapse } from 'antd';
import type { UploadProps } from 'antd';
import { useTicket, subscribeEvents, useRequester } from '../../api';
import { apiFetch } from '../../shared/api';
import type { AppEvent } from '../../api';
import {
  fetchComments,
  addComment,
  fetchAttachments,
  uploadAttachment,
  deleteAttachment,
  downloadAttachment,
  updateTicketStatus,
  updateRequester,
} from '../../shared/api';

export default function TicketDetail() {
  const { id = '' } = useParams();
  const [connected, setConnected] = useState(true);
  const { data: ticket, refetch: refetchTicket } = useTicket(id, {
    refetchInterval: connected ? false : 5000,
  });

  const comments = useQuery({
    queryKey: ['comments', id],
    queryFn: () => fetchComments(id),
    enabled: !!id,
    refetchInterval: connected ? false : 5000,
  });

  const attachments = useQuery({
    queryKey: ['attachments', id],
    queryFn: () => fetchAttachments(id),
    enabled: !!id,
    refetchInterval: connected ? false : 5000,
  });

  const features = useQuery({
    queryKey: ['features'],
    queryFn: async () => {
      try { return await apiFetch<{ attachments: boolean }>('/features'); } catch { return { attachments: false }; }
    },
  });

  const [pendingAtts, setPendingAtts] = useState<{ filename: string; bytes: number }[]>([]);

  useEffect(() => {
    const sub = subscribeEvents(setConnected);
    const off = sub.on('ticket_updated', (ev: AppEvent) => {
      if (String((ev.data as any)?.id) === id) {
        refetchTicket();
        comments.refetch();
        attachments.refetch();
      }
    });
    return () => {
      off();
      sub.close();
    };
  }, [id, refetchTicket, comments, attachments]);

  const updateStatus = useMutation({
    mutationFn: (status: string) => updateTicketStatus(id, status),
    onSuccess: () => refetchTicket(),
  });

  const addCommentMut = useMutation({
    mutationFn: (body: string) => addComment(id, body),
    onSuccess: () => comments.refetch(),
    onError: () => message.error('Failed to add comment'),
  });

  const requester = useRequester((ticket as any)?.requester_id || '');
  const updateReq = useMutation({
    mutationFn: (vals: { email?: string; display_name?: string }) =>
      updateRequester(String((ticket as any)?.requester_id), vals),
    onSuccess: () => requester.refetch(),
    onError: () => message.error('Failed to update requester'),
  });

  const uploadProps: UploadProps = {
    showUploadList: false,
    customRequest: async ({ file, onProgress, onSuccess, onError }) => {
      try {
        if (!features.data?.attachments) {
          throw new Error('Attachments disabled: storage not configured');
        }
        const f = file as File;
        setPendingAtts((p) => [...p, { filename: f.name, bytes: f.size }]);
        await uploadAttachment(id, f, {
          onProgress: (e) => onProgress?.({ percent: e.percent }),
        });
        onSuccess?.({});
        message.success('Attachment uploaded');
        attachments.refetch();
      } catch (err) {
        onError?.(err as Error);
      } finally {
        setPendingAtts((p) => p.filter((a) => a.filename !== (file as File).name));
      }
    },
  };

  if (!ticket) return null;

  return (
    <div>
      <Typography.Title level={4}>
        {(ticket as any).title || (ticket as any).number}{' '}
        {ticket.status && <Tag>{String(ticket.status)}</Tag>}
      </Typography.Title>
      {String((ticket as any).description || '').trim() && (
        <Typography.Paragraph style={{ whiteSpace: 'pre-wrap', marginTop: -8 }}>
          {String((ticket as any).description || '')}
        </Typography.Paragraph>
      )}
      <Select
        value={(ticket as any).status}
        style={{ width: 160, marginBottom: 16 }}
        onChange={(v) => updateStatus.mutate(v)}
        options={[
          { value: 'open', label: 'Open' },
          { value: 'pending', label: 'Pending' },
          { value: 'closed', label: 'Closed' },
        ]}
      />

      <Collapse style={{ marginBottom: 16 }}>
        <Collapse.Panel header="Requester" key="req">
          {requester.data && (
            <Form
              layout="vertical"
              initialValues={{
                display_name: requester.data.display_name,
                email: requester.data.email,
              }}
              onFinish={(vals) => updateReq.mutate(vals)}
            >
              <Form.Item name="display_name" label="Name">
                <Input />
              </Form.Item>
              <Form.Item name="email" label="Email">
                <Input />
              </Form.Item>
              <Button type="primary" htmlType="submit" loading={updateReq.isPending}>
                Save
              </Button>
            </Form>
          )}
        </Collapse.Panel>
      </Collapse>

      <Upload {...uploadProps} disabled={features.isLoading ? true : !features.data?.attachments}>
        <Button disabled={features.isLoading ? true : !features.data?.attachments}>
          {features.data?.attachments ? 'Upload Attachment' : 'Attachments disabled (no storage)'}
        </Button>
      </Upload>
      <List
        header="Attachments"
        dataSource={[
          ...pendingAtts.map((a) => ({ ...a, id: `pending-${a.filename}`, pending: true })),
          ...(attachments.data || []),
        ]}
        renderItem={(a: any) => (
          <List.Item
            key={String(a.id)}
            actions={
              a.pending
                ? undefined
                : [
                    <a key="dl" onClick={() => downloadAttachment(id, String(a.id))}>Download</a>,
                    <a
                      key="del"
                      onClick={async () => {
                        await deleteAttachment(id, String(a.id));
                        attachments.refetch();
                      }}
                    >
                      Delete
                    </a>,
                  ]
            }
          >
            {String(a.filename)}
            {a.pending
              ? ' (uploading...)'
              : ` (${((a.bytes || 0) / 1024).toFixed(1)} KB)`}
          </List.Item>
        )}
        style={{ marginTop: 16 }}
      />

      <List
        header="Comments"
        dataSource={comments.data || []}
        renderItem={(c: any) => (
          <List.Item key={String(c.id)}>
            {String(c.body_md || c.body || '')}
          </List.Item>
        )}
        style={{ marginTop: 16 }}
      />

      <Form
        layout="vertical"
        onFinish={(values: { body: string }) => addCommentMut.mutate(values.body)}
        style={{ marginTop: 16 }}
      >
        <Form.Item name="body" label="Add Comment" rules={[{ required: true }]}> 
          <Input.TextArea rows={3} />
        </Form.Item>
        <Button type="primary" htmlType="submit" loading={addCommentMut.isPending}>
          Add Comment
        </Button>
      </Form>
    </div>
  );
}
