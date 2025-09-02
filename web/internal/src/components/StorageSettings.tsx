import React from 'react';
import { Form, Input, Button, message } from 'antd';
import { useSettings } from '../api';
import { apiFetch } from '../../shared/api';

export default function StorageSettings() {
  const { data } = useSettings();

  const onFinish = async (values: any) => {
    await apiFetch('/settings/storage', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(values),
    });
    message.success('Storage settings saved');
  };

  return (
    <Form layout="vertical" onFinish={onFinish} initialValues={(data as any)?.storage}>
      <Form.Item label="Endpoint" name="endpoint">
        <Input />
      </Form.Item>
      <Form.Item label="Bucket" name="bucket">
        <Input />
      </Form.Item>
      <Form.Item>
        <Button type="primary" htmlType="submit">
          Save
        </Button>
      </Form.Item>
    </Form>
  );
}
