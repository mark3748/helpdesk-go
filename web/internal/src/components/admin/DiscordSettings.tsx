import { useEffect, useMemo } from 'react';
import { Alert, Button, Card, Form, Input, Space, Typography, message } from 'antd';
import { useSaveDiscordSettings, useSettings } from '../../api';

const { Paragraph, Text } = Typography;

export default function DiscordSettings() {
  const [form] = Form.useForm();
  const { data } = useSettings();
  const save = useSaveDiscordSettings();
  const discord = useMemo(
    () => (((data as any)?.discord ?? {}) as Record<string, string>),
    [data],
  );

  useEffect(() => {
    form.setFieldsValue(discord);
  }, [discord, form]);

  const onFinish = (values: Record<string, string>) => {
    save.mutate(values, {
      onSuccess: () => message.success('Discord settings saved. Restart the worker to apply them.'),
      onError: () => message.error('Failed to save Discord settings'),
    });
  };

  return (
    <Space direction="vertical" size="large" style={{ width: '100%' }}>
      <Alert
        type="warning"
        showIcon
        message="Restart the worker after saving"
        description="The Discord gateway connection and slash commands are initialized when the worker starts."
      />

      <Card title="Discord Bot Configuration">
        <Paragraph>
          Create a Discord application and bot, enable the Message Content intent, invite it to the
          server, and grant it permission to view the ticket channel, send messages, create public
          threads, and send messages in threads.
        </Paragraph>
        <Form form={form} layout="vertical" onFinish={onFinish}>
          <Form.Item
            label="Bot Token"
            name="bot_token"
            extra={discord.bot_token_configured === 'true' ? 'A bot token is configured.' : undefined}
          >
            <Input.Password placeholder="Leave blank to preserve the configured token" />
          </Form.Item>
          <Form.Item label="Server (Guild) ID" name="guild_id" rules={[{ required: true }]}>
            <Input placeholder="123456789012345678" />
          </Form.Item>
          <Form.Item label="Ticket Channel ID" name="channel_id" rules={[{ required: true }]}>
            <Input placeholder="123456789012345678" />
          </Form.Item>
          <Text type="secondary">
            Email account linking commands also require SMTP Host and From Address in Mail Settings.
          </Text>
          <div style={{ marginTop: 24 }}>
            <Button type="primary" htmlType="submit" loading={save.isPending}>
              Save Discord Settings
            </Button>
          </div>
        </Form>
      </Card>
    </Space>
  );
}
