import React from 'react'
import { Form, Input, Button, message } from 'antd'

export default function MailSettings() {
  const onFinish = async (values: any) => {
    await fetch('/settings/mail', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      credentials: 'include',
      body: JSON.stringify(values),
    })
    message.success('Email settings coming soon')
  }

  return (
    <Form layout="vertical" onFinish={onFinish}>
      <Form.Item label="SMTP Host" name="smtp_host">
        <Input />
      </Form.Item>
      <Form.Item label="IMAP Host" name="imap_host">
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
