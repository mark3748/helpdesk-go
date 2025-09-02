import { useEffect, useState } from 'react';
import TicketQueue from './TicketQueue';
import type { Ticket, Comment, Attachment } from '../api';
import { fetchComments, fetchAttachments, uploadAttachment, deleteAttachment, downloadAttachment, getMe, createTicket, addComment, logout } from '../api';
import {
  Layout,
  Button,
  Modal,
  Form,
  Input,
  Select,
  Typography,
  message,
  Upload,
  Progress,
} from 'antd';
import type { UploadProps } from 'antd';

export default function TicketWorkspace() {
  const [tabs, setTabs] = useState<Ticket[]>([]);
  const [activeId, setActiveId] = useState<string | null>(null);
  const [showNew, setShowNew] = useState(false);
  const [creating, setCreating] = useState(false);
  const [me, setMe] = useState<{ id: string } | null>(null);
  const [form] = Form.useForm();

  useEffect(() => {
    getMe().then((m) => setMe(m && m.id ? { id: m.id } : null)).catch(console.error);
  }, []);

  function openTicket(ticket: Ticket) {
    setTabs((prev) => {
      if (prev.find((t) => t.id === ticket.id)) return prev;
      return [...prev, ticket];
    });
    setActiveId(ticket.id);
  }

  function closeTicket(id: string) {
    setTabs((prev) => prev.filter((t) => t.id !== id));
    if (activeId === id) {
      const remaining = tabs.filter((t) => t.id !== id);
      setActiveId(remaining.length ? remaining[0].id : null);
    }
  }

    function newTicket() {
      setShowNew(true);
    }

    async function onCreateTicket(values: {
      title: string;
      description?: string;
      priority: number;
    }) {
      if (!me) {
        message.error('Authentication required. Please refresh the page and try again.');
        return;
      }
      setCreating(true);
      try {
        const res = await createTicket({
          title: values.title,
          description: values.description || '',
          priority: values.priority,
          requesterId: me.id,
        });
        if (res) {
          message.success(`Ticket ${res.number} created`);
          setShowNew(false);
          form.resetFields();
        } else {
          message.error('Failed to create ticket');
        }
      } catch {
        message.error('Failed to create ticket');
      } finally {
        setCreating(false);
      }
    }

  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Layout.Header style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', color: '#fff' }}>
        <Typography.Title level={4} style={{ color: '#fff', margin: 0 }}>Helpdesk</Typography.Title>
        <div style={{ display: 'flex', gap: 8 }}>
          <Button onClick={newTicket}>New Ticket</Button>
          <Button onClick={async () => { await logout(); window.location.reload(); }}>Logout</Button>
        </div>
      </Layout.Header>
      <Layout.Content style={{ padding: 16 }}>
        <TicketQueue onOpen={openTicket} onNewTicket={newTicket} />
        <div style={{ marginTop: 16 }}>
          {tabs.map((t) =>
            t.id === activeId ? (
              <TicketDetail key={t.id} ticket={t} currentUser={me} onClose={() => closeTicket(t.id)} />
            ) : null
          )}
        </div>
      </Layout.Content>

      <Modal
        title="New Ticket"
        open={showNew}
        onCancel={() => setShowNew(false)}
        onOk={() => form.submit()}
        confirmLoading={creating}
        okText={creating ? 'Creating…' : 'Create'}
      >
        <Form layout="vertical" form={form} onFinish={onCreateTicket}>
          <Form.Item name="title" label="Title" rules={[{ required: true, min: 3 }]}> 
            <Input placeholder="Issue title" />
          </Form.Item>
          <Form.Item name="description" label="Description"> 
            <Input.TextArea rows={4} placeholder="Describe the issue" />
          </Form.Item>
          <Form.Item name="priority" label="Priority" initialValue={2} rules={[{ required: true }]}> 
            <Select
              options={[
                { value: 1, label: '1 - Critical' },
                { value: 2, label: '2 - High' },
                { value: 3, label: '3 - Medium' },
                { value: 4, label: '4 - Low' },
              ]}
            />
          </Form.Item>
        </Form>
      </Modal>
    </Layout>
  );
}

function TicketDetail({ ticket, currentUser, onClose }: { ticket: Ticket; currentUser: { id: string } | null; onClose?: () => void }) {
  const [comments, setComments] = useState<Comment[]>([]);
  const [adding, setAdding] = useState(false);
  const [uploadPercent, setUploadPercent] = useState(0);
  const [attachments, setAttachments] = useState<Attachment[]>([]);
  const [form] = Form.useForm();
  useEffect(() => {
    fetchComments(ticket.id).then(setComments).catch(console.error);
    fetchAttachments(ticket.id).then(setAttachments).catch(console.error);
  }, [ticket.id]);

  const uploadProps: UploadProps = {
    showUploadList: false,
    customRequest: async ({ file, onProgress, onSuccess, onError }) => {
      try {
        await uploadAttachment(ticket.id, file as File, {
          onProgress: (evt) => {
            setUploadPercent(evt.percent);
            onProgress?.(evt);
          },
        });
        message.success('Uploaded');
        onSuccess?.({});
        // Refresh attachments after upload
        setAttachments(await fetchAttachments(ticket.id));
      } catch (err) {
        message.error('Upload failed');
        onError?.(err as Error);
      } finally {
        setUploadPercent(0);
      }
    },
  };

  return (
    <div style={{ border: '1px solid #eee', padding: 16, borderRadius: 8 }}>
      <div style={{ display: 'flex', justifyContent: 'space-between' }}>
        <Typography.Title level={5} style={{ margin: 0 }}>{ticket.subject}</Typography.Title>
        {onClose && <Button onClick={onClose}>Close</Button>}
      </div>
      <Upload {...uploadProps}>
        <Button>Upload Attachment</Button>
      </Upload>
      {uploadPercent > 0 && <Progress percent={uploadPercent} />}
      {attachments.length > 0 && (
        <div style={{ marginTop: 12 }}>
          <Typography.Text strong>Attachments</Typography.Text>
          <ul>
            {attachments.map((a) => (
              <li key={a.id} style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
                <span>{a.filename}</span>
                <span style={{ color: '#888' }}>({(a.bytes / 1024).toFixed(1)} KB)</span>
                <Button size="small" onClick={async () => {
                  try {
                    await downloadAttachment(ticket.id, a.id);
                  } catch {
                    message.error('Failed to download attachment');
                  }
                }}>Download</Button>
                <Button size="small" danger onClick={async () => {
                  const ok = await deleteAttachment(ticket.id, a.id);
                  if (ok) {
                    setAttachments(await fetchAttachments(ticket.id));
                  } else {
                    message.error('Failed to delete attachment');
                  }
                }}>Delete</Button>
              </li>
            ))}
          </ul>
        </div>
      )}
      <ul>
        {comments.map((c) => (
          <li key={c.id}>{c.body}</li>
        ))}
      </ul>
      <Form
        form={form}
        layout="vertical"
        onFinish={async (values: { body: string }) => {
          setAdding(true);
          // We need current user id; fetch from /me
          if (!currentUser) { message.error('Session expired. Please log in again.'); setAdding(false); return; }
          const ok = await addComment(ticket.id, currentUser.id, values.body || '');
          setAdding(false);
          if (ok) {
            form.resetFields();
            setComments(await fetchComments(ticket.id));
          } else {
            message.error('Failed to add comment');
          }
        }}
      >
        <Form.Item name="body" label="Add Comment" rules={[{ required: true }]}> 
          <Input.TextArea rows={3} placeholder="Type a comment" />
        </Form.Item>
        <Button type="primary" htmlType="submit" loading={adding}>Add Comment</Button>
      </Form>
    </div>
  );
}
