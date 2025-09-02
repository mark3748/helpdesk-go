import React from 'react'
import { Form, Input, Button, message } from 'antd'

export default function OIDCSettings() {
  const onFinish = async (values: any) => {
    await fetch('/settings/oidc', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      credentials: 'include',
      body: JSON.stringify(values),
    })
    message.success('OIDC settings coming soon')
  }

  return (
    <Form layout="vertical" onFinish={onFinish}>
      <Form.Item label="Issuer" name="issuer">
        <Input />
      </Form.Item>
      <Form.Item label="Client ID" name="client_id">
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
