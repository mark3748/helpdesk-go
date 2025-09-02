import React from 'react'
import { Form, Input, Button, message } from 'antd'

export default function StorageSettings() {
  const onFinish = async (values: any) => {
    await fetch('/settings/storage', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      credentials: 'include',
      body: JSON.stringify(values),
    })
    message.success('Storage settings coming soon')
  }

  return (
    <Form layout="vertical" onFinish={onFinish}>
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
  )
}
