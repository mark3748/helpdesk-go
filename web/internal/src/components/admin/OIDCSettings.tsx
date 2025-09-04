import { useEffect } from 'react';
import { Form, Input, Button, message, Space } from 'antd';
import { useSettings, useTestConnection, useSaveOIDCSettings } from '../../api';

export default function OIDCSettings() {
  const [form] = Form.useForm();
  const { data } = useSettings();
  const test = useTestConnection();
  const save = useSaveOIDCSettings();

  useEffect(() => {
    if ((data as any)?.oidc) {
      const oidc = (data as any).oidc as any;
      form.setFieldsValue({
        ...oidc,
        value_to_roles: oidc.value_to_roles
          ? Object.entries(oidc.value_to_roles).map(([value, roles]: any) => ({
              value,
              roles: (roles as string[]).join(','),
            }))
          : [],
      });
    }
  }, [data, form]);

  const onFinish = (values: any) => {
    const mappings: Record<string, string[]> = {};
    (values.value_to_roles || []).forEach((m: any) => {
      mappings[m.value] = m.roles
        .split(',')
        .map((r: string) => r.trim())
        .filter(Boolean);
    });
    const payload = { ...values, value_to_roles: mappings };
    save.mutate(payload, {
      onSuccess: () => message.success('OIDC settings saved'),
      onError: () => message.error('Failed to save'),
    });
  };

  return (
    <>
      <p>Log directory: {(data as any)?.log_path}</p>
      <p>Last test: {(data as any)?.last_test || 'never'}</p>
      <Button
        onClick={() =>
          test.mutate(undefined, {
            onSuccess: () => message.success('Test complete'),
            onError: () => message.error('Test failed'),
          })
        }
        loading={test.isPending}
        style={{ marginBottom: 16 }}
      >
        Test Connection
      </Button>
      <Form form={form} layout="vertical" onFinish={onFinish}>
        <Form.Item label="Issuer" name="issuer">
          <Input />
        </Form.Item>
        <Form.Item label="Client ID" name="client_id">
          <Input />
        </Form.Item>
        <Form.Item label="Role Claim Path" name="claim_path">
          <Input />
        </Form.Item>
        <Form.List name="value_to_roles">
          {(fields, { add, remove }) => (
            <>
              {fields.map(({ key, name, ...restField }) => (
                <Space key={key} align="baseline" style={{ marginBottom: 8 }}>
                  <Form.Item
                    {...restField}
                    name={[name, 'value']}
                    rules={[{ required: true }]}
                  >
                    <Input placeholder="Claim Value" />
                  </Form.Item>
                  <Form.Item
                    {...restField}
                    name={[name, 'roles']}
                    rules={[{ required: true }]}
                  >
                    <Input placeholder="Roles (comma separated)" />
                  </Form.Item>
                  <Button onClick={() => remove(name)}>Remove</Button>
                </Space>
              ))}
              <Form.Item>
                <Button type="dashed" onClick={() => add()}>
                  Add Mapping
                </Button>
              </Form.Item>
            </>
          )}
        </Form.List>
        <Form.Item>
          <Button type="primary" htmlType="submit" loading={save.isPending}>
            Save
          </Button>
        </Form.Item>
      </Form>
    </>
  );
}
