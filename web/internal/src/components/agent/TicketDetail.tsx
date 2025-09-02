import { useEffect, useState } from 'react';
import { useParams } from 'react-router-dom';
import { useQuery, useMutation } from '@tanstack/react-query';
import { Typography, Select, List, Form, Input, Button, Upload, message, Tag } from 'antd';
import type { UploadProps } from 'antd';
import { useTicket, subscribeEvents } from '../../api';
import type { AppEvent } from '../../api';
import {
  fetchComments,
  addComment,
  fetchAttachments,
  uploadAttachment,
  deleteAttachment,
  downloadAttachment,
  updateTicketStatus,
} from '../../../../shared/api';

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

  useEffect(() => {
    const stop = subscribeEvents((ev: AppEvent) => {
      if (ev.type === 'ticket_updated' && String((ev.data as any)?.id) === id) {
        refetchTicket();
        comments.refetch();
        attachments.refetch();
      }
    }, setConnected);
    return stop;
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

  const uploadProps: UploadProps = {
    showUploadList: false,
    customRequest: async ({ file, onProgress, onSuccess, onError }) => {
      try {
        await uploadAttachment(id, file as File, {
          onProgress: (e) => onProgress?.({ percent: e.percent }),
        });
        onSuccess?.({});
        attachments.refetch();
      } catch (err) {
        onError?.(err as Error);
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

      <Upload {...uploadProps}>
        <Button>Upload Attachment</Button>
      </Upload>
      <List
        header="Attachments"
        dataSource={attachments.data || []}
        renderItem={(a: any) => (
          <List.Item
            key={String(a.id)}
            actions={[
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
            ]}
          >
            {String(a.filename)} ({((a.bytes || 0) / 1024).toFixed(1)} KB)
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
