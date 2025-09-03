import { Modal, Form, Input, Select, message } from 'antd';
import { useEffect, useMemo, useState } from 'react';
import { apiFetch } from '../../shared/api';
import { useMutation } from '@tanstack/react-query';
import { createTicket } from '../../shared/api';

interface Props {
  open: boolean;
  onClose(): void;
  onCreated(): void;
}

export default function CreateTicketModal({ open, onClose, onCreated }: Props) {
  const [form] = Form.useForm();
  const [userOpts, setUserOpts] = useState<{ value: string; label: string }[]>([]);
  const [fetching, setFetching] = useState(false);
  const doSearch = useMemo(() => {
    let t: number | undefined;
    const runner = async (q: string) => {
      if (!q) { setUserOpts([]); return; }
      setFetching(true);
      try {
        const users = await apiFetch<any[]>(`/users?q=${encodeURIComponent(q)}`);
        setUserOpts(users.map(u => ({ value: String(u.id), label: u.display_name || u.email || u.username || u.id })));
      } catch {
        setUserOpts([]);
      } finally { setFetching(false); }
    };
    const debounced = (q: string) => {
      if (t) window.clearTimeout(t);
      t = window.setTimeout(() => runner(q), 300) as unknown as number;
    };
    (debounced as any).cancel = () => { if (t) window.clearTimeout(t); };
    return debounced as ((q: string) => void) & { cancel?: () => void };
  }, []);
  useEffect(() => () => { (doSearch as any).cancel?.(); }, [doSearch]);
  const create = useMutation({
    mutationFn: (values: {
      title: string;
      description?: string;
      priority: number;
      requester_id: string;
    }) =>
      createTicket({
        title: values.title,
        description: values.description || '',
        requester_id: values.requester_id,
        priority: values.priority,
      }),
    onSuccess: () => {
      message.success('Ticket created');
      form.resetFields();
      onCreated();
    },
    onError: () => message.error('Failed to create ticket'),
  });

  return (
    <Modal
      title="New Ticket"
      open={open}
      onCancel={onClose}
      onOk={() => form.submit()}
      confirmLoading={create.isPending}
    >
      <Form
        form={form}
        layout="vertical"
        onFinish={(values) => create.mutate(values as any)}
      >
        <Form.Item name="title" label="Title" rules={[{ required: true }]}>
          <Input />
        </Form.Item>
        <Form.Item name="description" label="Description">
          <Input.TextArea rows={4} />
        </Form.Item>
        <Form.Item name="requester_id" label="Requester" rules={[{ required: true }]}>
          <Select
            showSearch
            filterOption={false}
            onSearch={doSearch}
            options={userOpts}
            loading={fetching}
            placeholder="Search users by name/email"
          />
        </Form.Item>
        <Form.Item
          name="priority"
          label="Priority"
          initialValue={2}
          rules={[{ required: true }]}
        >
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
  );
}
