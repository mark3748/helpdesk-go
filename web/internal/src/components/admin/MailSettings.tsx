import { useEffect, useMemo, useState } from 'react';
import { Alert, Button, Card, Col, Form, Input, Row, Space, message } from 'antd';
import { useSettings, useSaveMailSettings, useSendTestEmail } from '../../api';

export default function MailSettings() {
  const [form] = Form.useForm();
  const [testRecipient, setTestRecipient] = useState('');
  const { data } = useSettings();
  const save = useSaveMailSettings();
  const sendTest = useSendTestEmail();
  const mail = useMemo(
    () => (((data as any)?.mail ?? {}) as Record<string, string>),
    [data],
  );

  useEffect(() => {
    if (mail) {
      form.setFieldsValue(mail);
      setTestRecipient((current) => current || mail.smtp_from || '');
    }
  }, [mail, form]);

  const onFinish = (values: any) => {
    save.mutate(values, {
      onSuccess: () => message.success('Email settings saved'),
      onError: () => message.error('Failed to save'),
    });
  };

  return (
    <Space direction="vertical" size="large" style={{ width: '100%' }}>
      <Alert
        type="info"
        showIcon
        message="Saved values override worker environment variables. Leave password fields blank to preserve the configured secret."
      />

      <Form form={form} layout="vertical" onFinish={onFinish}>
        <Card title="Outbound SMTP" style={{ marginBottom: 16 }}>
          <Row gutter={16}>
            <Col xs={24} md={16}>
              <Form.Item label="SMTP Host" name="smtp_host">
                <Input placeholder="smtp.example.com" />
              </Form.Item>
            </Col>
            <Col xs={24} md={8}>
              <Form.Item label="SMTP Port" name="smtp_port">
                <Input placeholder="587" />
              </Form.Item>
            </Col>
            <Col xs={24} md={12}>
              <Form.Item label="SMTP Username" name="smtp_user">
                <Input />
              </Form.Item>
            </Col>
            <Col xs={24} md={12}>
              <Form.Item
                label="SMTP Password"
                name="smtp_pass"
                extra={mail.smtp_pass_configured === 'true' ? 'A password is configured.' : undefined}
              >
                <Input.Password placeholder="Leave blank to preserve existing password" />
              </Form.Item>
            </Col>
            <Col xs={24}>
              <Form.Item label="From Address" name="smtp_from">
                <Input placeholder="helpdesk@example.com" />
              </Form.Item>
            </Col>
          </Row>
        </Card>

        <Card title="Inbound IMAP" style={{ marginBottom: 16 }}>
          <Row gutter={16}>
            <Col xs={24} md={16}>
              <Form.Item label="IMAP Host" name="imap_host">
                <Input placeholder="imap.example.com" />
              </Form.Item>
            </Col>
            <Col xs={24} md={8}>
              <Form.Item label="IMAP Port" name="imap_port">
                <Input placeholder="993" />
              </Form.Item>
            </Col>
            <Col xs={24} md={12}>
              <Form.Item label="IMAP Username" name="imap_user">
                <Input />
              </Form.Item>
            </Col>
            <Col xs={24} md={12}>
              <Form.Item
                label="IMAP Password"
                name="imap_pass"
                extra={mail.imap_pass_configured === 'true' ? 'A password is configured.' : undefined}
              >
                <Input.Password placeholder="Leave blank to preserve existing password" />
              </Form.Item>
            </Col>
            <Col xs={24}>
              <Form.Item label="IMAP Folder" name="imap_folder">
                <Input placeholder="INBOX" />
              </Form.Item>
            </Col>
          </Row>
        </Card>

        <Button type="primary" htmlType="submit" loading={save.isPending}>
          Save Mail Settings
        </Button>
      </Form>

      <Card title="Send Test Email">
        <Space.Compact style={{ width: '100%' }}>
          <Input
            value={testRecipient}
            onChange={(event) => setTestRecipient(event.target.value)}
            placeholder="recipient@example.com"
          />
          <Button
            onClick={() =>
              sendTest.mutate(testRecipient, {
                onSuccess: () => message.success('Test email queued'),
                onError: () => message.error('Failed to queue test email'),
              })
            }
            loading={sendTest.isPending}
            disabled={!testRecipient}
          >
            Send Test Email
          </Button>
        </Space.Compact>
        <p style={{ marginTop: 12, marginBottom: 0 }}>
          Last test queued: {(data as any)?.last_test || 'never'}
        </p>
      </Card>
    </Space>
  );
}
