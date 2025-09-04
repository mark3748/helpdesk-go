import { Modal, Form, Input, Select, message } from 'antd';
import { useMutation } from '@tanstack/react-query';
import { createTicket, createRequester } from '../../shared/api';

interface Props {
  open: boolean;
  onClose(): void;
  onCreated(): void;
}

export default function CreateTicketModal({ open, onClose, onCreated }: Props) {
  const [form] = Form.useForm();
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
      if (!requesterId) {
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
        <Form.Item
          name="requester_id"
          label="Requester ID"
          rules={[
            ({ getFieldValue }) => ({
              validator(_, value) {
                if (value || (getFieldValue('requester_email') && getFieldValue('requester_name')))
                  return Promise.resolve();
                return Promise.reject(new Error('Provide requester ID or name and email'));
              },
            }),
          ]}
        >
          <Input />
        </Form.Item>
        <Form.Item name="requester_email" label="Requester Email">
          <Input />
        </Form.Item>
        <Form.Item name="requester_name" label="Requester Name">
          <Input />
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
