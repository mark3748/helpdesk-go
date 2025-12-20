import { useState, useRef } from 'react';
import { Modal, Form, Input, Select, App, Spin } from 'antd';
import { useMutation } from '@tanstack/react-query';
import { createTicket, createRequester, searchRequesters } from '../../shared/api';

interface Props {
  open: boolean;
  onClose(): void;
  onCreated(): void;
}

export default function CreateTicketModal({ open, onClose, onCreated }: Props) {
  const [form] = Form.useForm();
  const { message } = App.useApp();
  const [options, setOptions] = useState<{ label: string; value: string }[]>([]);
  const [fetching, setFetching] = useState(false);
  const [isManual, setIsManual] = useState(false);
  const searchTimeout = useRef<ReturnType<typeof setTimeout> | null>(null);

  const fetchUser = (value: string) => {
    console.log('fetchUser called with:', value);
    if (searchTimeout.current) {
      clearTimeout(searchTimeout.current);
    }
    searchTimeout.current = setTimeout(() => {
      console.log('Executing search for:', value);
      setFetching(true);
      if (!value) {
        setOptions([]);
        setFetching(false);
        return;
      }
      searchRequesters(value).then((newOptions) => {
        console.log('Search results:', newOptions);
        setOptions(newOptions.map((r: any) => ({
          label: r.display_name ? `${r.display_name} <${r.email || 'no email'}>` : r.email || r.id || 'Unknown',
          value: r.id || '',
        })));
        setFetching(false);
      }).catch((err: any) => {
        console.error('Search failed:', err);
        setFetching(false);
      });
    }, 500);
  };

  const create = useMutation({
    mutationFn: async (values: {
      title: string;
      description?: string;
      priority: number;
      requester_id?: string;
      requester_email?: string;
      requester_name?: string;
    }) => {
      let requesterId = values.requester_id;
      // If manual mode or no ID selected
      if (!requesterId || isManual) {
        if (!values.requester_email || !values.requester_name) {
          throw new Error('Name and email required for new requester');
        }
        const r = await createRequester({
          email: String(values.requester_email),
          display_name: String(values.requester_name),
        });
        requesterId = r.id;
      }
      return createTicket({
        title: values.title,
        description: values.description || '',
        requester_id: String(requesterId),
        priority: values.priority,
      });
    },
    onSuccess: () => {
      message.success('Ticket created');
      form.resetFields();
      setIsManual(false);
      onCreated();
    },
    onError: (err) => message.error(`Failed to create ticket: ${err.message}`),
  });

  return (
    <Modal
      title="New Ticket"
      open={open}
      onCancel={onClose}
      onOk={() => form.submit()}
      confirmLoading={create.isPending}
    >
      <Form form={form} layout="vertical" onFinish={(values) => create.mutate(values as any)}>
        <Form.Item name="title" label="Title" rules={[{ required: true }]}>
          <Input />
        </Form.Item>
        <Form.Item name="description" label="Description">
          <Input.TextArea rows={4} />
        </Form.Item>

        {!isManual ? (
          <Form.Item
            name="requester_id"
            label="Requester"
            extra={<a onClick={() => setIsManual(true)}>Can't find? Create new requester</a>}
          >
            <Select
              showSearch
              placeholder="Search requester..."
              filterOption={false}
              onSearch={fetchUser}
              notFoundContent={fetching ? <Spin size="small" /> : null}
              options={options}
              allowClear
              onSelect={() => {
                form.setFieldsValue({ requester_email: undefined, requester_name: undefined });
              }}
            />
          </Form.Item>
        ) : (
          <div style={{ border: '1px solid #eee', padding: 8, marginBottom: 16, borderRadius: 4 }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 8 }}>
              <strong>New Requester</strong>
              <a onClick={() => setIsManual(false)}>Back to search</a>
            </div>
            <Form.Item name="requester_email" label="Requester Email" rules={[{ required: true }]}>
              <Input />
            </Form.Item>
            <Form.Item name="requester_name" label="Requester Name" rules={[{ required: true }]}>
              <Input />
            </Form.Item>
          </div>
        )}

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
  );
}
