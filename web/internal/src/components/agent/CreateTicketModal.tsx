import { Modal, Form, Input, Select, message } from 'antd';
import { useMutation } from '@tanstack/react-query';
import { createTicket } from '../../shared/api';
import { useMe } from '../../shared/auth';

interface Props {
  open: boolean;
  onClose(): void;
  onCreated(): void;
}

export default function CreateTicketModal({ open, onClose, onCreated }: Props) {
  const { data: me } = useMe();
  const [form] = Form.useForm();
  const create = useMutation({
    mutationFn: (values: { title: string; description?: string; priority: number }) => {
      if (!me) throw new Error('not authenticated');
      return createTicket({
        title: values.title,
        description: values.description || '',
        requester_id: String(me.id),
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
      <Form form={form} layout="vertical" onFinish={(values) => create.mutate(values as any)}>
        <Form.Item name="title" label="Title" rules={[{ required: true }]}> 
          <Input />
        </Form.Item>
        <Form.Item name="description" label="Description">
          <Input.TextArea rows={4} />
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
  );
}
